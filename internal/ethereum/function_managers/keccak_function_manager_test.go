package funcmanager

import (
	"encoding/hex"
	"fmt"
	"gscanner/internal/smt"
	"gscanner/internal/util"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

func Test_Kfm(t *testing.T) {
	yices2.Init()

	var terms = make([]yices2.TermT, 0)
	for i := 0; i < 32; i++ {
		p := math.BigPow(256, int64(i))
		pbits := p.Bits()
		v := make([]int32, len(pbits))
		for j := 0; j < len(pbits); j++ {
			v[j] = int32(p.Bit(j))
		}
		assert.Equal(t, len(pbits), len(v))
		terms = append(terms, yices2.BvconstFromArray(v))
	}
	assert.Equal(t, 32, len(terms))

	yices2.Exit()
}

func Test_Func(t *testing.T) {
	yices2.Init()
	var (
		power     = smt.NewFunction("Power", []uint32{256, 256}, 256)
		number256 = smt.NewBitVecValInt64(256, 256)
		terms     = make([]yices2.TermT, 0, 32)
	)
	for i := 0; i < 32; i++ {
		p := math.BigPow(256, int64(i))
		v := make([]int32, p.BitLen())
		for j := 0; j < p.BitLen(); j++ {
			v[j] = int32(p.Bit(j))
		}
		if len(v) < 256 {
			v = append(v, make([]int32, 256-len(v))...)
		}
		t := yices2.Application2(power.GetRaw(), number256.GetRaw(), yices2.BvconstInt32(256, int32(i)))
		f := yices2.BveqAtom(t, yices2.BvconstFromArray(v))
		if f == yices2.NullTerm {
			fmt.Println(yices2.ErrorString())
		}
		terms = append(terms, f)
	}

	solver := smt.NewSolver()
	status, model, err := solver.Check(yices2.And(terms))
	assert.Nil(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, yices2.StatusSat, status)
	yices2.Exit()
}

func Test_FindConcreteKeccak1(t *testing.T) {
	yices2.Init()

	bv := smt.NewBitVecValInt64(999, 999)
	bigBv := smt.GetBigBvValue(bv.GetRaw())
	bvBytes := fmt.Sprintf("%064s", hex.EncodeToString(bigBv.Bytes()))
	assert.Equal(t, "00000000000000000000000000000000000000000000000000000000000003e7", bvBytes)
	dataSha3, err := util.Sha3(bvBytes)
	assert.Nil(t, err)
	assert.Equal(t, "f2b4e536bd23bd6782833c997983bc4a576dc5faca807b4000f207eec069ebd4", hex.EncodeToString(dataSha3))

	s := new(big.Int)
	s.SetBytes(dataSha3)
	m := smt.NewBitVecValFromBigInt(s, 256)
	assert.True(t, m.GetBigInt().Cmp(s) == 0)
	fmt.Println(m.GetBigInt().String())

	yices2.Exit()
}

func Test_FindConcreteKeccak(t *testing.T) {
	yices2.Init()

	testCases := []struct {
		Int      int64
		Expected string
	}{
		{100, "17385872270140913825666367956517731270094621555228275961425792378517567244498"},
		{999, "109779323804479835882177185025246362569772444861502757923736492475615611055060"},
		{0, "18569430475105882587588266137607568536673111973893317399460219858819262702947"},
		{1111111111, "22409142075511682915821369844621669541799728939533748688245866698325363896872"},
	}
	for i := range testCases {
		m := Kfm.FindConcreteKeccak(smt.NewBitVecValInt64(testCases[i].Int, 256))
		assert.Equal(t, testCases[i].Expected, m.GetBigInt().String())
	}

	yices2.Exit()
}

func Test_emptyKeccakHash(t *testing.T) {
	yices2.Init()
	v := "89477152217924674838424037953991966239322087453347756267410168184682657981552"
	m := Kfm.GetEmptyKeccakHash()
	assert.Equal(t, v, m.String())
	yices2.Exit()
}

func Test_loop(t *testing.T) {
	m := map[string]func(){
		"A": func() {
			fmt.Println("A")
		},
		"B": func() {
			fmt.Println("B")
		},
		"C": func() {
			fmt.Println("C")
		},
	}
	var wg sync.WaitGroup
	for _, f := range m {
		wg.Add(1)
		go func(f func()) {
			defer wg.Done()
			f()
		}(f)
	}
	wg.Wait()
}
