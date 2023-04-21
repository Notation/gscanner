package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:   "gscanner",
	Short: "gscanner, solidity security scanner base symbolic execution",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func main() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	rootCmd.AddCommand(versionCommand)
	rootCmd.AddCommand(analyzeCommand)
	rootCmd.AddCommand(disassembleCommand)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
