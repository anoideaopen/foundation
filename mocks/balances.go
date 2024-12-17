package mocks

import (
	"encoding/json"
	"errors"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	pbfound "github.com/anoideaopen/foundation/proto"
)

func (ms *MockStub) addBalance(key string, amount *big.Int) error {
	rawBalance, err := ms.stub.GetState(key)
	if err != nil {
		return err
	}

	bal := new(big.Int).SetBytes(rawBalance)
	bal.Add(amount, bal)

	return ms.stub.PutState(key, bal.Bytes())
}

func (ms *MockStub) subBalance(key string, amount *big.Int) error {
	rawBalance, err := ms.stub.GetState(key)
	if err != nil {
		return err
	}

	bal := new(big.Int).SetBytes(rawBalance)
	if bal.Cmp(amount) == -1 {
		return errors.New("insufficient balance")
	}

	bal.Sub(bal, amount)

	return ms.stub.PutState(key, bal.Bytes())
}

func (ms *MockStub) getBalance(key string) (*big.Int, error) {
	rawBalance, err := ms.stub.GetState(key)
	if err != nil {
		return nil, err
	}

	bal := new(big.Int).SetBytes(rawBalance)
	return bal, nil
}

// Token Balance

func (ms *MockStub) AddTokenBalance(user *UserFoundation, amount *big.Int) error {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
	if err != nil {
		return err
	}

	return ms.addBalance(key, amount)
}

func (ms *MockStub) SubTokenBalance(user *UserFoundation, amount *big.Int) error {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
	if err != nil {
		return err
	}

	return ms.subBalance(key, amount)
}

func (ms *MockStub) GetTokenBalance(user *UserFoundation) (*big.Int, error) {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
	if err != nil {
		return nil, err
	}

	return ms.getBalance(key)
}

// Allowed Balance

func (ms *MockStub) AddAllowedBalance(user *UserFoundation, tokenName string, amount *big.Int) error {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, tokenName})
	if err != nil {
		return err
	}

	return ms.addBalance(key, amount)
}

func (ms *MockStub) SubAllowedBalance(user *UserFoundation, tokenName string, amount *big.Int) error {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, tokenName})
	if err != nil {
		return err
	}

	return ms.subBalance(key, amount)
}

func (ms *MockStub) GetAllowedBalance(user *UserFoundation, tokenName string) (*big.Int, error) {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, tokenName})
	if err != nil {
		return nil, err
	}

	return ms.getBalance(key)
}

// Token Locked Balance

func (ms *MockStub) GetTokenLockedBalance(user *UserFoundation) (*big.Int, error) {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
	if err != nil {
		return nil, err
	}

	return ms.getBalance(key)
}

// Allowed Locked Balance

func (ms *MockStub) GetAllowedLockedBalance(user *UserFoundation, tokenName string) (*big.Int, error) {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, tokenName})
	if err != nil {
		return nil, err
	}

	return ms.getBalance(key)
}

// Token Locked External

func (ms *MockStub) GetTokenExternalLockedInfo(id string) (*pbfound.TokenBalanceLock, error) {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{id})
	if err != nil {
		return nil, err
	}

	rawData, err := ms.stub.GetState(key)
	if err != nil {
		return nil, err
	}
	result := &pbfound.TokenBalanceLock{}
	err = json.Unmarshal(rawData, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Allowed Locked External

func (ms *MockStub) GetAllowedExternalLockedInfo(id string) (*pbfound.AllowedBalanceLock, error) {
	key, err := ms.stub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{id})
	if err != nil {
		return nil, err
	}

	rawData, err := ms.stub.GetState(key)
	if err != nil {
		return nil, err
	}
	result := &pbfound.AllowedBalanceLock{}
	err = json.Unmarshal(rawData, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
