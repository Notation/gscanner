package main

import (
	"fmt"
	"gscanner/internal/gscanner"

	"github.com/spf13/cobra"
)

var disassembleCommand = &cobra.Command{
	Use:   "disassemble",
	Short: "disassemble file and print easm",
	Long:  ``,
	Run: func(*cobra.Command, []string) {
		if err := disassemble(); err != nil {
			fmt.Printf("service err: %v", err)
		} else {
			fmt.Printf("service quit")
		}
	},
}

var (
	SolidityFile string
)

func init() {
	disassembleCommand.Flags().StringVar(&SolidityFile, "file", "", "disassemble file")
}

func disassemble() error {
	fmt.Printf("disassemble\n")

	dis := gscanner.NewDisassembler()
	err := dis.LoadFromSolidity([]string{SolidityFile})
	if err != nil {
		return err
	}
	fmt.Println("Disassembled runtime code:")
	fmt.Println(dis.GetContracts()[0].GetEASM())
	fmt.Println("Disassembled creation code:")
	fmt.Println(dis.GetContracts()[0].GetCreationEASM())

	return nil
}
