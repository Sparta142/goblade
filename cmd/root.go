package cmd

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	_ "github.com/sparta142/goblade/v2/ffxiv"
)

var verbose = false

var rootCmd = &cobra.Command{
	Use:                   "goblade",
	Short:                 "Lightweight, embeddable tool for capturing FINAL FANTASY XIV network traffic.",
	Version:               "0.3.0",
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
	rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", verbose, "Log more information to stderr")
}
