package state

import (
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"
	"strconv"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

var nextTxID int

func GetNextTxID() string {
	nextTxID++
	return strconv.Itoa(nextTxID)
}

func ResetCounter() {
	nextTxID = 0
}

type Transaction interface {
	Clone() Transaction
	GetGasLimit() *smt.BitVec
	GetTxID() string
	InitialGlobalState() (*GlobalState, error)
	GetCaller() *smt.BitVec
	GetOrigin() *smt.BitVec
	SetReturnData(string)
	GetReturnData() string
	GetCalldata() Calldata
	String() string
}

func InitialGlobalStateFromEnviroment(worldState *WorldState, enviroment *Enviroment, activeFunction string) (*GlobalState, error) {
	globalState := NewGlobalState(worldState, enviroment, nil, nil, nil)
	globalState.Enviroment.ActiveFuncName = activeFunction
	sender := enviroment.Sender
	termTypes := yices2.TypeOfTerm(sender.GetRaw())
	fmt.Println(termTypes)
	receiver := enviroment.ActiveAccount.Address
	value := enviroment.CallValue
	senderValue, err := globalState.WorldState.balances.Get(sender)
	if err != nil {
		return nil, err
	}
	receiverValue, err := globalState.WorldState.balances.Get(receiver)
	if err != nil {
		return nil, err
	}
	fmt.Println(sender.IsSymbolic(), receiver.IsSymbolic())
	globalState.WorldState.AddConstraint(*senderValue.Uge(value))
	senderValue.Sub(value)
	receiverValue.Add(value)
	err = globalState.WorldState.balances.Set(sender, senderValue)
	if err != nil {
		return nil, err
	}
	err = globalState.WorldState.balances.Set(receiver, receiverValue)
	if err != nil {
		return nil, err
	}
	return globalState, nil
}

type MessageCallTransaction struct {
	WorldState    *WorldState
	CalleeAccount *Account
	Caller        *smt.BitVec
	Calldata      Calldata
	ID            string
	GasPrice      *smt.BitVec
	GasLimit      *smt.BitVec
	Origin        *smt.BitVec
	Code          *disassembler.Disassembly
	CallValue     *smt.BitVec
	BaseFee       *smt.BitVec
	ReturnData    string
	InitCalldata  bool
	Static        bool
}

func (mct *MessageCallTransaction) Clone() Transaction {
	result := &MessageCallTransaction{
		WorldState:   mct.WorldState.Clone(),
		Caller:       mct.Caller.Clone().AsBitVec(),
		ID:           mct.ID,
		GasLimit:     mct.GasLimit,
		Static:       mct.Static,
		InitCalldata: mct.InitCalldata,
		ReturnData:   mct.ReturnData,
	}
	if mct.CalleeAccount != nil {
		result.CalleeAccount = mct.CalleeAccount.Clone()
	}
	if mct.Calldata != nil {
		result.Calldata = mct.Calldata.Clone()
	}
	if mct.CallValue != nil {
		result.CallValue = mct.CallValue.Clone().AsBitVec()
	}
	if mct.BaseFee != nil {
		result.BaseFee = mct.BaseFee.Clone().AsBitVec()
	}
	if mct.GasPrice != nil {
		result.GasPrice = mct.GasPrice.Clone().AsBitVec()
	}
	if mct.Origin != nil {
		result.Origin = mct.Origin.Clone().AsBitVec()
	}
	if mct.Code != nil {
		result.Code = mct.Code.Clone()
	}
	if mct.GasLimit != nil {
		result.GasLimit = mct.GasLimit.Clone().AsBitVec()
	}
	return result
}

func (mct *MessageCallTransaction) String() string {
	var callee string
	if mct.CalleeAccount != nil {
		callee = mct.CalleeAccount.Address.HexString()
	}
	if mct.Caller.IsSymbolic() {
		return fmt.Sprintf("MessageCallTransaction %s from %s to %s", mct.ID, mct.Caller.GetName(), callee)
	}
	return fmt.Sprintf("MessageCallTransaction %s from %s to %s", mct.ID, mct.Caller.HexString(), callee)
}

func (mct *MessageCallTransaction) GetReturnData() string {
	return mct.ReturnData
}

func (mct *MessageCallTransaction) GetCalldata() Calldata {
	return mct.Calldata
}

func (mct *MessageCallTransaction) GetGasLimit() *smt.BitVec {
	return mct.GasLimit
}

func (mct *MessageCallTransaction) GetCaller() *smt.BitVec {
	return mct.Caller
}

func (mct *MessageCallTransaction) GetOrigin() *smt.BitVec {
	return mct.Origin
}

func (mct *MessageCallTransaction) InitialGlobalState() (*GlobalState, error) {
	enviroment := &Enviroment{
		ActiveAccount: mct.CalleeAccount,
		Sender:        mct.Caller,
		CallData:      mct.Calldata,
		GasPrice:      mct.GasPrice,
		CallValue:     mct.CallValue,
		Origin:        mct.Origin,
		BaseFee:       mct.BaseFee,
		Code:          mct.Code,
		Static:        mct.Static,
	}
	return InitialGlobalStateFromEnviroment(mct.WorldState, enviroment, "fallback")
}

func (mct *MessageCallTransaction) GetTxID() string {
	return mct.ID
}

func (mct *MessageCallTransaction) SetReturnData(data string) {
	mct.ReturnData = data
}

type ContractCreationTransaction struct {
	PrevWorldState  *WorldState
	WorldState      *WorldState
	CalleeAccount   *Account
	Code            *disassembler.Disassembly
	Calldata        Calldata
	Caller          *smt.BitVec
	GasPrice        *smt.BitVec
	Origin          *smt.BitVec
	CallValue       *smt.BitVec
	ContractAddress *smt.BitVec
	BaseFee         *smt.BitVec
	ReturnData      string
	GasLimit        *smt.BitVec
	ID              string
	ContractName    string
}

func (cct *ContractCreationTransaction) InitialGlobalState() (*GlobalState, error) {
	enviroment := &Enviroment{
		ActiveAccount: cct.CalleeAccount,
		Sender:        cct.Caller,
		CallData:      cct.Calldata,
		GasPrice:      cct.GasPrice,
		CallValue:     cct.CallValue,
		Origin:        cct.Origin,
		Code:          cct.Code,
		BaseFee:       cct.BaseFee,
	}
	return InitialGlobalStateFromEnviroment(cct.WorldState, enviroment, "constructor")
}

func (cct *ContractCreationTransaction) Clone() Transaction {
	result := &ContractCreationTransaction{
		WorldState:    cct.WorldState.Clone(),
		CalleeAccount: cct.CalleeAccount.Clone(),
		Code:          cct.Code.Clone(),
		GasLimit:      cct.GasLimit,
		ID:            cct.ID,
		ContractName:  cct.ContractName,
	}
	if cct.Calldata != nil {
		result.Calldata = cct.Calldata.Clone()
	}
	if cct.Caller != nil {
		result.Caller = cct.Caller.Clone().AsBitVec()
	}
	if cct.GasPrice != nil {
		result.GasPrice = cct.GasPrice.Clone().AsBitVec()
	}
	if cct.Origin != nil {
		result.Origin = cct.Origin.Clone().AsBitVec()
	}
	if cct.CallValue != nil {
		result.CallValue = cct.CallValue.Clone().AsBitVec()
	}
	if cct.ContractAddress != nil {
		result.ContractAddress = cct.ContractAddress.Clone().AsBitVec()
	}
	if cct.BaseFee != nil {
		result.BaseFee = cct.BaseFee.Clone().AsBitVec()
	}
	if cct.GasLimit != nil {
		result.GasLimit = cct.GasLimit.Clone().AsBitVec()
	}
	return result
}

func (cct *ContractCreationTransaction) SetReturnData(data string) {
	cct.ReturnData = data
}

func (cct *ContractCreationTransaction) GetCalldata() Calldata {
	return cct.Calldata
}

func (cct *ContractCreationTransaction) GetTxID() string {
	return cct.ID
}

func (cct *ContractCreationTransaction) GetCaller() *smt.BitVec {
	return cct.Caller
}

func (cct *ContractCreationTransaction) GetOrigin() *smt.BitVec {
	return cct.Origin
}

func (cct *ContractCreationTransaction) GetGasLimit() *smt.BitVec {
	return cct.GasLimit
}

func (cct *ContractCreationTransaction) GetReturnData() string {
	return cct.ReturnData
}

func (cct *ContractCreationTransaction) String() string {
	var callee string
	if cct.CalleeAccount != nil {
		callee = cct.CalleeAccount.Address.HexString()
	}
	return fmt.Sprintf("ContractCreationTransaction %s from %s to %s", cct.ID, cct.Caller.HexString(), callee)
}
