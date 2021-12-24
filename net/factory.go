package net

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sparta142/goblade/ffxiv"
)

type tcpStreamFactory struct {
	out chan<- ffxiv.Bundle
	wg  sync.WaitGroup
}

func (fac *tcpStreamFactory) New(net, transport gopacket.Flow, tcp *layers.TCP, ac reassembly.AssemblerContext) reassembly.Stream {
	stream := &ffxivStream{
		net:       net,
		transport: transport,
		fsm: *reassembly.NewTCPSimpleFSM(reassembly.TCPSimpleFSMOptions{
			SupportMissingEstablishment: true,
		}),
		optCheck: reassembly.NewTCPOptionCheck(),
		toClient: newFfxivHalfStream(tcp.SrcPort, tcp.DstPort, fac.out),
		toServer: newFfxivHalfStream(tcp.DstPort, tcp.SrcPort, fac.out),
	}

	fac.wg.Add(2)
	go stream.toClient.Run(&fac.wg)
	go stream.toServer.Run(&fac.wg)

	return stream
}

func (fac *tcpStreamFactory) Wait() {
	fac.wg.Wait()
}
