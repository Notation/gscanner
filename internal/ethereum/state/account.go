package state

import (
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

type Storage struct {
	concrete bool
	storage  smt.Array

	knownKeys map[yices2.TermT]*smt.BitVec
}

func NewStorage() *Storage {
	return &Storage{
		storage: smt.NewArray(),
	}
}

func NewConcreteStorage() *Storage {
	return &Storage{
		storage:   smt.NewArray(),
		concrete:  true,
		knownKeys: make(map[yices2.TermT]*smt.BitVec),
	}
}

func (s *Storage) Get(key *smt.BitVec) (*smt.BitVec, error) {
	if s.concrete {
		_, ok := s.knownKeys[key.GetRaw()]
		if !ok {
			// 返回默认值
			return smt.NewBitVecValInt64(0, s.storage.GetRange()), nil
		}
	}
	return s.storage.Get(key)
}

func (s *Storage) Set(key, value *smt.BitVec) error {
	err := s.storage.Set(key, value)
	if err != nil {
		return err
	}
	if s.concrete {
		s.knownKeys[key.GetRaw()] = value.Clone().AsBitVec()
	}
	return nil
}

func (s *Storage) Clone() *Storage {
	result := &Storage{
		concrete: s.concrete,
		storage:  s.storage,
	}
	if s.concrete {
		result.knownKeys = make(map[yices2.TermT]*smt.BitVec)
		for k, v := range s.knownKeys {
			result.knownKeys[k] = v
		}
		fmt.Println("copy concrete storage ", len(result.knownKeys), "elements")
	}
	return result
}

type Account struct {
	Nonce        int64
	Code         *disassembler.Disassembly
	Address      *smt.BitVec
	Storage      *Storage
	ContractName string
	Deleted      bool
	Balances     smt.Array
}

func NewAccount(address *smt.BitVec,
	code *disassembler.Disassembly,
	balances smt.Array,
	nonce int64,
	contractName string,
	concrete bool) *Account {
	account := &Account{
		Address:  address,
		Nonce:    nonce,
		Balances: balances,
	}
	if concrete {
		account.Storage = NewConcreteStorage()
	} else {
		account.Storage = NewStorage()
	}
	if contractName == "" {
		if address.IsSymbolic() {
			account.ContractName = "unknown"
		} else {
			account.ContractName = fmt.Sprintf("%s", account.Address.HexString())
		}
	} else {
		account.ContractName = contractName
	}
	if code == nil {
		account.Code = disassembler.NewDisassembly("")
	} else {
		account.Code = code
	}
	return account
}

func (account *Account) GetBalance() (*smt.BitVec, error) {
	return account.Balances.Get(account.Address)
}

func (account *Account) SetBalance(value *smt.BitVec) {
	err := account.Balances.Set(account.Address, value)
	if err != nil {
		// fmt.Println(fmt.Errorf("SetBalance %v", err))
	}
}

func (account *Account) StorageGet(key *smt.BitVec) (*smt.BitVec, error) {
	return account.Storage.Get(key)
}

func (account *Account) StorageSet(key, value *smt.BitVec) error {
	return account.Storage.Set(key, value)
}

func (account *Account) SerialisedCode() string {
	return account.Code.GetBytecode()
}

func (account *Account) Clone() *Account {
	return &Account{
		Nonce:        account.Nonce,
		Code:         account.Code.Clone(),
		Address:      account.Address.Clone().AsBitVec(),
		Storage:      account.Storage.Clone(),
		ContractName: account.ContractName,
		Deleted:      account.Deleted,
		Balances:     account.Balances,
	}
}
