package state

import (
	"encoding/hex"
	"fmt"
	"gscanner/internal/disassembler"
	funcmanager "gscanner/internal/ethereum/function_managers"
	"gscanner/internal/opcode"
	"gscanner/internal/smt"
	"gscanner/internal/util"
	"os"
	"reflect"
	"strconv"
	"strings"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// 更多资料可以参考以下链接：
// https://ethervm.io/
// https://ethereum.org/en/developers/docs/evm/opcodes/

type Hook func(*GlobalState) error

type TxStart struct {
	GlobalState *GlobalState
	Tx          Transaction
	OPCode      string
}

type TxEnd struct {
	GlobalState *GlobalState
	Revert      bool
	ReturnData  *ReturnData
}

func NewTxStart(globalState *GlobalState, opCode string, tx Transaction) *TxStart {
	return &TxStart{
		GlobalState: globalState,
		OPCode:      opCode,
		Tx:          tx,
	}
}

func NewTxEnd(globalState *GlobalState, returnData *ReturnData, revert bool) (*TxEnd, error) {
	txInfo := globalState.GetCurrentTransaction()
	if txInfo == nil {
		return nil, fmt.Errorf("no current tx")
	}
	defer fmt.Println("--------------------------------------------------------tx end----------------------------------------")
	// MessageCallTransaction
	if _, ok := txInfo.Tx.(*MessageCallTransaction); ok {
		if returnData == nil || returnData.Size.Value() <= 0 {
			txInfo.Tx.SetReturnData("")
			return &TxEnd{
				GlobalState: globalState,
				Revert:      revert,
				ReturnData:  nil,
			}, nil
		}
		dataBv := smt.Concats(returnData.Data...)
		if dataBv != nil {
			fmt.Println("MessageCallTransaction end with return data: ", dataBv.String())
			txInfo.Tx.SetReturnData(dataBv.HexString())
		} else {
			fmt.Println("MessageCallTransaction end with nil data")
			txInfo.Tx.SetReturnData("")
		}
		return &TxEnd{
			GlobalState: globalState,
			Revert:      revert,
			ReturnData:  returnData,
		}, nil
	}
	// ContractCreationTransaction
	if returnData == nil || returnData.Size.Value() <= 0 {
		txInfo.Tx.SetReturnData("")
		return &TxEnd{
			GlobalState: globalState,
			Revert:      revert,
			ReturnData:  nil,
		}, nil
	}
	// 合约代码
	contractCode := smt.Concats(returnData.Data...).HexString()
	err := globalState.Enviroment.ActiveAccount.Code.AssignBytecode(contractCode)
	if err != nil {
		return nil, errors.Wrapf(err, "ActiveAccount AssignBytecode")
	}
	account := globalState.WorldState.GetAccount(globalState.Enviroment.ActiveAccount.Address.Value())
	err = account.Code.AssignBytecode(contractCode)
	if err != nil {
		return nil, errors.Wrapf(err, "AssignBytecode")
	}
	// 合约地址
	txInfo.Tx.SetReturnData(globalState.Enviroment.ActiveAccount.Address.HexString())
	return &TxEnd{
		GlobalState: globalState,
		Revert:      revert,
		ReturnData:  returnData,
	}, nil
}

type EvaluateResult struct {
	GlobalStates []*GlobalState
	TxStart      *TxStart
	TxEnd        *TxEnd
}

func checkGasLimit(globalState *GlobalState) error {
	err := globalState.MachineState.CheckGas()
	if err != nil {
		return err
	}
	txInfo := globalState.GetCurrentTransaction()
	if txInfo == nil {
		return fmt.Errorf("no tx found")
	}
	if txInfo.Tx.GetGasLimit().Value() <= globalState.MachineState.GetGasUsedMin() {
		return fmt.Errorf("out of gas")
	}
	return nil
}

func calculateGas(globalStates []*GlobalState) error {
	for _, globalState := range globalStates {
		currentInstruction, err := globalState.GetCurrentInstruction()
		if err != nil {
			return errors.Wrapf(err, "globalState.GetCurrentInstruction")
		}
		globalState.GasUsedMinAdd(currentInstruction.GasMin)
		globalState.GasUsedMaxAdd(currentInstruction.GasMax)
		err = checkGasLimit(globalState)
		if err != nil {
			return err
		}
	}
	return nil
}

func transferETH(globalState *GlobalState, sender, receiver, value *smt.BitVec) error {
	senderBalance, err := globalState.WorldState.GetBalance(sender)
	if err != nil {
		return errors.Wrapf(err, "GetBalance sender")
	}
	receiverBalance, err := globalState.WorldState.GetBalance(receiver)
	if err != nil {
		return errors.Wrapf(err, "GetBalance receiver")
	}
	globalState.WorldState.AddConstraint(*senderBalance.Uge(value))
	senderNewBalance := senderBalance.Sub(value)
	receiverNewBalance := receiverBalance.Add(value)
	err = globalState.WorldState.SetBalance(sender, senderNewBalance)
	if err != nil {
		return errors.Wrapf(err, "SetBalance sender")
	}
	err = globalState.WorldState.SetBalance(receiver, receiverNewBalance)
	if err != nil {
		return errors.Wrapf(err, "SetBalance receiver")
	}
	return nil
}

type Instruction struct {
	OPCode string

	preHook  []Hook
	postHook []Hook
}

func NewInstruction(opCode string, preHook, postHook []Hook) *Instruction {
	return &Instruction{
		OPCode:   opCode,
		preHook:  preHook,
		postHook: postHook,
	}
}

func (ins *Instruction) execPreHooks(globalState *GlobalState) {
	for _, hook := range ins.preHook {
		hook(globalState)
	}
}

func (ins *Instruction) execPostHooks(globalState *GlobalState) {
	for _, hook := range ins.postHook {
		hook(globalState)
	}
}

// Evaluate 执行OPCode的具体逻辑
func (ins *Instruction) Evaluate(globalState *GlobalState) (evaluateResult *EvaluateResult, err error) {
	op := strings.Title(strings.ToLower(ins.OPCode))
	if strings.HasPrefix(ins.OPCode, "PUSH") {
		op = strings.Title("push")
	} else if strings.HasPrefix(ins.OPCode, "DUP") {
		op = strings.Title("dup")
	} else if strings.HasPrefix(ins.OPCode, "SWAP") {
		op = strings.Title("swap")
	} else if strings.HasPrefix(ins.OPCode, "LOG") {
		op = strings.Title("log")
	}
	currentInstruction, _ := globalState.GetCurrentInstruction()
	fmt.Printf("Evluating %v(%s) #[%s] at %d\n", ins.OPCode, op,
		hex.EncodeToString(currentInstruction.Argument),
		globalState.MachineState.GetPC())
	t := reflect.TypeOf(ins)
	method, ok := t.MethodByName(op)
	if !ok {
		return nil, fmt.Errorf("not implemented: %s", ins.OPCode)
	}
	// pc := globalState.MachineState.pc
	ins.execPreHooks(globalState)
	callResult := method.Func.Call([]reflect.Value{reflect.ValueOf(ins), reflect.ValueOf(globalState)})
	if len(callResult) != 2 {
		panic(fmt.Errorf("wrong op handler definition: args length %d != 2", len(callResult)))
	}
	evaluateResult, ok = callResult[0].Interface().(*EvaluateResult)
	if !ok {
		panic(fmt.Errorf("wrong op handler definition: args 1 type"))
	}
	if callResult[1].Interface() == nil {
		err = nil
	} else {
		err, ok = callResult[1].Interface().(error)
		if !ok {
			panic(fmt.Errorf("wrong op handler definition: args 2 type, %s", reflect.TypeOf(callResult[1])))
		}
	}
	ins.execPostHooks(globalState)

	if len(globalState.WorldState.TransactionSequence) <= 0 {
		fmt.Println()
	}

	// globalState.MachineState.Print(pc)

	return evaluateResult, err
}

// Jumpdest
// 指令: JUMPDEST Mark a valid destination for jumps
// gas: 1
// 是否改变worldstate: false
func (ins *Instruction) Jumpdest(globalState *GlobalState) (*EvaluateResult, error) {
	// 计算gas
	err := calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	// pc自增
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Push
// 指令: PUSH{n} 1<=n<=32 Place n byte item on stack
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Push(globalState *GlobalState) (*EvaluateResult, error) {
	pushInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, err
	}
	a := smt.NewBitVecValFromBytes(pushInstruction.Argument, 256)
	fmt.Println(hex.EncodeToString(pushInstruction.Argument), hex.EncodeToString(a.Bytes()))
	err = globalState.MachineState.PushStack(smt.NewBitVecValFromBytes(pushInstruction.Argument, 256))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Dup
// 指令: DUP{n} 1<=n<=16 Duplicate n th stack item
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Dup(globalState *GlobalState) (*EvaluateResult, error) {
	instruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(strings.TrimPrefix(instruction.OPCode, "DUP"))
	if err != nil {
		return nil, errors.Wrapf(err, "Atoi")
	}
	err = globalState.MachineState.Dup(n)
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Swap
// 指令: SWAP{n} 1<=n<=16 Exchange 1st and nth stack items
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Swap(globalState *GlobalState) (*EvaluateResult, error) {
	instruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, err
	}
	depth, err := strconv.Atoi(strings.TrimPrefix(instruction.OPCode, "SWAP"))
	if err != nil {
		return nil, errors.Wrapf(err, "strconv.Atoi")
	}
	err = globalState.MachineState.stack.Swap(depth)
	if err != nil {
		return nil, errors.Wrapf(err, "swap")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Pop
// 指令: POP Remove word from stack
// gas: 2
// 是否改变worldstate: true
func (ins *Instruction) Pop(globalState *GlobalState) (*EvaluateResult, error) {
	op, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	yices2.PpTerm(os.Stdout, op.GetRaw(), 1000, 80, 0)
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// And
// 指令: AND Bitwise AND operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) And(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	data := op1.AsBitVec().And(op2.AsBitVec())
	yices2.PpTerm(os.Stdout, data.GetRaw(), 1000, 80, 0)
	err = globalState.MachineState.PushStack(data)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Or
// 指令: OR Bitwise OR operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Or(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Or(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// XOR
// 指令: XOR Bitwise XOR operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Xor(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Xor(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Not
// 指令: NOT Bitwise NOT operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Not(globalState *GlobalState) (*EvaluateResult, error) {
	op, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}

	err = globalState.MachineState.PushStack(op.Not())
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Byte
// 指令: BYTE Retrieve single byte from word
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Byte(globalState *GlobalState) (*EvaluateResult, error) {
	// index
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	// word
	op2, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	if op1.IsSymbolic() {
		fmt.Println("unsupported symbolic offset")
		result := globalState.NewBitVec(op1.String()+"["+op2.String()+"]", 256)
		err = globalState.MachineState.PushStack(result)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
	} else {
		var (
			result *smt.BitVec
			index  = op1.Value()
			offset = (31 - index) * 8
		)
		if offset < 0 {
			result = smt.NewBitVecValInt64(0, 256)
		} else {
			// 需要填充为256bits
			v := yices2.Bvextract(op2.GetRaw(), uint32(offset), uint32(offset+7))
			bvFill := smt.NewBitVecValInt64(0, 248)
			result = bvFill.Concat(smt.NewBitVecFromTerm(v, 8))
		}
		err = globalState.MachineState.PushStack(result)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Add
// 指令: ADD Addition operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Add(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Add(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Sub
// 指令: SUB Subtraction operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Sub(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Sub(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Mul
// 指令: MUL Multiplication operation
// gas: 5
// 是否改变worldstate: true
func (ins *Instruction) Mul(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Mul(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Div
// 指令: DIV Integer division operation
// gas: 5
// 是否改变worldstate: true
func (ins *Instruction) Div(globalState *GlobalState) (*EvaluateResult, error) {
	op1, op2, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	if op2.Eq(smt.NewBitVecValInt64(0, 256)).IsTrue() {
		bv := smt.NewBitVecValInt64(0, 256)
		err = globalState.MachineState.PushStack(bv)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
	} else {
		err = globalState.MachineState.PushStack(op1.Div(op2))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// SDiv
// 指令: SDIV Signed integer division operation (truncated)
// gas: 5
// 是否改变worldstate: true
func (ins *Instruction) SDiv(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	if op2.AsBitVec().Eq(smt.NewBitVecValInt64(0, 256)).IsTrue() {
		bv := smt.NewBitVecValInt64(0, 256)
		err = globalState.MachineState.PushStack(bv)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
	} else {
		err = globalState.MachineState.PushStack(op1.AsBitVec().SDiv(op2.AsBitVec()))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Mod
// 指令: MOD Modulo remainder operation
// gas: 5
// 是否改变worldstate: true
func (ins *Instruction) Mod(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().URem(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Shl
// 指令: Shl Shift Left
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Shl(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Shl(op2.AsBitVec()))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Shr
// 指令: Shr Logical Shift Right
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Shr(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Shr(op2.AsBitVec()))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Ashr
// 指令: SAR Arithmetic Shift Right
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Ashr(globalState *GlobalState) (*EvaluateResult, error) {
	op1, op2, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	// fmt.Println(op1.Value(), " ", op2.Value())
	err = globalState.MachineState.PushStack(op1.AShr(op2))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Smod
// 指令: SAR Signed modulo remainder operation
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) SMod(globalState *GlobalState) (*EvaluateResult, error) {
	op1, op2, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.SRem(op2))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// AddMod
// 指令: ADDMOD Modulo addition operation
// gas: 8
// 是否改变worldstate: true
func (ins *Instruction) AddMod(globalState *GlobalState) (*EvaluateResult, error) {
	op1, op2, op3, err := globalState.MachineState.PopBitVec3()
	if err != nil {
		return nil, err
	}
	f1 := op1.URem(op3)
	f2 := op2.URem(op3)
	err = globalState.MachineState.PushStack(f1.Add(f2).URem(op3))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// MulMod
// 指令: MULMOD Modulo multiplication operation
// gas: 8
// 是否改变worldstate: true
func (ins *Instruction) MulMod(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	op3, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	f1 := op1.URem(op3)
	f2 := op2.URem(op3)
	err = globalState.MachineState.PushStack(f1.Mul(f2).URem(op3))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Exp
// 指令: EXP Exponential operation
// gas: 10*
// 是否改变worldstate: true
func (ins *Instruction) Exp(globalState *GlobalState) (*EvaluateResult, error) {
	// base exponent
	op1, op2, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	yices2.PpTerm(os.Stdout, op1.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, op2.GetRaw(), 1000, 80, 0)
	exponent, constraint := funcmanager.Efm.CreateCondition(op1, op2)
	err = globalState.MachineState.PushStack(exponent)
	if err != nil {
		return nil, err
	}
	globalState.WorldState.AddConstraint(constraint)
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// SignExtend
// 指令: SIGNEXTEND Extend length of two's complement signed integer
// gas: 5
// 是否改变worldstate: true
func (ins *Instruction) SIGNEXTEND(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}

	testbit := op1.Mul(smt.NewBitVecValInt64(8, 256)).Add(smt.NewBitVecValInt64(7, 256))
	setTestBit := smt.NewBitVecValInt64(1, 256).Shl(testbit)
	signBitSet := op2.And(testbit).Eq(smt.NewBitVecValInt64(0, 256)).AsBitVec().Not()
	f1 := op1.Lt(smt.NewBitVecValInt64(31, 256))
	f2 := op2.Or(smt.NewBitVecValInt64(0, 256).Sub(testbit)).AsBitVec()
	f3 := op2.And(setTestBit.Sub(smt.NewBitVecValInt64(1, 256))).AsBitVec()
	f4 := yices2.Ite(signBitSet.GetRaw(), f2.GetRaw(), f3.GetRaw())
	f5 := yices2.Ite(f1.GetRaw(), f4, op1.GetRaw())

	err = globalState.MachineState.PushStack(smt.NewBitVecFromTerm(f5, 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Lt
// 指令: LT Less-than comparison
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Lt(globalState *GlobalState) (*EvaluateResult, error) {
	op1, op2, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	fmt.Println(op1.IsSymbolic(), op2.IsSymbolic())
	// if !op1.IsSymbolic() {
	// 	fmt.Println("op1 value: ", op1.Value())
	// }
	// if !op2.IsSymbolic() {
	// 	fmt.Println("op1 value: ", op2.Value())
	// }
	yices2.PpTerm(os.Stdout, op1.Ult(op2).GetRaw(), 1000, 80, 0)
	err = globalState.MachineState.PushStack(op1.Ult(op2))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Gt
// 指令: GT Greater-than comparison
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Gt(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Ugt(op2.AsBitVec()))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Slt
// 指令: SLT Signed less-than comparison
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Slt(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(op1.AsBitVec().Lt(op2.AsBitVec()))
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Sgt
// 指令: STT Signed greater-than comparison
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Sgt(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	globalState.MachineState.PushStack(op1.AsBitVec().Gt(op2.AsBitVec()))
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Eq
// 指令: EQ Equality comparison
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Eq(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	op2, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	//condition := op1.AsBitVec().Eq(op2.AsBitVec())
	condition := op1.AsBitVec().Eq(op2.AsBitVec())
	// fmt.Println(op1.AsBitVec().HexString(), op2.AsBitVec().Size())
	yices2.PpTerm(os.Stdout, op1.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, op2.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, condition.GetRaw(), 1000, 80, 0)
	err = globalState.MachineState.PushStack(condition)
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Iszero
// 指令: ISZERO Simple not operator
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Iszero(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopStack()
	if err != nil {
		return nil, err
	}
	var exp *smt.Bool
	if op1.Type() == smt.BitVecType {
		bv := op1.AsBitVec()
		exp = bv.Eq(smt.NewBitVecValInt64(0, op1.Size()))
	} else if op1.Type() == smt.BoolType {
		b := op1.AsBool()
		exp = &b
		exp = exp.AsBitVec().Eq(smt.NewBitVecValInt64(0, 256))
	} else {
		return nil, fmt.Errorf("unknown storable type")
	}
	yices2.PpTerm(os.Stdout, exp.GetRaw(), 1000, 80, 0)
	err = globalState.MachineState.PushStack(exp)
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Callvalue
// 指令: CALLVALUE CALLVALUE Get deposited value by the instruction/transaction responsible for this execution
// gas: 2
// 是否改变worldstate: true
func (ins *Instruction) Callvalue(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.CallValue.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// CalldataLoad
// 指令: CALLDATALOAD Get input data of current environment
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Calldataload(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	yices2.PpTerm(os.Stdout, op1.GetRaw(), 1000, 80, 0)
	value, err := globalState.Enviroment.CallData.GetWordAt(op1)
	if err != nil {
		return nil, err
	}
	// yices2.PpTerm(os.Stdout, value.GetRaw(), 1000, 80, 0)
	err = globalState.MachineState.PushStack(value)
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// CalldataSize
// 指令: CALLDATASIZE Get size of input data in current environment
// gas: 3
// 是否改变worldstate: true
func (ins *Instruction) Calldatasize(globalState *GlobalState) (*EvaluateResult, error) {
	size := globalState.Enviroment.CallData.CalldataSize()
	err := globalState.MachineState.PushStack(size.Clone())
	if err != nil {
		return nil, err
	}
	yices2.PpTerm(os.Stdout, size.GetRaw(), 1000, 80, 0)
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) CalldataCopy(globalState *GlobalState) (*EvaluateResult, error) {
	op1, op2, op3, err := globalState.MachineState.PopBitVec3()
	if err != nil {
		return nil, err
	}
	yices2.PpTerm(os.Stdout, op1.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, op2.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, op3.GetRaw(), 1000, 80, 0)
	txInfo := globalState.GetCurrentTransaction()
	if txInfo == nil {
		return nil, fmt.Errorf("no tx")
	}
	if _, ok := txInfo.Tx.(*ContractCreationTransaction); ok {
		log.Errorf("CALLDATACOPY on contract reate tx is unsupported")
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}

	return ins.calldataCopyHelper(globalState, op1, op2, op3)
}

func (ins *Instruction) calldataCopyHelper(globalState *GlobalState, memOffset, calldataOffset, size *smt.BitVec) (*EvaluateResult, error) {
	if memOffset.IsSymbolic() {
		log.Infof("unsupported symbolic memory offset in CALLDATACOPY")
		err := calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if calldataOffset.IsSymbolic() {
		log.Infof("unsupported symbolic calldata offset in CALLDATACOPY")
		err := calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	var (
		concreteMemOffset = memOffset.Value()
		// concreteCalldataOffset = calldataOffset.Value()
		concreteSize int64
	)
	if size.IsSymbolic() {
		log.Infof("unsupported symbolic size in CALLDATACOPY")
		concreteSize = 320
	} else {
		concreteSize = size.Value()
	}

	err := globalState.MachineState.MemExtend(concreteMemOffset, concreteSize)
	if err != nil {
		log.Errorf("MemExtend: %v", err)
		err = globalState.MachineState.MemExtend(concreteMemOffset, 1)
		if err != nil {
			return nil, errors.Wrapf(err, "MemExtend")
		}
		name := fmt.Sprintf("calldata(%s)[%s:%s]",
			globalState.Enviroment.ActiveAccount.ContractName,
			calldataOffset, size)
		err = globalState.MachineState.MemWriteByte(memOffset, globalState.NewBitVec(name, 8))
		if err != nil {
			log.Errorf("MemWriteByte: %v", err)
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}

	var (
		calldataIndex = calldataOffset
		memIndex      = memOffset
	)
	for i := int64(0); i < concreteSize; i++ {
		data, err := globalState.Enviroment.CallData.GetByteAt(calldataIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "CallData GetByteAt")
		}
		err = globalState.MachineState.MemWriteByte(memIndex, data)
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
		calldataIndex = calldataIndex.AddInt64(1)
		memIndex = memIndex.AddInt64(1)
	}

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Address(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.ActiveAccount.Address.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	yices2.PpTerm(os.Stdout, globalState.Enviroment.ActiveAccount.Address.GetRaw(), 1000, 80, 0)
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Balance(globalState *GlobalState) (*EvaluateResult, error) {
	// address
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	var balance *smt.BitVec
	if op1.IsSymbolic() {
		// 构建判断分支
		balance = smt.NewBitVecValInt64(0, 256)
		for _, account := range globalState.WorldState.GetAccounts() {
			accountBalance, err := account.GetBalance()
			if err != nil {
				return nil, errors.Wrapf(err, "GetBalance symbolic")
			}
			condition := account.Address.Eq(op1).GetRaw()
			positiveBranch := accountBalance.GetRaw()
			negativeBranch := balance.GetRaw()
			balance = smt.NewBitVecFromTerm(yices2.Ite(condition, positiveBranch, negativeBranch), 256)
		}
	} else {
		account := globalState.WorldState.AccountsExistOrLoad(op1)
		balance, err = account.GetBalance()
		if err != nil {
			return nil, errors.Wrapf(err, "GetBalance")
		}
	}
	yices2.PpTerm(os.Stdout, op1.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, balance.GetRaw(), 1000, 80, 0)
	err = globalState.MachineState.PushStack(balance)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Origin(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.Origin.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	yices2.PpTerm(os.Stdout, globalState.Enviroment.Origin.GetRaw(), 1000, 80, 0)
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Caller(globalState *GlobalState) (*EvaluateResult, error) {
	caller := globalState.Enviroment.Sender.Clone()
	yices2.PpTerm(os.Stdout, caller.GetRaw(), 1000, 80, 0)
	err := globalState.MachineState.PushStack(caller)
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) ChainID(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.ChainID.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) SelfBalance(globalState *GlobalState) (*EvaluateResult, error) {
	balance, err := globalState.Enviroment.ActiveAccount.GetBalance()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(balance)
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) CodeSize(globalState *GlobalState) (*EvaluateResult, error) {
	txInfo := globalState.GetCurrentTransaction()
	var size int64
	if _, ok := txInfo.Tx.(*ContractCreationTransaction); ok {
		size = int64(len(globalState.Enviroment.Code.GetBytecode()) / 2)
		if _, ok := globalState.Enviroment.CallData.(*ConcreteCalldata); ok {
			size += globalState.Enviroment.CallData.Size().Value()
		} else {
			size += 16
			sizeBv := smt.NewBitVecValInt64(size, 256)
			constraint := globalState.Enviroment.CallData.Size().Eq(sizeBv)
			globalState.WorldState.AddConstraint(*constraint)
		}
	} else {
		size = int64(len(globalState.Enviroment.Code.GetBytecode()) / 2)
	}
	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(int64(size), 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) sha3GasHelper(globalState *GlobalState, length int64) (*GlobalState, error) {
	gasMin, gasMax := CalculateSha3Gas(length)
	globalState.GasUsedMinAdd(gasMin)
	globalState.GasUsedMaxAdd(gasMax)
	err := checkGasLimit(globalState)
	if err != nil {
		return nil, err
	}
	return globalState, nil
}

func (ins *Instruction) sha3(globalState *GlobalState) (*EvaluateResult, error) {
	index, op1, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec2")
	}
	var length int64
	if op1.IsSymbolic() {
		length = 64
		globalState.WorldState.AddConstraint(*op1.Eq(smt.NewBitVecValInt64(length, 256)))
	} else {
		length = op1.Value()
	}
	globalState, err = ins.sha3GasHelper(globalState, length)
	if err != nil {
		return nil, errors.Wrapf(err, "sha3GasHelper")
	}
	err = globalState.MachineState.MemExtend(index.Value(), length)
	if err != nil {
		return nil, errors.Wrapf(err, "MemExtend")
	}

	var data *smt.BitVec
	for i := int64(0); i < length; i++ {
		b := globalState.MachineState.MemGetByteAt(index.AddInt64(i))
		if data == nil {
			data = b
		} else {
			data = data.Concat(b)
		}
	}
	if data == nil {
		data = funcmanager.Kfm.GetEmptyKeccakHash()
	} else {
		data = funcmanager.Kfm.CreateKeccak(data)
	}
	err = globalState.MachineState.PushStack(data)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) GasPrice(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.GasPrice.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Basefee(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.BaseFee.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Codecopy(globalState *GlobalState) (*EvaluateResult, error) {
	memOffset, codeOffset, size, err := globalState.MachineState.PopBitVec3()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec3")
	}
	// yices2.PpTerm(os.Stdout, memOffset.GetRaw(), 1000, 80, 0)
	// yices2.PpTerm(os.Stdout, codeOffset.GetRaw(), 1000, 80, 0)
	// yices2.PpTerm(os.Stdout, size.GetRaw(), 1000, 80, 0)
	fmt.Println(memOffset.IsSymbolic(), codeOffset.IsSymbolic(), size.IsSymbolic())
	fmt.Println(memOffset.Value(), codeOffset.Value(), size.Value())
	var (
		code   = globalState.Enviroment.Code.GetBytecode()
		txInfo = globalState.GetCurrentTransaction()
		ok     bool
	)
	if _, ok = txInfo.Tx.(*ContractCreationTransaction); !ok {
		states, err := ins.codeCopyHelper(globalState, code, "CODECOPY", memOffset, codeOffset, size, true)
		if err != nil {
			return nil, errors.Wrapf(err, "codeCopyHelper")
		}
		return &EvaluateResult{
			GlobalStates: states,
		}, nil
	}
	var (
		concretCodeOffset = codeOffset.Value()
		concretSize       = size.Value()
		codeSize          = int64(len(code) / 2)
	)
	log.Infof("copying from code offset: %d %d with size: %d", concretCodeOffset, codeSize, concretSize)
	if _, ok := globalState.Enviroment.CallData.(*SymbolicCalldata); ok {
		if concretCodeOffset > concretSize {
			return ins.calldataCopyHelper(globalState, memOffset, codeOffset, size)
		}
	} else {
		var (
			calldataCopyOffset int64
			codeCopySize       int64
		)
		// 调整size值防止越界
		if concretCodeOffset+concretSize <= codeSize {
			codeCopySize = concretSize
		} else if codeSize-concretCodeOffset > 0 {
			codeCopySize = codeSize - concretCodeOffset
		}
		if concretCodeOffset-codeSize > 0 {
			calldataCopyOffset = concretCodeOffset - codeSize
		}
		var calldataCopySize int64
		if concretCodeOffset+concretSize-codeSize > 0 {
			calldataCopySize = concretCodeOffset + concretSize - codeSize
		}
		codeCopyResult, err := ins.codeCopyHelper(
			globalState, code, "CODECOPY", memOffset.Clone().AsBitVec(), codeOffset, size, false)
		if err != nil {
			return nil, errors.Wrapf(err, "codeCopyHelper")
		}
		newMemOffset := memOffset.Clone().AsBitVec()
		newMemOffset = newMemOffset.AddInt64(codeCopySize)
		fmt.Println(newMemOffset.Value(), calldataCopyOffset, calldataCopySize)
		return ins.calldataCopyHelper(
			codeCopyResult[0],
			newMemOffset,
			smt.NewBitVecValInt64(calldataCopyOffset, 256),
			smt.NewBitVecValInt64(calldataCopySize, 256))
	}

	codeCopyResult, err := ins.codeCopyHelper(
		globalState, code, "CODECOPY", memOffset.Clone().AsBitVec(), codeOffset, size, true)
	if err != nil {
		return nil, errors.Wrapf(err, "codeCopyHelper")
	}
	return &EvaluateResult{
		GlobalStates: codeCopyResult,
	}, nil
}

func (ins *Instruction) ExtCodeSize(globalState *GlobalState) ([]*GlobalState, error) {
	address, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec")
	}
	if address.IsSymbolic() {
		log.Errorf("symbolic address of EXTCODESIZE not supported")
		err = globalState.MachineState.PushStack(smt.NewBitVec(fmt.Sprintf("extcodesize_%s", address.String()), 256))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return []*GlobalState{globalState}, nil
	}
	account := globalState.WorldState.AccountsExistOrLoad(address)
	code := account.Code.GetBytecode()
	size := int64(len(code) / 2)
	err = globalState.MachineState.PushStack(smt.NewBitVecValInt64(size, 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return []*GlobalState{globalState}, nil
}

func (ins *Instruction) codeCopyHelper(
	globalState *GlobalState,
	code, opCode string,
	memOffset, codeOffset, size *smt.BitVec, checkGasAndIncreasePC bool) ([]*GlobalState, error) {
	if memOffset.IsSymbolic() {
		log.Infof("unsupported symoblic offset of %s", opCode)
		if checkGasAndIncreasePC {
			err := calculateGas([]*GlobalState{globalState})
			if err != nil {
				return nil, errors.Wrapf(err, "calculateGas")
			}
			globalState.MachineState.IncreasePC()
		}
		return []*GlobalState{globalState}, nil
	}
	concreteMemOffset := memOffset.Value()
	if size.IsSymbolic() {
		globalState.MachineState.MemExtend(concreteMemOffset, 1)
		data := globalState.NewBitVec(fmt.Sprintf("code(%s)",
			globalState.Enviroment.ActiveAccount.ContractName), 8)
		err := globalState.MachineState.MemWriteByte(memOffset, data)
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
		if checkGasAndIncreasePC {
			err := calculateGas([]*GlobalState{globalState})
			if err != nil {
				return nil, errors.Wrapf(err, "calculateGas")
			}
			globalState.MachineState.IncreasePC()
		}
		return []*GlobalState{globalState}, nil
	}

	concreteSize := size.Value()
	if codeOffset.IsSymbolic() {
		log.Infof("unsupported symoblic code offset of %s", opCode)
		err := globalState.MachineState.MemExtend(concreteMemOffset, concreteSize)
		if err != nil {
			return nil, errors.Wrapf(err, "MemExtend")
		}
		for i := 0; i < int(concreteSize); i++ {
			index := smt.NewBitVecValInt64(int64(i)+concreteMemOffset, 256)
			data := globalState.NewBitVec(fmt.Sprintf("code(%s)",
				globalState.Enviroment.ActiveAccount.ContractName), 8)
			err = globalState.MachineState.MemWriteByte(index, data)
			if err != nil {
				return nil, errors.Wrapf(err, "MemWriteByte")
			}
		}
		if checkGasAndIncreasePC {
			err := calculateGas([]*GlobalState{globalState})
			if err != nil {
				return nil, errors.Wrapf(err, "calculateGas")
			}
			globalState.MachineState.IncreasePC()
		}
		return []*GlobalState{globalState}, nil
	}

	concreteCodeOffset := codeOffset.Value()
	codeBytes, err := hex.DecodeString(code)
	if err != nil {
		return nil, errors.Wrapf(err, "DecodeString")
	}

	for i := 0; i < int(concreteSize); i++ {
		if int(concreteCodeOffset)+i >= len(codeBytes) {
			log.Infof("copy end")
			break
		}
		index := smt.NewBitVecValInt64(concreteMemOffset+int64(i), 256)
		data := smt.NewBitVecValFromBytes([]byte{codeBytes[int(concreteCodeOffset)+i]}, 8)
		err = globalState.MachineState.MemWriteByte(index, data)
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
	}

	if checkGasAndIncreasePC {
		err := calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
	}
	return []*GlobalState{globalState}, nil
}

func (ins *Instruction) Extcodecopy(globalState *GlobalState) (*EvaluateResult, error) {
	address, memOffset, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec2")
	}
	codeOffset, size, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec2")
	}
	if address.IsSymbolic() {
		err := calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		log.Infof("symbolic adddress of EXTCODECOPY is unsupported")
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	fmt.Println(address.Value(), " ", memOffset.Value(), " ", codeOffset.Value(), " ", size.Value())
	bytecode := globalState.WorldState.AccountsExistOrLoad(address).Code.GetBytecode()
	states, err := ins.codeCopyHelper(globalState, bytecode, "EXTCODECOPY", memOffset, codeOffset, size, false)
	if err != nil {
		return nil, errors.Wrapf(err, "codeCopyHelper")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: states,
	}, nil
}

func (ins *Instruction) Extcodehash(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec")
	}
	var (
		address  = smt.NewBitVecFromTerm(yices2.Bvextract(op1.GetRaw(), 0, 159), 160)
		codeHash *smt.BitVec
	)
	if address.IsSymbolic() {
		_, data, err := util.GetCodeHash("")
		if err != nil {
			return nil, errors.Wrapf(err, "GetCodeHash")
		}
		codeHash = smt.NewBitVecValFromBytes(data, 256)
	} else if globalState.WorldState.GetAccount(address.Value()) == nil {
		codeHash = smt.NewBitVecValInt64(0, 256)
	} else {
		bytecode := globalState.WorldState.AccountsExistOrLoad(address).Code.GetBytecode()
		_, data, err := util.GetCodeHash(bytecode)
		if err != nil {
			return nil, errors.Wrapf(err, "GetCodeHash")
		}
		codeHash = smt.NewBitVecValFromBytes(data, 256)
	}
	err = globalState.MachineState.PushStack(codeHash)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Returndatacopy(globalState *GlobalState) (*EvaluateResult, error) {
	memOffset, reuturnOffset, size, err := globalState.MachineState.PopBitVec3()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec3")
	}
	if memOffset.IsSymbolic() {
		log.Errorf("symbolic memoffset in RETURNDATACOPY is unsupported")
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if reuturnOffset.IsSymbolic() {
		log.Errorf("symbolic reuturnOffset in RETURNDATACOPY is unsupported")
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if size.IsSymbolic() {
		log.Errorf("symbolic size in RETURNDATACOPY is unsupported")
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if globalState.LastReturnData == nil {
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}

	var (
		concreteMemOffset          = memOffset.Value()
		concreteReturnOffset       = reuturnOffset.Value()
		concreteSize               = size.Value()
		concreteLastReturnDataSize = globalState.LastReturnData.Size.Value()
		zeroData                   = smt.NewBitVecValInt64(0, 8)
	)
	err = globalState.MachineState.MemExtend(concreteMemOffset, concreteSize)
	if err != nil {
		return nil, errors.Wrapf(err, "MemExtend")
	}
	for i := int64(0); i < concreteSize; i++ {
		var data *smt.BitVec
		if i < concreteLastReturnDataSize {
			data = globalState.LastReturnData.Data[concreteReturnOffset+i]
		} else {
			data = zeroData
		}
		err = globalState.MachineState.MemWriteByte(memOffset.AddInt64(i), data.Clone().AsBitVec())
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
	}

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Returndatasize(globalState *GlobalState) (*EvaluateResult, error) {
	if globalState.LastReturnData != nil {
		globalState.MachineState.PushStack(globalState.LastReturnData.Size.Clone())
	} else {
		globalState.MachineState.PushStack(smt.NewBitVecValInt64(0, 256))
	}
	err := calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) BlockHash(globalState *GlobalState) (*EvaluateResult, error) {
	// block number
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.PushStack(globalState.NewBitVec(
		fmt.Sprintf("blockhash_block_%d", op1.Value()), 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) CoinBase(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.NewBitVec("coinbase", 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) TimeStamp(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.NewBitVec("timestamp", 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Number(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.Enviroment.BlockNumber)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Difficulty(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(globalState.NewBitVec("block_difficulty", 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) GasLimit(globalState *GlobalState) (*EvaluateResult, error) {
	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(globalState.MachineState.gasLimit, 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Mload(globalState *GlobalState) (*EvaluateResult, error) {
	// offset
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	err = globalState.MachineState.MemExtend(op1.Value(), 32)
	if err != nil {
		return nil, err
	}
	data := globalState.MachineState.MemGetWordAt(op1)
	err = globalState.MachineState.PushStack(data)
	if err != nil {
		return nil, err
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Mstore(globalState *GlobalState) (*EvaluateResult, error) {
	// offset
	offset, value, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	fmt.Println("Mstore valuesize", value.Size())
	err = globalState.MachineState.MemExtend(offset.Value(), 32)
	if err != nil {
		return nil, errors.Wrapf(err, "MemExtend")
	}
	err = globalState.MachineState.MemWriteWordAt(offset, value)
	if err != nil {
		return nil, errors.Wrapf(err, "WriteWordAt")
	}
	// v := globalState.MachineState.MemGetWordAt(offset)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "MemGetWordAt")
	// }
	// fmt.Println("mstore ", offset.Value(), value.Value(), " get->", v.Value())
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) MStore8(globalState *GlobalState) (*EvaluateResult, error) {
	// offset
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	// value
	op2, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	globalState.MachineState.MemExtend(op1.Value(), 1)
	v := yices2.Bvextract(op2.GetRaw(), uint32(0), uint32(7))
	// 端序？
	err = globalState.MachineState.PushStack(smt.NewBitVecFromTerm(v, 8))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Sload(globalState *GlobalState) (*EvaluateResult, error) {
	index, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec")
	}
	data, err := globalState.Enviroment.ActiveAccount.StorageGet(index)
	if err != nil {
		return nil, errors.Wrapf(err, "StorageGet")
	}
	fmt.Printf("sload %+v\n", globalState.Enviroment.ActiveAccount)
	yices2.PpTerm(os.Stdout, index.GetRaw(), 1000, 80, 0)
	yices2.PpTerm(os.Stdout, data.GetRaw(), 1000, 80, 0)
	fmt.Println(index.Value(), data.Value())
	err = globalState.MachineState.PushStack(data)
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Sstore(globalState *GlobalState) (*EvaluateResult, error) {
	if globalState.Enviroment.Static {
		return nil, fmt.Errorf("write protected")
	}
	key, value, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec2")
	}
	fmt.Println(key.IsSymbolic(), value.IsSymbolic())
	fmt.Println(key.TermType(), value.TermType())
	// yices2.PpTerm(os.Stdout, key.GetRaw(), 1000, 90, 0)
	// yices2.PpTerm(os.Stdout, value.GetRaw(), 1000, 90, 0)
	// fmt.Println(key.Value(), value.HexString())
	err = globalState.Enviroment.ActiveAccount.StorageSet(key, value)
	if err != nil {
		return nil, errors.Wrapf(err, "StorageSet")
	}
	fmt.Printf("sstore %+v\n", globalState.Enviroment.ActiveAccount)
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Jump(globalState *GlobalState) (*EvaluateResult, error) {
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec")
	}
	if op1.IsSymbolic() {
		log.Errorf("invalid jump to symbolic address")
		return nil, fmt.Errorf("invalid jump to symbolic address")
	}

	yices2.PpTerm(os.Stdout, op1.GetRaw(), 1000, 80, 0)
	fmt.Println(yices2.TermConstructor(op1.GetRaw()))
	jumpAddress := op1.Value()
	index := util.GetInstructionIndex(globalState.Enviroment.Code.GetInstructions(), int(jumpAddress))
	if index == -1 {
		log.Errorf("invalid jump destination")
		return nil, fmt.Errorf("invalid jump destination")
	}
	fmt.Println("jump to ", jumpAddress, index)
	dstInstruction := globalState.Enviroment.Code.GetInstructions()[index]
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}
	if dstInstruction.OPCode != "JUMPDEST" {
		log.Errorf("jump destination is not JUMPDEST")
		return nil, errors.Wrapf(err, "jump destination is not JUMPDEST")
	}
	opCodeInfo, _ := opcode.GetOPCodeInfoByOperation("JUMP")
	// newState := globalState.Clone()
	// newState.GasUsedMinAdd(int64(opCodeInfo.GasMin))
	// newState.GasUsedMaxAdd(int64(opCodeInfo.GasMax))
	// newState.MachineState.Jump(index)

	globalState.GasUsedMinAdd(int64(opCodeInfo.GasMin))
	globalState.GasUsedMaxAdd(int64(opCodeInfo.GasMax))
	globalState.MachineState.Jump(index)
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

// Jumpi
// 指令: JUMPI Conditionally alter the program counter
// gas: 10
// 是否改变worldstate: true
func (ins *Instruction) Jumpi(globalState *GlobalState) (*EvaluateResult, error) {
	opCodeInfo, ok := opcode.GetOPCodeInfoByOperation("JUMPI")
	if !ok {
		return nil, fmt.Errorf("opcode not found")
	}
	// jump address
	jumpAddress, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	log.Infof("jumpi to %d", jumpAddress.Value())
	if jumpAddress.Value() == 76 {
		fmt.Println()
	}
	// condition
	conditionBool, err := globalState.MachineState.PopBool()
	if err != nil {
		return nil, err
	}
	yices2.PpTerm(os.Stdout, conditionBool.GetRaw(), 1000, 80, 0)
	if jumpAddress.IsSymbolic() {
		fmt.Println("skipped invalid jumpi")
		globalState.GasUsedMinAdd(int64(opCodeInfo.GasMin))
		globalState.GasUsedMaxAdd(int64(opCodeInfo.GasMax))
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}

	trueCondition := yices2.False() != conditionBool.GetRaw()
	falseCondition := yices2.True() != conditionBool.GetRaw()
	var states []*GlobalState
	if falseCondition {
		fmt.Println("falseCondition")
		newState := globalState.Clone()
		newState.GasUsedAdd(int64(opCodeInfo.GasMin), int64(opCodeInfo.GasMax))

		newState.IncreaseDepth()
		newState.MachineState.IncreasePC()
		condition := conditionBool.Not()
		yices2.PpTerm(os.Stdout, condition.GetRaw(), 1000, 80, 0)
		newState.WorldState.AddConstraint(*condition)
		states = append(states, newState)
	}
	index := util.GetInstructionIndex(globalState.Enviroment.Code.GetInstructions(), int(jumpAddress.Value()))
	if index == -1 {
		fmt.Println("invalid jump destination: ", jumpAddress.Value())
		return &EvaluateResult{
			GlobalStates: states,
		}, nil
	}
	destInstruction := globalState.Enviroment.Code.GetInstructions()[index]
	if destInstruction.OPCode == "JUMPDEST" {
		if trueCondition {
			fmt.Println("trueCondition")
			newState := globalState.Clone()
			newState.GasUsedAdd(int64(opCodeInfo.GasMin), int64(opCodeInfo.GasMax))
			newState.MachineState.Jump(index)
			newState.IncreaseDepth()
			newState.WorldState.AddConstraint(*conditionBool)
			states = append(states, newState)
		}
	}
	return &EvaluateResult{
		GlobalStates: states,
	}, nil
}

func (ins *Instruction) BeginSub(globalState *GlobalState) (*EvaluateResult, error) {
	return nil, fmt.Errorf("beginsub")
}

func (ins *Instruction) JumpSub(globalState *GlobalState) (*EvaluateResult, error) {
	// location
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	if op1.IsSymbolic() {
		return nil, fmt.Errorf("cannot jump to symbolic location")
	}
	location := op1.Value()
	index := util.GetInstructionIndex(globalState.Enviroment.Code.GetInstructions(), int(location))
	if index == -1 {
		return nil, fmt.Errorf("invalid jump destination: %d", index)
	}
	destInstruction := globalState.Enviroment.Code.GetInstructions()[index]
	if destInstruction.OPCode != "BEGINSUB" {
		return nil, fmt.Errorf("invalid jumpsub location: %d", index)
	}
	newPC := globalState.MachineState.GetPC()
	globalState.MachineState.subroutineStack.Push(smt.NewBitVecValInt64(int64(newPC), 256))
	globalState.MachineState.Jump(int(location))

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return nil, fmt.Errorf("beginsub")
}

func (ins *Instruction) ReturnSub(globalState *GlobalState) (*EvaluateResult, error) {
	location, err := globalState.MachineState.subroutineStack.Pop()
	if err != nil {
		return nil, errors.Wrapf(err, "subroutineStack.Pop")
	}
	globalState.MachineState.Jump(int(location.AsBitVec().Value()))

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) PC(globalState *GlobalState) (*EvaluateResult, error) {
	index := globalState.MachineState.GetPC()
	pc := globalState.Enviroment.Code.GetInstructions()[index].Address
	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(int64(pc), 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) MSize(globalState *GlobalState) (*EvaluateResult, error) {
	size := globalState.MachineState.MemSize()
	err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(size, 256))
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) Gas(globalState *GlobalState) (*EvaluateResult, error) {
	gas := globalState.NewBitVec("gas", 256)
	err := globalState.MachineState.PushStack(gas)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) LOg(globalState *GlobalState) (*EvaluateResult, error) {
	if globalState.Enviroment.Static {
		return nil, fmt.Errorf("write protected")
	}
	depth, err := strconv.Atoi(strings.TrimPrefix(ins.OPCode, "LOG"))
	if err != nil {
		return nil, fmt.Errorf("strconv.Atoi: %v", err)
	}
	fmt.Println("log stack pop times: ", depth+2)
	for i := 0; i < depth+2; i++ {
		_, err = globalState.MachineState.PopStack()
		if err != nil {
			return nil, errors.Wrapf(err, "PopStack")
		}
	}

	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}

	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) createTxHelper(
	globalState *GlobalState,
	callvalue, memOffset, memSize, salt *smt.BitVec, opCode string) (*EvaluateResult, error) {

	var (
		calldata      = getCalldata(globalState, memOffset, memSize)
		size, codeEnd int64
		codeRaw       []*smt.BitVec
	)
	if memSize.IsSymbolic() {
		size = 100000
	} else {
		size = memSize.Value()
	}
	for i := int64(0); i < size; i++ {
		data, err := calldata.GetByteAt(smt.NewBitVecValInt64(i, 256))
		if err != nil {
			return nil, errors.Wrapf(err, "GetByteAt")
		}
		if data.IsSymbolic() {
			codeEnd = i
			break
		}
		codeRaw = append(codeRaw, data)
	}
	if len(codeRaw) == 0 {
		log.Infof("trying to exec a create type instruction with no code")
		err := globalState.MachineState.PushStack(smt.NewBitVecValInt64(1, 256))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	var (
		codeBv              = smt.Concats(codeRaw...)
		codeStr             = codeBv.HexString()
		nextTxID            = GetNextTxID()
		constructorArgument = NewConcreteCalldata(nextTxID, codeBv.Bytes()[codeEnd:])
		code                = disassembler.NewDisassembly(codeStr)
		caller              = globalState.Enviroment.ActiveAccount.Address
		origin              = globalState.Enviroment.Origin
		contractAddress     *smt.BitVec
	)
	globalState, err := ins.sha3GasHelper(globalState, int64(len(codeStr)/2))
	if err != nil {
		return nil, errors.Wrapf(err, "sha3GasHelper")
	}
	codeHashStr, codeHashBytes, err := util.GetCodeHash(code.GetBytecode())
	if err != nil {
		return nil, errors.Wrapf(err, "GetCodeHash")
	}
	if salt != nil {
		if salt.IsSymbolic() {
			if salt.Size() != 256 {
				salt = salt.PadToSize(256)
			}
			dataToCompute := smt.Concats(
				smt.NewBitVecValInt64(256, 8),
				caller,
				salt,
				smt.NewBitVecValFromBytes(codeHashBytes, 256),
			)
			address := funcmanager.Kfm.CreateKeccak(dataToCompute)
			contractAddress = smt.NewBitVecFromTerm(yices2.Bvextract(address.GetRaw(), 0, 159), 160)
		} else {
			saltStr := salt.HexString()
			addressStr := caller.HexString()
			_, bytes, err := util.GetCodeHash("0xff" + saltStr + addressStr + codeHashStr)
			if err != nil {
				return nil, errors.Wrapf(err, "GetCodeHash")
			}
			contractAddress = smt.NewBitVecValFromBytes(bytes, 256)
		}
	}

	tx := &ContractCreationTransaction{
		PrevWorldState:  globalState.WorldState.Clone(),
		WorldState:      globalState.WorldState,
		CalleeAccount:   globalState.WorldState.CreateAccount(0, caller, true, caller, code, 0),
		Caller:          caller,
		Code:            code,
		Calldata:        constructorArgument,
		GasPrice:        globalState.Enviroment.GasPrice,
		GasLimit:        smt.NewBitVecValFromInt64(globalState.MachineState.gasLimit, 256),
		Origin:          origin,
		CallValue:       callvalue,
		ContractAddress: contractAddress,
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
		TxStart:      NewTxStart(globalState, opCode, tx),
	}, nil
}

func (ins *Instruction) Create(globalState *GlobalState) (*EvaluateResult, error) {
	if globalState.Enviroment.Static {
		return nil, fmt.Errorf("write protected")
	}

	callvalue, memOffset, memSize, err := globalState.MachineState.PopBitVec3()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec3")
	}
	fmt.Println(callvalue.Value(), memOffset.Value(), memSize.Value())

	return ins.createTxHelper(globalState, callvalue, memOffset, memSize, nil, "CREATE")
}

func (ins *Instruction) createTxPost(globalState *GlobalState) (*EvaluateResult, error) {
	states, err := ins.handleCreateTypePost(globalState, "CREATE_TX_POST")
	if err != nil {
		return nil, errors.Wrapf(err, "handleCreateTypePost")
	}
	err = calculateGas(states)
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: states,
	}, nil
}

func (ins *Instruction) Create2(globalState *GlobalState) (*EvaluateResult, error) {
	if globalState.Enviroment.Static {
		return nil, fmt.Errorf("write protected")
	}
	callvalue, memOffset, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec2")
	}
	memSize, salt, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, errors.Wrapf(err, "PopBitVec2")
	}
	return ins.createTxHelper(globalState, callvalue, memOffset, memSize, salt, "CREATE")
}

func (ins *Instruction) handleCreateTypePost(globalState *GlobalState, opCode string) ([]*GlobalState, error) {
	if opCode == "create2" {
		_, _, err := globalState.MachineState.PopBitVec2()
		if err != nil {
			return nil, errors.Wrapf(err, "PopBitVec2")
		}
		_, _, err = globalState.MachineState.PopBitVec2()
		if err != nil {
			return nil, errors.Wrapf(err, "PopBitVec2")
		}
	} else {
		_, _, _, err := globalState.MachineState.PopBitVec3()
		if err != nil {
			return nil, errors.Wrapf(err, "PopBitVec3")
		}
	}
	var returnVal *smt.BitVec
	if globalState.LastReturnData != nil {
		returnVal = smt.Concats(globalState.LastReturnData.Data...)
	} else {
		returnVal = smt.NewBitVecValInt64(0, 256)
	}
	err := globalState.MachineState.PushStack(returnVal)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	return []*GlobalState{globalState}, nil
}

func (ins *Instruction) Return(globalState *GlobalState) (*EvaluateResult, error) {
	// offset length
	op1, op2, err := globalState.MachineState.PopBitVec2()
	if err != nil {
		return nil, err
	}
	returnData := &ReturnData{
		Data: make([]*smt.BitVec, 0),
		Size: op2,
	}
	if op2.IsSymbolic() {
		returnData.Data = []*smt.BitVec{globalState.NewBitVec("return_data", 8)}
		fmt.Println("return with symbolic length unsupported")
	} else {
		var (
			offset = op1.Value()
			length = op2.Value()
			index  = smt.NewBitVecValInt64(offset, 256)
		)
		fmt.Println(op1.Value(), op2.Value())
		err := globalState.MachineState.MemExtend(op1.Value(), length)
		if err != nil {
			return nil, errors.Wrapf(err, "MemExtend")
		}
		err = checkGasLimit(globalState)
		if err != nil {
			return nil, errors.Wrapf(err, "checkGasLimit")
		}

		for i := int64(0); i < length; i++ {
			byt := globalState.MachineState.MemGetByteAt(index)
			returnData.Data = append(returnData.Data, byt)
			index = index.AddInt64(1)
		}
	}
	if op1.Value() == 128 && op2.Value() == 32 {
		fmt.Println()
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	txEnd, err := NewTxEnd(globalState, returnData, false)
	if err != nil {
		return nil, errors.Wrapf(err, "NewTxEnd")
	}
	return &EvaluateResult{
		GlobalStates: nil,
		TxEnd:        txEnd,
	}, nil
}

func (ins *Instruction) Selfdestruct(globalState *GlobalState) (*EvaluateResult, error) {
	if globalState.Enviroment.Static {
		return nil, fmt.Errorf("write protected")
	}
	// offset
	target, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	amount, err := globalState.Enviroment.ActiveAccount.GetBalance()
	if err != nil {
		return nil, errors.Wrapf(err, "GetBalance")
	}

	err = globalState.WorldState.TransferETH(target, amount)
	if err != nil {
		return nil, errors.Wrapf(err, "TransferETH")
	}

	globalState.Accounts[globalState.Enviroment.ActiveAccount.Address.Value()] = globalState.Enviroment.ActiveAccount.Clone()
	globalState.Enviroment.ActiveAccount.SetBalance(smt.NewBitVecValInt64(0, 256))
	globalState.Enviroment.ActiveAccount.Deleted = true
	txEnd, err := NewTxEnd(globalState, nil, false)
	if err != nil {
		return nil, errors.Wrapf(err, "NewTxEnd")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: nil,
		TxEnd:        txEnd,
	}, nil
}

// Revert
// 指令: REVERT Stop execution and revert state changes, without consuming all provided gas and providing a reason
// gas: 0
// 是否改变worldstate: true
func (ins *Instruction) Revert(globalState *GlobalState) (*EvaluateResult, error) {
	// offset
	op1, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	// length
	op2, err := globalState.MachineState.PopBitVec()
	if err != nil {
		return nil, err
	}
	if op1.IsSymbolic() {
		return nil, fmt.Errorf("symbolic offset is not supported")
	}
	returnData := &ReturnData{}
	if op2.IsSymbolic() {
		// 可能长度不匹配？
		fmt.Println("encounter symbolic lenght, try fill 300 byte")
		// 符号数值，最多填充300的byte
		for i := int64(0); i < 300; i++ {
			var (
				iBv            = smt.NewBitVecValInt64(i, 256)
				condition      = iBv.Lt(op2).GetRaw()
				positiveBranch = globalState.NewBitVec(fmt.Sprintf("return_data_%d", i), 8).GetRaw()
				negativeBranch = smt.NewBitVecValInt64(0, 8).GetRaw()
			)
			fmt.Printf("%d %d", iBv.AsBitVec().Size(), iBv.Lt(op2).Size())
			returnData.Data = append(returnData.Data, smt.NewBitVecFromTerm(
				yices2.Ite(condition, positiveBranch, negativeBranch), 8))
		}
		returnData.Size = smt.NewBitVecValInt64(300, 256)
	} else {
		// 具体数值，从offset处读取length长度的byte
		length := op2.Value()
		fmt.Println(length)
		for i := 0; i < int(length); i++ {
			returnData.Data = append(returnData.Data, globalState.NewBitVec(fmt.Sprintf("return_data_%d", i), 8))
		}
		returnData.Size = op2
	}
	// 写入
	err = globalState.MachineState.MemWriteBytes(op1, returnData.Data...)
	if err != nil {
		for i, data := range returnData.Data {
			fmt.Println(i, "->", data.Size())
		}
		return nil, errors.Wrapf(err, "MemWriteBytes")
	}

	txEnd, err := NewTxEnd(globalState, returnData, true)
	if err != nil {
		return nil, errors.Wrapf(err, "NewTxEnd")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: nil,
		TxEnd:        txEnd,
	}, nil
}

func (ins *Instruction) AssertFail(globalState *GlobalState) (*EvaluateResult, error) {
	return nil, fmt.Errorf("invalid instruction")
}

func (ins *Instruction) Invalid(globalState *GlobalState) (*EvaluateResult, error) {
	return nil, fmt.Errorf("invalid instruction")
}

// Stop
// 指令: STOP Halts execution
// gas: 0
// 是否改变worldstate: true
func (ins *Instruction) Stop(globalState *GlobalState) (*EvaluateResult, error) {
	txEnd, err := NewTxEnd(globalState, nil, false)
	if err != nil {
		return nil, errors.Wrapf(err, "NewTxEnd")
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: nil,
		TxEnd:        txEnd,
	}, nil
}

func (ins *Instruction) writeSymoblicData(globalState *GlobalState, offsetBv, sizeBv *smt.BitVec) error {
	if offsetBv.IsSymbolic() || sizeBv.IsSymbolic() {
		return nil
	}

	var (
		offset         = offsetBv.Value()
		size           = sizeBv.Value()
		returnData     = make([]*smt.BitVec, 0)
		returnDataSize = smt.NewBitVec("returndatasize", 256)
	)
	for i := int64(0); i < size; i++ {
		returnData = append(returnData, globalState.NewBitVec(
			fmt.Sprintf("call_output_var(%d)_%d", offset+i, globalState.MachineState.GetPC()), 8))
	}
	for i := int64(0); i < size; i++ {
		iBv := smt.NewBitVecValInt64(i, 256)
		f1 := iBv.Lt(returnDataSize)
		oldData := globalState.MachineState.MemGetByteAt(iBv)
		dataToWrite := smt.NewBitVecFromTerm(yices2.Ite(f1.GetRaw(), returnData[i].GetRaw(), oldData.GetRaw()), 8)
		err := globalState.MachineState.MemWriteByte(iBv, dataToWrite)
		if err != nil {
			return errors.Wrapf(err, "MemWriteByte")
		}
	}
	globalState.LastReturnData = &ReturnData{
		Data: returnData,
		Size: returnDataSize,
	}
	return nil
}

// Call
// 指令: CALL Message-call into an account
// gas: *
// 是否改变worldstate: true
func (ins *Instruction) Call(globalState *GlobalState) (*EvaluateResult, error) {
	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}
	callParams, err := GetCallParameters(globalState, true)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	if callParams.CalleeAccount != nil && callParams.CalleeAccount.Code.GetBytecode() == "" {
		fmt.Println("this call is about eth transfer")
		sender := globalState.Enviroment.ActiveAccount.Address
		receiver := callParams.CalleeAccount.Address
		err := transferETH(globalState, sender, receiver, callParams.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "transferETH")
		}
		err = ins.writeSymoblicData(globalState, callParams.MemoryOutOffset, callParams.MemoryOutSize)
		if err != nil {
			return nil, errors.Wrapf(err, "writeSymoblicData")
		}
		err = globalState.MachineState.PushStack(globalState.NewBitVec(fmt.Sprintf("retval_%d", currentInstruction.Address), 256))
		if err != nil {
			return nil, errors.Wrapf(err, "writeSymoblicData")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if globalState.Enviroment.Static {
		if callParams.Value.IsSymbolic() {
			globalState.WorldState.AddConstraint(*smt.NewBitVecValInt64(0, 256).Eq(callParams.Value))
		} else if callParams.Value.Value() > 0 {
			return nil, fmt.Errorf("write protection")
		}
	}
	nativeResult, err := NativeCall(
		globalState,
		callParams.CalleeAddress,
		callParams.Calldata,
		callParams.MemoryOutOffset,
		callParams.MemoryOutSize)
	if err != nil {
		return nil, errors.Wrapf(err, "NativeCall")
	}
	if nativeResult != nil {
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: nativeResult,
		}, nil
	}
	/*
			        transaction = MessageCallTransaction(
		            world_state=global_state.world_state,
		            gas_price=environment.gasprice,
		            gas_limit=gas,
		            origin=environment.origin,
		            caller=environment.active_account.address,
		            callee_account=callee_account,
		            call_data=call_data,
		            call_value=value,
		            static=environment.static,
		        )
	*/
	tx := &MessageCallTransaction{
		WorldState:    globalState.WorldState,
		GasPrice:      globalState.Enviroment.GasPrice,
		GasLimit:      callParams.Gas,
		Origin:        globalState.Enviroment.Origin,
		Caller:        globalState.Enviroment.ActiveAccount.Address,
		CalleeAccount: callParams.CalleeAccount,
		Code:          callParams.CalleeAccount.Code,
		Calldata:      callParams.Calldata,
		CallValue:     callParams.Value,
		Static:        globalState.Enviroment.Static,
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
		TxStart:      NewTxStart(globalState, ins.OPCode, tx),
	}, nil
}

func (ins *Instruction) callCode(globalState *GlobalState) (*EvaluateResult, error) {
	stackSize := globalState.MachineState.StackSize()
	memOffset, outSize, err := globalState.MachineState.GetBitVec2(stackSize-7, stackSize-6)
	if err != nil {
		return nil, errors.Wrapf(err, "GetBitVec2")
	}
	callParams, err := GetCallParameters(globalState, true)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	if callParams.CalleeAccount != nil && callParams.CalleeAccount.Code.GetBytecode() != "" {
		log.Infof("call abount eth transfer")
		sender := globalState.Enviroment.ActiveAccount.Address
		receiver := callParams.CalleeAccount.Address
		err = transferETH(globalState, sender, receiver, callParams.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "transferETH")
		}
		err = ins.writeSymoblicData(globalState, memOffset, outSize)
		if err != nil {
			return nil, errors.Wrapf(err, "writeSymoblicData")
		}
		currentInstruction, err := globalState.GetCurrentInstruction()
		if err != nil {
			return nil, errors.Wrapf(err, "GetCurrentInstruction")
		}
		err = globalState.MachineState.PushStack(globalState.NewBitVec(
			fmt.Sprintf("retval_%d", currentInstruction.Address),
			256,
		))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	states, err := NativeCall(
		globalState,
		callParams.CalleeAddress,
		callParams.Calldata,
		callParams.MemoryOutOffset,
		callParams.MemoryOutSize,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "NativeCall")
	}
	if len(states) > 0 {
		err = calculateGas(states)
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: states,
		}, nil
	}

	/*
			        transaction = MessageCallTransaction(
		            world_state=global_state.world_state,
		            gas_price=environment.gasprice,
		            gas_limit=gas,
		            origin=environment.origin,
		            code=callee_account.code,
		            caller=environment.address,
		            callee_account=environment.active_account,
		            call_data=call_data,
		            call_value=value,
		            static=environment.static,
		        )
	*/

	tx := &MessageCallTransaction{
		WorldState:    globalState.WorldState,
		GasPrice:      globalState.Enviroment.GasPrice,
		GasLimit:      callParams.Gas,
		Origin:        globalState.Enviroment.Origin,
		Code:          callParams.CalleeAccount.Code,
		Caller:        globalState.Enviroment.ActiveAccount.Address,
		CalleeAccount: callParams.CalleeAccount,
		Calldata:      callParams.Calldata,
		CallValue:     callParams.Value,
		Static:        globalState.Enviroment.Static,
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
		TxStart:      NewTxStart(globalState, ins.OPCode, tx),
	}, nil
}

func (ins *Instruction) delegatecall(globalState *GlobalState) (*EvaluateResult, error) {
	stackSize := globalState.MachineState.StackSize()
	memOffset, err := globalState.MachineState.GetBitVec(stackSize - 6)
	if err != nil {
		return nil, errors.Wrapf(err, "GetBitVec")
	}
	outSize, err := globalState.MachineState.GetBitVec(stackSize - 5)
	if err != nil {
		return nil, errors.Wrapf(err, "GetBitVec")
	}
	callParams, err := GetCallParameters(globalState, true)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}
	if callParams.CalleeAccount != nil && callParams.CalleeAccount.Code.GetBytecode() != "" {
		log.Infof("call abount eth transfer")
		sender := globalState.Enviroment.ActiveAccount.Address
		receiver := callParams.CalleeAccount.Address
		err = transferETH(globalState, sender, receiver, callParams.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "transferETH")
		}
		err = ins.writeSymoblicData(globalState, memOffset, outSize)
		if err != nil {
			return nil, errors.Wrapf(err, "writeSymoblicData")
		}
		err = globalState.MachineState.PushStack(globalState.NewBitVec(
			fmt.Sprintf("retval_%d", currentInstruction.Address),
			256,
		))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	states, err := NativeCall(
		globalState,
		callParams.CalleeAddress,
		callParams.Calldata,
		callParams.MemoryOutOffset,
		callParams.MemoryOutSize,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "NativeCall")
	}
	if len(states) > 0 {
		err = calculateGas(states)
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: states,
		}, nil
	}
	/*
			        transaction = MessageCallTransaction(
		            world_state=global_state.world_state,
		            gas_price=environment.gasprice,
		            gas_limit=gas,
		            origin=environment.origin,
		            code=callee_account.code,
		            caller=environment.sender,
		            callee_account=environment.active_account,
		            call_data=call_data,
		            call_value=environment.callvalue,
		            static=environment.static,
		        )
	*/
	tx := &MessageCallTransaction{
		WorldState:    globalState.WorldState,
		GasPrice:      globalState.Enviroment.GasPrice,
		GasLimit:      callParams.Gas,
		Origin:        globalState.Enviroment.Origin,
		Code:          callParams.CalleeAccount.Code,
		Caller:        globalState.Enviroment.Sender,
		CalleeAccount: globalState.Enviroment.ActiveAccount,
		Calldata:      callParams.Calldata,
		CallValue:     globalState.Enviroment.CallValue,
		Static:        globalState.Enviroment.Static,
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
		TxStart:      NewTxStart(globalState, ins.OPCode, tx),
	}, nil
}

func (ins *Instruction) callCodePost(globalState *GlobalState) (*EvaluateResult, error) {
	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}

	callParams, err := GetCallParameters(globalState, true)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	var (
		memOffset = callParams.MemoryOutOffset
		size      = callParams.MemoryOutSize
	)
	// 没有真实的数据，需要手动填充返回数据
	if globalState.LastReturnData == nil {
		name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
		returnData := globalState.NewBitVec(name, 256)
		err = globalState.MachineState.PushStack(returnData)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = ins.writeSymoblicData(globalState, memOffset, size)
		if err != nil {
			return nil, errors.Wrapf(err, "writeSymoblicData")
		}
		globalState.WorldState.AddConstraint(*returnData.Eq(smt.NewBitVecValInt64(0, 256)))
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if memOffset.IsSymbolic() || size.IsSymbolic() {
		name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
		returnData := globalState.NewBitVec(name, 256)
		err = globalState.MachineState.PushStack(returnData)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	// 有真实数据，填充具体数据
	var (
		concreteMemOffset = memOffset.Value()
		concreteSize      = size.Value()
		returnSize        int64
	)
	if globalState.LastReturnData.Size.IsSymbolic() {
		returnSize = 500
	} else {
		if concreteSize > globalState.LastReturnData.Size.Value() {
			concreteSize = globalState.LastReturnData.Size.Value()
		}
		err = globalState.MachineState.MemExtend(concreteMemOffset, concreteSize)
		if err != nil {
			return nil, errors.Wrapf(err, "MemExtend")
		}
	}
	for i := int64(0); i < returnSize; i++ {
		err = globalState.MachineState.MemWriteByte(memOffset.AddInt64(i), globalState.LastReturnData.Data[i].Clone().AsBitVec())
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
	}
	name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
	returnData := globalState.NewBitVec(name, 256)
	err = globalState.MachineState.PushStack(returnData)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	globalState.WorldState.AddConstraint(*returnData.Eq(smt.NewBitVecValInt64(1, 256)))
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) delegatecallPost(globalState *GlobalState) (*EvaluateResult, error) {
	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}

	callParams, err := GetCallParameters(globalState, true)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	var (
		memOffset = callParams.MemoryOutOffset
		size      = callParams.MemoryOutSize
	)
	// 没有真实的数据，需要手动填充返回数据
	if globalState.LastReturnData == nil {
		name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
		returnData := globalState.NewBitVec(name, 256)
		err = globalState.MachineState.PushStack(returnData)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		globalState.WorldState.AddConstraint(*returnData.Eq(smt.NewBitVecValInt64(0, 256)))
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	if memOffset.IsSymbolic() || size.IsSymbolic() {
		name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
		returnData := globalState.NewBitVec(name, 256)
		err = globalState.MachineState.PushStack(returnData)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	// 有真实数据，填充具体数据
	var (
		concreteMemOffset = memOffset.Value()
		concreteSize      = size.Value()
		returnSize        int64
	)
	if globalState.LastReturnData.Size.IsSymbolic() {
		returnSize = 500
	} else {
		if concreteSize > globalState.LastReturnData.Size.Value() {
			concreteSize = globalState.LastReturnData.Size.Value()
		}
		err = globalState.MachineState.MemExtend(concreteMemOffset, concreteSize)
		if err != nil {
			return nil, errors.Wrapf(err, "MemExtend")
		}
	}
	for i := int64(0); i < returnSize; i++ {
		err = globalState.MachineState.MemWriteByte(memOffset.AddInt64(i), globalState.LastReturnData.Data[i].Clone().AsBitVec())
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
	}
	name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
	returnData := globalState.NewBitVec(name, 256)
	err = globalState.MachineState.PushStack(returnData)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	globalState.WorldState.AddConstraint(*returnData.Eq(smt.NewBitVecValInt64(1, 256)))
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}

func (ins *Instruction) staticcall(globalState *GlobalState) (*EvaluateResult, error) {
	callParams, err := GetCallParameters(globalState, false)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}
	if callParams.CalleeAccount != nil && callParams.CalleeAccount.Code.GetBytecode() != "" {
		log.Infof("call abount eth transfer")
		sender := globalState.Enviroment.ActiveAccount.Address
		receiver := callParams.CalleeAccount.Address
		err = transferETH(globalState, sender, receiver, callParams.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "transferETH")
		}
		err = ins.writeSymoblicData(globalState, callParams.MemoryOutOffset, callParams.MemoryOutSize)
		if err != nil {
			return nil, errors.Wrapf(err, "writeSymoblicData")
		}
		err = globalState.MachineState.PushStack(globalState.NewBitVec(
			fmt.Sprintf("retval_%d", currentInstruction.Address),
			256,
		))
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	states, err := NativeCall(
		globalState,
		callParams.CalleeAddress,
		callParams.Calldata,
		callParams.MemoryOutOffset,
		callParams.MemoryOutSize,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "NativeCall")
	}
	if len(states) > 0 {
		err = calculateGas(states)
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: states,
		}, nil
	}
	/*
			        transaction = MessageCallTransaction(
		            world_state=global_state.world_state,
		            gas_price=environment.gasprice,
		            gas_limit=gas,
		            origin=environment.origin,
		            code=callee_account.code,
		            caller=environment.address,
		            callee_account=callee_account,
		            call_data=call_data,
		            call_value=value,
		            static=True,
		        )
	*/
	tx := &MessageCallTransaction{
		WorldState:    globalState.WorldState,
		GasPrice:      globalState.Enviroment.GasPrice,
		GasLimit:      callParams.Gas,
		Origin:        globalState.Enviroment.Origin,
		Code:          callParams.CalleeAccount.Code,
		Caller:        globalState.Enviroment.ActiveAccount.Address,
		CalleeAccount: callParams.CalleeAccount,
		Calldata:      callParams.Calldata,
		CallValue:     callParams.Value,
		Static:        true,
	}
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
		TxStart:      NewTxStart(globalState, ins.OPCode, tx),
	}, nil
}

func (ins *Instruction) callPost(globalState *GlobalState) (*EvaluateResult, error) {
	return ins.postHandler(globalState, "call")
}

func (ins *Instruction) staticcallPost(globalState *GlobalState) (*EvaluateResult, error) {
	return ins.postHandler(globalState, "staticcall")
}

func (ins *Instruction) postHandler(globalState *GlobalState, funcName string) (*EvaluateResult, error) {
	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}
	var (
		memOffset *smt.BitVec
		size      *smt.BitVec
		withVale  = funcName != "staticcall"
	)
	if funcName == "staticcall" || funcName == "delegatecall" {
		memOffset, size, err = globalState.MachineState.GetBitVec2(6, 5)
	} else {
		// callcode、delegatecall
		memOffset, size, err = globalState.MachineState.GetBitVec2(7, 6)
	}
	callParams, err := GetCallParameters(globalState, withVale)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCallParameters")
	}
	memOffset, size = callParams.MemoryOutOffset, callParams.MemoryOutSize
	// 没有真实的数据，需要手动填充返回数据
	if globalState.LastReturnData == nil || memOffset.IsSymbolic() || size.IsSymbolic() {
		name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
		returnData := globalState.NewBitVec(name, 256)
		err = globalState.MachineState.PushStack(returnData)
		if err != nil {
			return nil, errors.Wrapf(err, "PushStack")
		}
		err = calculateGas([]*GlobalState{globalState})
		if err != nil {
			return nil, errors.Wrapf(err, "calculateGas")
		}
		globalState.MachineState.IncreasePC()
		return &EvaluateResult{
			GlobalStates: []*GlobalState{globalState},
		}, nil
	}
	// 有真实数据，填充具体数据
	var (
		concreteMemOffset = memOffset.Value()
		concreteSize      = size.Value()
		returnSize        int64
	)
	if globalState.LastReturnData.Size.IsSymbolic() {
		returnSize = 500
	} else {
		if concreteSize > globalState.LastReturnData.Size.Value() {
			concreteSize = globalState.LastReturnData.Size.Value()
		}
		err = globalState.MachineState.MemExtend(concreteMemOffset, concreteSize)
		if err != nil {
			return nil, errors.Wrapf(err, "MemExtend")
		}
	}
	for i := int64(0); i < returnSize; i++ {
		err = globalState.MachineState.MemWriteByte(memOffset.AddInt64(i), globalState.LastReturnData.Data[i].Clone().AsBitVec())
		if err != nil {
			return nil, errors.Wrapf(err, "MemWriteByte")
		}
	}
	name := fmt.Sprintf("returnvalue_%d", currentInstruction.Address)
	returnData := globalState.NewBitVec(name, 256)
	err = globalState.MachineState.PushStack(returnData)
	if err != nil {
		return nil, errors.Wrapf(err, "PushStack")
	}
	globalState.WorldState.AddConstraint(*returnData.Eq(smt.NewBitVecValInt64(1, 256)))
	err = calculateGas([]*GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "calculateGas")
	}
	globalState.MachineState.IncreasePC()
	return &EvaluateResult{
		GlobalStates: []*GlobalState{globalState},
	}, nil
}
