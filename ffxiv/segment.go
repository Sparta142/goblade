package ffxiv

import (
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

type SegmentType uint16

const (
	SegmentIpc             = SegmentType(3)
	SegmentClientKeepAlive = SegmentType(7)
	SegmentServerKeepAlive = SegmentType(8)
)

func (s SegmentType) String() string {
	switch s {
	case SegmentIpc:
		return "Ipc"
	case SegmentClientKeepAlive:
		return "ClientKeepAlive"
	case SegmentServerKeepAlive:
		return "ServerKeepAlive"
	default:
		return fmt.Sprint(uint16(s))
	}
}

var (
	segmentHeaderSize = int(unsafe.Sizeof(segmentHeader{}))
	ipcHeaderSize     = int(unsafe.Sizeof(ipcHeader{}))
)

type segmentHeader struct {
	// The total length of the segment, in bytes (including this header).
	Length uint32

	// The ID of the actor that sent the segment.
	Source uint32

	// The ID of the actor that received the segment.
	Target uint32

	// The segment type. Usually SegmentIpc.
	Type SegmentType

	_ [2]byte
}

type Segment struct {
	segmentHeader
	Payload interface{}
}

func (s *Segment) payloadLength() int {
	return int(s.segmentHeader.Length) - segmentHeaderSize
}

type ipcHeader struct {
	Magic    uint16
	Type     uint16
	_        [2]byte
	ServerId uint16
	Epoch    uint32
	_        [4]byte
}

type Ipc struct {
	ipcHeader
	Data []byte
}

type KeepAlive struct {
	Id    uint32
	Epoch uint32
}

func ReadSegment(rd io.Reader, s *Segment) error {
	// Read the Segment header
	if err := binary.Read(rd, byteOrder, &s.segmentHeader); err != nil {
		return fmt.Errorf("read segment header: %w", err)
	}

	// Decode the Segment payload depending on the type
	switch s.segmentHeader.Type {
	case SegmentIpc:
		ipc := &Ipc{}
		if err := binary.Read(rd, byteOrder, &ipc.ipcHeader); err != nil {
			return fmt.Errorf("read ipc header: %w", err)
		}

		ipc.Data = make([]byte, s.payloadLength()-ipcHeaderSize)
		if _, err := io.ReadFull(rd, ipc.Data); err != nil {
			return fmt.Errorf("read ipc data: %w", err)
		}

		s.Payload = ipc
	case SegmentClientKeepAlive, SegmentServerKeepAlive:
		keepAlive := &KeepAlive{}
		if err := binary.Read(rd, byteOrder, keepAlive); err != nil {
			return fmt.Errorf("read keep-alive: %w", err)
		}
		s.Payload = keepAlive
	default:
		data := make([]byte, s.payloadLength())
		if _, err := io.ReadFull(rd, data); err != nil {
			return fmt.Errorf("read raw segment: %w", err)
		}
		s.Payload = data
	}

	return nil
}
