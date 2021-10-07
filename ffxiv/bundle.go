package ffxiv

import (
	"bufio"
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

// Magic string indicating that a Bundle contains IPC segments.
const IpcMagicString = "\x52\x52\xa0\x41\xff\x5d\x46\xe2\x7f\x2a\x64\x4d\x7b\x99\xc4\x75"

// Magic string indicating that a Bundle contains keep-alive segments.
const KeepAliveMagicString = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

var (
	ErrBadMagicString = errors.New("ffxiv: bad magic string")
	ErrBadCompression = errors.New("ffxiv: bad compression type")
	ErrBadSegmentType = errors.New("ffxiv: bad segment type")
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

type Bundle struct {
	bundleHeader
	Segments []Segment
}

// Reads a single Bundle from data into bundle.
// Returns the number of bytes read or the error encountered.
func ReadBundle(rd io.Reader, b *Bundle) error {
	// Read the Bundle header
	if err := binary.Read(rd, byteOrder, &b.bundleHeader); err != nil {
		return fmt.Errorf("read bundle header: %w", err)
	}

	// Validate magic string in Bundle header.
	// Note: It is a programming error for the magic string to be invalid.
	if s := string(b.Magic[:]); s != IpcMagicString && s != KeepAliveMagicString {
		return ErrBadMagicString
	}

	// Make a reader to decompress the payload if needed
	var rr io.Reader

	switch b.Compression {
	case CompressionNone:
		rr = rd
	case CompressionZlib:
		zr, err := zlib.NewReader(bufio.NewReader(rd))
		if err != nil {
			return err
		}

		defer zr.Close()
		rr = zr
	}

	// Read all segments from the decompressed payload
	b.Segments = make([]Segment, b.SegmentCount)

	for i := 0; i < len(b.Segments); i++ {
		if err := ReadSegment(rr, &b.Segments[i]); err != nil {
			return fmt.Errorf("read segment: %w", err)
		}
	}

	return nil
}

func ReadBundleLength(data []byte) int {
	offset := unsafe.Offsetof(bundleHeader{}.Length)
	size := unsafe.Sizeof(bundleHeader{}.Length)

	if len(data) < int(offset+size) {
		return -1
	}

	return int(byteOrder.Uint16(data[offset:]))
}

func (b *Bundle) Time() time.Time {
	return time.UnixMilli(int64(b.Epoch)).UTC()
}

func (b *Bundle) IsCompressed() bool {
	return b.Compression != CompressionNone
}
