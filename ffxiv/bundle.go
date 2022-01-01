package ffxiv

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
	"unsafe"
)

type EncodingType uint8
type CompressionType uint8

const (
	CompressionNone = CompressionType(0)
	CompressionZlib = CompressionType(1)
)

const (
	// Magic string indicating that a Bundle contains IPC segments.
	IpcMagicString = "\x52\x52\xa0\x41\xff\x5d\x46\xe2\x7f\x2a\x64\x4d\x7b\x99\xc4\x75"

	// Magic string indicating that a Bundle contains keep-alive segments.
	KeepAliveMagicString = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

var (
	ErrBadMagicString = errors.New("ffxiv: bad magic string")
	ErrBadCompression = errors.New("ffxiv: bad compression type")
	ErrBadSegmentType = errors.New("ffxiv: bad segment type")
	ErrNotEnoughData  = errors.New("ffxiv: not enough data")
)

var (
	bundleLengthOffset = unsafe.Offsetof(bundleHeader{}.Length)
	bundleLengthSize   = unsafe.Sizeof(bundleHeader{}.Length)
	bundleHeaderSize   = unsafe.Sizeof(bundleHeader{})
)

var byteOrder = binary.LittleEndian

type bundleHeader struct {
	// Magic string header - all IPC bundles share the same magic string
	// and all keep-alive bundles have a magic string of all null bytes.
	Magic [16]byte

	// The number of milliseconds since the Unix epoch time.
	Epoch uint64

	// The total length of the bundle, in bytes (including this header).
	Length uint16

	_ [2]byte

	// The connection type. Usually 0.
	ConnectionType uint16

	// The number of segments in the bundle.
	SegmentCount uint16

	// The encoding type of the bundle payload.
	Encoding EncodingType

	// The compression type of the bundle payload.
	Compression CompressionType

	_ [6]byte
}

func (h *bundleHeader) UnmarshalBinary(data []byte) error {
	if len(data) < int(bundleHeaderSize) {
		return ErrNotEnoughData
	}

	copy(h.Magic[:], data)

	// Validate magic string in Bundle header
	if s := string(h.Magic[:]); s != IpcMagicString && s != KeepAliveMagicString {
		return ErrBadMagicString
	}

	h.Epoch = byteOrder.Uint64(data[16:24])
	h.Length = byteOrder.Uint16(data[24:26])
	h.ConnectionType = byteOrder.Uint16(data[28:30])
	h.SegmentCount = byteOrder.Uint16(data[30:32])
	h.Encoding = EncodingType(data[32])
	h.Compression = CompressionType(data[33])

	return nil
}

type Bundle struct {
	bundleHeader
	Segments []Segment
}

func (b *Bundle) UnmarshalBinary(data []byte) error {
	// Read the Bundle header
	if err := b.bundleHeader.UnmarshalBinary(data); err != nil {
		return err
	}

	// A reader for the decompressed payload, which is just a reader
	// for the original payload if it isn't compressed to begin with.
	var r io.Reader

	switch b.Compression {
	case CompressionNone:
		r = bytes.NewReader(data[bundleHeaderSize:])
	case CompressionZlib:
		zr, err := zlib.NewReader(bytes.NewReader(data[bundleHeaderSize:]))
		if err != nil {
			return err
		}
		defer zr.Close()
		r = zr
	default:
		return ErrBadCompression
	}

	// Read all segments from the decompressed payload
	b.Segments = make([]Segment, b.SegmentCount)

	for i := 0; i < len(b.Segments); i++ {
		if err := ReadSegment(r, &b.Segments[i]); err != nil {
			return fmt.Errorf("read segment: %w", err)
		}
	}

	return nil
}

func ReadBundleLength(data []byte) int {
	if len(data) < int(bundleLengthOffset+bundleLengthSize) {
		return -1
	}

	return int(byteOrder.Uint16(data[bundleLengthOffset:]))
}

func (b *Bundle) Time() time.Time {
	return time.UnixMilli(int64(b.Epoch)).UTC()
}

func (b *Bundle) IsCompressed() bool {
	return b.Compression != CompressionNone
}
