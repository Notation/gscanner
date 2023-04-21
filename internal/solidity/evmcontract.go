package solidity

import (
	"gscanner/internal/disassembler"
	"gscanner/internal/util"
	"regexp"
	"strings"
)

const (
	ContractAddressPattern = `(_{2}.{38})`
)

var regCode *regexp.Regexp

func init() {
	regCode, _ = regexp.Compile(ContractAddressPattern)
}

type EVMContract struct {
	Name string

	Code        string
	Disassembly *disassembler.Disassembly

	CreationCode        string
	CreationDisassembly *disassembler.Disassembly
}

func NewEVMContract(code, creationCode, name string) *EVMContract {
	return &EVMContract{
		Name:                name,
		Code:                replaceAddress(code),
		Disassembly:         disassembler.NewDisassembly(code),
		CreationCode:        replaceAddress(creationCode),
		CreationDisassembly: disassembler.NewDisassembly(creationCode),
	}
}

func (c *EVMContract) BytecodeHash() (string, error) {
	codeStr, _, err := util.GetCodeHash(c.Code)
	return codeStr, err
}

func (c *EVMContract) CreationCodeHash() (string, error) {
	codeStr, _, err := util.GetCodeHash(c.CreationCode)
	return codeStr, err
}

func (c *EVMContract) GetEASM() string {
	return c.Disassembly.GetEASM()
}

func (c *EVMContract) GetCreationEASM() string {
	return c.CreationDisassembly.GetEASM()
}

func replaceAddress(code string) string {
	return regCode.ReplaceAllString(code, strings.Repeat("aa", 20))
}
