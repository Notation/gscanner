package smt

// func Test_model(t *testing.T) {
// 	yices2.Init()
// 	s := NewSolver()
// 	x := yices2.NewUninterpretedTerm(yices2.IntType())
// 	y := yices2.NewUninterpretedTerm(yices2.IntType())
// 	f := yices2.And2(yices2.ArithEqAtom(x, y), yices2.ArithGtAtom(x, yices2.Int32(2)))
// 	status, model, err := s.Check(f)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, model)
// 	assert.Equal(t, yices2.StatusSat, status)

// 	_, m, _ := model.Eval(f)
// 	fmt.Println(GetInt64Value(m, x))
// 	fmt.Println(GetInt64Value(m, y))

// 	yices2.Exit()
// }

// func Test_modelbv(t *testing.T) {
// 	yices2.Init()
// 	s := NewSolver()
// 	x := yices2.NewUninterpretedTerm(yices2.BvType(256))
// 	y := yices2.NewUninterpretedTerm(yices2.BvType(256))
// 	z := yices2.BvconstInt32(256, 8)
// 	f := yices2.And2(yices2.BveqAtom(x, y), yices2.BveqAtom(yices2.Bvadd(x, y), z))
// 	status, model, err := s.Check(f)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, model)
// 	assert.Equal(t, yices2.StatusSat, status)

// 	status, m, err := model.Eval(f)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, model)
// 	assert.Equal(t, yices2.StatusSat, status)
// 	fmt.Println(GetBitVecTermValue(m, x))
// 	fmt.Println(GetBitVecTermValue(m, y))
// 	fmt.Println(GetBitVecTermValue(m, z))

// 	yices2.Exit()
// }
