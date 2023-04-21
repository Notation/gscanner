package state

import (
	"github.com/ethereum/go-ethereum/params"
)

func CalculateSha3Gas(length int64) (int64, int64) {
	gas := params.Keccak256Gas + params.Keccak256WordGas*uint64(Ceil32(length)/32)
	return int64(gas), int64(gas)
}

func CalculateNativeGas(size int64, contract string) (int64, int64) {
	var gas uint64
	wordNum := uint64(Ceil32(size) / 32)
	if contract == "ecrecover" {
		gas = params.EcrecoverGas
	} else if contract == "sha256" {
		gas = params.Sha256BaseGas + wordNum*params.Sha256PerWordGas
	} else if contract == "ripemd160" {
		gas = params.Ripemd160BaseGas + wordNum*params.Ripemd160PerWordGas
	} else if contract == "identity" {
		gas = params.IdentityBaseGas + wordNum*params.IdentityPerWordGas
	}
	return int64(gas), int64(gas)
}
