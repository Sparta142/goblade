package ffxiv

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
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
	IpcMagicBytes = []byte{82, 82, 160, 65, 255, 93, 70, 226, 127, 42, 100, 77, 123, 153, 196, 117}

	// Magic bytes indicating that a Bundle contains keep-alive segments.
	KeepAliveMagicBytes = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

var (
	ErrBadMagicBytes  = errors.New("ffxiv: bad magic bytes")
	ErrBadCompression = errors.New("ffxiv: bad compression type")
	ErrNotEnoughData  = errors.New("ffxiv: not enough data")
)

const (
	bundleLengthOffset  = 24
	bundleLengthSize    = unsafe.Sizeof(*(*uint32)(nil))
	bundleHeaderSize    = 40
	maxUncompressedSize = 1 << 16
)

// The byte order used by all FFXIV network data.
var byteOrder = binary.LittleEndian

// Pool of []byte used as scratch space for deserializing Bundle payloads.
var slicePool = sync.Pool{
	New: func() any {
		return make([]byte, maxUncompressedSize)
	},
}

type Bundle struct {
	// The number of milliseconds since the Unix epoch time.
	Epoch uint64 `json:"epoch"`

	// The connection type. Usually 0.
	ConnectionType uint16 `json:"connectionType"`

	// The encoding type of the bundle payload.
	Encoding EncodingType `json:"-"`

	Segments []Segment `json:"segments"`
}

func (b *Bundle) UnmarshalBinary(data []byte) error {
	// Is there enough bytes in data to contain a Bundle header?
	if len(data) < bundleHeaderSize {
		return fmt.Errorf("check length for header: %w", ErrNotEnoughData)
	}

	_ = data[bundleHeaderSize-1]

	// Validate the magic bytes
	if !bytes.HasPrefix(data, IpcMagicBytes) && !bytes.HasPrefix(data, KeepAliveMagicBytes) {
		return ErrBadMagicBytes
	}

	// Read the Bundle header
	b.Epoch = byteOrder.Uint64(data[16:24])
	b.ConnectionType = byteOrder.Uint16(data[28:30])
	b.Segments = make([]Segment, byteOrder.Uint16(data[30:32]))
	b.Encoding = EncodingType(data[32])

	// Get the info that describes how to read the payload
	length := byteOrder.Uint32(data[24:28])
	compression := CompressionType(data[33])
	uncompressedLength := byteOrder.Uint32(data[36:40])

	// Is there enough bytes in data to contain the entire Bundle?
	if len(data) < int(length) {
		return fmt.Errorf("check length for bundle: %w", ErrNotEnoughData)
	}

	rental := slicePool.Get()
	defer slicePool.Put(rental)

	// Decompress the Bundle payload
	payloadData, err := compression.Decompress(
		data[bundleHeaderSize:],
		rental.([]byte)[:uncompressedLength],
	)
	if err != nil {
		return fmt.Errorf("decompress payload: %w", err)
	}

	// Read all segments from the decompressed payload
	for i := range b.Segments {
		segment := &b.Segments[i]

		if err := segment.UnmarshalBinary(payloadData); err != nil {
			return fmt.Errorf("read segment: %w", err)
		}

		// Advance payloadData by the size of the Segment we just read
		payloadData = payloadData[segment.Length:]
	}

	// Sanity check: the entire payload should have been consumed
	if len(payloadData) != 0 {
		panic("did not consume entire payload")
	}

	return nil
}

// Get the UTC time that this Bundle was sent.
func (b *Bundle) Time() time.Time {
	return time.UnixMilli(int64(b.Epoch)).UTC()
}

// Decompresses src according to this compression type. dst may not be used.
func (c CompressionType) Decompress(src, dst []byte) ([]byte, error) {
	switch c {
	case CompressionNone:
		return src, nil

	case CompressionZlib:
		reader, err := zlib.NewReader(bytes.NewReader(src))
		if err != nil {
			return nil, fmt.Errorf("create zlib reader: %w", err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read all from zlib reader: %w", err)
		}

		return decompressed, nil

	case CompressionOodle:
		if err := oodle.Decode(src, dst); err != nil {
			return nil, fmt.Errorf("oodle decode: %w", err)
		}

		return dst, nil

	default:
		return nil, ErrBadCompression
	}
}

func PeekBundleLength(data []byte) int {
	if len(data) < int(bundleLengthOffset+bundleLengthSize) {
		return -1
	}

	return int(byteOrder.Uint16(data[bundleLengthOffset:]))
}

var _ encoding.BinaryUnmarshaler = (*Bundle)(nil)
