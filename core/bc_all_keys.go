package core

import (
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/cctransfer"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
)

type ListPaginatedKeys struct {
	Bookmark string   `json:"bookmark"`
	Keys     []string `json:"keys"`
}

// QueryAllKeys returns total keys of state
func (bc *BaseContract) QueryAllKeys(sender *types.Sender, pageSize int32, bookmark string) (*ListPaginatedKeys, error) {
	// Checks
	if !bc.ContractConfig().IsAdminSet() {
		return nil, cctransfer.ErrAdminNotSet
	}

	if admin, err := types.AddrFromBase58Check(bc.ContractConfig().GetAdmin().GetAddress()); err == nil {
		if !sender.Equal(admin) {
			return nil, ErrUnauthorisedNotAdmin
		}
	} else {
		return nil, fmt.Errorf("creating admin address: %w", err)
	}

	keys, b, err := stubGetKeys(bc.GetStub(), bookmark, pageSize, false)
	if err != nil {
		return nil, err
	}

	return &ListPaginatedKeys{Bookmark: b, Keys: keys}, nil
}

// QueryAllCompositeKeys returns total keys of state
func (bc *BaseContract) QueryAllCompositeKeys(sender *types.Sender, pageSize int32, bookmark string) (*ListPaginatedKeys, error) {
	// Checks
	if !bc.ContractConfig().IsAdminSet() {
		return nil, cctransfer.ErrAdminNotSet
	}

	if admin, err := types.AddrFromBase58Check(bc.ContractConfig().GetAdmin().GetAddress()); err == nil {
		if !sender.Equal(admin) {
			return nil, ErrUnauthorisedNotAdmin
		}
	} else {
		return nil, fmt.Errorf("creating admin address: %w", err)
	}

	keys, b, err := stubGetKeys(bc.GetStub(), bookmark, pageSize, true)
	if err != nil {
		return nil, err
	}

	return &ListPaginatedKeys{Bookmark: b, Keys: keys}, nil
}

func stubGetKeys(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize int32,
	isComposit bool,
) ([]string, string, error) {
	var (
		iter shim.StateQueryIteratorInterface
		meta *peer.QueryResponseMetadata
		err  error
	)

	if isComposit {
		iter, meta, err = stub.GetAllStatesCompositeKeyWithPagination(pageSize, bookmark)
	} else {
		iter, meta, err = stub.GetStateByRangeWithPagination("", "", pageSize, bookmark)
	}
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = iter.Close()
	}()

	keys := make([]string, 0, pageSize)

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, "", err
		}

		k := kv.GetKey()
		if isComposit {
			keyObjectType, keyAttrs, err := stub.SplitCompositeKey(k)
			if err != nil {
				return nil, "", err
			}
			k = strings.Join(append([]string{keyObjectType}, keyAttrs...), "_")
		}

		keys = append(keys, k)
	}

	b := ""
	if meta != nil {
		b = meta.GetBookmark()
	}

	return keys, b, nil
}
