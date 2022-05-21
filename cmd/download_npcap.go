//go:build windows

package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const url = "https://nmap.org/npcap/#download"

//nolint:forbidigo // fmt.Printf is used to print a message to the user
var downloadNpcapCmd = &cobra.Command{
	Use:   "download-npcap",
	Short: "Open the Npcap library download webpage in a web browser",
	Run: func(*cobra.Command, []string) {
		if err := browser.OpenURL(url); err != nil {
			log.WithError(err).Debug("Failed to open browser")

			fmt.Println("Unable to open a web browser. Please visit this link to download Npcap:")
			fmt.Printf("\n    %s\n\n", url)
		}
	},
}

func init() {
	rootCmd.AddCommand(downloadNpcapCmd)
}
