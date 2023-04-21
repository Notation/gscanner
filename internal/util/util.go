package util

import (
	"encoding/hex"
	"gscanner/internal/disassembler"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func GetCodeHash(code string) (string, []byte, error) {
	data, err := hex.DecodeString(strings.TrimPrefix(code, "0x"))
	if err != nil {
		return "", nil, err
	}
	result := crypto.Keccak256(data)
	return hex.EncodeToString(result), result, nil
}

func Sha3(data string) ([]byte, error) {
	value, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(value), nil
}

func GetInstructionIndex(instructions []disassembler.EvmInstruction, address int) int {
	for index, instruction := range instructions {
		if instruction.Address >= address {
			return index
		}
	}
	return -1
}
