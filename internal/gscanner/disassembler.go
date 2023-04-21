package gscanner

import (
	"gscanner/internal/solidity"
)

type Disassembler struct {
	solcBinary  string
	solcVersion string
	contracts   []*solidity.SolidityContract
}

func NewDisassembler() *Disassembler {
	disassembler := &Disassembler{}
	return disassembler
}

func (md *Disassembler) GetContracts() []*solidity.SolidityContract {
	return md.contracts
}

func (md *Disassembler) LoadFromSolidity(solidityFiles []string) error {
	for _, file := range solidityFiles {
		contracts, err := solidity.GetConstractsFromFile(file)
		if err != nil {
			return err
		}
		md.contracts = append(md.contracts, contracts...)
	}
	return nil
}
