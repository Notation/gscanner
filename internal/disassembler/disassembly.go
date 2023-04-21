package disassembler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Disassembly 汇编信息管理
type Disassembly struct {
	bytecode          string
	instructions      []EvmInstruction
	funcHashes        []string
	funcNameToAddress map[string]int
	funcAddressToName map[int]string
}

func NewDisassembly(bytecode string) *Disassembly {
	d := Disassembly{
		bytecode:          bytecode,
		funcNameToAddress: make(map[string]int),
		funcAddressToName: make(map[int]string),
	}
	d.AssignBytecode(bytecode)
	return &d
}

func (d *Disassembly) AssignBytecode(bytecode string) error {
	instructions, err := disassemble(bytecode)
	if err != nil {
		return errors.Wrap(err, "disassemble")
	}
	d.instructions = instructions
	d.bytecode = bytecode
	jumpTableIndices := FindOPCodeSequence([][]string{{"PUSH1", "PUSH2", "PUSH3", "PUSH4"}, {"EQ"}}, d.instructions)
	for _, index := range jumpTableIndices {
		functionHash, jumpTarget, functionName := getFunctionInfo(index, d.instructions)
		d.funcHashes = append(d.funcHashes, functionHash)
		if jumpTarget != 0 && functionName != "" {
			d.funcNameToAddress[functionName] = jumpTarget
			d.funcAddressToName[jumpTarget] = functionName
		}
	}
	return nil
}

func (d *Disassembly) GetBytecode() string {
	return d.bytecode
}

func (d *Disassembly) GetEASM() string {
	return instructionListToEASM(d.instructions)
}

func (d *Disassembly) GetInstructions() []EvmInstruction {
	return d.instructions
}

func getFunctionInfo(index int, instructions []EvmInstruction) (
	string, int, string) {
	// fmt.Println(instructions[index].FormatArgument())
	var (
		funcHash        = instructions[index].FormatArgument()
		funcName        = "_function_" + funcHash
		entryPoint, err = strconv.ParseInt(strings.TrimPrefix(instructions[index+2].FormatArgument(), "0x"), 16, 64)
	)
	if err != nil {
		fmt.Println(err)
	}
	return funcHash, int(entryPoint), funcName
}

func (d *Disassembly) Clone() *Disassembly {
	result := &Disassembly{
		funcHashes:        make([]string, len(d.funcHashes)),
		funcNameToAddress: make(map[string]int),
		funcAddressToName: make(map[int]string),
		instructions:      d.instructions, // instructions不会变更，使用同一个即可
		bytecode:          d.bytecode,
	}
	copy(result.funcHashes, d.funcHashes)
	for k, v := range d.funcNameToAddress {
		result.funcNameToAddress[k] = v
	}
	for k, v := range d.funcAddressToName {
		result.funcAddressToName[k] = v
	}
	return result
}
