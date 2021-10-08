//go:build windows

package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

const url = "https://nmap.org/npcap/#download"

var downloadNpcapCmd = &cobra.Command{
	Use:   "download-npcap",
	Short: "Open the Npcap library download webpage",
	Run: func(*cobra.Command, []string) {
		if err := browser.OpenURL(url); err == nil {
			fmt.Println("Unable to open a web browser automatically.")
			fmt.Println("Please visit this link to download Npcap:")
			fmt.Printf("\n%s\n\n", url)
		}
	},
}

func init() {
	rootCmd.AddCommand(downloadNpcapCmd)
}
