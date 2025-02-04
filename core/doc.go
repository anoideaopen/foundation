package core

import (
	"encoding/json"
	"errors"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
)

// DocsKey is a key for documents
const DocsKey = "documents"

// Doc json struct
type Doc struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
}

// DocumentsList returns list of documents
func DocumentsList(stub shim.ChaincodeStubInterface) ([]Doc, error) {
	iter, err := stub.GetStateByPartialCompositeKey(DocsKey, []string{})
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = iter.Close()
	}()

	var result []Doc

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var doc Doc
		err = json.Unmarshal(res.GetValue(), &doc)
		if err != nil {
			return nil, err
		}

		result = append(result, doc)
	}

	return result, nil
}

// AddDocs adds documents to the ledger
func AddDocs(stub shim.ChaincodeStubInterface, rawDocs string) error {
	if rawDocs == "" {
		return errors.New("wrong docs parameters")
	}

	var docs []*Doc

	err := json.Unmarshal([]byte(rawDocs), &docs)
	if err != nil {
		return err
	}

	for _, doc := range docs {
		if doc.ID == "" || doc.Hash == "" {
			return errors.New("empty value of doc parameters")
		}

		key, err := stub.CreateCompositeKey(DocsKey, []string{doc.ID})
		if err != nil {
			return err
		}

		// check for the same docID
		rawDoc, err := stub.GetState(key)
		if err != nil {
			return err
		}
		if len(rawDoc) > 0 {
			continue
		}

		docJSON, err := json.Marshal(doc)
		if err != nil {
			return err
		}

		if err = stub.PutState(key, docJSON); err != nil {
			return err
		}
	}

	return nil
}

// DeleteDoc deletes document from the ledger
func DeleteDoc(stub shim.ChaincodeStubInterface, docID string) error {
	key, err := stub.CreateCompositeKey(DocsKey, []string{docID})
	if err != nil {
		return err
	}

	return stub.DelState(key)
}
