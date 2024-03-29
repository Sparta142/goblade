package net

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/reassembly"
	"github.com/sparta142/goblade/ffxiv"
	"github.com/sparta142/goblade/oodle"
)

// Filters for potential FFXIV ports and known data center networks.
var bpfFilter = fmt.Sprintf(
	"tcp and src portrange 49152-65535 and dst portrange 49152-65535 and (net %s)",
	strings.Join(ffxiv.DataCenterCIDRs[:], " or "),
)

// How often to attempt to flush TCP connections.
const flushInterval = 1 * time.Minute

// How old the data in an out-of-order TCP stream should be before flushing that stream.
const flushStreamAge = 3 * time.Minute

func Capture(handle *pcap.Handle, out chan<- ffxiv.Bundle) error {
	return CaptureContext(context.Background(), handle, out)
}

func CaptureContext(ctx context.Context, handle *pcap.Handle, out chan<- ffxiv.Bundle) error {
	// Configure pcap handle
	if err := handle.SetBPFFilter(bpfFilter); err != nil {
		return fmt.Errorf("set bpf packet filter: %w", err)
	}

	// Setup packet source
	src := gopacket.NewPacketSource(handle, handle.LinkType())
	src.NoCopy = true
	src.Lazy = true

	// Create TCP reassembler
	factory := &tcpStreamFactory{out: out}
	pool := reassembly.NewStreamPool(factory)
	assembler := reassembly.NewAssembler(pool)
	assembler.MaxBufferedPagesPerConnection = 512
	assembler.MaxBufferedPagesTotal = 2048

	// Ticker to flush the reassembler periodically
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	// Setup Oodle decompression
	if err := oodle.Setup(); err != nil {
		log.WithError(err).Errorf("Failed to set up Oodle decompression. This won't end well.")
	} else {
		defer oodle.Shutdown()
	}

Outer:
	for {
		select {
		case packet, ok := <-src.Packets():
			if !ok {
				log.Info("No more packets available")
				break Outer
			}

			handlePacket(packet, assembler)

		case <-ticker.C:
			handleTick(assembler)

		case <-ctx.Done():
			break Outer
		}
	}

	// Clean up everything
	flushed := assembler.FlushAll()
	log.WithField("count", flushed).Info("Flushed/closed all streams")
	factory.Wait()
	close(out)

	return nil
}

type captureContext gopacket.CaptureInfo

func (c *captureContext) GetCaptureInfo() gopacket.CaptureInfo {
	return gopacket.CaptureInfo(*c)
}

func handlePacket(packet gopacket.Packet, assembler *reassembly.Assembler) {
	tcp := packet.TransportLayer().(*layers.TCP)
	net := packet.NetworkLayer()

	if err := tcp.SetNetworkLayerForChecksum(net); err != nil {
		// This is probably not a fatal error for our purposes
		log.WithError(err).Warn("Failed to set network layer for checksum")
	}

	ctx := captureContext(packet.Metadata().CaptureInfo)
	assembler.AssembleWithContext(net.NetworkFlow(), tcp, &ctx)
}

func handleTick(assembler *reassembly.Assembler) {
	log.Debug("Starting periodic stream maintenance")

	flushed, closed := assembler.FlushWithOptions(reassembly.FlushOptions{
		T: time.Now().Add(-flushStreamAge),
	})

	log.WithFields(log.Fields{
		"flushed": flushed,
		"closed":  closed,
	}).Debug("Stream maintenance finished")
}

var _ reassembly.AssemblerContext = (*captureContext)(nil)
