package cmd

import (
	"fmt"
	"strings"

	"github.com/inconshreveable/mousetrap"
	"github.com/sirupsen/logrus"
	"github.com/sparta142/goblade/ffxiv"
	"github.com/spf13/cobra"
)

var (
	verbose = false
	region  = string(ffxiv.RegionGlobal)
)

// Version info from ldflags.
var (
	version    = "dev"
	gitSummary = "unknown"
)

var rootCmd = &cobra.Command{
	Use:                   "goblade",
	Short:                 "Lightweight, embeddable tool for capturing FINAL FANTASY XIV network traffic.",
	Version:               fmt.Sprintf("%s (%s)", version, gitSummary),
	DisableFlagsInUseLine: true,
	Example: strings.Join([]string{
		"goblade live",
		"goblade live enp0s2",
		"goblade file ./packets.pcapng",
	}, "\n"),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	PersistentPreRun: func(*cobra.Command, []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}

func Execute() {
	// Default to executing "goblade live" if opened from Explorer
	cobra.MousetrapHelpText = ""
	if mousetrap.StartedByExplorer() {
		rootCmd.SetArgs([]string{"live"})
	}

	rootCmd.Execute() //nolint:errcheck
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		verbose,
		"log more information to stderr",
	)

	rootCmd.PersistentFlags().StringVarP(
		&region,
		"region",
		"r",
		region,
		"the opcode region to decode IPCs for",
	)
}
