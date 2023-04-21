package state

import (
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"
)

type Enviroment struct {
	ActiveAccount  *Account
	Sender         *smt.BitVec
	GasPrice       *smt.BitVec
	CallValue      *smt.BitVec
	Origin         *smt.BitVec
	BaseFee        *smt.BitVec
	BlockNumber    *smt.BitVec
	ChainID        *smt.BitVec
	CallData       Calldata
	Code           *disassembler.Disassembly
	Static         bool
	ActiveFuncName string
}

func (env *Enviroment) Clone() *Enviroment {
	newEnv := &Enviroment{
		ActiveAccount:  env.ActiveAccount.Clone(),
		Static:         env.Static,
		ActiveFuncName: env.ActiveFuncName,
	}
	if env.Code != nil {
		newEnv.Code = env.Code.Clone()
	}
	if env.Sender != nil {
		newEnv.Sender = env.Sender.Clone().AsBitVec()
	}
	if env.GasPrice != nil {
		newEnv.GasPrice = env.GasPrice.Clone().AsBitVec()
	}
	if env.CallValue != nil {
		newEnv.CallValue = env.CallValue.Clone().AsBitVec()
	}
	if env.Origin != nil {
		newEnv.Origin = env.Origin.Clone().AsBitVec()
	}
	if env.BaseFee != nil {
		newEnv.BaseFee = env.BaseFee.Clone().AsBitVec()
	}
	if env.BlockNumber != nil {
		newEnv.BlockNumber = env.BlockNumber.Clone().AsBitVec()
	}
	if env.ChainID != nil {
		newEnv.ChainID = env.ChainID.Clone().AsBitVec()
	}
	if env.CallData != nil {
		newEnv.CallData = env.CallData.Clone()
	}
	return newEnv
}
