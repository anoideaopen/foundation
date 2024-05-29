package document

import (
	"encoding/json"
	"errors"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// DocumentsKey is a key for documents
const DocumentsKey = "documents"

// Document json struct
type Document struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
}

// DocumentsList returns list of documents
func DocumentsList(stub shim.ChaincodeStubInterface) ([]Document, error) {
	iter, err := stub.GetStateByPartialCompositeKey(DocumentsKey, []string{})
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = iter.Close()
	}()

	var result []Document

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var doc Document
		err = json.Unmarshal(res.GetValue(), &doc)
		if err != nil {
			return nil, err
		}

		result = append(result, doc)
	}

	return result, nil
}

// AddDocuments adds documents to the ledger
func AddDocuments(stub shim.ChaincodeStubInterface, rawDocuments string) error {
	if rawDocuments == "" {
		return errors.New("wrong documents parameters")
	}

	var documents []*Document

	err := json.Unmarshal([]byte(rawDocuments), &documents)
	if err != nil {
		return err
	}

	for _, document := range documents {
		if document.ID == "" || document.Hash == "" {
			return errors.New("empty value of document parameters")
		}

		key, err := stub.CreateCompositeKey(DocumentsKey, []string{document.ID})
		if err != nil {
			return err
		}

		// check for the same documentID
		rawDoc, err := stub.GetState(key)
		if err != nil {
			return err
		}
		if len(rawDoc) > 0 {
			continue
		}

		documentJSON, err := json.Marshal(document)
		if err != nil {
			return err
		}

		if err = stub.PutState(key, documentJSON); err != nil {
			return err
		}
	}

	return nil
}

// DeleteDocument deletes document from the ledger
func DeleteDocument(stub shim.ChaincodeStubInterface, documentID string) error {
	key, err := stub.CreateCompositeKey(DocumentsKey, []string{documentID})
	if err != nil {
		return err
	}

	return stub.DelState(key)
}
