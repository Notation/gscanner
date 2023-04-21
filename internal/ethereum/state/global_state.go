package state

import (
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"

	log "github.com/sirupsen/logrus"
)

type TxInfo struct {
	State *GlobalState
	Tx    Transaction
}

func (txInfo *TxInfo) Clone() *TxInfo {
	result := &TxInfo{
		// 注意，这里没有复制state
		State: txInfo.State,
		Tx:    txInfo.Tx.Clone(),
	}
	return result
}

type TransactionStack struct {
	stack []*TxInfo
}

func NewTransactionStack() *TransactionStack {
	return &TransactionStack{
		stack: make([]*TxInfo, 0),
	}
}

func (ts *TransactionStack) Push(tx *TxInfo) {
	ts.stack = append(ts.stack, tx)
}

func (ts *TransactionStack) Size() int {
	return len(ts.stack)
}

func (ts *TransactionStack) Pop() (*TxInfo, error) {
	if len(ts.stack) == 0 {
		return nil, fmt.Errorf("stack underflow")
	}
	tx := ts.stack[len(ts.stack)]
	ts.stack = ts.stack[:len(ts.stack)-1]
	return tx, nil
}

func (ts *TransactionStack) Top() (*TxInfo, error) {
	if len(ts.stack) == 0 {
		return nil, fmt.Errorf("stack underflow")
	}
	return ts.stack[len(ts.stack)-1], nil
}

func (ts *TransactionStack) Clone() *TransactionStack {
	result := &TransactionStack{
		stack: make([]*TxInfo, len(ts.stack)),
	}
	for i, txInfo := range ts.stack {
		result.stack[i] = txInfo.Clone()
	}
	return result
}

type GlobalState struct {
	WorldState       *WorldState
	Enviroment       *Enviroment
	MachineState     *MachineState
	LastReturnData   *ReturnData
	TransactionStack *TransactionStack
	Annotations      []smt.Annotation
	Accounts         map[int64]*Account
}

func NewGlobalState(worldState *WorldState,
	enviroment *Enviroment,
	machineState *MachineState,
	txStack []Transaction,
	lastReturnData *ReturnData,
	Annotations ...smt.Annotation) *GlobalState {
	gs := &GlobalState{
		WorldState: worldState,
		Enviroment: enviroment,
		TransactionStack: &TransactionStack{
			make([]*TxInfo, 0),
		},
		LastReturnData: lastReturnData,
		Annotations:    Annotations,
		Accounts:       make(map[int64]*Account),
	}
	if machineState != nil {
		gs.MachineState = machineState
	} else {
		gs.MachineState = NewMachineState(1000000000)
	}
	return gs
}

func (gs *GlobalState) IncreaseDepth() {
	gs.MachineState.depth += 1
}

func (gs *GlobalState) GasUsedMinAdd(gas int64) {
	gs.MachineState.GasUsedMinAdd(gas)
}

func (gs *GlobalState) GasUsedMaxAdd(gas int64) {
	gs.MachineState.GasUsedMaxAdd(gas)
}

func (gs *GlobalState) GasUsedAdd(gasMin, gasMax int64) {
	gs.MachineState.GasUsedMinAdd(gasMin)
	gs.MachineState.GasUsedMaxAdd(gasMax)
}

func (gs *GlobalState) AddAnnotations(Annotations ...smt.Annotation) {
	gs.Annotations = append(gs.Annotations, Annotations...)
}

func (gs *GlobalState) Clone() *GlobalState {
	newState := &GlobalState{
		WorldState:       gs.WorldState.Clone(),
		Enviroment:       gs.Enviroment.Clone(),
		MachineState:     gs.MachineState.Clone(),
		TransactionStack: gs.TransactionStack.Clone(),
		Annotations:      make([]smt.Annotation, len(gs.Annotations)),
		Accounts:         make(map[int64]*Account),
	}
	if gs.LastReturnData != nil {
		newState.LastReturnData = gs.LastReturnData.Clone()
	}
	for i, Annotation := range gs.Annotations {
		newState.Annotations[i] = Annotation.Clone()
	}
	for i, account := range gs.Accounts {
		newState.Accounts[i] = account.Clone()
	}
	return newState
}

func (gs *GlobalState) GetAccounts() map[int64]*Account {
	return gs.WorldState.GetAccounts()
}

func (gs *GlobalState) GetCurrentInstruction() (disassembler.EvmInstruction, error) {
	ins := gs.Enviroment.Code.GetInstructions()
	if gs.MachineState.pc >= len(ins) {
		return disassembler.EvmInstruction{
			Address: gs.MachineState.pc,
			OPCode:  "STOP",
		}, fmt.Errorf("index out of bound")
	}
	return ins[gs.MachineState.pc], nil
}

func (gs *GlobalState) GetCurrentTransaction() *TxInfo {
	txInfo, err := gs.TransactionStack.Top()
	if err != nil {
		log.Errorf("no tx")
		return nil
	}
	return txInfo
}
func (gs *GlobalState) NewBitVec(name string, size int, Annotations ...smt.Annotation) *smt.BitVec {
	txInfo := gs.GetCurrentTransaction()
	if txInfo == nil {
		fmt.Println("no tx")
		return nil
	}
	return smt.NewBitVec(fmt.Sprintf("%s_%s", txInfo.Tx.GetTxID(), name), uint32(size), Annotations...)
}
func (gs *GlobalState) AddAnnotation(Annotation smt.Annotation) {
	gs.Annotations = append(gs.Annotations, Annotation)
}

func (gs *GlobalState) GetAnnotations() []smt.Annotation {
	return gs.Annotations
}

func (gs *GlobalState) FilterAnnotations() []smt.Annotation {
	return nil
}
