package ffxiv

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

const (
	segmentHeaderSize = 16
	ipcHeaderSize     = 16
	keepAliveSize     = 8
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

type Segment struct {
	// The total length of the segment, in bytes (including the header).
	Length uint32 `json:"-"`

	// The ID of the actor that sent the segment.
	Source uint32 `json:"source"`

	// The ID of the actor that received the segment.
	Target uint32 `json:"target"`

	// The segment type. Usually SegmentIpc.
	Type SegmentType `json:"type"`

	Payload interface{} `json:"payload"`
}

func (s *Segment) UnmarshalBinary(data []byte) error {
	if len(data) < segmentHeaderSize {
		return ErrNotEnoughData
	}

	// Read the Segment header
	s.Length = byteOrder.Uint32(data[0:4])
	s.Source = byteOrder.Uint32(data[4:8])
	s.Target = byteOrder.Uint32(data[8:12])
	s.Type = SegmentType(byteOrder.Uint16(data[12:14]))

	// Decode the Segment payload depending on the type
	payloadData := data[segmentHeaderSize:s.Length]

	switch s.Type {
	case SegmentIpc:
		s.Payload = &Ipc{}
		if err := s.Payload.(*Ipc).UnmarshalBinary(payloadData); err != nil {
			return err
		}

	case SegmentClientKeepAlive, SegmentServerKeepAlive:
		s.Payload = &KeepAlive{}
		if err := s.Payload.(*KeepAlive).UnmarshalBinary(payloadData); err != nil {
			return err
		}

	default:
		log.Debugf("Segment has unknown type %d; storing payload as []byte", s.Type)
		s.Payload = payloadData
	}

	return nil
}

type Ipc struct {
	Magic    uint16 `json:"-"`
	Type     uint16 `json:"type"`
	ServerID uint16 `json:"serverId"`
	Epoch    uint32 `json:"epoch"`

	Data []byte `json:"data"`
}

func (i *Ipc) UnmarshalBinary(data []byte) error {
	if len(data) < ipcHeaderSize {
		return ErrNotEnoughData
	}

	// Read the IPC header
	i.Magic = byteOrder.Uint16(data[0:2])
	i.Type = byteOrder.Uint16(data[2:4])
	i.ServerID = byteOrder.Uint16(data[6:8])
	i.Epoch = byteOrder.Uint32(data[8:12])

	i.Data = data[16:]
	return nil
}

type KeepAlive struct {
	ID    uint32 `json:"id"`
	Epoch uint32 `json:"epoch"`
}

func (k *KeepAlive) UnmarshalBinary(data []byte) error {
	if len(data) < keepAliveSize {
		return ErrNotEnoughData
	}

	k.ID = byteOrder.Uint32(data[0:4])
	k.ID = byteOrder.Uint32(data[4:8])
	return nil
}
