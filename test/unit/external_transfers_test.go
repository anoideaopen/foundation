package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAdminTransferBalance(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &CustomToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user2 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)
	user2.AddBalance("cc", 500)

	transferRequest := &proto.TransferRequest{
		RequestId:       "",
		Basis:           proto.TransferBasis_TRANSFER_BASIS_INHERITANCE,
		AdministratorId: owner.Address(),
		DocumentType:    proto.DocumentType_DOCUMENT_TYPE_INHERITANCE,
		DocumentNumber:  "1",
		DocumentDate:    timestamppb.New(time.Now()),
		DocumentHashes: []string{
			"hash1", "hash2",
		},
		FromAddress:    user1.Address(),
		ToAddress:      user2.Address(),
		Token:          "",
		Amount:         "600",
		Reason:         "test transfer",
		BalanceType:    proto.BalanceType_BALANCE_TYPE_TOKEN,
		AdditionalInfo: nil,
	}

	data, err := json.Marshal(transferRequest)
	require.NoError(t, err)

	t.Log(string(data))

	err = owner.RawSignedInvokeWithErrorReturned("cc", "transferBalance", string(data))
	require.NoError(t, err)

	user1.BalanceShouldBe("cc", 400)
	user2.BalanceShouldBe("cc", 1100)
}

func TestNotAdminFailedTransfer(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)
	initMsg := ledger.NewCC("cc", &CustomToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user2 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)
	user2.AddBalance("cc", 500)

	transferRequest := &proto.TransferRequest{
		RequestId:       "",
		Basis:           proto.TransferBasis_TRANSFER_BASIS_INHERITANCE,
		AdministratorId: owner.Address(),
		DocumentType:    proto.DocumentType_DOCUMENT_TYPE_INHERITANCE,
		DocumentNumber:  "1",
		DocumentDate:    timestamppb.New(time.Now()),
		DocumentHashes: []string{
			"hash1", "hash2",
		},
		FromAddress:    user1.Address(),
		ToAddress:      user2.Address(),
		Token:          "",
		Amount:         "600",
		Reason:         "test transfer",
		BalanceType:    proto.BalanceType_BALANCE_TYPE_TOKEN,
		AdditionalInfo: nil,
	}

	data, err := json.Marshal(transferRequest)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("cc", "transferBalance", string(data))
	require.EqualError(t, err, core.ErrUnauthorisedNotAdmin.Error())

	user1.BalanceShouldBe("cc", 1000)
	user2.BalanceShouldBe("cc", 500)
}

func TestInsufficientFundsTransfer(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)
	initMsg := ledger.NewCC("cc", &CustomToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user2 := ledger.NewWallet()
	user1.AddBalance("cc", 500)
	user2.AddBalance("cc", 500)

	transferRequest := &proto.TransferRequest{
		RequestId:       "",
		Basis:           proto.TransferBasis_TRANSFER_BASIS_INHERITANCE,
		AdministratorId: owner.Address(),
		DocumentType:    proto.DocumentType_DOCUMENT_TYPE_INHERITANCE,
		DocumentNumber:  "1",
		DocumentDate:    timestamppb.New(time.Now()),
		DocumentHashes: []string{
			"hash1", "hash2",
		},
		FromAddress:    user1.Address(),
		ToAddress:      user2.Address(),
		Token:          "",
		Amount:         "600",
		Reason:         "test transfer",
		BalanceType:    proto.BalanceType_BALANCE_TYPE_TOKEN,
		AdditionalInfo: nil,
	}

	data, err := json.Marshal(transferRequest)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "transferBalance", string(data))
	require.EqualError(t, err, balance.ErrInsufficientBalance.Error())

	user1.BalanceShouldBe("cc", 500)
	user2.BalanceShouldBe("cc", 500)
}

func TestNegativeAmountTransfer(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)
	initMsg := ledger.NewCC("cc", &CustomToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user2 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)
	user2.AddBalance("cc", 500)

	transferRequest := &proto.TransferRequest{
		RequestId:       "",
		Basis:           proto.TransferBasis_TRANSFER_BASIS_INHERITANCE,
		AdministratorId: owner.Address(),
		DocumentType:    proto.DocumentType_DOCUMENT_TYPE_INHERITANCE,
		DocumentNumber:  "1",
		DocumentDate:    timestamppb.New(time.Now()),
		DocumentHashes: []string{
			"hash1", "hash2",
		},
		FromAddress:    user1.Address(),
		ToAddress:      user2.Address(),
		Token:          "",
		Amount:         "-100",
		Reason:         "test transfer",
		BalanceType:    proto.BalanceType_BALANCE_TYPE_TOKEN,
		AdditionalInfo: nil,
	}

	data, err := json.Marshal(transferRequest)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "transferBalance", string(data))
	require.EqualError(t, err, "amount must be greater than zero")

	user1.BalanceShouldBe("cc", 1000)
	user2.BalanceShouldBe("cc", 500)
}

func TestZeroAmountTransfer(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)
	initMsg := ledger.NewCC("cc", &CustomToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user2 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)
	user2.AddBalance("cc", 500)

	transferRequest := &proto.TransferRequest{
		RequestId:       "",
		Basis:           proto.TransferBasis_TRANSFER_BASIS_INHERITANCE,
		AdministratorId: owner.Address(),
		DocumentType:    proto.DocumentType_DOCUMENT_TYPE_INHERITANCE,
		DocumentNumber:  "1",
		DocumentDate:    timestamppb.New(time.Now()),
		DocumentHashes: []string{
			"hash1", "hash2",
		},
		FromAddress:    user1.Address(),
		ToAddress:      user2.Address(),
		Token:          "",
		Amount:         "0",
		Reason:         "test transfer",
		BalanceType:    proto.BalanceType_BALANCE_TYPE_TOKEN,
		AdditionalInfo: nil,
	}

	data, err := json.Marshal(transferRequest)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "transferBalance", string(data))
	require.EqualError(t, err, "amount must be greater than zero")

	user1.BalanceShouldBe("cc", 1000)
	user2.BalanceShouldBe("cc", 500)
}
