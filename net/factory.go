package net

import (
	"net"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"github.com/sparta142/goblade/ffxiv"
)

type tcpStreamFactory struct {
	wg  sync.WaitGroup
	out chan<- ffxiv.Bundle
}

func (fac *tcpStreamFactory) New( //nolint:ireturn // To satisfy the reassembly.StreamFactory interface
	netFlow, transport gopacket.Flow,
	_ *layers.TCP,
	_ reassembly.AssemblerContext,
) reassembly.Stream {
	src := net.JoinHostPort(netFlow.Src().String(), transport.Src().String())
	dst := net.JoinHostPort(netFlow.Dst().String(), transport.Dst().String())

	stream := &ffxivStream{
		fsm: *reassembly.NewTCPSimpleFSM(reassembly.TCPSimpleFSMOptions{
			SupportMissingEstablishment: true,
		}),
		toClient: newFfxivHalfStream(src, dst, fac.out),
		toServer: newFfxivHalfStream(dst, src, fac.out),
	}

	fac.wg.Add(2)
	go stream.toClient.Run(&fac.wg)
	go stream.toServer.Run(&fac.wg)

	return stream
}

func (fac *tcpStreamFactory) Wait() {
	fac.wg.Wait()
}
