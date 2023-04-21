package main

import (
	"fmt"
	funcmanager "gscanner/internal/ethereum/function_managers"
	"gscanner/internal/ethereum/state"
	"gscanner/internal/gscanner"
	"gscanner/internal/module"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/spf13/cobra"
)

var analyzeCommand = &cobra.Command{
	Use:   "analyze",
	Short: "analyze contract",
	Long:  ``,
	Run: func(*cobra.Command, []string) {
		if err := analyzeExec(); err != nil {
			fmt.Printf("service err: %v", err)
		} else {
			fmt.Printf("service quit")
		}
	},
}

func init() {
	analyzeCommand.Flags().StringVar(&SolidityFile, "file", "", "disassemble file")
}

func analyzeExec() error {
	fmt.Printf("analyze exec\n")
	yices2.Init()
	defer yices2.Exit()

	state.Init()
	funcmanager.Init()

	disassembler := gscanner.NewDisassembler()
	err := disassembler.LoadFromSolidity([]string{SolidityFile})
	if err != nil {
		fmt.Println("disassembler.LoadFromSolidity ", err)
		return err
	}
	moduleManager := module.NewModuleManager()
	moduleManager.AddModule(module.NewArbitraryJump())
	moduleManager.AddModule(module.NewTxOrigin())
	moduleManager.AddModule(module.NewUncheckedRetval())
	moduleManager.AddModule(module.NewAccidentallyKillable())
	analyzer := gscanner.NewAnalyzer(moduleManager, disassembler)
	return analyzer.Run()
}
