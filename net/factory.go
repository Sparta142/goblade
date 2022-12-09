package net

import (
	"net"
	"net/netip"
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

// New implements reassembly.StreamFactory.
func (fac *tcpStreamFactory) New( //nolint:ireturn
	netFlow, transport gopacket.Flow,
	_ *layers.TCP,
	_ reassembly.AssemblerContext,
) reassembly.Stream {
	src := toAddrPort(netFlow.Src(), transport.Src())
	dst := toAddrPort(netFlow.Dst(), transport.Dst())

	stream := &tcpStream{
		fsm: *reassembly.NewTCPSimpleFSM(reassembly.TCPSimpleFSMOptions{
			SupportMissingEstablishment: true,
		}),
		toClient: newTCPFlow(src, dst, fac.out),
		toServer: newTCPFlow(dst, src, fac.out),
	}

	fac.wg.Add(2)
	go stream.toClient.Run(&fac.wg)
	go stream.toServer.Run(&fac.wg)

	return stream
}

func (fac *tcpStreamFactory) Wait() {
	fac.wg.Wait()
}

func toAddrPort(network, transport gopacket.Endpoint) netip.AddrPort {
	s := net.JoinHostPort(network.String(), transport.String())
	return netip.MustParseAddrPort(s)
}

var _ reassembly.StreamFactory = (*tcpStreamFactory)(nil)
