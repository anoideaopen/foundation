package industrialtoken

import (
	"errors"
	"fmt"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
)

type Metadata struct {
	Name            string          `json:"name"`
	Symbol          string          `json:"symbol"`
	Decimals        uint            `json:"decimals"`
	UnderlyingAsset string          `json:"underlying_asset"`
	Issuer          string          `json:"issuer"`
	DeliveryForm    string          `json:"deliveryForm"`
	UnitOfMeasure   string          `json:"unitOfMeasure"`
	TokensForUnit   string          `json:"tokensForUnit"`
	PaymentTerms    string          `json:"paymentTerms"`
	Price           string          `json:"price"`
	Methods         []string        `json:"methods"`
	Groups          []MetadataGroup `json:"groups"`
	Fee             fee             `json:"fee"`
	Rates           []metadataRate  `json:"rates"`
}

// MetadataGroup struct
type MetadataGroup struct {
	Name         string    `json:"name"`
	Amount       *big.Int  `json:"amount"`
	MaturityDate time.Time `json:"maturityDate"`
	Note         string    `json:"note"`
}

type fee struct {
	Address  string   `json:"address"`
	Currency string   `json:"currency"`
	Fee      *big.Int `json:"fee"`
	Floor    *big.Int `json:"floor"`
	Cap      *big.Int `json:"cap"`
}

type metadataRate struct {
	DealType string   `json:"deal_type"`
	Currency string   `json:"currency"`
	Rate     *big.Int `json:"rate"`
	Min      *big.Int `json:"min"`
	Max      *big.Int `json:"max"`
}

// QueryMetadata returns token Metadata
func (it *IndustrialToken) QueryMetadata() (Metadata, error) {
	m := Metadata{
		Name:            it.extConfig.GetName(),
		Symbol:          it.ContractConfig().GetSymbol(),
		Decimals:        uint(it.extConfig.GetDecimals()),
		UnderlyingAsset: it.extConfig.GetUnderlyingAsset(),
		DeliveryForm:    it.extConfig.GetDeliveryForm(),
		UnitOfMeasure:   it.extConfig.GetUnitOfMeasure(),
		TokensForUnit:   it.extConfig.GetTokensForUnit(),
		PaymentTerms:    it.extConfig.GetPaymentTerms(),
		Price:           it.extConfig.GetPrice(),
		Issuer:          it.Issuer().String(),
		Methods:         it.GetMethods(it),
	}

	if err := it.loadConfigUnlessLoaded(); err != nil {
		return Metadata{}, err
	}

	for _, group := range it.config.GetGroups() {
		m.Groups = append(m.Groups, MetadataGroup{
			Name:         group.GetId(),
			Amount:       new(big.Int).SetBytes(group.GetEmission()),
			MaturityDate: time.Unix(group.GetMaturity(), 0),
			Note:         group.GetNote(),
		})
	}
	if len(it.config.GetFeeAddress()) == 32 {
		m.Fee.Address = types.AddrFromBytes(it.config.GetFeeAddress()).String()
	}

	if it.config.GetFee() != nil {
		m.Fee.Currency = it.config.GetFee().GetCurrency()
		m.Fee.Fee = new(big.Int).SetBytes(it.config.GetFee().GetFee())
		m.Fee.Floor = new(big.Int).SetBytes(it.config.GetFee().GetFloor())
		m.Fee.Cap = new(big.Int).SetBytes(it.config.GetFee().GetCap())
	}

	for _, r := range it.config.GetRates() {
		m.Rates = append(m.Rates, metadataRate{
			DealType: r.GetDealType(),
			Currency: r.GetCurrency(),
			Rate:     new(big.Int).SetBytes(r.GetRate()),
			Min:      new(big.Int).SetBytes(r.GetMin()),
			Max:      new(big.Int).SetBytes(r.GetMax()),
		})
	}

	return m, nil
}

// ChangeGroupMetadata changes metadata for a group of token
func (it *IndustrialToken) ChangeGroupMetadata(groupName string, maturityDate time.Time, note string) error {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !it.config.GetInitialized() {
		return errors.New("token is not initialized")
	}
	notFound := true
	for _, group := range it.config.GetGroups() {
		if group.GetId() == groupName {
			notFound = false
			bChanged := false

			nilTime := time.Time{}

			if maturityDate != nilTime && maturityDate != time.Unix(group.GetMaturity(), 0) {
				bChanged = true
				group.Maturity = maturityDate.Unix()
			}

			if note != "" && note != group.GetNote() {
				bChanged = true
				group.Note = note
			}

			if bChanged {
				return it.saveConfig()
			}

			break
		}
	}
	if notFound {
		return fmt.Errorf("token group %s not found", groupName)
	}

	return nil
}

// QueryIndustrialBalanceOf - returns balance of the token for user address
// WARNING: DO NOT USE CODE LIKE THIS IN REAL TOKENS AS `map[string]string` IS NOT ORDERED
// AND WILL CAUSE ENDORSEMENT MISMATCH ON PEERS. THIS IS FOR TESTING PURPOSES ONLY.
// NOTE: THIS APPROACH IS USED DUE TO LEGACY CODE IN THE FOUNDATION LIBRARY.
// IMPLEMENTING A PROPER SOLUTION WOULD REQUIRE SIGNIFICANT CHANGES.
func (it *IndustrialToken) QueryIndustrialBalanceOf(address *types.Address) (map[string]string, error) {
	return it.IndustrialBalanceGet(address)
}

// QueryAllowedBalanceOf - returns allowed balance of the token for user address
func (it *IndustrialToken) QueryAllowedBalanceOf(address *types.Address, token string) (*big.Int, error) {
	return it.AllowedBalanceGet(token, address)
}

// QueryDocumentsList - returns list of emission documents
func (it *IndustrialToken) QueryDocumentsList() ([]core.Doc, error) {
	return core.DocumentsList(it.GetStub())
}

// TxAddDocs - adds docs to a token
func (it *IndustrialToken) TxAddDocs(sender *types.Sender, rawDocs string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unathorized")
	}

	return core.AddDocs(it.GetStub(), rawDocs)
}

// TxDeleteDoc - deletes doc from state
func (it *IndustrialToken) TxDeleteDoc(sender *types.Sender, docID string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unathorized")
	}

	return core.DeleteDoc(it.GetStub(), docID)
}

// TxSetRate sets token rate to an asset for a type of deal
func (it *IndustrialToken) TxSetRate(sender *types.Sender, dealType string, currency string, rate *big.Int) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}
	// TODO - check if it may be helpful in business logic
	if rate.Sign() == 0 {
		return errors.New("trying to set rate = 0")
	}
	if it.ContractConfig().GetSymbol() == currency {
		return errors.New("currency is equals token: it is impossible")
	}
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	for i, r := range it.config.GetRates() {
		if r.GetDealType() == dealType && r.GetCurrency() == currency {
			it.config.Rates[i].Rate = rate.Bytes()
			return it.saveConfig()
		}
	}
	it.config.Rates = append(it.config.Rates, &proto.TokenRate{
		DealType: dealType,
		Currency: currency,
		Rate:     rate.Bytes(),
		Max:      new(big.Int).SetUint64(0).Bytes(), // todo maybe needs different solution
		Min:      new(big.Int).SetUint64(0).Bytes(),
	})
	return it.saveConfig()
}

// TxSetLimits sets limits for a deal type and an asset
func (it *IndustrialToken) TxSetLimits(sender *types.Sender, dealType string, currency string, min *big.Int, max *big.Int) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}
	if min.Cmp(max) > 0 && max.Cmp(big.NewInt(0)) > 0 {
		return errors.New("min limit is greater than max limit")
	}
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	unknownDealType := true
	for i, r := range it.config.GetRates() {
		if r.GetDealType() == dealType {
			unknownDealType = false
			if r.GetCurrency() == currency {
				it.config.GetRates()[i].Max = max.Bytes()
				it.config.GetRates()[i].Min = min.Bytes()
				return it.saveConfig()
			}
		}
	}
	if unknownDealType {
		return errors.New("unknown DealType")
	}
	return errors.New("unknown currency")
}
