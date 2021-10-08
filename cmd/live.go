package cmd

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/jackpal/gateway"
	"github.com/sparta142/goblade/v2/net"
	"github.com/spf13/cobra"
)

var errNoDefaultInterface = errors.New("no default interface found")

var promiscuous bool

var liveCmd = &cobra.Command{
	Use:                   "live [--promiscuous] [device]",
	Short:                 "Decode traffic from a network interface in real time",
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, args []string) error {
		var device string

		if len(args) == 0 {
			var err error
			device, err = getDefaultDeviceName()

			if err != nil {
				return err
			}

			log.Printf("Capturing on default device: %s\n", device)
		} else {
			device = args[0]
			log.Printf("Capturing on given device: %s\n", device)
		}

		handle, err := pcap.OpenLive(device, 2048, promiscuous, pcap.BlockForever)
		if err != nil {
			return err
		}
		defer handle.Close()

		for bnd := range net.Capture(handle) {
			fmt.Printf("%v (latency = %s)\n", bnd, time.Since(bnd.Time())) // TODO
		}

		return nil
	},
}

func getDefaultDeviceName() (string, error) {
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
			if ip.Equal(addr.IP) {
				return iface.Name, nil
			}
		}
	}

	return "", errNoDefaultInterface
}

func init() {
	rootCmd.AddCommand(liveCmd)

	liveCmd.Flags().BoolVar(
		&promiscuous,
		"promiscuous",
		false,
		"whether to capture all traffic instead of just this computer's")
}
