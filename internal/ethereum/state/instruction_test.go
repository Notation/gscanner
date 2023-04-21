package state

import (
	"encoding/hex"
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"
	"gscanner/internal/util"
	"os"
	"strings"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

// func Test_getInstructionHandlerFunc(t *testing.T) {
// 	opcode := "PUSH"
// 	ins := Instruction{
// 		OPCode: opcode,
// 	}
// 	_, err := ins.Evaluate(&GlobalState{})
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// }

func Test_Instruction(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	var (
		cfg yices2.ConfigT
		ctx yices2.ContextT
	)
	yices2.InitConfig(&cfg)
	yices2.InitContext(cfg, &ctx)

	bv1 := smt.NewBitVecValInt64(0, 256)
	bv3 := smt.NewBitVecValInt64(0, 256)
	bv2 := smt.NewBitVecValInt64(0, 32)

	f1 := yices2.BveqAtom(bv1.GetRaw(), bv3.GetRaw())
	f2 := yices2.BveqAtom(bv2.GetRaw(), yices2.Zero())

	m := smt.NewModel(ctx)
	status, _, err := m.Eval(f1)
	assert.Nil(t, err)
	assert.Equal(t, yices2.StatusSat, status)
	status, _, err = m.Eval(f2)
	assert.Nil(t, err)
	assert.Equal(t, yices2.StatusSat, status)
}

func Test_InstructionNot(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	a := smt.NewBitVecValInt64(128, 8)
	b := a.Not()
	fmt.Println(a.TermType(), a.String())
	fmt.Println(b.TermType(), b.String())
}

func Test_Ite(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	testData := smt.NewBitVecValInt64(7, 256)
	returnData := make([]*smt.BitVec, 0)
	result := make([]*smt.BitVec, 0)
	returnDataSize := smt.NewBitVec("returndatasize", 256)
	for i := 0; i < 10; i++ {
		returnData = append(returnData, smt.NewBitVec(fmt.Sprintf("test_%d", i), 256))
	}
	for i := int64(0); i < 10; i++ {
		iBv := smt.NewBitVecValInt64(i, 256)
		f1 := iBv.Lt(returnDataSize)
		result = append(result, smt.NewBitVecFromTerm(yices2.Ite(f1.GetRaw(), returnData[i].GetRaw(), testData.GetRaw()), 256))
	}
	fmt.Println(len(result))
	for i, data := range result {
		termType := yices2.TypeOfTerm(data.GetRaw())
		termSize := yices2.TermBitsize(data.GetRaw())
		fmt.Println(yices2.TermIsBitvector(data.GetRaw()))
		fmt.Println(i, "-->", data.String(), " type ", termType, " size ", termSize)
	}
}

func Test_CodeCopy(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606040")
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})
	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(2, 8))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(2, 8))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(2, 8))
	assert.Nil(t, err)
	result, err := NewInstruction("codecopy", nil, nil).Evaluate(globalState)
	assert.Nil(t, err)
	a := result.GlobalStates[0].MachineState.MemGetByteAt(smt.NewBitVecValInt64(2, 256))
	assert.NotNil(t, a)
	assert.Equal(t, int64(96), a.Value())
	b := result.GlobalStates[0].MachineState.MemGetByteAt(smt.NewBitVecValInt64(3, 256))
	assert.NotNil(t, b)
	assert.Equal(t, int64(64), b.Value())
}

func Test_Create(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606060")
	bytecode := "606060606060"
	bytes, err := hex.DecodeString(bytecode)
	assert.Nil(t, err)
	calldata := NewConcreteCalldata("1", bytes)
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
		CallData:      calldata,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(6, 8))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(0, 8))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(3, 8))
	assert.Nil(t, err)

	err = globalState.MachineState.MemExtend(0, 100)
	assert.Nil(t, err)
	dataWriteToMem := make([]*smt.BitVec, 0)
	for _, b := range bytes {
		dataWriteToMem = append(dataWriteToMem, smt.NewBitVecValInt64(int64(b), 8))
	}
	err = globalState.MachineState.MemWriteBytes(smt.NewBitVecValInt64(0, 256), dataWriteToMem...)
	assert.Nil(t, err)
	// for i := 0; i < len(bytes); i++ {
	// 	data := globalState.MachineState.MemGetByteAt(smt.NewBitVecValInt64(int64(i), 256))
	// 	typ := yices2.TypeOfTerm(data.GetRaw())
	// 	fmt.Println(data.Value(), yices2.TypeIsBitvector(typ), " ", yices2.TypeIsScalar(typ))
	// }
	result, err := NewInstruction("create", nil, nil).Evaluate(globalState)
	assert.Nil(t, err)
	assert.NotNil(t, result.TxStart)
	tx, ok := result.TxStart.Tx.(*ContractCreationTransaction)
	assert.NotNil(t, ok)
	assert.Equal(t, int64(3), tx.CallValue.Value())
	assert.Equal(t, bytecode, tx.Code.GetBytecode())
	assert.Equal(t, account.Address.HexString(), tx.CalleeAccount.Address.HexString())
}

func Test_Create2(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606060")
	bytecode := "606060606060"
	bytes, err := hex.DecodeString(bytecode)
	assert.Nil(t, err)
	calldata := NewConcreteCalldata("1", bytes)
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
		CallData:      calldata,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})

	var (
		value = smt.NewBitVecValInt64(3, 256)
		salt  = smt.NewBitVecValInt64(10, 256)
	)
	err = globalState.MachineState.PushStack(salt)
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(6, 256))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(0, 256))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(value)
	assert.Nil(t, err)

	err = globalState.MachineState.MemExtend(0, 100)
	assert.Nil(t, err)
	dataWriteToMem := make([]*smt.BitVec, 0)
	for _, b := range bytes {
		dataWriteToMem = append(dataWriteToMem, smt.NewBitVecValInt64(int64(b), 8))
	}
	err = globalState.MachineState.MemWriteBytes(smt.NewBitVecValInt64(0, 256), dataWriteToMem...)
	assert.Nil(t, err)
	// for i := 0; i < len(bytes); i++ {
	// 	data := globalState.MachineState.MemGetByteAt(smt.NewBitVecValInt64(int64(i), 256))
	// 	typ := yices2.TypeOfTerm(data.GetRaw())
	// 	fmt.Println(data.Value(), yices2.TypeIsBitvector(typ), " ", yices2.TypeIsScalar(typ))
	// }
	result, err := NewInstruction("create2", nil, nil).Evaluate(globalState)
	assert.Nil(t, err)
	assert.NotNil(t, result.TxStart)
	tx, ok := result.TxStart.Tx.(*ContractCreationTransaction)
	assert.NotNil(t, ok)
	assert.Equal(t, value.Value(), tx.CallValue.Value())
	assert.Equal(t, bytecode, tx.Code.GetBytecode())
	assert.Equal(t, account.Address.HexString(), tx.CalleeAccount.Address.HexString())
}

func Test_ExtCodecopy(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	newAccount := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	newAccount.Code = disassembler.NewDisassembly("60616240")
	extAccount := worldState.CreateAccount(1000, smt.NewBitVecValInt64(121, 256), false, nil, nil, 0)
	extAccount.Code = disassembler.NewDisassembly("6040404040")

	enviroment := &Enviroment{
		ActiveAccount: newAccount,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          newAccount.Code,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})

	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(3, 256))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(0, 256))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(0, 256))
	assert.Nil(t, err)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(121, 256))
	assert.Nil(t, err)

	result, err := NewInstruction("extcodecopy", nil, nil).Evaluate(globalState)
	assert.Nil(t, err)

	state := result.GlobalStates[0]
	a := state.MachineState.MemGetByteAt(smt.NewBitVecValInt64(0, 256))
	assert.NotNil(t, a)
	assert.Equal(t, int64(96), a.Value())
	b := state.MachineState.MemGetByteAt(smt.NewBitVecValInt64(1, 256))
	assert.NotNil(t, b)
	assert.Equal(t, int64(64), b.Value())
	c := state.MachineState.MemGetByteAt(smt.NewBitVecValInt64(2, 256))
	assert.NotNil(t, c)
	assert.Equal(t, int64(64), c.Value())
}

func Test_ExtCodehashNoAccount(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606040")
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})
	ins := NewInstruction("extcodehash", nil, nil)

	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(1, 256))
	assert.Nil(t, err)
	result, err := ins.Evaluate(globalState)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), top.Value())
}

func Test_ExtCodehashNoCode(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606040")
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})
	ins := NewInstruction("extcodehash", nil, nil)

	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(1000, 256))
	assert.Nil(t, err)
	result, err := ins.Evaluate(globalState)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
	assert.Nil(t, err)
	codeHashStr, _, err := util.GetCodeHash("")
	assert.Nil(t, err)
	resultCodeHashStr, _, err := util.GetCodeHash(top.HexString())
	assert.Equal(t, codeHashStr, resultCodeHashStr)
}

func Test_ExtCodehashAccountExists(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606040")
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})
	ins := NewInstruction("extcodehash", nil, nil)

	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(101, 256))
	assert.Nil(t, err)
	result, err := ins.Evaluate(globalState)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
	assert.Nil(t, err)
	codeHashStr, _, err := util.GetCodeHash("60606040")
	assert.Nil(t, err)
	assert.Equal(t, codeHashStr, top.HexString())
}

func Test_Ashr(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	tcs := []struct {
		Op1      string
		Op2      string
		Expected string
	}{
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x00",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x01",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0x01",
			"0xc000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0xff",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0x0100",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0x0101",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x00",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x01",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0xff",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x0100",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x01",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x4000000000000000000000000000000000000000000000000000000000000000",
			"0xfe",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0xf8",
			"0x000000000000000000000000000000000000000000000000000000000000007f",
		},
		{
			"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0xfe",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0xff",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x0100",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
	for _, tc := range tcs {
		worldState := NewWorldState()
		account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
		account.Code = disassembler.NewDisassembly("60606040")
		enviroment := &Enviroment{
			ActiveAccount: account,
			BlockNumber:   smt.NewBitVec("block_number", 256),
			ChainID:       smt.NewBitVec("chain_id", 256),
			Code:          account.Code,
		}
		globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
		globalState.TransactionStack.Push(&TxInfo{
			State: nil,
			Tx: &MessageCallTransaction{
				WorldState: worldState,
				GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
			},
		})
		ins := NewInstruction("ashr", nil, nil)
		op1, err := hex.DecodeString(strings.TrimPrefix(tc.Op1, "0x"))
		assert.Nil(t, err)
		op2, err := hex.DecodeString(strings.TrimPrefix(tc.Op2, "0x"))
		assert.Nil(t, err)
		expected, err := hex.DecodeString(strings.TrimPrefix(tc.Expected, "0x"))
		assert.Nil(t, err)
		op1Bv := smt.NewBitVecValFromBytes(op1, 256)
		op2Bv := smt.NewBitVecValFromBytes(op2, 256)
		expectedBv := smt.NewBitVecValFromBytes(expected, 256)

		fmt.Println(op1Bv.HexString(), " ", op2Bv.HexString(), " ", expectedBv.HexString())

		err = globalState.MachineState.PushStack(op2Bv)
		assert.Nil(t, err)
		err = globalState.MachineState.PushStack(op1Bv)
		assert.Nil(t, err)
		result, err := ins.Evaluate(globalState)
		assert.Nil(t, err)
		top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
		assert.Nil(t, err)
		assert.Equal(t, expectedBv.HexString(), top.HexString())
	}
}

func Test_Shl(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	tcs := []struct {
		Op1      string
		Op2      string
		Expected string
	}{
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x00",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x01",
			"0x0000000000000000000000000000000000000000000000000000000000000002",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0xff",
			"0x8000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x0100",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x0101",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x00",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x01",
			"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0xff",
			"0x8000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x0100",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x01",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x01",
			"0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
		},
	}
	for _, tc := range tcs {
		worldState := NewWorldState()
		account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
		account.Code = disassembler.NewDisassembly("60606040")
		enviroment := &Enviroment{
			ActiveAccount: account,
			BlockNumber:   smt.NewBitVec("block_number", 256),
			ChainID:       smt.NewBitVec("chain_id", 256),
			Code:          account.Code,
		}
		globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
		globalState.TransactionStack.Push(&TxInfo{
			State: nil,
			Tx: &MessageCallTransaction{
				WorldState: worldState,
				GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
			},
		})
		ins := NewInstruction("shl", nil, nil)
		op1, err := hex.DecodeString(strings.TrimPrefix(tc.Op1, "0x"))
		assert.Nil(t, err)
		op2, err := hex.DecodeString(strings.TrimPrefix(tc.Op2, "0x"))
		assert.Nil(t, err)
		expected, err := hex.DecodeString(strings.TrimPrefix(tc.Expected, "0x"))
		assert.Nil(t, err)
		op1Bv := smt.NewBitVecValFromBytes(op1, 256)
		op2Bv := smt.NewBitVecValFromBytes(op2, 256)
		expectedBv := smt.NewBitVecValFromBytes(expected, 256)

		err = globalState.MachineState.PushStack(op2Bv)
		assert.Nil(t, err)
		err = globalState.MachineState.PushStack(op1Bv)
		assert.Nil(t, err)
		result, err := ins.Evaluate(globalState)
		assert.Nil(t, err)
		top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
		assert.Nil(t, err)
		assert.Equal(t, expectedBv.HexString(), top.HexString())
	}
}

func Test_Shr(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	tcs := []struct {
		Op1      string
		Op2      string
		Expected string
	}{
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x00",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x01",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0x01",
			"0x4000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0xff",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0x0100",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x8000000000000000000000000000000000000000000000000000000000000000",
			"0x0101",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x00",
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x01",
			"0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0xff",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"0x0100",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x01",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
	for _, tc := range tcs {
		worldState := NewWorldState()
		account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
		account.Code = disassembler.NewDisassembly("60606040")
		enviroment := &Enviroment{
			ActiveAccount: account,
			BlockNumber:   smt.NewBitVec("block_number", 256),
			ChainID:       smt.NewBitVec("chain_id", 256),
			Code:          account.Code,
		}
		globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
		globalState.TransactionStack.Push(&TxInfo{
			State: nil,
			Tx: &MessageCallTransaction{
				WorldState: worldState,
				GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
			},
		})
		ins := NewInstruction("shr", nil, nil)
		op1, err := hex.DecodeString(strings.TrimPrefix(tc.Op1, "0x"))
		assert.Nil(t, err)
		op2, err := hex.DecodeString(strings.TrimPrefix(tc.Op2, "0x"))
		assert.Nil(t, err)
		expected, err := hex.DecodeString(strings.TrimPrefix(tc.Expected, "0x"))
		assert.Nil(t, err)
		op1Bv := smt.NewBitVecValFromBytes(op1, 256)
		op2Bv := smt.NewBitVecValFromBytes(op2, 256)
		expectedBv := smt.NewBitVecValFromBytes(expected, 256)

		err = globalState.MachineState.PushStack(op2Bv)
		assert.Nil(t, err)
		err = globalState.MachineState.PushStack(op1Bv)
		assert.Nil(t, err)
		result, err := ins.Evaluate(globalState)
		assert.Nil(t, err)
		top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
		assert.Nil(t, err)
		assert.Equal(t, expectedBv.HexString(), top.HexString())
	}
}

func Test_Basefee(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606040")
	fee := smt.NewBitVec("fee", 256)
	enviroment := &Enviroment{
		ActiveAccount: account,
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
		BaseFee:       fee,
	}
	globalState := NewGlobalState(worldState, enviroment, NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx: &MessageCallTransaction{
			WorldState: worldState,
			GasLimit:   smt.NewBitVecValFromInt64(8000000, 256),
		},
	})
	ins := NewInstruction("basefee", nil, nil)
	result, err := ins.Evaluate(globalState)
	assert.Nil(t, err)
	top, err := result.GlobalStates[0].MachineState.GetBitVec(0)
	assert.Nil(t, err)
	assert.Equal(t, top.GetRaw(), fee.GetRaw())
}

func Test_Basefee222(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	bytes := []byte{0x00, 0x10}
	bv := smt.NewBitVecValFromBytes(bytes, 256)
	fmt.Println(hex.EncodeToString(bytes))
	fmt.Println(hex.EncodeToString(bv.Bytes()))

	a := smt.NewBitVec("a", 256)
	t1 := yices2.Eq(a.GetRaw(), smt.NewBitVecValInt64(0, a.Size()).GetRaw())
	t2 := yices2.Not(t1)
	yices2.PpTerm(os.Stdout, t1, 1000, 80, 0)
	yices2.PpTerm(os.Stdout, t2, 1000, 80, 0)

	b := smt.NewBitVecValInt64(1, 256)
	c := smt.NewBitVecValInt64(1000, 256)
	f1 := yices2.Eq(b.GetRaw(), c.GetRaw())
	fmt.Println(f1 == yices2.True())
	fmt.Println(f1 == yices2.False())

	yices2.PpTerm(os.Stdout, b.And(c).GetRaw(), 1000, 80, 0)
}
