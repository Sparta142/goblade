package ffxiv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/klauspost/compress/zlib"
)

type EncodingType uint8
type CompressionType uint8

const (
	CompressionNone = CompressionType(0)
	CompressionZlib = CompressionType(1)
)

var (
	// Magic bytes indicating that a Bundle contains IPC segments.
	IpcMagicBytes = []byte("\x52\x52\xa0\x41\xff\x5d\x46\xe2\x7f\x2a\x64\x4d\x7b\x99\xc4\x75")

	// Magic bytes indicating that a Bundle contains keep-alive segments.
	KeepAliveMagicBytes = []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
)

var (
	ErrBadMagicBytes  = errors.New("ffxiv: bad magic bytes")
	ErrBadCompression = errors.New("ffxiv: bad compression type")
	ErrBadSegmentType = errors.New("ffxiv: bad segment type")
	ErrNotEnoughData  = errors.New("ffxiv: not enough data")
	ErrTooMuchData    = errors.New("ffxiv: too much data in bundle slice")
)

const (
	bundleLengthOffset = 24
	bundleLengthSize   = 2 // sizeof(uint16)
	bundleHeaderSize   = 40
)

var byteOrder = binary.LittleEndian

type Bundle struct {
	// Magic bytes header - all IPC bundles share the same magic bytes
	// and all keep-alive bundles have a magic string of all null bytes.
	Magic [16]byte `json:"-"`

	// The number of milliseconds since the Unix epoch time.
	Epoch uint64 `json:"epoch"`

	// The total length of the bundle, in bytes (including the header).
	Length uint16 `json:"length"`

	// The connection type. Usually 0.
	ConnectionType uint16 `json:"connectionType"`

	// The encoding type of the bundle payload.
	Encoding EncodingType `json:"-"`

	// The compression type of the bundle payload.
	Compression CompressionType `json:"-"`

	Segments []Segment `json:"segments"`
}

func (b *Bundle) UnmarshalBinary(data []byte) error {
	if len(data) < bundleHeaderSize {
		return ErrNotEnoughData
	}

	// Read and validate the Bundle header
	copy(b.Magic[:], data)

	if !bytes.Equal(b.Magic[:], IpcMagicBytes) && !bytes.Equal(b.Magic[:], KeepAliveMagicBytes) {
		return ErrBadMagicBytes
	}

	b.Epoch = byteOrder.Uint64(data[16:24])
	b.Length = byteOrder.Uint16(data[24:26])
	b.ConnectionType = byteOrder.Uint16(data[28:30])
	b.Encoding = EncodingType(data[32])
	b.Compression = CompressionType(data[33])

	// Sanity check
	if int(b.Length) != len(data) {
		return ErrNotEnoughData
	}

	// Read the Bundle payload
	var payloadData []byte

	switch b.Compression {
	case CompressionNone:
		payloadData = data[bundleHeaderSize:]

	case CompressionZlib:
		zr, err := zlib.NewReader(bytes.NewReader(data[bundleHeaderSize:]))
		if err != nil {
			return fmt.Errorf("create zlib reader: %w", err)
		}
		defer zr.Close()

		if payloadData, err = ioutil.ReadAll(zr); err != nil {
			return fmt.Errorf("read all from zlib reader: %w", err)
		}

	default:
		return ErrBadCompression
	}

	// Read all segments from the decompressed payload
	segmentCount := byteOrder.Uint16(data[30:32])
	b.Segments = make([]Segment, segmentCount)

	for i := 0; i < int(segmentCount); i++ { //nolint:varnamelen
		if err := b.Segments[i].UnmarshalBinary(payloadData); err != nil {
			return fmt.Errorf("read segment: %w", err)
		}

		// Advance payloadData by the size of the just-read Segment
		payloadData = payloadData[b.Segments[i].Length:]
	}

	// Sanity check: the entire payload should have been consumed
	if len(payloadData) != 0 {
		return ErrTooMuchData
	}

	return nil
}

func (b *Bundle) Time() time.Time {
	return time.UnixMilli(int64(b.Epoch)).UTC()
}

func (b *Bundle) IsCompressed() bool {
	return b.Compression != CompressionNone
}

func PeekBundleLength(data []byte) int {
	if len(data) < int(bundleLengthOffset+bundleLengthSize) {
		return -1
	}

	return int(byteOrder.Uint16(data[bundleLengthOffset:]))
}
