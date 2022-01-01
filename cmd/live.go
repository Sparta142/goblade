package cmd

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket/pcap"
	"github.com/jackpal/gateway"
	"github.com/sparta142/goblade/ffxiv"
	"github.com/sparta142/goblade/net"
	"github.com/spf13/cobra"
)

// Flag indicating that a network interface is loopback.
const pcapIfLoopback = uint32(0x00000001)

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

		handle, err := pcap.OpenLive(ifname, 2048, promiscuous, pcap.BlockForever)
		if err != nil {
			return err
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
		return "", err
	}

	devs, err := pcap.FindAllDevs()
	if err != nil {
		return "", err
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
	table, ok := ffxiv.GetOpcodeTable(ffxiv.RegionGlobal)
	if !ok {
		log.Fatal("Failed to load global opcode table")
	}

	bundles := make(chan ffxiv.Bundle, 100)
	go net.Capture(handle, bundles)

	for bnd := range bundles {
		fmt.Printf("* Bundle (%d bytes, at %s)\n", bnd.Length, bnd.Time())

		for i, seg := range bnd.Segments {
			fmt.Printf("    [%d] Segment - %s (%d bytes, 0x%X -> 0x%X)\n", i+1, seg.Type, seg.Length, seg.Source, seg.Target)

			if ipc, ok := seg.Payload.(*ffxiv.Ipc); ok {
				fmt.Printf(
					"    ServerZone: %q | ClientZone: %q\n",
					table.GetOpcodeName(ffxiv.ServerZoneIpcType, int(ipc.Type)),
					table.GetOpcodeName(ffxiv.ClientZoneIpcType, int(ipc.Type)),
				)
			}
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
