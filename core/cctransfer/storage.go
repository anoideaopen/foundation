package cctransfer

import (
	"fmt"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"google.golang.org/protobuf/encoding/protojson"
)

// LoadCCFromTransfer returns entry by id.
func LoadCCFromTransfer(stub shim.ChaincodeStubInterface, idArg string) (*pb.CCTransfer, error) {
	key := CCFromTransfer(idArg)
	data, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}

	cct := new(pb.CCTransfer)
	if len(data) == 0 {
		return nil, ErrNotFound
	}

	if err = protojson.Unmarshal(data, cct); err != nil {
		if err = proto.Unmarshal(data, cct); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
	}
	return cct, nil
}

// LoadCCFromTransfers returns entries by range.
func LoadCCFromTransfers(
	stub shim.ChaincodeStubInterface,
	startKey, endKey, bookmark string,
	pageSize int32,
) (*pb.CCTransfers, error) {
	iter, meta, err := stub.GetStateByRangeWithPagination(startKey, endKey, pageSize, bookmark)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = iter.Close()
	}()

	ccts := new(pb.CCTransfers)

	for iter.HasNext() {
		var kv *queryresult.KV
		kv, err = iter.Next()
		if err != nil {
			return nil, err
		}

		cct := new(pb.CCTransfer)
		if err = protojson.Unmarshal(kv.GetValue(), cct); err != nil {
			if err = proto.Unmarshal(kv.GetValue(), cct); err != nil {
				return nil, fmt.Errorf("unmarshal: %w", err)
			}
		}

		ccts.Ccts = append(ccts.Ccts, cct)
	}

	if meta != nil {
		ccts.Bookmark = meta.GetBookmark()
	}

	return ccts, nil
}

// SaveCCFromTransfer saves entry.
func SaveCCFromTransfer(stub shim.ChaincodeStubInterface, cct *pb.CCTransfer) error {
	if cct == nil {
		return ErrSaveNilTransfer
	}

	if cct.GetId() == "" {
		return ErrEmptyIDTransfer
	}

	data, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(cct)
	if err != nil {
		return err
	}

	return stub.PutState(CCFromTransfer(cct.GetId()), data)
}

// DelCCFromTransfer deletes entry.
func DelCCFromTransfer(stub shim.ChaincodeStubInterface, idArg string) error {
	key := CCFromTransfer(idArg)
	return stub.DelState(key)
}

// LoadCCToTransfer returns entry by id.
func LoadCCToTransfer(stub shim.ChaincodeStubInterface, idArg string) (*pb.CCTransfer, error) {
	key := CCToTransfer(idArg)
	data, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}

	cct := new(pb.CCTransfer)
	if len(data) == 0 {
		return nil, ErrNotFound
	}

	if err = protojson.Unmarshal(data, cct); err != nil {
		if err = proto.Unmarshal(data, cct); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
	}
	return cct, nil
}

// SaveCCToTransfer saves entry.
func SaveCCToTransfer(stub shim.ChaincodeStubInterface, cct *pb.CCTransfer) error {
	if cct == nil {
		return ErrSaveNilTransfer
	}

	if cct.GetId() == "" {
		return ErrEmptyIDTransfer
	}

	data, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(cct)
	if err != nil {
		return err
	}

	return stub.PutState(CCToTransfer(cct.GetId()), data)
}

// DelCCToTransfer deletes entry.
func DelCCToTransfer(stub shim.ChaincodeStubInterface, idArg string) error {
	key := CCToTransfer(idArg)
	return stub.DelState(key)
}
