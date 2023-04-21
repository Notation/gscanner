package funcmanager

import (
	"encoding/hex"
	"fmt"
	"gscanner/internal/smt"
	"gscanner/internal/util"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/pkg/errors"
)

var (
	TotalParts         = math.BigPow(10, 40) // 10**40
	Part               = math.BigPow(10, 40)
	InternalDifference = math.BigPow(10, 30) // 10**30
)

func init() {
	Part = Part.Sub(Part, big.NewInt(1))
	Part = Part.Sub(Part, TotalParts) // (2**256 - 1)
}

type FuncInfo struct {
	Func        *smt.Function
	FuncReverse *smt.Function
}

type KeccakFunctionManager struct {
	HasMatcher          string
	storeFunction       map[uint32]FuncInfo
	internalHookForSize map[uint32]*big.Int
	hashResultStore     map[uint32][]*smt.BitVec
	concreteHashes      map[*smt.BitVec]*smt.BitVec
	symbolicInputs      map[uint32][]*smt.BitVec
	indexCounter        *big.Int
}

func NewKeccakFunctionManager() *KeccakFunctionManager {
	kfm := &KeccakFunctionManager{
		storeFunction:       make(map[uint32]FuncInfo),
		internalHookForSize: make(map[uint32]*big.Int),
		indexCounter:        TotalParts.Sub(TotalParts, big.NewInt(34534)),
		hashResultStore:     make(map[uint32][]*smt.BitVec),
		concreteHashes:      make(map[*smt.BitVec]*smt.BitVec),
		symbolicInputs:      make(map[uint32][]*smt.BitVec),
	}
	return kfm
}

func (kfm *KeccakFunctionManager) FindConcreteKeccak(data *smt.BitVec) *smt.BitVec {
	bigBv := smt.GetBigBvValue(data.GetRaw())
	bvBytes := fmt.Sprintf("%064s", hex.EncodeToString(bigBv.Bytes()))
	dataSha3, err := util.Sha3(bvBytes)
	if err != nil {
		fmt.Println(errors.Wrap(err, "util.Sha3"))
	}
	s := new(big.Int)
	s.SetBytes(dataSha3)
	return smt.NewBitVecValFromBigInt(s, 256)
}

func (kfm *KeccakFunctionManager) GetFunction(length uint32) FuncInfo {
	funcInfo, ok := kfm.storeFunction[length]
	if ok {
		return funcInfo
	}
	function := smt.NewFunction(fmt.Sprintf("keccak256_%d", length), []uint32{length}, 256)
	inverse := smt.NewFunction(fmt.Sprintf("keccak256_%d-1", length), []uint32{length}, 256)
	funcInfo = FuncInfo{
		function,
		inverse,
	}
	kfm.storeFunction[length] = funcInfo
	kfm.hashResultStore[length] = make([]*smt.BitVec, 0)
	return funcInfo
}

func (kfm *KeccakFunctionManager) GetEmptyKeccakHash() *smt.BitVec {
	s := new(big.Int)
	s.SetString("89477152217924674838424037953991966239322087453347756267410168184682657981552", 10)
	return smt.NewBitVecValFromBigInt(s, 256)
}

func (kfm *KeccakFunctionManager) CreateKeccak(data *smt.BitVec) *smt.BitVec {
	length := data.Size()
	funcInfo := kfm.GetFunction(length)
	if !data.IsSymbolic() {
		concreteHash := kfm.FindConcreteKeccak(data)
		kfm.concreteHashes[data] = concreteHash
		return concreteHash
	}
	if _, ok := kfm.symbolicInputs[length]; !ok {
		kfm.symbolicInputs[length] = make([]*smt.BitVec, 1)
	}
	returnData := funcInfo.Func.Call(data)
	kfm.symbolicInputs[length] = append(kfm.symbolicInputs[length], data)
	kfm.hashResultStore[length] = append(kfm.hashResultStore[length],
		returnData)
	return returnData
}

func (kfm *KeccakFunctionManager) CreateConditions() smt.Bool {
	condition := smt.NewBoolVal(true)
	for _, inputs := range kfm.symbolicInputs {
		for _, input := range inputs {
			c := kfm.createConditions(input)
			condition = smt.NewBoolFromTerm(yices2.And2(
				condition.GetRaw(),
				c.GetRaw(),
			))
		}
	}
	for concreteInput, concreteHash := range kfm.concreteHashes {
		funcInfo := kfm.GetFunction(concreteInput.Size())
		funcReturnData := funcInfo.Func.Call(concreteInput)
		invFuncReturnData := funcInfo.FuncReverse.Call(funcReturnData)
		condition = smt.NewBoolFromTerm(yices2.And3(
			condition.GetRaw(),
			yices2.BveqAtom(funcReturnData.GetRaw(), concreteHash.GetRaw()),
			yices2.BveqAtom(invFuncReturnData.GetRaw(), concreteHash.GetRaw()),
		))
	}
	return condition
}

func (kfm *KeccakFunctionManager) GetConcreteHashData(model *smt.Model) map[uint32][]int {
	concreteHashes := make(map[uint32][]int)
	for size, hashes := range kfm.hashResultStore {
		concreteHashes[size] = make([]int, 0)
		for _, val := range hashes {
			status, m, err := model.Eval(val.GetRaw())
			if err != nil || status != yices2.StatusSat || m == nil {
				continue
			}
			v := smt.GetBvValue(m, val.GetRaw())
			concreteHashes[size] = append(concreteHashes[size], int(v))
		}
	}
	return concreteHashes
}

func (kfm *KeccakFunctionManager) createConditions(funcInput *smt.BitVec) smt.Bool {
	length := funcInput.Size()
	index, ok := kfm.internalHookForSize[length]
	if !ok {
		size := new(big.Int)
		size.SetBytes(kfm.indexCounter.Bytes())
		kfm.internalHookForSize[length] = size
		index = kfm.indexCounter
		kfm.indexCounter = kfm.indexCounter.Sub(kfm.indexCounter, InternalDifference)
	}
	var (
		funcInfo   = kfm.GetFunction(length)
		lowerBound = index.Mul(index, Part)
		upperBound = lowerBound.Add(lowerBound, Part)
	)

	funcReturnData := funcInfo.Func.Call(funcInput)
	invFuncReturnData := funcInfo.FuncReverse.Call(funcReturnData)
	lowerBoundBv := smt.NewBitVecValFromBigInt(lowerBound, 256)
	upperBoundBv := smt.NewBitVecValFromBigInt(upperBound, 256)
	number64 := smt.NewBitVecValInt64(64, 256)
	urmVal := yices2.Bvrem(funcReturnData.GetRaw(), number64.GetRaw())

	f1 := yices2.BveqAtom(invFuncReturnData.GetRaw(), funcInput.GetRaw())
	f2 := yices2.BvsleAtom(lowerBoundBv.GetRaw(), funcReturnData.GetRaw())
	f3 := yices2.BvltAtom(funcReturnData.GetRaw(), upperBoundBv.GetRaw())
	f4 := yices2.BveqAtom(urmVal, yices2.Zero())
	cond := yices2.And([]yices2.TermT{f1, f2, f3, f4})
	concreteCond := smt.NewBoolVal(false)
	for key, keccak := range kfm.concreteHashes {
		if key.Size() == funcInput.Size() {
			hashEq := yices2.And([]yices2.TermT{
				yices2.BveqAtom(funcReturnData.GetRaw(), keccak.GetRaw()),
				yices2.BveqAtom(key.GetRaw(), funcInput.GetRaw()),
			})
			concreteCond = smt.NewBoolFromTerm(yices2.Or2(concreteCond.GetRaw(), hashEq))
		}
	}
	return smt.NewBoolFromTerm(yices2.And2(cond, concreteCond.GetRaw()))
}
