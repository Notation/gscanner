package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	BuildBranch  string
	BuildVersion string
	BuildTime    string
	Builder      string
)

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "show version",
	Long:  ``,
	Run: func(*cobra.Command, []string) {
		if err := printVersion(); err != nil {
			fmt.Printf("service err: %v", err)
		} else {
			fmt.Printf("service quit")
		}
	},
}

func printVersion() error {
	fmt.Printf("\033[36m%-16s\033[0m %s\n", "BuildBranch", BuildBranch)
	fmt.Printf("\033[36m%-16s\033[0m %s\n", "BuildVersion", BuildVersion)
	fmt.Printf("\033[36m%-16s\033[0m %s\n", "BuildTime", BuildTime)
	fmt.Printf("\033[36m%-16s\033[0m %s\n", "Builder", Builder)
	return nil
}
