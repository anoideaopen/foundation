package ledger

import (
	"sort"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
)

// RecordBalance - one entry
type RecordBalance struct {
	Key   string   `json:"key"`
	Value *big.Int `json:"value"`
}

type ListBalancePaginatedResponse struct {
	Bookmark         string           `json:"bookmark"`
	Sum              []*RecordBalance `json:"sum"`
	RecordsOfBalance []*RecordBalance `json:"records"`
}

func GivenBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeGiven.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSum(rs),
		RecordsOfBalance: rs,
	}, nil
}

func TokenBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeToken.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSum(rs),
		RecordsOfBalance: rs,
	}, nil
}

func LockedTokenBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeTokenLocked.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSum(rs),
		RecordsOfBalance: rs,
	}, nil
}

func AllowedBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeAllowed.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSums(stub, rs),
		RecordsOfBalance: rs,
	}, nil
}

func LockedAllowedBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeAllowedLocked.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSums(stub, rs),
		RecordsOfBalance: rs,
	}, nil
}

func IndustrialBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeToken.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSums(stub, rs),
		RecordsOfBalance: rs,
	}, nil
}

func LockedIndustrialBalancesGetWithPagination(
	stub shim.ChaincodeStubInterface,
	bookmark string,
	pageSize uint64,
) (*ListBalancePaginatedResponse, error) {
	rs, b, err := getBalancesWithPagination(
		stub,
		balance.BalanceTypeTokenLocked.String(),
		bookmark,
		pageSize,
	)
	if err != nil {
		return nil, err
	}

	return &ListBalancePaginatedResponse{
		Bookmark:         b,
		Sum:              calcSums(stub, rs),
		RecordsOfBalance: rs,
	}, nil
}

func getBalancesWithPagination(
	stub shim.ChaincodeStubInterface,
	prefix string,
	bookmark string,
	pageSize uint64,
) ([]*RecordBalance, string, error) {
	iter, meta, err := stub.GetStateByPartialCompositeKeyWithPagination(
		prefix,
		[]string{},
		int32(pageSize),
		bookmark,
	)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = iter.Close()
	}()

	result := make([]*RecordBalance, 0, meta.GetFetchedRecordsCount())

	for iter.HasNext() {
		var res *queryresult.KV
		res, err = iter.Next()
		if err != nil {
			return nil, "", err
		}

		rec := &RecordBalance{
			Key:   res.GetKey(),
			Value: new(big.Int).SetBytes(res.GetValue()),
		}

		result = append(result, rec)
	}

	return result, meta.GetBookmark(), nil
}

func calcSum(rs []*RecordBalance) []*RecordBalance {
	if len(rs) == 0 {
		return []*RecordBalance{}
	}

	sum := new(big.Int)
	for _, record := range rs {
		sum = new(big.Int).Add(sum, record.Value)
	}

	return []*RecordBalance{
		{Key: "", Value: sum},
	}
}

func calcSums(
	stub shim.ChaincodeStubInterface,
	rs []*RecordBalance,
) []*RecordBalance {
	m := make(map[string]*big.Int)
	for _, record := range rs {
		_, components, err := stub.SplitCompositeKey(record.Key)
		if err != nil {
			return nil
		}

		key := components[len(components)-1]
		if len(components) < 2 {
			key = ""
		}

		sum, ok := m[key]
		if !ok {
			sum = big.NewInt(0)
		}

		sum = new(big.Int).Add(sum, record.Value)
		m[key] = sum
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	records := make([]*RecordBalance, 0, len(keys))
	for _, k := range keys {
		records = append(records, &RecordBalance{Key: k, Value: m[k]})
	}

	return records
}
