package net

import (
	"bufio"
	"bytes"
	"sync"
	"sync/atomic"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sparta142/goblade/ffxiv"
)

const kibibytes = 1024

type ffxivStream struct {
	fsm                reassembly.TCPSimpleFSM
	toClient, toServer ffxivHalfStream
}

type ffxivHalfStream struct {
	// Whether the reassembler missed a TCP segment
	lostData atomic.Value

	r *nio.PipeReader
	w *nio.PipeWriter

	bundles chan<- ffxiv.Bundle

	Src, Dst string
}

func newFfxivHalfStream(src, dst string, bundles chan<- ffxiv.Bundle) ffxivHalfStream {
	log.Debugf("Creating half stream for %s->%s", src, dst)

	halfStream := ffxivHalfStream{
		bundles: bundles,
		Src:     src,
		Dst:     dst,
	}
	halfStream.lostData.Store(false)
	halfStream.r, halfStream.w = nio.Pipe(buffer.New(2 * kibibytes))

	return halfStream
}

func (s *ffxivStream) Accept(
	tcp *layers.TCP,
	_ gopacket.CaptureInfo,
	dir reassembly.TCPFlowDirection,
	_ reassembly.Sequence,
	start *bool,
	_ reassembly.AssemblerContext,
) bool {
	if !s.fsm.CheckState(tcp, dir) {
		log.Debug("Packet failed state check, ignoring")
		return false
	}

	*start = true

	return true
}

func (s *ffxivStream) ReassembledSG(sg reassembly.ScatterGather, _ reassembly.AssemblerContext) {
	available, _ := sg.Lengths()
	if available == 0 {
		return
	}

	direction, _, _, skip := sg.Info()
	half := s.getHalf(direction)

	if skip > 0 {
		half.lostData.Store(true)
		log.Warnf("Lost %d bytes in stream", skip)

		return
	}

	// Queue the packets to the Bundle reading logic
	p := sg.Fetch(available)
	if _, err := half.w.Write(p); err != nil {
		log.WithError(err).Fatal("Failed to write data to half stream")
	}
}

func (s *ffxivStream) ReassemblyComplete(_ reassembly.AssemblerContext) bool {
	log.Debugf("Closing stream %v", s)

	s.toClient.w.Close()
	s.toServer.w.Close()

	return true
}

func (s *ffxivStream) getHalf(direction reassembly.TCPFlowDirection) *ffxivHalfStream {
	switch direction {
	case reassembly.TCPDirServerToClient:
		return &s.toClient
	case reassembly.TCPDirClientToServer:
		return &s.toServer
	}

	// Unreachable as long as TCPFlowDirection is bool
	panic("unknown TCP direction")
}

func (hs *ffxivHalfStream) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	log.Debugf("Starting half stream processing for %v", hs)

	scanner := bufio.NewScanner(hs.r)
	scanner.Split(hs.splitBundles)
	defer hs.r.Close()

	var bundle ffxiv.Bundle

	for scanner.Scan() {
		if err := bundle.UnmarshalBinary(scanner.Bytes()); err != nil {
			log.WithError(err).Fatal("Failed to read bundle")
		}

		hs.bundles <- bundle
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Fatal("Failed while scanning for bundle")
	}
}

func (hs *ffxivHalfStream) splitBundles(data []byte, _ bool) (advance int, token []byte, err error) {
	// There are 3 failure modes to be aware of when considering lost bytes:
	//     1) The magic header was (partially) lost, so its delimited bundle will
	//        not be found by the Scanner and *not* cause any issues.
	//	   2) Part of the payload (i.e., not the magic header) was lost, so the scanner
	//		  will misinterpret the remaining data as part of the original bundle.
	//		  This can cause major issues due to "unaligned" bundle decoding and probably
	//        an error of some sort after a sequence of completely invalid bundles.
	//	   3) The entire bundle as a whole was sent as one TCP segment, and was completely lost.
	//		  This will also *not* cause any issues with misinterpreting the data stream.
	if hs.lostData.Swap(false) == true {
		log.Warnf("Discarding all %d bytes in scanner as a safety measure", len(data))
		return len(data), nil, nil
	}

	// Find the magic bytes in `data`
	idx := indexFirst(data, ffxiv.IpcMagicBytes, ffxiv.KeepAliveMagicBytes)
	if idx == -1 {
		return 0, nil, nil
	}

	// The chunk of `data` that starts with the magic string (found above)
	chunk := data[idx:]

	length := ffxiv.PeekBundleLength(chunk) // The (probable) length of the Bundle
	if length > len(chunk) || length == -1 {
		return idx, nil, nil
	}

	return idx + length, chunk[:length], nil
}

// Get the index of the earliest instance of any slice in seps,
// or -1 if no slice in seps is present in s.
func indexFirst(s []byte, seps ...[]byte) int {
	idx := -1
	for _, sep := range seps {
		index := bytes.Index(s, sep)

		if index != -1 && (idx == -1 || index < idx) {
			idx = index
		}
	}

	return idx
}

var _ reassembly.Stream = (*ffxivStream)(nil)
