package state

import (
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

const FunctionHashByteLength = 4

var Actors = map[string]*smt.BitVec{}

func Init() {
	Actors = map[string]*smt.BitVec{
		"CREATOR":  smt.NewBitVecValFromInt64(9999, 256),
		"ATTACKER": smt.NewBitVecValFromInt64(8888, 256),
		"SOMEGUY":  smt.NewBitVecValFromInt64(7777, 256),
	}
}

func GenerateFunctionConstraints(calldata Calldata, funcHashes [][]int64) ([]smt.Bool, error) {
	if len(funcHashes) == 0 {
		return nil, nil
	}
	constraints := make([]smt.Bool, 0)
	for i := 0; i < FunctionHashByteLength; i++ {
		constraint := smt.NewBoolVal(false)
		for _, funcHash := range funcHashes {
			data, err := calldata.GetByteAt(smt.NewBitVecValInt64(int64(i), 256))
			if err != nil {
				return nil, err
			}
			term := yices2.Or2(constraint.GetRaw(), data.Eq(smt.NewBitVecValInt64(funcHash[i], 256)).GetRaw())
			constraint = smt.NewBoolFromTerm(term)
		}
		constraints = append(constraints, constraint)
	}
	return constraints, nil
}

func PrepareMessageCall(worldStates []*WorldState, calleeAddress *smt.BitVec, funcHashes [][]int64) (result []*GlobalState, err error) {
	for _, worldState := range worldStates {
		address := calleeAddress.Value()
		calleeAccount := worldState.GetAccount(address)
		if calleeAccount.Deleted {
			fmt.Println("skipped dead contract")
			continue
		}
		nextTxID := GetNextTxID()
		externalSender := smt.NewBitVec(fmt.Sprintf("sender_%s", nextTxID), 256)
		calldata := NewSymbolicCalldata(fmt.Sprintf("%s", nextTxID))
		tx := &MessageCallTransaction{
			WorldState:    worldState,
			ID:            nextTxID,
			GasPrice:      smt.NewBitVec(fmt.Sprintf("gas_price%s", nextTxID), 256),
			GasLimit:      smt.NewBitVecValFromInt64(8000000, 256),
			Origin:        externalSender,
			Caller:        externalSender,
			CalleeAccount: calleeAccount,
			Code:          calleeAccount.Code.Clone(),
			Calldata:      calldata,
			CallValue:     smt.NewBitVec(fmt.Sprintf("call_value%s", nextTxID), 256),
		}
		var (
			constraints []smt.Bool
			err         error
		)
		if funcHashes != nil {
			constraints, err = GenerateFunctionConstraints(calldata, funcHashes)
			if err != nil {
				return nil, err
			}
		}
		globalState, err := setupGlobalStateForExecution(tx, constraints)
		if err != nil {
			return nil, err
		}
		result = append(result, globalState)
	}
	return result, nil
}

func PrepareContractCreation(
	contractCreationCode string,
	contractName string,
	worldState *WorldState,
) (*GlobalState, *Account, error) {
	openState := worldState
	if openState == nil {
		openState = NewWorldState()
	}
	nextTxID := GetNextTxID()
	calleeAccount := openState.CreateConcreteStorageAccount(
		0,
		nil,
		true,
		Actors["CREATOR"].Clone().AsBitVec(),
		disassembler.NewDisassembly(contractCreationCode),
		0)
	tx := &ContractCreationTransaction{
		PrevWorldState: openState.Clone(),
		WorldState:     openState,
		ID:             nextTxID,
		GasPrice:       smt.NewBitVec(fmt.Sprintf("gas_price%s", nextTxID), 256),
		GasLimit:       smt.NewBitVecValFromInt64(8000000, 256),
		Code:           disassembler.NewDisassembly(contractCreationCode),
		Caller:         Actors["CREATOR"].Clone().AsBitVec(),
		CalleeAccount:  calleeAccount,
		Origin:         Actors["CREATOR"].Clone().AsBitVec(),
		ContractName:   contractName,
		Calldata:       NewSymbolicCalldata(nextTxID),
		CallValue:      smt.NewBitVec(fmt.Sprintf("call_value%s", nextTxID), 256),
	}
	globalState, err := setupGlobalStateForExecution(tx, nil)
	if err != nil {
		return nil, nil, err
	}
	return globalState, tx.CalleeAccount, nil
}

func setupGlobalStateForExecution(tx Transaction, initialConstrains []smt.Bool) (*GlobalState, error) {
	globalState, err := tx.InitialGlobalState()
	if err != nil {
		return nil, err
	}
	globalState.TransactionStack.Push(&TxInfo{
		State: nil,
		Tx:    tx,
	})
	globalState.WorldState.AddConstraints(initialConstrains...)
	globalState.WorldState.TransactionSequence = append(globalState.WorldState.TransactionSequence, tx)
	term := yices2.Or3(Actors["CREATOR"].Eq(tx.GetCaller()).GetRaw(), Actors["ATTACKER"].Eq(tx.GetCaller()).GetRaw(), Actors["SOMEGUY"].Eq(tx.GetCaller()).GetRaw())
	constraint := smt.NewBoolFromTerm(term)
	globalState.WorldState.AddConstraint(constraint)
	return globalState, nil
}
