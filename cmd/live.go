package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket/pcap"
	"github.com/jackpal/gateway"
	"github.com/sparta142/goblade/ffxiv"
	"github.com/sparta142/goblade/net"
	"github.com/spf13/cobra"
)

// Flag indicating that a network interface is loopback.
const pcapIfLoopback = uint32(0x00000001)

const defaultSnaplen = 2048

var errNoDefaultInterface = errors.New("no default interface found")

var promiscuous bool

var liveCmd = &cobra.Command{
	Use:                   "live [--promiscuous] [INTERFACE]",
	Short:                 "Decode traffic from a network interface in real time",
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, args []string) error {
		var ifname string

		if len(args) == 0 {
			var err error
			ifname, err = getDefaultInterfaceName()

			if err != nil {
				return err
			}

			log.Infof("Capturing on default device: %s", ifname)
		} else {
			ifname = args[0]
			log.Infof("Capturing on specified device: %s", ifname)
		}

		handle, err := pcap.OpenLive(ifname, defaultSnaplen, promiscuous, pcap.BlockForever)
		if err != nil {
			return fmt.Errorf("open live pcap device: %w", err)
		}
		defer handle.Close()
		handlePackets(handle)

		return nil
	},
}

// Gets the name of the non-loopback network interface for the default gateway.
func getDefaultInterfaceName() (string, error) {
	ip, err := gateway.DiscoverInterface()
	if err != nil {
		return "", fmt.Errorf("discover default network interface ip: %w", err)
	}

	devs, err := pcap.FindAllDevs()
	if err != nil {
		return "", fmt.Errorf("find all network interfaces: %w", err)
	}

	for _, iface := range devs {
		for _, addr := range iface.Addresses {
			if ip.Equal(addr.IP) && (iface.Flags&pcapIfLoopback) == 0 {
				return iface.Name, nil
			}
		}
	}

	return "", errNoDefaultInterface
}

func handlePackets(handle *pcap.Handle) {
	bundles := make(chan ffxiv.Bundle)
	go func() {
		err := net.Capture(handle, bundles)
		if err != nil {
			log.Fatal(err)
		}
	}()

	e := json.NewEncoder(os.Stdout)
	e.SetEscapeHTML(false)
	e.SetIndent("", "")

	for bnd := range bundles {
		err := e.Encode(bnd)
		if err != nil {
			log.WithError(err).Fatal("Failed to encode bundle")
		}
	}
}

func init() {
	rootCmd.AddCommand(liveCmd)

	liveCmd.Flags().BoolVar(
		&promiscuous,
		"promiscuous",
		false,
		"capture all network traffic instead of just this computer's",
	)
}
