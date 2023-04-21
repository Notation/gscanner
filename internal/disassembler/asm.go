package disassembler

import (
	"encoding/hex"
	"gscanner/internal/opcode"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const PatternPush = `^PUSH(\d*)$`

var regPush *regexp.Regexp

func init() {
	regPush, _ = regexp.Compile(PatternPush)
}

type EvmInstruction struct {
	GasMin            int64  // 最小gas消耗量
	GasMax            int64  // 最大gas消耗量
	Address           int    // 地址
	RequiredArguments int    // 指令参数数量
	OPCode            string // OPCode
	Argument          []byte // 指令参数
}

func (ei *EvmInstruction) String() string {
	var builder strings.Builder
	builder.WriteString(strconv.Itoa(ei.Address))
	builder.WriteString(" ")
	builder.WriteString(ei.OPCode)
	if len(ei.Argument) > 0 {
		builder.WriteString(" ")
		builder.Write([]byte("0x" + hex.EncodeToString(ei.Argument)))
	}
	builder.WriteString("\n")
	return builder.String()
}

// FormatArgument 将argument格式化成16进制的字符串
// 格式化之后的长度至少是8，不足补0
// 如[0x1,0x2]格式化之后为0x00000102
func (ei *EvmInstruction) FormatArgument() string {
	if len(ei.Argument) <= 0 {
		return ""
	}
	var (
		prefix     = "0x"
		placehoder = "0"
		data       = hex.EncodeToString(ei.Argument)
	)
	if len(ei.Argument) < 8 {
		var builder strings.Builder
		builder.WriteString(prefix)
		if len(data) > 8 {
			builder.WriteString(strings.Repeat(placehoder, 16-len(data)))
		} else {
			builder.WriteString(strings.Repeat(placehoder, 8-len(data)))
		}
		builder.WriteString(data)
		return builder.String()
	}
	return prefix + data
}

func instructionListToEASM(instructions []EvmInstruction) string {
	var builder strings.Builder
	for i := range instructions {
		builder.WriteString(strconv.Itoa(instructions[i].Address))
		builder.WriteString(" ")
		builder.WriteString(instructions[i].OPCode)
		if len(instructions[i].Argument) > 0 {
			builder.WriteString(" ")
			builder.Write([]byte("0x" + hex.EncodeToString(instructions[i].Argument)))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

// patterns从0开始，instructions从index开始，依次匹配
func isSequenceMatch(patterns [][]string, instructions []EvmInstruction, index int) bool {
	for i, pattern := range patterns {
		if index+i >= len(instructions) {
			return false
		}
		var foundOPCode bool
		for _, p := range pattern {
			if instructions[index+i].OPCode == p {
				foundOPCode = true
			}
		}
		if !foundOPCode {
			return false
		}
	}
	return true
}

func FindOPCodeSequence(patterns [][]string, instructions []EvmInstruction) []int {
	result := make([]int, 0)
	for i := 0; i < len(instructions)-len(patterns)+1; i++ {
		if isSequenceMatch(patterns, instructions, i) {
			result = append(result, i)
		}
	}
	return result
}

// disassemble 解码code为EVMInstruction
// data 仅支持hex string
func disassemble(data string) (instructions []EvmInstruction, err error) {
	var (
		bytecode = []byte(data)
	)
	bytecode, err = hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		return nil, errors.Wrap(err, "decode string")
	}
	var (
		length   = len(bytecode)
		partCode string
		address  int
	)
	if length >= 43 {
		partCode = string(bytecode[length-43:])
	}
	if strings.Contains(partCode, "bzzr") {
		length -= 43
	}
	for {
		if address >= length {
			break
		}
		opCodeInfo, ok := opcode.GetOPCodeInfoByAddress(int(bytecode[address]))
		if !ok {
			instructions = append(instructions, []EvmInstruction{{
				Address: address,
				OPCode:  "INVALID",
			}}...)
			address++
			continue
		}
		currentInstruction := EvmInstruction{
			Address:           address,
			OPCode:            string(opCodeInfo.OPCode),
			GasMin:            int64(opCodeInfo.GasMin),
			GasMax:            int64(opCodeInfo.GasMax),
			RequiredArguments: opCodeInfo.RequiredElements,
			Argument:          getPUSHArguments(opCodeInfo.OPCode.String(), bytecode, address),
		}
		instructions = append(instructions, []EvmInstruction{currentInstruction}...)
		address++
		address += len(currentInstruction.Argument)
	}
	return instructions, nil
}

// getPUSHArguments 获取PUSH指令的参数
// PUSH指令处理 eg.
// PUSH1 0x80
// PUSH21 0x11B464736F6C634300081100330000000000000000
// 这里取PUSH后面的数字，即参数数量，然后把参数取出来
func getPUSHArguments(opCode string, bytecode []byte, address int) []byte {
	if !regPush.Match([]byte(opCode)) {
		return nil
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(opCode, "PUSH"))
	arugments := make([]byte, n)
	copy(arugments, bytecode[address+1:address+1+n])
	return arugments
}
