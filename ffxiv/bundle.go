package ffxiv

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
	"unsafe"

	"github.com/klauspost/compress/zlib"
	log "github.com/sirupsen/logrus"
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
)

const (
	bundleLengthOffset = 24
	bundleLengthSize   = unsafe.Sizeof(Bundle{}.Length)
	bundleHeaderSize   = 40
)

var byteOrder = binary.LittleEndian

type Bundle struct {
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

	// Sanity check
	if len(data) < int(b.Length) {
		return ErrNotEnoughData
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

	_ = data[bundleHeaderSize-1]

	// Validate the magic bytes
	if !bytes.HasPrefix(data, IpcMagicBytes) && !bytes.HasPrefix(data, KeepAliveMagicBytes) {
		return ErrBadMagicBytes
	}

	// Read and validate the Bundle header
	b.Epoch = byteOrder.Uint64(data[16:24])
	b.Length = byteOrder.Uint32(data[24:28])
	b.ConnectionType = byteOrder.Uint16(data[28:30])
	b.Segments = make([]Segment, byteOrder.Uint16(data[30:32]))
	b.Encoding = EncodingType(data[32])
	b.Compression = CompressionType(data[33])
	b.UncompressedLength = byteOrder.Uint32(data[36:40])

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
		panic("did not consume entire payload")
	}

	return nil
}

func decompressBytes(data []byte, compression CompressionType, rawLen uint32) ([]byte, error) {
	var decompressed []byte

	switch compression {
	case CompressionNone:
		if rawLen != 0 && rawLen != uint32(len(data)) {
			log.WithFields(log.Fields{
				"raw_length":  rawLen,
				"data_length": len(data),
			}).Warnf("Mismatched lengths for uncompressed data")
		}

		return data, nil

	case CompressionZlib:
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("create zlib reader: %w", err)
		}
		defer reader.Close()

		if decompressed, err = io.ReadAll(reader); err != nil {
			return nil, fmt.Errorf("read all from zlib reader: %w", err)
		}

	case CompressionOodle:
		decompressed = make([]byte, rawLen)
		if err := oodle.Decode(data, decompressed); err != nil {
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

var _ encoding.BinaryUnmarshaler = (*Bundle)(nil)
