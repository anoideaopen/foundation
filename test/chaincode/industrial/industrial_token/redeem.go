package industrialtoken

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

// RedeemRequest base struct
type RedeemRequest struct {
	TransactionID string         `json:"transactionId"`
	UserAddress   *types.Address `json:"userAddress"`
	GroupName     string         `json:"groupName"`
	Amount        *big.Int       `json:"amounts"`
	Ref           string         `json:"ref"`
}

const redeemRequestKey = "it_redeem_req"

// TxCreateRedeemRequest creates redeem request
func (it *IndustrialToken) TxCreateRedeemRequest(sender *types.Sender, groupName string, amount *big.Int, ref string) error {
	stub := it.GetStub()
	txID := stub.GetTxID()

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	key, err := stub.CreateCompositeKey(redeemRequestKey, []string{txID})
	if err != nil {
		return err
	}

	if err = it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !it.config.GetInitialized() {
		return errors.New("token is not initialized")
	}
	notFound := true
	for _, group := range it.config.GetGroups() {
		if group.GetId() == groupName {
			notFound = false
			break
		}
	}
	if notFound {
		return fmt.Errorf("token group %s not found", groupName)
	}

	jsonRequest, err := json.Marshal(RedeemRequest{
		TransactionID: txID,
		UserAddress:   sender.Address(),
		GroupName:     groupName,
		Amount:        amount,
		Ref:           ref,
	})
	if err != nil {
		return err
	}

	if err = stub.PutState(key, jsonRequest); err != nil {
		return err
	}

	return it.IndustrialBalanceSub(groupName, sender.Address(), amount, "Redeem hold")
}

// QueryRedeemRequestsList returns list of redemption requests
func (it *IndustrialToken) QueryRedeemRequestsList() ([]RedeemRequest, error) {
	stub := it.GetStub()

	iter, err := stub.GetStateByPartialCompositeKey(redeemRequestKey, []string{})
	if err != nil {
		return nil, err
	}

	defer func() {
		err = iter.Close()
	}()

	var result []RedeemRequest

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var req RedeemRequest
		err = json.Unmarshal(res.GetValue(), &req)
		if err != nil {
			return nil, err
		}

		result = append(result, req)
	}

	return result, nil
}

// TxAcceptRedeemRequest - accepts request for tokens redemption
func (it *IndustrialToken) TxAcceptRedeemRequest(sender *types.Sender, requestID string, amount *big.Int, _ string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	bigIntZero := new(big.Int).SetInt64(0)
	// Check limits
	rate, _, err := it.GetRateAndLimits("redeem", "")
	if err != nil {
		return err
	}
	maxRate := new(big.Int).SetBytes(rate.GetMax())
	minRate := new(big.Int).SetBytes(rate.GetMin())
	if amount.Cmp(minRate) < 0 || (maxRate.Cmp(bigIntZero) > 0 && amount.Cmp(maxRate) > 0) {
		return errors.New("incorrect amount")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(redeemRequestKey, []string{requestID})
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

	var req RedeemRequest
	err = json.Unmarshal(rawRequest, &req)
	if err != nil {
		return err
	}

	// delete request from state
	if err = stub.DelState(key); err != nil {
		return err
	}

	if amount.Cmp(req.Amount) == 1 {
		return errors.New("wrong amount to redeem")
	}

	returnAmount := new(big.Int).Sub(req.Amount, amount)

	if amount.Cmp(bigIntZero) > 0 {
		if err = it.changeEmissionInGroup(req.GroupName, amount); err != nil {
			return err
		}
	}

	if returnAmount.Cmp(bigIntZero) > 0 {
		// returning to user not accepted amount of tokens
		return it.IndustrialBalanceAdd(it.ContractConfig().GetSymbol()+"_"+req.GroupName, req.UserAddress, returnAmount, "Redeem returned")
	}

	return nil
}

// TxDenyRedeemRequest - denys request for tokens redemption
func (it *IndustrialToken) TxDenyRedeemRequest(sender *types.Sender, requestID string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(redeemRequestKey, []string{requestID})
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
	if err = stub.DelState(key); err != nil {
		return err
	}

	var req RedeemRequest
	err = json.Unmarshal(rawRequest, &req)
	if err != nil {
		return err
	}

	// returning to user not accepted amount of tokens
	return it.IndustrialBalanceAdd(it.ContractConfig().GetSymbol()+"_"+req.GroupName, req.UserAddress, req.Amount, "Redeem denied")
}

func (it *IndustrialToken) changeEmissionInGroup(groupName string, amount *big.Int) error {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !it.config.GetInitialized() {
		return errors.New("token is not initialized")
	}

	for _, group := range it.config.GetGroups() {
		if group.GetId() == groupName {
			if group.Emission == nil {
				group.Emission = new(big.Int).Bytes()
			}

			if new(big.Int).SetBytes(group.GetEmission()).Cmp(amount) < 0 {
				return errors.New("emission can't become negative")
			}

			group.Emission = new(big.Int).Sub(new(big.Int).SetBytes(group.GetEmission()), amount).Bytes()

			return it.saveConfig()
		}
	}

	return fmt.Errorf("token group %s not found", groupName)
}
