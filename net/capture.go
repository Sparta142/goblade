package net

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/reassembly"
	"github.com/sparta142/goblade/v2/ffxiv"
)

// BPF filter that selects known FFXIV ports and game server subnets.
const defaultBpfFilter = "tcp and " +
	"src portrange 49152-65535 and " +
	"dst portrange 49152-65535 and " +
	"(net 204.2.229.0/24 or 195.82.50.0/24 or 124.50.157.0/24)"

// How often to attempt to flush TCP connections.
const flushInterval = 1 * time.Minute

// How old the data in an out-of-order TCP stream should be before flushing that stream.
const flushStreamAge = 3 * time.Minute

func Capture(handle *pcap.Handle) <-chan ffxiv.Bundle {
	return CaptureContext(context.Background(), handle)
}

func CaptureContext(ctx context.Context, handle *pcap.Handle) <-chan ffxiv.Bundle {
	out := make(chan ffxiv.Bundle)

	go func() {
		// Configure pcap handle
		handle.SetBPFFilter(bpfFilter())
		handle.SetDirection(pcap.DirectionInOut)

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

		defer func() {
			log.Debugf("Closing all streams")
			closed := assembler.FlushAll()
			log.Infof("Closed %d streams", closed)

			factory.Wait()
			close(out)
		}()

		for {
			select {
			case packet, ok := <-src.Packets():
				if !ok {
					log.Info("No more packets to handle")
					return
				}

				tcp := packet.TransportLayer().(*layers.TCP)
				net := packet.NetworkLayer()

				if err := tcp.SetNetworkLayerForChecksum(net); err != nil {
					log.WithError(err).Warn("Failed to set network layer for checksum")
					return
				}

				assembler.AssembleWithContext(net.NetworkFlow(), tcp, newCaptureContext(packet))

			case <-ticker.C:
				log.Debugf("Starting periodic flush")

				flushed, closed := assembler.FlushWithOptions(reassembly.FlushOptions{
					T: time.Now().Add(-flushStreamAge),
				})

				log.Debugf("Flushed %d streams, closed %d", flushed, closed)

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

type captureContext struct {
	gopacket.CaptureInfo
}

func (cc *captureContext) GetCaptureInfo() gopacket.CaptureInfo {
	return cc.CaptureInfo
}

func newCaptureContext(packet gopacket.Packet) *captureContext {
	return &captureContext{
		CaptureInfo: packet.Metadata().CaptureInfo,
	}
}

// Gets the BPF filter set in the environment variables,
// or the default filter if none has been set.
func bpfFilter() string {
	if expr, present := os.LookupEnv("GOBLADE_BPF"); present {
		log.WithField("new_bpf", expr).Warn("Default BPF overridden in environment variables")
		return expr
	}

	return defaultBpfFilter
}
