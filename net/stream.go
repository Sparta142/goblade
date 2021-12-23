package net

import (
	"bufio"
	"bytes"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sparta142/goblade/v2/ffxiv"
	"github.com/sparta142/goblade/v2/internal"
)

type ffxivStream struct {
	net, transport     gopacket.Flow
	fsm                reassembly.TCPSimpleFSM
	optCheck           reassembly.TCPOptionCheck
	toClient, toServer ffxivHalfStream
}

type ffxivHalfStream struct {
	srcPort, dstPort layers.TCPPort
	in               chan []byte
	out              chan<- ffxiv.Bundle

	// Whether the reassembler missed a TCP segment
	lostData bool
}

func newFfxivHalfStream(srcPort, dstPort layers.TCPPort, out chan<- ffxiv.Bundle) ffxivHalfStream {
	return ffxivHalfStream{
		srcPort:  srcPort,
		dstPort:  dstPort,
		in:       make(chan []byte),
		out:      out,
		lostData: false,
	}
}

func (stream *ffxivStream) Accept(tcp *layers.TCP, ci gopacket.CaptureInfo, dir reassembly.TCPFlowDirection, nextSeq reassembly.Sequence, start *bool, ac reassembly.AssemblerContext) bool {
	if !stream.fsm.CheckState(tcp, dir) {
		return false // Failed TCP state check
	}

	if err := stream.optCheck.Accept(tcp, ci, dir, nextSeq, start); err != nil {
		return false // Failed options check
	}

	*start = true
	return true
}

func (stream *ffxivStream) ReassembledSG(sg reassembly.ScatterGather, ac reassembly.AssemblerContext) {
	available, _ := sg.Lengths()
	if available == 0 {
		return
	}

	direction, _, _, skip := sg.Info()
	half := stream.getHalf(direction)

	if skip > 0 {
		half.lostData = true
		log.Warnf("Lost %d bytes in stream", skip)
		return
	}

	// Queue the packets to the Bundle reading logic
	half.in <- sg.Fetch(available)
}

func (stream *ffxivStream) ReassemblyComplete(ac reassembly.AssemblerContext) bool {
	log.Debugf("Closing stream %v", stream)
	close(stream.toClient.in)
	close(stream.toServer.in)
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

	scanner := bufio.NewScanner(internal.NewChannelReader(hs.in))
	scanner.Split(hs.splitBundles)

	var bundle ffxiv.Bundle

	for scanner.Scan() {
		r := bytes.NewReader(scanner.Bytes())
		if err := ffxiv.ReadBundle(r, &bundle); err != nil {
			log.WithError(err).Fatal("Failed to read bundle") // TODO: Handle gracefully
		}

		hs.out <- bundle
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Error("Failed while scanning for bundle") // TODO: Should this crash?
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
	//	   3) The entire bundle as a whole was sent as one segment, and was completely lost.
	//		  This will also *not* cause any issues with misinterpreting the data stream.
	if hs.lostData {
		hs.lostData = false
		log.Warnf("Discarding %d bytes in scanner as a safety measure", len(data))
		return len(data), nil, nil
	}

	s := string(data)

	// Find the magic string
	i := indexFirst(s, ffxiv.IpcMagicString, ffxiv.KeepAliveMagicString)
	if i == -1 {
		return 0, nil, nil
	}

	// Get the length of the bundle that is signaled by the magic string
	chunk := data[i:]
	if len(chunk) < 2 {
		return i, nil, nil
	}

	length := int(ffxiv.ReadBundleLength(chunk)) // The (probable) length of the Bundle
	if length > len(chunk) || length == -1 {
		return i, nil, nil
	}

	return i + length, chunk[:length], nil
}

// Get the index of the earliest occurrence of any substring in substrs.
func indexFirst(s string, substrs ...string) int {
	i := -1
	for _, substr := range substrs {
		index := strings.Index(s, substr)

		if index != -1 && (i == -1 || index < i) {
			i = index
		}
	}

	return i
}
