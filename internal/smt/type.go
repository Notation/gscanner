package smt

import yices2 "github.com/ianamason/yices2_go_bindings/yices_api"

type StorableType interface {
	GetRaw() yices2.TermT
	Clone() StorableType
	AsBitVec() *BitVec
	AsBool() Bool
	Type() string
	Size() uint32
}

// Annotation 注解类型，用于标记执行逻辑
type Annotation interface {
	Clone() Annotation
	PersistToWorldState() bool
	PersistOverCalls() bool
}

type TxOriginAnnotation struct{}

func (txAnotation *TxOriginAnnotation) Clone() Annotation         { return &TxOriginAnnotation{} }
func (txAnotation *TxOriginAnnotation) PersistToWorldState() bool { return false }
func (txAnotation *TxOriginAnnotation) PersistOverCalls() bool    { return false }

type ReturnValue struct {
	Address int
	Value   *BitVec
}

func (rv *ReturnValue) Clone() *ReturnValue {
	result := &ReturnValue{
		Address: rv.Address,
		Value:   rv.Value.Clone().AsBitVec(),
	}
	return result
}

type UncheckedRetvalAnnotation struct {
	ReturnValues []*ReturnValue
}

func (ur *UncheckedRetvalAnnotation) Clone() Annotation {
	result := &UncheckedRetvalAnnotation{
		ReturnValues: make([]*ReturnValue, len(ur.ReturnValues)),
	}
	for i, val := range ur.ReturnValues {
		result.ReturnValues[i] = val.Clone()
	}
	return result
}
func (ur *UncheckedRetvalAnnotation) PersistToWorldState() bool { return false }
func (ur *UncheckedRetvalAnnotation) PersistOverCalls() bool    { return false }
