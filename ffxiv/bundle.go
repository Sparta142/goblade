package ffxiv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
	"unsafe"

	"github.com/klauspost/compress/zlib"
	"github.com/sparta142/goblade/oodle"
)

type EncodingType uint8

type CompressionType uint8

const (
	CompressionNone  = CompressionType(0)
	CompressionZlib  = CompressionType(1)
	CompressionOodle = CompressionType(2)
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
	bundleLengthSize   = unsafe.Sizeof(Bundle{}.Length)
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
	Length uint32 `json:"length"`

	UncompressedLength uint32 `json:"uncompressedLength"`

	// The connection type. Usually 0.
	ConnectionType uint16 `json:"connectionType"`

	// The encoding type of the bundle payload.
	Encoding EncodingType `json:"-"`

	// The compression type of the bundle payload.
	Compression CompressionType `json:"-"`

	Segments []Segment `json:"segments"`
}

func (b *Bundle) UnmarshalBinary(data []byte) error {
	if err := b.unmarshalHeader(data); err != nil {
		return err
	}

	if err := b.unmarshalPayload(data[bundleHeaderSize:]); err != nil {
		return err
	}

	return nil
}

func (b *Bundle) Time() time.Time {
	return time.UnixMilli(int64(b.Epoch)).UTC()
}

func (b *Bundle) IsCompressed() bool {
	return b.Compression != CompressionNone
}

func (b *Bundle) unmarshalHeader(data []byte) error {
	if len(data) < bundleHeaderSize {
		return ErrNotEnoughData
	}

	// Read and validate the Bundle header
	copy(b.Magic[:], data)

	if !bytes.Equal(b.Magic[:], IpcMagicBytes) && !bytes.Equal(b.Magic[:], KeepAliveMagicBytes) {
		return ErrBadMagicBytes
	}

	b.Epoch = byteOrder.Uint64(data[16:24])
	b.Length = byteOrder.Uint32(data[24:28])
	b.ConnectionType = byteOrder.Uint16(data[28:30])
	b.Encoding = EncodingType(data[32])
	b.Compression = CompressionType(data[33])
	b.Segments = make([]Segment, byteOrder.Uint16(data[30:32]))
	b.UncompressedLength = byteOrder.Uint32(data[36:40])

	// Sanity check
	if len(data) != int(b.Length) {
		return ErrNotEnoughData
	}

	return nil
}

func (b *Bundle) unmarshalPayload(data []byte) error {
	// Read the Bundle payload
	payloadData, err := decompressBytes(data, b.Compression, b.UncompressedLength)
	if err != nil {
		return err
	}

	// Read all segments from the decompressed payload
	for i := range b.Segments {
		segment := &b.Segments[i]

		if err := segment.UnmarshalBinary(payloadData); err != nil {
			return fmt.Errorf("read segment: %w", err)
		}

		// Advance payloadData by the size of the just-read Segment
		payloadData = payloadData[segment.Length:]
	}

	// Sanity check: the entire payload should have been consumed
	if len(payloadData) != 0 {
		return ErrTooMuchData
	}

	return nil
}

func decompressBytes(data []byte, compression CompressionType, rawLen uint32) ([]byte, error) {
	var decompressed []byte

	switch compression {
	case CompressionNone:
		return data, nil

	case CompressionZlib:
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("create zlib reader: %w", err)
		}
		defer reader.Close()

		if decompressed, err = ioutil.ReadAll(reader); err != nil {
			return nil, fmt.Errorf("read all from zlib reader: %w", err)
		}

	case CompressionOodle:
		var err error
		decompressed = make([]byte, rawLen)
		if err = oodle.Decode(data, decompressed); err != nil {
			return nil, fmt.Errorf("oodle decode: %w", err)
		}

	default:
		return nil, ErrBadCompression
	}

	return decompressed, nil
}

func PeekBundleLength(data []byte) int {
	if len(data) < int(bundleLengthOffset+bundleLengthSize) {
		return -1
	}

	return int(byteOrder.Uint16(data[bundleLengthOffset:]))
}
