package state

import (
	"fmt"
	"gscanner/internal/smt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	log "github.com/sirupsen/logrus"
)

const (
	SymbolicCalldataSize = 320
)

type CallParameters struct {
	CalleeAddress   *smt.BitVec
	CalleeAccount   *Account
	Value           *smt.BitVec
	Gas             *smt.BitVec
	MemoryOutOffset *smt.BitVec
	MemoryOutSize   *smt.BitVec
	Calldata        Calldata
}

func GetCallParameters(globalState *GlobalState, withValue bool) (*CallParameters, error) {
	gas, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	to, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	var value *smt.BitVec
	if withValue {
		value, err = globalState.MachineState.PopBitVec()
		if err != nil {
			return nil, err
		}
	}
	memoryInputOffset, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	memoryInputSize, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	memoryOutOffset, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	memoryOutSize, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}

	calldata := getCalldata(globalState, memoryInputOffset, memoryInputSize)
	calleeAddress := getCalleeAddress(globalState, to)
	calleeAccount := getCalleeAccount(globalState, calleeAddress)

	if value != nil {
		condition := value.Gt(smt.NewBitVecValInt64(0, value.Size()))
		term := yices2.Ite(condition.GetRaw(),
			value.AddInt64(int64(params.CallStipend)).GetRaw(),
			smt.NewBitVecValInt64(0, value.Size()).GetRaw())
		value = smt.NewBitVecFromTerm(term, value.Size())
	}

	return &CallParameters{
		CalleeAddress:   calleeAddress,
		CalleeAccount:   calleeAccount,
		Value:           value,
		Gas:             gas,
		MemoryOutOffset: memoryOutOffset,
		MemoryOutSize:   memoryOutSize,
		Calldata:        calldata,
	}, nil
}

func getCalleeAddress(globalState *GlobalState, symbolicToAddress *smt.BitVec) *smt.BitVec {
	// var (
	// 	enviroment    = globalState.Enviroment
	// 	calleeAddress string
	// )

	// if !symbolicToAddress.IsSymbolic() {
	// 	return fmt.Sprintf("0x%040s", symbolicToAddress.String())
	// }
	// fmt.Println("symbolic call")

	return symbolicToAddress
}

func getCalldata(globalState *GlobalState, memOffset, size *smt.BitVec) Calldata {
	txInfo := globalState.GetCurrentTransaction()
	txID := fmt.Sprintf("%s_internalcall", txInfo.Tx.GetTxID())
	if size.IsSymbolic() {
		size = smt.NewBitVecValInt64(SymbolicCalldataSize, 256)
	}
	if memOffset.IsSymbolic() {
		log.Infof("symbolic offset is unsupported")
		return NewSymbolicCalldata(txID)
	}

	var (
		data         = make([]byte, 0)
		concreteSize = size.Value()
	)
	for i := int64(0); i < concreteSize; i++ {
		bv := globalState.MachineState.MemGetByteAt(memOffset.AddInt64(i))
		if bv.Size() != 8 {
			log.Errorf("bv bytesize should be 8 but got %d", bv.Size())
		}
		data = append(data, byte(bv.Value()))
	}
	return NewConcreteCalldata(txID, data)
}

func getCalleeAccount(globalState *GlobalState, calleeAddress *smt.BitVec) *Account {
	if calleeAddress.IsSymbolic() {
		return NewAccount(calleeAddress, nil, globalState.WorldState.balances, 0, "", false)
	}
	return globalState.WorldState.AccountsExistOrLoad(calleeAddress)
}

func insertReturnValue(globalState *GlobalState) error {
	instruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("retval_%d", instruction.Address)
	retval := globalState.NewBitVec(name, 256)
	err = globalState.MachineState.PushStack(retval)
	if err != nil {
		return err
	}
	globalState.WorldState.AddConstraint(*retval.Eq(smt.NewBitVecValInt64(0, 256)))
	return nil
}

func NativeCall(globalState *GlobalState,
	calleeAddress *smt.BitVec,
	calldata Calldata,
	memoryOutOffset, memoryOutSize *smt.BitVec) ([]*GlobalState, error) {
	if calleeAddress.IsSymbolic() {
		return nil, nil
	}
	address := calleeAddress.Value()
	if address > int64(len(vm.PrecompiledContractsIstanbul)) || address <= 0 {
		return nil, nil
	}
	fmt.Printf("native contract call: %d", address)
	if memoryOutOffset.IsSymbolic() || memoryOutSize.IsSymbolic() {
		fmt.Printf("call with symbolic offset or size is unsupported")
		return []*GlobalState{globalState}, nil
	}
	var (
		offset = memoryOutOffset.Value()
		size   = memoryOutSize.Value()
	)
	gasMin, gasMax := CalculateNativeGas(
		globalState.MachineState.CalcExtensionSize(offset, size),
		nativeFuncName[common.BytesToAddress([]byte{byte(address)})],
	)
	globalState.MachineState.gasUsedMin += gasMin
	globalState.MachineState.gasUsedMax += gasMax

	data, err := nativeContractCall(address, calldata)
	if err != nil {
		for i := int64(0); i < size; i++ {
			k := smt.NewBitVecValInt64(offset+i, 256)
			name := fmt.Sprintf("%s(%s)", nativeFuncName[common.BytesToAddress([]byte{byte(address)})], "")
			v := globalState.NewBitVec(name, 8)
			err := globalState.MachineState.MemWriteByte(k, v)
			if err != nil {
				return nil, err
			}
		}
		err := insertReturnValue(globalState)
		if err != nil {
			return nil, err
		}
		return []*GlobalState{globalState}, nil
	}
	err = insertReturnValue(globalState)
	if err != nil {
		return nil, err
	}
	var minSize = int64(len(data))
	if minSize > size {
		minSize = size
	}
	values := make([]*smt.BitVec, minSize)
	for i := int64(0); i < minSize; i++ {
		s := new(big.Int)
		s.SetBytes([]byte{data[i]})
		values[i] = smt.NewBitVecValFromBigInt(s, 8)
	}
	err = globalState.MachineState.MemWriteBytes(smt.NewBitVecValInt64(offset, 256), values...)
	if err != nil {
		return nil, err
	}
	return []*GlobalState{globalState}, nil
}
