package state

// import (
// 	"fmt"
// 	"testing"

// 	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
// 	"github.com/stretchr/testify/assert"
// )

// func Test_Clone(t *testing.T) {
// 	yices2.Init()

// 	m := NewMemory()
// 	assert.Equal(t, m.Size(), 0)

// 	m.Extend(10)
// 	assert.Equal(t, m.Size(), 10)

// 	intType := yices2.IntType()
// 	x := yices2.NewVariable(intType)
// 	y := yices2.NewVariable(intType)
// 	z := yices2.NewVariable(intType)

// 	m.memory[x] = smt.NewBitVecFromTerm(y, 256)

// 	newM := m.Clone()
// 	y1 := m.memory[x]
// 	y2 := newM.memory[x]

// 	assert.Equal(t, y1, y2)

// 	newM.memory[x] = smt.NewBitVecFromTerm(z, 256)
// 	y3 := newM.memory[x]
// 	assert.NotEqual(t, y1, y3)
// 	assert.NotEqual(t, y2, y3)

// 	yices2.Exit()
// }

// func Test_GetWordAt(t *testing.T) {
// 	var (
// 		cfg yices2.ConfigT
// 		ctx yices2.ContextT
// 	)
// 	yices2.Init()
// 	yices2.InitConfig(&cfg)
// 	yices2.InitContext(cfg, &ctx)

// 	model := yices2.GetModel(ctx, 1)
// 	assert.NotEqual(t, nil, model)

// 	m := NewMemory()
// 	m.Extend(128)
// 	// 0x1 0x2 0x3 0x4
// 	k1 := 32
// 	v1 := smt.NewBitVecValInt64(1, 256)
// 	m.memory[k1] = v1
// 	assert.Equal(t, len(m.memory), 128)

// 	term := m.GetWordAt(0)
// 	fmt.Println(term)
// 	assert.Equal(t, term, v1)

// 	yices2.Exit()
// }

// func Test_WriteWordAt(t *testing.T) {
// 	m := NewMemory()
// 	m.Extend(33)
// 	var (
// 		cfg yices2.ConfigT
// 		ctx yices2.ContextT
// 	)
// 	yices2.Init()
// 	yices2.InitConfig(&cfg)
// 	yices2.InitContext(cfg, &ctx)

// 	x := yices2.BvconstUint32(256, 1)
// 	y := yices2.BvconstUint32(256, 0)
// 	g := yices2.ArithEq0Atom(x)
// 	q := yices2.Ite(g, x, y)
// 	m.WriteWordAt(0, q)
// 	rq := m.GetWordAt(0)
// 	assert.Equal(t, q, rq)
// 	printTypeInfo(q)
// 	printTypeInfo(rq)

// 	var (
// 		vq  int32
// 		vrq int32
// 	)
// 	assert.Equal(t, yices2.BoolConstValue(q, &vq), yices2.BoolConstValue(q, &vrq))
// 	fmt.Printf("vq %d, vrq %d\n", vq, vrq)

// 	yices2.Exit()
// }

// func printTypeInfo(term yices2.TermT) {
// 	fmt.Println(yices2.TermToString(term, 80, 30, 0))
// 	fmt.Println("bitsize is ", yices2.TermBitsize(term), ", type is ", yices2.TypeOfTerm(term))
// 	fmt.Println()
// }

// func Test_MemoryWrite(t *testing.T) {
// 	yices2.Init()
// 	var (
// 		cfg yices2.ConfigT
// 		ctx yices2.ContextT
// 	)
// 	yices2.InitConfig(&cfg)
// 	yices2.InitContext(cfg, &ctx)
// 	mem := NewMemory()
// 	mem.Extend(32)

// 	assert.Equal(t, 32, mem.Size())

// 	c := yices2.NewVariable(yices2.BvType(256))
// 	mem.WriteWordAt(0, c)

// 	nc := mem.GetWordAt(0)
// 	assert.Equal(t, c, nc)

// 	fmt.Printf("%+v\n", mem.memory[:])

// 	yices2.Exit()
// }

// func Test_Basic(t *testing.T) {
// 	var (
// 		cfg yices2.ConfigT
// 		ctx yices2.ContextT
// 	)
// 	yices2.Init()
// 	yices2.InitConfig(&cfg)
// 	yices2.InitContext(cfg, &ctx)

// 	var constraints []yices2.TermT
// 	x := yices2.NewUninterpretedTerm(yices2.IntType())
// 	y := yices2.NewUninterpretedTerm(yices2.IntType())

// 	constraints = append(constraints, yices2.ArithEq0Atom(x))
// 	constraints = append(constraints, yices2.ArithGtAtom(y, x))

// 	errorcode := yices2.AssertFormulas(ctx, constraints)
// 	if errorcode < 0 {
// 		fmt.Printf("Assert failed: code = %d, error = %d\n", errorcode, yices2.ErrorCode())
// 	}

// 	status := yices2.CheckContext(ctx, yices2.ParamT{})
// 	switch status {
// 	case yices2.StatusSat:
// 		fmt.Println("The formula is satisfiable")
// 		model := yices2.GetModel(ctx, 1)
// 		if model == nil {
// 			fmt.Println("Error in get_model")
// 		} else {
// 			fmt.Println(yices2.ModelToString(*model, 80, 30, 0))
// 			var value int32
// 			code := yices2.GetInt32Value(*model, x, &value)
// 			if code < 0 {
// 				fmt.Println("Error in get_int32_value for 'x'")
// 			} else {
// 				fmt.Printf("Value of x = %v\n", value)
// 			}
// 			code = yices2.GetInt32Value(*model, y, &value)
// 			if code < 0 {
// 				fmt.Println("Error in get_int32_value for 'y'")
// 			} else {
// 				fmt.Printf("Value of y = %v\n", value)
// 			}
// 		}
// 		yices2.CloseModel(model)
// 	case yices2.StatusUnsat:
// 		fmt.Println("The formula is not satisfiable")
// 	case yices2.StatusIdle:
// 		fallthrough
// 	case yices2.StatusSearching:
// 		fallthrough
// 	case yices2.StatusInterrupted:
// 		fallthrough
// 	case yices2.StatusError:
// 		fmt.Println("Error in check_context")
// 	}

// 	yices2.Exit()
// }

// func Test_WriteMemoryOfMuiltyTypes(t *testing.T) {
// 	m := NewMemory()
// 	// 256 byte
// 	m.Extend(256)
// 	var (
// 		cfg yices2.ConfigT
// 		ctx yices2.ContextT
// 	)
// 	yices2.Init()
// 	yices2.InitConfig(&cfg)
// 	yices2.InitContext(cfg, &ctx)

// 	// write int
// 	// func() {
// 	// 	value := 8
// 	// 	term := yices2.BvconstInt32(256, int32(value))
// 	// 	m.WriteIntAt(0, value)
// 	// 	result := m.GetWordAt(0)
// 	// 	assert.Equal(t, term, result)
// 	// }()

// 	// write bool
// 	func() {
// 		return
// 		var rv, tv int32
// 		value := true

// 		x := yices2.BvconstUint32(256, 1)
// 		y := yices2.BvconstUint32(256, 0)
// 		term := yices2.Ite(yices2.True(), x, y)

// 		m.WriteBoolAt(64, value)
// 		result := m.GetWordAt(64)
// 		yices2.BoolConstValue(result, &rv)
// 		yices2.BoolConstValue(term, &tv)
// 		assert.Equal(t, tv, rv)
// 		fmt.Println(yices2.TermToString(term, 512, 30, 0))
// 		fmt.Printf("term %d, rv %d\n", tv, rv)
// 	}()

// 	func() {
// 		return
// 		var rv, tv int32
// 		value := false

// 		x := yices2.BvconstUint32(256, 1)
// 		y := yices2.BvconstUint32(256, 0)
// 		term := yices2.Ite(yices2.False(), x, y)

// 		m.WriteBoolAt(64, value)
// 		result := m.GetWordAt(64)
// 		errcode := yices2.BoolConstValue(result, &rv)
// 		if errcode < 0 {
// 			fmt.Println(errcode)
// 		}
// 		errcode = yices2.BoolConstValue(term, &tv)
// 		if errcode < 0 {
// 			fmt.Println(errcode)
// 		}
// 		assert.Equal(t, tv, rv)

// 		fmt.Printf("term %d, rv %d\n", tv, rv)
// 	}()

// 	func() {
// 		x := yices2.BvconstUint32(256, 1)
// 		y := yices2.BvconstUint32(256, 0)
// 		q := yices2.Ite(yices2.True(), x, y)

// 		m.WriteWordAt(0, q)
// 		rq := m.GetWordAt(0)
// 		assert.Equal(t, q, rq)
// 		printTypeInfo(q)
// 		printTypeInfo(rq)

// 		var (
// 			vq  int32
// 			vrq int32
// 		)
// 		j := [1]int32{}
// 		errcode := yices2.BvConstValue(q, j[:])
// 		if errcode < 0 {
// 			fmt.Println(errcode)
// 		}
// 		k := [1]int32{}
// 		errcode = yices2.BvConstValue(rq, k[:])
// 		if errcode < 0 {
// 			fmt.Println(errcode)
// 		}
// 		assert.Equal(t, vq, vrq)
// 		fmt.Printf("vq %d, vrq %d\n", vq, vrq)
// 		fmt.Println(j)
// 		fmt.Println(k)
// 	}()

// 	yices2.Exit()
// }
