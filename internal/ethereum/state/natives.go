package state

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var s = vm.PrecompiledContractsIstanbul

var nativeFuncName = map[common.Address]string{
	common.BytesToAddress([]byte{1}): "ecrecover",
	common.BytesToAddress([]byte{2}): "sha256",
	common.BytesToAddress([]byte{3}): "ripemd160",
	common.BytesToAddress([]byte{4}): "identity",
	common.BytesToAddress([]byte{5}): "mod_exp",
	common.BytesToAddress([]byte{6}): "ec_add",
	common.BytesToAddress([]byte{7}): "ec_mul",
	common.BytesToAddress([]byte{8}): "ec_pair",
	common.BytesToAddress([]byte{9}): "blake2b_fcompress",
}

func nativeContractCall(address int64, calldata Calldata) ([]byte, error) {
	concreteData, ok := calldata.(*ConcreteCalldata)
	if !ok {
		return nil, fmt.Errorf("concrete calldata required")
	}
	function, ok := vm.PrecompiledContractsIstanbul[common.BytesToAddress([]byte{byte(address)})]
	if !ok {
		return nil, fmt.Errorf("no function at address %d", address)
	}
	return function.Run(concreteData.Concrete(nil))
}
