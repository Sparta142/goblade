package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket/pcap"
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:                   "file FILENAME",
	Short:                 "Decode traffic from a pcap-compatible file",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, args []string) error {
		handle, err := pcap.OpenOffline(args[0])
		if err != nil {
			return err
		}
		defer handle.Close()

		log.Infof("Parsing capture file: %s", args[0])

		handlePackets(handle)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
}
