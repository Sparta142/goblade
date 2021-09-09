package ffxiv

import (
	"encoding/binary"
	"io"
	"unsafe"
)

type Unmarshaler interface {
	Unmarshal(rd io.Reader) error
}

type EncodingType uint8
type CompressionType uint8
type SegmentType uint16

const (
	CompressionNone CompressionType = 0
	CompressionZlib CompressionType = 1
)

const (
	SegmentIpc             SegmentType = 3
	SegmentClientKeepAlive SegmentType = 7
	SegmentServerKeepAlive SegmentType = 8
)

var BundleHeaderSize = int(unsafe.Sizeof(BundleHeader{}))
var SegmentHeaderSize = int(unsafe.Sizeof(SegmentHeader{}))

var MagicBytes = []byte{
	0x52, 0x52, 0xa0, 0x41, // 0x41a05252 (little endian)
	0xff, 0x5d, 0x46, 0xe2, // 0xe2465dff
	0x7f, 0x2a, 0x64, 0x4d, // 0x4d642a7f
	0x7b, 0x99, 0xc4, 0x75, // 0x75c4997b
}

var KeepAliveMagicBytes = []byte{
	0x00, 0x00, 0x00, 0x00, // 0x00000000
	0x00, 0x00, 0x00, 0x00, // 0x00000000
	0x00, 0x00, 0x00, 0x00, // 0x00000000
	0x00, 0x00, 0x00, 0x00, // 0x00000000
}

type BundleHeader struct {
	// Magic header bytes - always the same for IPC bundles.
	Magic [16]byte

	// The number of milliseconds since the Unix epoch time.
	Epoch uint64

	// The total length of the bundle, in bytes (including this header).
	Length uint16

	_ [2]byte

	// The connection type. Usually 0.
	ConnType uint16

	// The number of segments in the bundle.
	SegmentCount uint16

	// The encoding type of the bundle payload.
	Encoding EncodingType

	// The compression type of the bundle payload.
	Compression CompressionType

	_ [6]byte
}

func (bh *BundleHeader) Unmarshal(rd io.Reader) error {
	return binary.Read(rd, binary.LittleEndian, bh) // TODO: Optimize
}

func (bh *BundleHeader) IsCompressed() bool {
	return bh.Compression != CompressionNone
}

func (bh *BundleHeader) PayloadLength() int {
	return int(bh.Length) - BundleHeaderSize
}

type Bundle struct {
	BundleHeader
	Segments []Segment
}

func (b *Bundle) Unmarshal(rd io.Reader) error {
	b.BundleHeader.Unmarshal(rd)
	panic("not implemented") // TODO: Unmarshal payload
}

type SegmentHeader struct {
	// The total length of the segment, in bytes (including this header).
	Length uint32

	// The ID of the actor that sent the segment.
	Source uint32

	// The ID of the actor that received the segment.
	Target uint32

	// The segment type/opcode.
	Type SegmentType

	_ [2]byte
}

func (sh *SegmentHeader) Unmarshal(rd io.Reader) error {
	return binary.Read(rd, binary.LittleEndian, sh) // TODO: Optimize
}

func (sh *SegmentHeader) PayloadLength() int {
	return int(sh.Length) - SegmentHeaderSize
}

type Segment struct {
	SegmentHeader
	Data []byte
}

func (s *Segment) Unmarshal(rd io.Reader) error {
	s.SegmentHeader.Unmarshal(rd)
	panic("not implemented") // TODO: Unmarshal payload
}
