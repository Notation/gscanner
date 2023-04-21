package state

import (
	"fmt"
	"gscanner/internal/disassembler"
	"gscanner/internal/smt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type WorldState struct {
	accounts            map[int64]*Account
	balances            smt.Array
	startingBalances    smt.Array
	constraint          *Constraint
	TransactionSequence []Transaction
	annotations         []smt.Annotation
}

func NewWorldState() *WorldState {
	ws := &WorldState{
		accounts:         make(map[int64]*Account),
		balances:         smt.NewArrayWithNameAndRange("balance", 256),
		startingBalances: smt.NewArrayWithNameAndRange("starting_balance", 256),
		constraint:       NewConstraints(),
		annotations:      make([]smt.Annotation, 0),
	}
	return ws
}
func (ws *WorldState) GetBalance(address *smt.BitVec) (*smt.BitVec, error) {
	return ws.balances.Get(address)
}

func (ws *WorldState) SetBalance(address, value *smt.BitVec) error {
	return ws.balances.Set(address, value)
}

func (ws *WorldState) TransferETH(address, amount *smt.BitVec) error {
	oldBalance, err := ws.balances.Get(address)
	if err != nil {
		return errors.Wrapf(err, "Get")
	}
	newBalance := oldBalance.Add(amount)
	return ws.balances.Set(address, newBalance)
}

func (ws *WorldState) IsConstraintPossible() bool {
	return ws.constraint.IsPossible()
}

func (ws *WorldState) SetConstraint(constrain *Constraint) {
	ws.constraint = constrain
	if len(ws.constraint.constraints) >= 6 {
		fmt.Println()
	}
}

func (ws *WorldState) GetConstraint() *Constraint {
	return ws.constraint
}

func (ws *WorldState) AddConstraint(constrain smt.Bool) {
	ws.constraint.AppendBool(constrain)
	fmt.Println("AddConstraint", len(ws.constraint.constraints))
	if len(ws.constraint.constraints) >= 6 {
		fmt.Println()
	}
}

func (ws *WorldState) AddConstraints(constrains ...smt.Bool) {
	ws.constraint.AppendBools(constrains...)
	fmt.Println("AddConstraints", len(ws.constraint.constraints))
	if len(ws.constraint.constraints) >= 6 {
		fmt.Println()
	}
}

func (ws *WorldState) GetAccounts() map[int64]*Account {
	return ws.accounts
}

func (ws *WorldState) GetAccount(index int64) *Account {
	return ws.accounts[index]
}

func (ws *WorldState) Get(index *smt.BitVec) *Account {
	account, ok := ws.accounts[index.Value()]
	if ok {
		return account
	}
	newAccount := NewAccount(index, disassembler.NewDisassembly(""), ws.balances, 0, "", false)
	ws.accounts[index.Value()] = newAccount
	return newAccount
}

func (ws *WorldState) generateAddress(creator, nonce int64) *smt.BitVec {
	creatorAddress := common.BigToAddress(big.NewInt(creator))
	fmt.Println("creator ", creatorAddress)
	newAddress := crypto.CreateAddress(creatorAddress, uint64(nonce))
	fmt.Println("new     ", newAddress)
	b := new(big.Int)
	b.SetBytes(newAddress[:])
	return smt.NewBitVecValFromBigInt(b, 256)
}

func (ws *WorldState) generateNewAddress(creator *smt.BitVec, nonce int64) *smt.BitVec {
	if creator != nil {
		return ws.generateAddress(creator.Value(), nonce)
	}
	for i := int64(0); i < 66; i++ {
		if _, ok := ws.accounts[i]; ok {
			continue
		}
		return smt.NewBitVecValInt64(i, 256)
	}
	log.Errorf("address for testing has reached searhc limit")
	return nil
}

func (ws *WorldState) Clone() *WorldState {
	newWorldState := &WorldState{
		accounts:            make(map[int64]*Account),
		balances:            ws.balances,
		startingBalances:    ws.startingBalances,
		constraint:          ws.constraint.Clone(),
		TransactionSequence: make([]Transaction, len(ws.TransactionSequence)),
		annotations:         make([]smt.Annotation, len(ws.annotations)),
	}
	copy(newWorldState.TransactionSequence, ws.TransactionSequence)
	copy(newWorldState.annotations, ws.annotations)
	for k, v := range ws.accounts {
		newWorldState.accounts[k] = v.Clone()
	}
	return newWorldState
}

func (ws *WorldState) AccountsExistOrLoad(address *smt.BitVec) *Account {
	account, ok := ws.accounts[address.Value()]
	if ok {
		return account
	}
	return ws.CreateAccount(0, address, false, nil, nil, 0)
}

func (ws *WorldState) CreateConcreteStorageAccount(
	balance int,
	address *smt.BitVec,
	concreteStorage bool,
	creatorAddress *smt.BitVec,
	code *disassembler.Disassembly,
	nonce int64) *Account {
	var (
		creator *Account
		ok      bool
	)
	if address == nil {
		address = ws.generateNewAddress(creatorAddress, nonce)
	}
	if creatorAddress != nil {
		if creator, ok = ws.accounts[creatorAddress.Value()]; ok {
			nonce = creator.Nonce
		} else {
			creator = ws.CreateAccount(0, creatorAddress, false, nil, nil, 0)
		}
	}
	newAccount := NewAccount(address.Clone().AsBitVec(), code, ws.balances, 0, "", true)
	// fmt.Println(newAccount.Address.String())
	newAccount.SetBalance(smt.NewBitVecValInt64(int64(balance), 256))

	ws.PutAccount(newAccount)

	return newAccount
}

func (ws *WorldState) CreateAccount(
	balance int,
	address *smt.BitVec,
	concreteStorage bool,
	creatorAddress *smt.BitVec,
	code *disassembler.Disassembly,
	nonce int64) *Account {
	var (
		creator *Account
		ok      bool
	)
	if address == nil {
		address = ws.generateNewAddress(creatorAddress, nonce)
	}
	if creatorAddress != nil {
		if creator, ok = ws.accounts[creatorAddress.Value()]; ok {
			nonce = creator.Nonce
		} else {
			creator = ws.CreateAccount(0, creatorAddress, false, nil, nil, 0)
		}
	}
	newAccount := NewAccount(address.Clone().AsBitVec(), code, ws.balances, 0, "", false)
	// fmt.Println(newAccount.Address.String())
	newAccount.SetBalance(smt.NewBitVecValInt64(int64(balance), 256))

	ws.PutAccount(newAccount)

	return newAccount
}

func (ws *WorldState) AddAnnotation(Annotations ...smt.Annotation) {
	ws.annotations = append(ws.annotations, Annotations...)
}

func (ws *WorldState) GetAnnotations() []smt.Annotation {
	return ws.GetAnnotations()
}

func (ws *WorldState) FilterAnnotations() {

}

func (ws *WorldState) PutAccount(account *Account) {
	account.Balances = ws.balances
	ws.accounts[account.Address.Value()] = account
}
