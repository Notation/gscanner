package state

import (
	"fmt"
	"gscanner/internal/smt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/pkg/errors"
)

type Calldata interface {
	CalldataSize() *smt.BitVec
	GetWordAt(*smt.BitVec) (*smt.BitVec, error)
	GetByteAt(*smt.BitVec) (*smt.BitVec, error)
	Concrete(model *smt.Model) []byte
	Size() *smt.BitVec
	Type() string
	Clone() Calldata
}

const (
	CalldataBasic    = "basic"
	CalldataConcrete = "concrete"
	CalldataSymbolic = "symbolic"
)

// ===============================
type ConcreteCalldata struct {
	TxID string

	concreteCalldata []byte
	calldata         smt.Array
}

func NewConcreteCalldata(txID string, calldata []byte) *ConcreteCalldata {
	cc := &ConcreteCalldata{
		TxID:             txID,
		concreteCalldata: make([]byte, len(calldata)),
		calldata:         smt.NewConcreteArrayWithNameAndRange(fmt.Sprintf("concrete_calldata_%s", txID), 8),
	}
	copy(cc.concreteCalldata, calldata)
	for i, data := range calldata {
		// index: 256bits -> value: 8bits
		err := cc.calldata.Set(smt.NewBitVecValInt64(int64(i), 256), smt.NewBitVecValInt64(int64(data), 8))
		if err != nil {
			fmt.Println(err)
		}
	}
	return cc
}
func (cc *ConcreteCalldata) Clone() Calldata {
	result := &ConcreteCalldata{
		TxID:             cc.TxID,
		concreteCalldata: make([]byte, len(cc.concreteCalldata)),
		calldata:         cc.calldata,
	}
	copy(result.concreteCalldata, cc.concreteCalldata)
	return result
}

func (cc *ConcreteCalldata) CalldataSize() *smt.BitVec {
	return cc.Size()
}

// word of 32 bytes
func (cc *ConcreteCalldata) GetWordAt(index *smt.BitVec) (*smt.BitVec, error) {
	offset := index.Value()
	if offset+32 > int64(len(cc.concreteCalldata)) || offset < 0 {
		return nil, fmt.Errorf("wrong offset")
	}
	var (
		currentIndex = smt.NewBitVecValInt64(int64(offset), 256)
		stop         = smt.NewBitVecValInt64(int64(offset+32), 256)
		parts        = make([]*smt.BitVec, 0)
	)
	for {
		solver := smt.NewSolver()
		f := yices2.BveqAtom(currentIndex.GetRaw(), stop.GetRaw())
		status, _, err := solver.Check(f)
		if err != nil {
			return nil, errors.Wrap(err, "solver.Check")
		}
		if status != yices2.StatusSat {
			break
		}
		element, err := cc.calldata.Get(currentIndex)
		if err != nil {
			return nil, errors.Wrap(err, "calldata.GetConcrete")
		}
		parts = append(parts, element)
		currentIndex = currentIndex.AddInt64(1)
	}
	result := smt.Concats(parts...)
	return result, nil
}

func (cc *ConcreteCalldata) GetByteAt(index *smt.BitVec) (*smt.BitVec, error) {
	val := index.Value()
	if val > int64(len(cc.concreteCalldata)) || val < 0 {
		return nil, fmt.Errorf("wrong index")
	}
	return cc.calldata.Get(index)
}

func (cc *ConcreteCalldata) Concrete(model *smt.Model) []byte {
	return cc.concreteCalldata
}

func (cc *ConcreteCalldata) Size() *smt.BitVec {
	return smt.NewBitVecValInt64(int64(len(cc.concreteCalldata)), 256)
}

func (cc *ConcreteCalldata) Type() string {
	return CalldataConcrete
}

// ===============================
type SymbolicCalldata struct {
	TxID     string
	size     *smt.BitVec
	calldata smt.Array
}

func NewSymbolicCalldata(txID string) *SymbolicCalldata {
	cc := &SymbolicCalldata{
		TxID:     txID,
		size:     smt.NewBitVec(txID+"_calldatasize", 256),
		calldata: smt.NewArrayWithNameAndRange(txID+"_calldata", 8),
	}
	return cc
}

func (sc *SymbolicCalldata) Clone() Calldata {
	return &SymbolicCalldata{
		TxID:     sc.TxID,
		size:     sc.size.Clone().AsBitVec(),
		calldata: sc.calldata,
	}
}

func (sc *SymbolicCalldata) CalldataSize() *smt.BitVec {
	return sc.size
}

func (sc *SymbolicCalldata) GetWordAt(index *smt.BitVec) (*smt.BitVec, error) {
	var (
		currentIndex = index.Clone().AsBitVec()
		stop         = index.AddInt64(32)
		parts        = make([]*smt.BitVec, 0)
	)
	for {
		solver := smt.NewSolver()
		f := yices2.BvneqAtom(currentIndex.GetRaw(), stop.GetRaw())
		status, _, err := solver.Check(f)
		if err != nil {
			return nil, errors.Wrap(err, "solver.Check")
		}
		if status != yices2.StatusSat {
			break
		}
		element, err := sc.calldata.Get(currentIndex)
		if err != nil {
			return nil, errors.Wrap(err, "calldata.Get")
		}
		parts = append(parts, element)
		currentIndex = currentIndex.AddInt64(1)
	}
	result := smt.Concats(parts...)
	return result, nil
}

func (cc *SymbolicCalldata) GetByteAt(index *smt.BitVec) (*smt.BitVec, error) {
	return cc.calldata.Get(index)
}

func (sc *SymbolicCalldata) Concrete(model *smt.Model) []byte {
	_, m, _ := model.ModelCompletionEval(sc.size.GetRaw())
	concreteLength := smt.GetInt64Value(m, sc.size.GetRaw())
	result := make([]byte, concreteLength)
	for i := 0; i < int(concreteLength); i++ {
		v, _ := sc.calldata.Get(smt.NewBitVecValInt64(int64(i), 256))
		result = append(result, byte(v.Value()))
	}
	return result
}

func (cc *SymbolicCalldata) Size() *smt.BitVec {
	return cc.size
}

func (sc *SymbolicCalldata) Type() string {
	return CalldataSymbolic
}
