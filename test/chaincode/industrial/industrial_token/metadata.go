package industrialtoken

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anoideaopen/foundation/core/types"
)

// MaturityChangeRequest base struct
type MaturityChangeRequest struct {
	TransactionID string         `json:"transactionId"`
	UserAddress   *types.Address `json:"userAddress"`
	GroupName     string         `json:"groupName"`
	MaturityDate  time.Time      `json:"maturityDate"`
	Ref           string         `json:"ref"`
}

const (
	mcRequestKey = "it_maturity_req"
	timeFormat   = "02.01.2006 15:04:05"
)

// TxCreateMCRequest creates maturity date change request
func (it *IndustrialToken) TxCreateMCRequest(sender *types.Sender, groupName, maturityDateString, ref string) error {
	stub := it.GetStub()
	txID := stub.GetTxID()

	key, err := stub.CreateCompositeKey(mcRequestKey, []string{txID})
	if err != nil {
		return err
	}

	cfg, err := it.loadConfig()
	if err != nil {
		return err
	}
	if !cfg.GetInitialized() {
		return errors.New("token is not initialized")
	}

	maturityDate, err := time.Parse(timeFormat, maturityDateString)
	if err != nil {
		return err
	}

	oldMaturityDate := time.Time{}
	notFound := true
	for _, group := range cfg.GetGroups() {
		if group.GetId() == groupName {
			oldMaturityDate = time.Unix(group.GetMaturity(), 0)
			notFound = false
			break
		}
	}
	if notFound {
		return fmt.Errorf("token group %s not found", groupName)
	}

	timeNow := time.Now()

	if maturityDate.Before(timeNow) || maturityDate.Equal(timeNow) {
		return errors.New("maturity date should be greater than now")
	}

	if maturityDate.Before(oldMaturityDate) || maturityDate.Equal(timeNow) {
		return fmt.Errorf("maturity date should be greater than %v", oldMaturityDate)
	}

	jsonRequest, err := json.Marshal(MaturityChangeRequest{
		TransactionID: txID,
		UserAddress:   sender.Address(),
		GroupName:     groupName,
		MaturityDate:  maturityDate,
		Ref:           ref,
	})
	if err != nil {
		return err
	}

	return stub.PutState(key, jsonRequest)
}

// QueryMCRequestsList returns list of maturity dates change requests
func (it *IndustrialToken) QueryMCRequestsList() ([]MaturityChangeRequest, error) {
	stub := it.GetStub()

	iter, err := stub.GetStateByPartialCompositeKey(mcRequestKey, []string{})
	if err != nil {
		return nil, err
	}

	defer func() {
		err = iter.Close()
	}()

	var result []MaturityChangeRequest

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var req MaturityChangeRequest
		err = json.Unmarshal(res.GetValue(), &req)
		if err != nil {
			return nil, err
		}

		result = append(result, req)
	}

	return result, nil
}

// TxAcceptMCRequest - accepts request for tokens maturity date change
func (it *IndustrialToken) TxAcceptMCRequest(sender *types.Sender, requestID, _ string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(mcRequestKey, []string{requestID})
	if err != nil {
		return err
	}

	rawRequest, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawRequest) == 0 {
		return errors.New("request with this key not found")
	}

	var req MaturityChangeRequest
	err = json.Unmarshal(rawRequest, &req)
	if err != nil {
		return err
	}

	// delete request from state
	if err = stub.DelState(key); err != nil {
		return err
	}

	return it.ChangeGroupMetadata(req.GroupName, req.MaturityDate, "")
}

// TxDenyMCRequest - denys request for tokens maturity date change
func (it *IndustrialToken) TxDenyMCRequest(sender *types.Sender, requestID string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(mcRequestKey, []string{requestID})
	if err != nil {
		return err
	}

	rawRequest, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawRequest) == 0 {
		return errors.New("request with this key not found")
	}

	// delete request from state
	return stub.DelState(key)
}

// TxChangeGroupNote changes token group note
func (it *IndustrialToken) TxChangeGroupNote(sender *types.Sender, groupName, note string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	return it.ChangeGroupMetadata(groupName, time.Time{}, note)
}
