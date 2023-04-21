package module

import (
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/ethereum/state"
	"gscanner/internal/smt"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

func Test_UniqueJump(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	worldState := state.NewWorldState()
	account := worldState.CreateAccount(10, smt.NewBitVecValInt64(101, 256), false, nil, nil, 0)
	account.Code = disassembler.NewDisassembly("60606040")
	enviroment := &state.Enviroment{
		ActiveAccount: account,
		CallData:      state.NewSymbolicCalldata("2"),
		BlockNumber:   smt.NewBitVec("block_number", 256),
		ChainID:       smt.NewBitVec("chain_id", 256),
		Code:          account.Code,
	}
	globalState := state.NewGlobalState(worldState, enviroment, state.NewMachineState(8000000), nil, nil)
	globalState.TransactionStack.Push(&state.TxInfo{
		State: nil,
		Tx: &state.MessageCallTransaction{
			WorldState:    worldState,
			GasLimit:      smt.NewBitVecValFromInt64(8000000, 256),
			Calldata:      state.NewSymbolicCalldata("2"),
			CallValue:     smt.NewBitVec("call_value", 256),
			Caller:        smt.NewBitVecValFromInt64(8888, 256),
			CalleeAccount: account,
		},
	})

	f := smt.NewBitVec("jump_dest", 256)
	err := globalState.MachineState.PushStack(f)
	assert.Nil(t, err)

	term1 := f.Eq(smt.NewBitVecValFromInt64(666, 256))
	globalState.WorldState.SetConstraint(state.NewConstraints(*term1))

	arbitraryJump := NewArbitraryJump()
	result, err := arbitraryJump.Execute(globalState)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(result))

	term2 := f.Gt(smt.NewBitVecValFromInt64(66, 256))
	globalState.WorldState.SetConstraint(state.NewConstraints(*term2))
	result, err = arbitraryJump.Execute(globalState)
	assert.Nil(t, err)
	assert.Greater(t, len(result), 0)
}

func Test_UniqueJumpRT(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	// a := smt.NewBitVecValInt64(1, 256)
	b := smt.NewBitVec("b", 256)
	c := smt.NewBitVecValFromInt64(3, 256)

	f1 := b.Ge(c)

	solver := smt.NewSolver()
	_, model, err := solver.Check(f1.GetRaw())
	if err != nil {
		fmt.Println(err)
		fmt.Println()
	}
	assert.NotNil(t, model)
	assert.NotNil(t, model)
}
