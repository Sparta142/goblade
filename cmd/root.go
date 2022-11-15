package cmd

import (
	"fmt"
	"strings"

	"github.com/inconshreveable/mousetrap"
	log "github.com/sirupsen/logrus"
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
			log.SetLevel(log.DebugLevel)
		}
	},
}

func Execute() {
	log.StandardLogger().Formatter = &log.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	// Default to executing "goblade live" if opened from Explorer
	cobra.MousetrapHelpText = ""
	if mousetrap.StartedByExplorer() {
		rootCmd.SetArgs([]string{"live"})
	}

	_ = rootCmd.Execute()
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
