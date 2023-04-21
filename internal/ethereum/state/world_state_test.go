package state

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

func Test_getNewAddress(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()
	worldState := &WorldState{}

	addrbv := worldState.generateAddress(9, 1)
	fmt.Println(addrbv.String())

	addbig := addrbv.GetBigInt()
	addrRet := common.BigToAddress(addbig)
	fmt.Println(addrRet.String())
	fmt.Println(hex.EncodeToString(addrbv.Bytes()))
}

// 451621489038867907991407253636460980703141628156
