package opcode

import (
	"fmt"
)

// Operation EVM操作码
// https://ethereum.org/en/developers/docs/evm/opcodes
type Operation string

func (op Operation) String() string {
	return string(op)
}

const (
	STOP           Operation = "STOP"
	ADD            Operation = "ADD"
	MUL            Operation = "MUL"
	SUB            Operation = "SUB"
	DIV            Operation = "DIV"
	SDIV           Operation = "SDIV"
	MOD            Operation = "MOD"
	SMOD           Operation = "SMOD"
	ADDMOD         Operation = "ADDMOD"
	MULMOD         Operation = "MULMOD"
	EXP            Operation = "EXP"
	SIGNEXTEND     Operation = "SIGNEXTEND"
	LT             Operation = "LT"
	GT             Operation = "GT"
	SLT            Operation = "SLT"
	SGT            Operation = "SGT"
	EQ             Operation = "EQ"
	ISZERO         Operation = "ISZERO"
	AND            Operation = "AND"
	OR             Operation = "OR"
	XOR            Operation = "XOR"
	NOT            Operation = "NOT"
	BYTE           Operation = "BYTE"
	SHL            Operation = "SHL"
	SHR            Operation = "SHR"
	SAR            Operation = "SAR"
	SHA3           Operation = "SHA3"
	ADDRESS        Operation = "ADDRESS"
	BALANCE        Operation = "BALANCE"
	ORIGIN         Operation = "ORIGIN"
	CALLER         Operation = "CALLER"
	CALLVALUE      Operation = "CALLVALUE"
	CALLDATALOAD   Operation = "CALLDATALOAD"
	CALLDATASIZE   Operation = "CALLDATASIZE"
	CALLDATACOPY   Operation = "CALLDATACOPY"
	CODESIZE       Operation = "CODESIZE"
	CODECOPY       Operation = "CODECOPY"
	GASPRICE       Operation = "GASPRICE"
	EXTCODESIZE    Operation = "EXTCODESIZE"
	EXTCODECOPY    Operation = "EXTCODECOPY"
	EXTCODEHASH    Operation = "EXTCODEHASH"
	RETURNDATASIZE Operation = "RETURNDATASIZE"
	RETURNDATACOPY Operation = "RETURNDATACOPY"
	BLOCKHASH      Operation = "BLOCKHASH"
	COINBASE       Operation = "COINBASE"
	TIMESTAMP      Operation = "TIMESTAMP"
	NUMBER         Operation = "NUMBER"
	DIFFICULTY     Operation = "DIFFICULTY"
	GASLIMIT       Operation = "GASLIMIT"
	CHAINID        Operation = "CHAINID"
	SELFBALANCE    Operation = "SELFBALANCE"
	BASEFEE        Operation = "BASEFEE"
	POP            Operation = "POP"
	MLOAD          Operation = "MLOAD"
	MSTORE         Operation = "MSTORE"
	MSTORE8        Operation = "MSTORE8"
	SLOAD          Operation = "SLOAD"
	SSTORE         Operation = "SSTORE"
	JUMP           Operation = "JUMP"
	JUMPI          Operation = "JUMPI"
	PC             Operation = "PC"
	MSIZE          Operation = "MSIZE"
	GAS            Operation = "GAS"
	JUMPDEST       Operation = "JUMPDEST"
	BEGINSUB       Operation = "BEGINSUB"
	RETURNSUB      Operation = "RETURNSUB"
	JUMPSUB        Operation = "JUMPSUB"
	LOG0           Operation = "LOG0"
	LOG1           Operation = "LOG1"
	LOG2           Operation = "LOG2"
	LOG3           Operation = "LOG3"
	LOG4           Operation = "LOG4"
	CREATE         Operation = "CREATE"
	CREATE2        Operation = "CREATE2"
	CALL           Operation = "CALL"
	CALLCODE       Operation = "CALLCODE"
	RETURN         Operation = "RETURN"
	DELEGATECALL   Operation = "DELEGATECALL"
	STATICCALL     Operation = "STATICCALL"
	REVERT         Operation = "REVERT"
	SELFDESTRUCT   Operation = "SELFDESTRUCT"
	INVALID        Operation = "INVALID"
	// PUSH{1~32}
	// SWAP{1~16}
	// DUP{1~16}
)

// https://github.com/wolflo/evm-opcodes/blob/main/gas.md
type OPCodeInfo struct {
	GasMin           int
	GasMax           int
	RequiredElements int
	Address          int
	OPCode           Operation
}

var opCodeInfos = map[Operation]OPCodeInfo{
	STOP: {
		GasMin:           0,
		GasMax:           0,
		RequiredElements: 0,
		Address:          0x00,
	},
	ADD: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x01,
	},
	MUL: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 2,
		Address:          0x02,
	},
	SUB: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x03,
	},
	DIV: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 2,
		Address:          0x04,
	},
	SDIV: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 2,
		Address:          0x05,
	},
	MOD: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 2,
		Address:          0x06,
	},
	SMOD: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 2,
		Address:          0x07,
	},
	ADDMOD: {
		GasMin:           8,
		GasMax:           8,
		RequiredElements: 2,
		Address:          0x08,
	},
	MULMOD: {
		GasMin:           8,
		GasMax:           8,
		RequiredElements: 3,
		Address:          0x09,
	},
	EXP: {
		GasMin:           10,
		GasMax:           340,
		RequiredElements: 2,
		Address:          0x0A,
	},
	SIGNEXTEND: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 2,
		Address:          0x0B,
	},
	LT: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x10,
	},
	GT: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x11,
	},
	SLT: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x12,
	},
	SGT: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x13,
	},
	EQ: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x14,
	},
	ISZERO: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 1,
		Address:          0x15,
	},
	AND: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x16,
	},
	OR: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x17,
	},
	XOR: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x18,
	},
	NOT: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 1,
		Address:          0x19,
	},
	BYTE: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x1A,
	},
	SHL: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x1B,
	},
	SHR: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x1C,
	},
	SAR: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 2,
		Address:          0x1D,
	},
	SHA3: {
		GasMin:           30,
		GasMax:           30 + 6*8,
		RequiredElements: 2,
		Address:          0x20,
	},
	ADDRESS: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x30,
	},
	BALANCE: {
		GasMin:           700,
		GasMax:           700,
		RequiredElements: 1,
		Address:          0x31,
	},
	ORIGIN: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x32,
	},
	CALLER: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x33,
	},
	CALLVALUE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x34,
	},
	CALLDATALOAD: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 1,
		Address:          0x35,
	},
	CALLDATASIZE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x36,
	},
	CALLDATACOPY: {
		GasMin:           2,
		GasMax:           2 + 3*768, // https://ethereum.stackexchange.com/a/47556
		RequiredElements: 0,
		Address:          0x37,
	},
	CODESIZE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x38,
	},
	CODECOPY: {
		GasMin:           2,
		GasMax:           2 + 3*768, // https://ethereum.stackexchange.com/a/47556
		RequiredElements: 0,
		Address:          0x39,
	},
	GASPRICE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x3A,
	},
	EXTCODESIZE: {
		GasMin:           700,
		GasMax:           700,
		RequiredElements: 0,
		Address:          0x3B,
	},
	EXTCODECOPY: {
		GasMin:           700,
		GasMax:           700 + 3*768, // https://ethereum.stackexchange.com/a/47556
		RequiredElements: 4,
		Address:          0x3C,
	},
	RETURNDATASIZE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x3D,
	},
	RETURNDATACOPY: {
		GasMin:           3,
		GasMax:           3,
		RequiredElements: 3,
		Address:          0x3E,
	},
	EXTCODEHASH: {
		GasMin:           700,
		GasMax:           700,
		RequiredElements: 1,
		Address:          0x3F,
	},
	BLOCKHASH: {
		GasMin:           20,
		GasMax:           20,
		RequiredElements: 1,
		Address:          0x40,
	},
	COINBASE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x41,
	},
	TIMESTAMP: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x42,
	},
	NUMBER: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x43,
	},
	DIFFICULTY: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x44,
	},
	GASLIMIT: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x45,
	},
	CHAINID: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x46,
	},
	SELFBALANCE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x47,
	},
	BASEFEE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x48,
	},
	POP: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 1,
		Address:          0x50,
	},
	MLOAD: {
		GasMin:           3,
		GasMax:           96,
		RequiredElements: 1,
		Address:          0x51,
	},
	MSTORE: {
		GasMin:           3,
		GasMax:           98,
		RequiredElements: 2,
		Address:          0x52,
	},
	MSTORE8: {
		GasMin:           3,
		GasMax:           98,
		RequiredElements: 2,
		Address:          0x53,
	},
	SLOAD: {
		GasMin:           800,
		GasMax:           800,
		RequiredElements: 1,
		Address:          0x54,
	},
	SSTORE: {
		GasMin:           5000,
		GasMax:           5000,
		RequiredElements: 1,
		Address:          0x55,
	},
	JUMP: {
		GasMin:           8,
		GasMax:           8,
		RequiredElements: 1,
		Address:          0x56,
	},
	JUMPI: {
		GasMin:           10,
		GasMax:           10,
		RequiredElements: 2,
		Address:          0x57,
	},
	PC: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x58,
	},
	MSIZE: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x59,
	},
	GAS: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x5A,
	},
	JUMPDEST: {
		GasMin:           1,
		GasMax:           1,
		RequiredElements: 0,
		Address:          0x5B,
	},
	BEGINSUB: {
		GasMin:           2,
		GasMax:           2,
		RequiredElements: 0,
		Address:          0x5C,
	},
	RETURNSUB: {
		GasMin:           5,
		GasMax:           5,
		RequiredElements: 0,
		Address:          0x5D,
	},
	JUMPSUB: {
		GasMin:           10,
		GasMax:           10,
		RequiredElements: 1,
		Address:          0x5E,
	},
	LOG0: {
		GasMin:           375,
		GasMax:           375 + 8*32, // https://ethereum.stackexchange.com/a/1691
		RequiredElements: 2,
		Address:          0xA0,
	},
	LOG1: {
		GasMin:           2 * 375,
		GasMax:           2*375 + 8*32,
		RequiredElements: 3,
		Address:          0xA1,
	},
	LOG2: {
		GasMin:           3 * 375,
		GasMax:           3*375 + 8*32,
		RequiredElements: 4,
		Address:          0xA2,
	},
	LOG3: {
		GasMin:           4 * 375,
		GasMax:           4*375 + 8*32,
		RequiredElements: 5,
		Address:          0xA3,
	},
	LOG4: {
		GasMin:           5 * 375,
		GasMax:           5*375 + 8*32,
		RequiredElements: 6,
		Address:          0xA4,
	},
	CREATE: {
		GasMin:           32000,
		GasMax:           32000,
		RequiredElements: 3,
		Address:          0xF0,
	},
	CREATE2: {
		GasMin:           32000,
		GasMax:           32000,
		RequiredElements: 4,
		Address:          0xF5,
	},
	CALL: {
		GasMin:           700,
		GasMax:           700 + 9000 + 25000,
		RequiredElements: 7,
		Address:          0xF1,
	},
	CALLCODE: {
		GasMin:           700,
		GasMax:           700 + 9000 + 25000,
		RequiredElements: 7,
		Address:          0xF2,
	},
	RETURN: {
		GasMin:           0,
		GasMax:           0,
		RequiredElements: 2,
		Address:          0xF3,
	},
	DELEGATECALL: {
		GasMin:           700,
		GasMax:           700 + 9000 + 25000,
		RequiredElements: 6,
		Address:          0xF4,
	},
	STATICCALL: {
		GasMin:           700,
		GasMax:           700 + 9000 + 25000,
		RequiredElements: 6,
		Address:          0xFA,
	},
	REVERT: {
		GasMin:           0,
		GasMax:           0,
		RequiredElements: 2,
		Address:          0xFD,
	},
	SELFDESTRUCT: {
		GasMin:           5000,
		GasMax:           30000,
		RequiredElements: 1,
		Address:          0xFF,
	},
	INVALID: {
		GasMin:           0,
		GasMax:           0,
		RequiredElements: 0,
		Address:          0xFE,
	},
}

var opCodes map[int]OPCodeInfo

func init() {
	// PUSH{1~32}
	for i := 1; i < 33; i++ {
		opCodeInfos[Operation(fmt.Sprintf("PUSH%d", i))] = OPCodeInfo{
			GasMin:           3,
			GasMax:           3,
			RequiredElements: 0,
			Address:          0x5F + i,
		}
	}
	// SWAP{1~16} DUP{1~16}
	for i := 1; i < 17; i++ {
		opCodeInfos[Operation(fmt.Sprintf("DUP%d", i))] = OPCodeInfo{
			GasMin:           3,
			GasMax:           3,
			RequiredElements: 0,
			Address:          0x7F + i,
		}
		opCodeInfos[Operation(fmt.Sprintf("SWAP%d", i))] = OPCodeInfo{
			GasMin:           3,
			GasMax:           3,
			RequiredElements: 0,
			Address:          0x8F + i,
		}
	}
	for k, info := range opCodeInfos {
		info.OPCode = k
		opCodeInfos[k] = info
	}

	opCodes = make(map[int]OPCodeInfo)
	for _, info := range opCodeInfos {
		opCodes[info.Address] = info
	}
}

func GetOPCodeInfoByAddress(key int) (OPCodeInfo, bool) {
	info, ok := opCodes[key]
	return info, ok
}

func GetOPCodeInfoByOperation(key Operation) (OPCodeInfo, bool) {
	info, ok := opCodeInfos[key]
	return info, ok
}
