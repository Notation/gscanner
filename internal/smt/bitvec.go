package smt

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

const (
	BitVecType = "bitvec"
	BoolType   = "bool"
)

type BitVec struct {
	name        string
	value       yices2.TermT
	annotations *Set
}

func Concat(lhv, rhv *BitVec) *BitVec {
	result := &BitVec{
		name:        "",
		value:       yices2.Bvconcat2(lhv.value, rhv.value),
		annotations: lhv.annotations.Union(rhv.annotations),
	}
	return result
}

// Concats 拼接一组bv，他们必须是同样的size
func Concats(values ...*BitVec) *BitVec {
	if len(values) == 0 {
		return nil
	}
	terms := make([]yices2.TermT, len(values))
	for i, _ := range values {
		terms[i] = values[i].GetRaw()
	}
	term := yices2.Bvconcat(terms)
	result := NewBitVecFromTerm(term, values[0].Size()*uint32(len(values)))
	fmt.Println("Concats ", yices2.TypeOfTerm(result.GetRaw()), " size ", yices2.TermBitsize(result.GetRaw()))
	return result
}

func NewBitVecValFromInt64(value int64, size uint32) *BitVec {
	return &BitVec{
		name:        "",
		value:       yices2.BvconstInt64(size, value),
		annotations: NewSet(),
	}
}

func NewBitVecVal(value *big.Int, size uint32, annotations ...Annotation) *BitVec {
	bvv := newBitVecValFromBigInt(value, size)
	bvv.annotations = NewSet(annotations...)
	return bvv
}

func NewBitVecValFromBigInt(value *big.Int, size uint32) *BitVec {
	return newBitVecValFromBigInt(value, size)
}

func NewBitVecValFromBytes(bytes []byte, size uint32) *BitVec {
	value := new(big.Int)
	value.SetBytes(bytes)
	bv := newBitVecValFromBigInt(value, size)
	// fmt.Println("NewBitVecValFromBytes ", yices2.TypeOfTerm(bv.GetRaw()))
	return bv
}

func NewBitVecValInt64(value int64, size uint32) *BitVec {
	val := big.NewInt(value)
	bv := newBitVecValFromBigInt(val, size)
	// fmt.Println("NewBitVecValInt64 ", yices2.TypeOfTerm(bv.GetRaw()))
	return bv
}

func NewBitVecValInt64WithName(name string, value int64, size uint32) *BitVec {
	val := big.NewInt(value)
	bv := newBitVecValFromBigInt(val, size)
	// fmt.Println("NewBitVecValInt64 ", yices2.TypeOfTerm(bv.GetRaw()))
	bv.name = name
	return bv
}

func newBitVecValFromBigInt(value *big.Int, size uint32) *BitVec {
	v := make([]int32, value.BitLen())
	for j := 0; j < value.BitLen(); j++ {
		v[j] = int32(value.Bit(j))
	}
	// padding
	// fmt.Println("newBitVecValFromBigInt ", len(v))
	if uint32(len(v)) < size {
		v = append(v, make([]int32, size-uint32(len(v)))...)
	}
	if uint32(len(v)) != size {
		panic(fmt.Errorf("bvsize not %d", size))
	}

	bv := &BitVec{
		name:        "",
		value:       yices2.BvconstFromArray(v),
		annotations: NewSet(),
	}
	// fmt.Println("newBitVecValFromBigInt ", bv.Value(), " -type-> ", yices2.TypeOfTerm(bv.value))
	return bv
}

func NewBitVec(name string, size uint32, annotations ...Annotation) *BitVec {
	term := yices2.NewUninterpretedTerm(yices2.BvType(size))
	errcode := yices2.SetTermName(term, name)
	if errcode < 0 {
		fmt.Println("set term name ", errcode)
	}
	return &BitVec{
		name:        name,
		value:       term,
		annotations: NewSet(annotations...),
	}
}

func NewBitVecFromTerm(value yices2.TermT, size uint32) *BitVec {
	return &BitVec{
		name:        "",
		value:       value,
		annotations: NewSet(),
	}
}

func getBitVecValue(value yices2.TermT) int {
	intVal := make([]int32, 256)
	errorcode := yices2.BvConstValue(value, intVal)
	if errorcode != 0 {
		fmt.Println("BvConstValue ", yices2.ErrorString())
		return 0
	}
	intBytes := make([]byte, 0, 32)
	for i := 0; i < len(intVal); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if intVal[i+j] == 1 {
				b |= 1 << j
			}
		}
		intBytes = append(intBytes, b)
	}
	ret := binary.LittleEndian.Uint32(intBytes)
	return int(ret)
}

func GetBitVecTermValue(model *yices2.ModelT, value yices2.TermT) int64 {
	intVal := make([]int32, 256)
	errorcode := yices2.GetBvValue(*model, value, intVal)
	if errorcode != 0 {
		fmt.Println("GetBvValue ", errorcode)
		return 0
	}
	intBytes := make([]byte, 0, 32)
	for i := 0; i < len(intVal); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if intVal[i+j] == 1 {
				b |= 1 << j
			}
		}
		intBytes = append(intBytes, b)
	}
	ret := binary.LittleEndian.Uint32(intBytes)
	return int64(ret)
}

func GetBigBvValue(value yices2.TermT) *big.Int {
	intVal := make([]int32, yices2.TermBitsize(value))
	errorcode := yices2.BvConstValue(value, intVal)
	if errorcode != 0 {
		fmt.Printf("BvConstValue %s, type %d, size %d\n",
			yices2.ErrorString(), yices2.TypeOfTerm(value), yices2.TermBitsize(value))

		intVal := make([]int32, yices2.TermBitsize(value))
		fmt.Println(len(intVal))
		result := big.NewInt(0)
		for i := 0; i < len(intVal); i++ {
			b := yices2.Bitextract(value, uint32(i))
			// yices2.PpTerm(os.Stdout, b, 1000, 80, 0)
			if yices2.True() == b {
				result = result.SetBit(result, i, 1)
			} else {
				result = result.SetBit(result, i, 0)
			}
		}
		return result
	}

	result := big.NewInt(0)
	for i := 0; i < len(intVal); i++ {
		result = result.SetBit(result, i, uint(intVal[i]))
	}
	return result
}

func (bv *BitVec) Anotate(annotaion Annotation) {
	bv.annotations.Add(annotaion)
}

func (bv *BitVec) GetAnnotations() *Set {
	return bv.annotations
}

func (bv *BitVec) PadToSize(size uint32) *BitVec {
	var (
		oldSize = bv.Size()
		newBv   *BitVec
	)
	// padding
	if oldSize < size {
		newBv = bv.Concat(NewBitVecValInt64(0, size-oldSize))
	} else {
		newBv = bv
	}
	return &BitVec{
		name:        newBv.name,
		value:       newBv.value,
		annotations: newBv.annotations.Clone(),
	}
}

func (bv *BitVec) Clone() StorableType {
	return &BitVec{
		name:        bv.name,
		value:       bv.value,
		annotations: bv.annotations.Clone(),
	}
}

func (bv *BitVec) AsBitVec() *BitVec {
	return bv
}

func (bv *BitVec) AsBool() Bool {
	bvTrue := NewBitVecValInt64(1, 256)
	return Bool{
		name:        bv.name,
		value:       yices2.Eq(bvTrue.value, bv.value),
		annotations: bv.annotations.Clone(),
	}
}

func (bv *BitVec) TermType() int {
	return int(yices2.TypeOfTerm(bv.value))
}

func (bv *BitVec) Bytes() []byte {
	// 大端序
	return bv.GetBigInt().Bytes()
}

func (bv *BitVec) Type() string {
	return BitVecType
}

func (bv *BitVec) AddInt64(n int64) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvadd(bv.GetRaw(), yices2.BvconstInt64(bv.Size(), n)),
		annotations: bv.annotations.Clone(),
	}
}

func (bv *BitVec) GetBigInt() *big.Int {
	return GetBigBvValue(bv.GetRaw())
}

// String 返回byte string
func (bv *BitVec) String() string {
	return GetBigBvValue(bv.GetRaw()).String()
}

// HexString返回16进制编码的string
func (bv *BitVec) HexString() string {
	// 大端序
	bg := bv.GetBigInt()
	return hex.EncodeToString(bg.Bytes())
}

func (bv *BitVec) Size() uint32 {
	return yices2.TermBitsize(bv.value)
}

func (bv *BitVec) IsSymbolic() bool {
	// return yices2.TrmCnstrVariable == yices2.TermConstructor(bv.value)
	return yices2.TermConstructor(bv.value) > 2
}

func (bv *BitVec) Value() int64 {
	// 符号变量没有具体值
	bg := GetBigBvValue(bv.GetRaw())
	return bg.Int64()
}

func (bv *BitVec) GetRaw() yices2.TermT {
	return bv.value
}

func (bv *BitVec) GetName() string {
	return bv.name
}

// Concat 计算结果的size是两者之和
func (bv *BitVec) Concat(other *BitVec) *BitVec {

	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvconcat2(bv.GetRaw(), other.GetRaw()),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// RotateLeft 翻转bits
func (bv *BitVec) RotateLeft() {
	bv.value = yices2.RotateLeft(bv.value, bv.Size())
}

//Not bitwisze not
func (bv *BitVec) Not() *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvnot(bv.value),
		annotations: bv.annotations.Clone(),
	}
}

//Add
func (bv *BitVec) Add(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvadd(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Sub
func (bv *BitVec) Sub(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvsub(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Mul
func (bv *BitVec) Mul(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvmul(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Div
func (bv *BitVec) Div(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvdiv(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) SDiv(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvsdiv(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// And
func (bv *BitVec) And(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvand2(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Or
func (bv *BitVec) Or(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvor2(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Xor
func (bv *BitVec) Xor(other *BitVec) *BitVec {
	return &BitVec{
		name:        bv.name,
		value:       yices2.Bvxor2(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Lt
// Bvs{xxxx} 有符号
// Bv{xxxx} 无符号
func (bv *BitVec) Lt(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvsltAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) Ult(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvltAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Gt
func (bv *BitVec) Gt(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvsgtAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) Ugt(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvgtAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Le
func (bv *BitVec) Le(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvsleAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) Ule(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvleAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Ge
func (bv *BitVec) Ge(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvsgeAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) Uge(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvgeAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) Eq(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.Eq(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) BvEq(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BveqAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) Ne(other *BitVec) *Bool {
	return &Bool{
		name:        bv.name,
		value:       yices2.BvneqAtom(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Lshift
func (bv *BitVec) Shl(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvshl(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// Rshift
// 逻辑右移
func (bv *BitVec) Shr(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvlshr(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

// 算术右移
func (bv *BitVec) AShr(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvashr(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) URem(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvrem(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) SRem(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvsrem(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}

func (bv *BitVec) UDiv(other *BitVec) *BitVec {
	return &BitVec{
		name: bv.name,

		value:       yices2.Bvdiv(bv.value, other.value),
		annotations: bv.annotations.Union(other.annotations),
	}
}
