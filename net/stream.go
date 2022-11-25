package net

import (
	"bufio"
	"bytes"
	"fmt"
	"net/netip"
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

// The number of bytes in one kibibyte (1 KiB).
const kibibytes = 1024

type tcpStream struct {
	fsm                reassembly.TCPSimpleFSM
	toClient, toServer *tcpFlow
}

type tcpFlow struct {
	// Whether the reassembler missed a TCP segment
	lostData atomic.Bool

	reader *nio.PipeReader
	writer *nio.PipeWriter

	bundles chan<- ffxiv.Bundle

	Src, Dst netip.AddrPort
}

func newTCPFlow(src, dst netip.AddrPort, bundles chan<- ffxiv.Bundle) *tcpFlow {
	flow := &tcpFlow{
		bundles: bundles,
		Src:     src,
		Dst:     dst,
	}
	flow.lostData.Store(false)
	flow.reader, flow.writer = nio.Pipe(buffer.New(2 * kibibytes))

	log.Debugf("Created TCP flow for %s", flow)

	return flow
}

func (stream *tcpStream) Accept(
	tcp *layers.TCP,
	_ gopacket.CaptureInfo,
	dir reassembly.TCPFlowDirection,
	_ reassembly.Sequence,
	start *bool,
	_ reassembly.AssemblerContext,
) bool {
	if !stream.fsm.CheckState(tcp, dir) {
		log.Warn("Packet failed state check, ignoring")
		return false
	}

	*start = true

	return true
}

func (stream *tcpStream) ReassembledSG(sg reassembly.ScatterGather, _ reassembly.AssemblerContext) {
	available, _ := sg.Lengths()
	if available == 0 {
		return
	}

	direction, _, _, skip := sg.Info()
	flow := stream.getFlow(direction)

	if skip > 0 {
		flow.lostData.Store(true)
		log.Warnf("Lost %d bytes in stream", skip)

		return
	}

	// Queue the packets to the Bundle reading logic
	p := sg.Fetch(available)
	if _, err := flow.writer.Write(p); err != nil {
		log.WithError(err).Fatal("Failed to write data to TCP flow")
	}
}

func (stream *tcpStream) ReassemblyComplete(_ reassembly.AssemblerContext) bool {
	log.Debugf("Closing stream %v", stream)

	stream.toClient.writer.Close()
	stream.toServer.writer.Close()

	return true
}

func (stream *tcpStream) getFlow(direction reassembly.TCPFlowDirection) *tcpFlow {
	switch direction {
	case reassembly.TCPDirServerToClient:
		return stream.toClient
	case reassembly.TCPDirClientToServer:
		return stream.toServer
	}

	// Unreachable as long as TCPFlowDirection is bool
	panic("unknown TCP direction")
}

func (flow *tcpFlow) String() string {
	return fmt.Sprintf("%s->%s", flow.Src, flow.Dst)
}

func (flow *tcpFlow) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	log.Debugf("Starting TCP flow processing for %s", flow)

	scanner := bufio.NewScanner(flow.reader)
	scanner.Split(flow.splitBundles)
	defer flow.reader.Close()

	var bundle ffxiv.Bundle

	for scanner.Scan() {
		if err := bundle.UnmarshalBinary(scanner.Bytes()); err != nil {
			log.WithError(err).Fatal("Failed to read bundle")
		}

		flow.bundles <- bundle
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Fatal("Failed while scanning for bundle")
	}
}

func (flow *tcpFlow) splitBundles(data []byte, _ bool) (advance int, token []byte, err error) {
	// There are 3 failure modes to be aware of when considering lost bytes:
	//     1) The magic header was (partially) lost, so its delimited bundle will
	//        not be found by the Scanner and *not* cause any issues.
	//	   2) Part of the payload (i.e., not the magic header) was lost, so the scanner
	//		  will misinterpret the remaining data as part of the original bundle.
	//		  This can cause major issues due to "unaligned" bundle decoding and probably
	//        an error of some sort after a sequence of completely invalid bundles.
	//	   3) The entire bundle as a whole was sent as one TCP segment, and was completely lost.
	//		  This will also *not* cause any issues with misinterpreting the data stream.
	if flow.lostData.Swap(false) {
		log.Warnf("Discarding all %d bytes in scanner as a safety measure", len(data))
		return len(data), nil, nil
	}

	var idx int

	// Find the magic bytes in `data`.
	// We explicitly check index 0 first since the magic bytes will always
	// be there unless something went wrong. When we're correct,
	// this function will run about 11x faster (according to pprof).
	if bytes.HasPrefix(data, ffxiv.IpcMagicBytes) || bytes.HasPrefix(data, ffxiv.KeepAliveMagicBytes) {
		idx = 0
	} else {
		idx = indexFirst(data, ffxiv.IpcMagicBytes, ffxiv.KeepAliveMagicBytes)
		if idx == -1 {
			return 0, nil, nil
		}
	}

	// The chunk of `data` that starts with the magic bytes (found above)
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

		// Did we match and is it earlier than what we already matched, if any?
		if index != -1 && (idx == -1 || index < idx) {
			idx = index
		}
	}

	return idx
}

var _ reassembly.Stream = (*tcpStream)(nil)
