package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types/big"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// AddressLength is expected bytes len for business entity Address
const AddressLength = 32

// Address might be more complicated structure
// contains fields like isIndustrial bool or isMultisig bool
type Address pb.Address

// AddrFromBytes creates address from bytes
func AddrFromBytes(in []byte) *Address {
	addr := &Address{}
	addrBytes := make([]byte, AddressLength)
	copy(addrBytes, in[:32])
	addr.Address = addrBytes
	return addr
}

// AddrFromBase58Check creates address from base58 string
func AddrFromBase58Check(in string) (*Address, error) {
	value, ver, err := base58.CheckDecode(in)
	if err != nil {
		return &Address{}, fmt.Errorf("decoding base58 '%s' failed, err: %w", in, err)
	}

	addr := &Address{}
	addrBytes := make([]byte, AddressLength)
	copy(addrBytes, append([]byte{ver}, value...)[:32])
	addr.Address = addrBytes

	return addr, nil
}

// Equal compares two addresses
func (a *Address) Equal(b *Address) bool {
	return bytes.Equal(a.Address, b.Address)
}

// Bytes returns address bytes
func (a *Address) Bytes() []byte {
	return a.Address
}

// Empty return true if len(a.Address) == 0
func (a *Address) Empty() bool {
	return len(a.Address) == 0
}

// String returns address string
func (a *Address) String() string {
	return base58.CheckEncode(a.Address[1:], a.Address[0])
}

// MarshalJSON marshals address to json
func (a *Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// CheckWithStub checks if the address is blacklisted by querying the account
// information from the provided ChaincodeStubInterface.
func (a *Address) CheckWithStub(stub shim.ChaincodeStubInterface) error {
	if a.Empty() {
		return nil
	}

	accInfo, err := helpers.GetAccountInfo(stub, a.String())
	if err != nil {
		return err
	}

	if accInfo.GetBlackListed() {
		return fmt.Errorf("address %s is blacklisted", a.String())
	}

	return nil
}

// UnmarshalJSON unmarshals address from json
func (a *Address) UnmarshalJSON(data []byte) error {
	var tmp string
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	a1, err := AddrFromBase58Check(tmp)
	if len(tmp) == 0 {
		err = nil
	}
	a.UserID = a1.UserID
	a.Address = a1.Address
	a.IsIndustrial = a1.IsIndustrial
	a.IsMultisig = a1.IsMultisig
	return err
}

func (a *Address) UnmarshalText(text []byte) error {
	addr, err := AddrFromBase58Check(string(text))
	if err != nil && len(string(text)) != 0 {
		return err
	}

	a.UserID = addr.UserID
	a.Address = addr.Address
	a.IsIndustrial = addr.IsIndustrial
	a.IsMultisig = addr.IsMultisig
	return nil
}

// IsUserIDSame checks if userIDs are the same
func (a *Address) IsUserIDSame(b *Address) bool {
	if a.UserID == "" || b.UserID == "" {
		return false
	}
	return a.UserID == b.UserID
}

// Sender is a wrapper for address
type Sender struct {
	addr *Address
}

func (s *Sender) UnmarshalText(text []byte) error {
	addr, err := AddrFromBase58Check(string(text))
	if err != nil {
		return err
	}

	s.addr = addr
	return nil
}

// NewSenderFromAddr creates sender from address
func NewSenderFromAddr(addr *Address) *Sender {
	return &Sender{addr: addr}
}

// Address returns address
func (s *Sender) Address() *Address {
	return s.addr
}

// Equal compares two senders
func (s *Sender) Equal(addr *Address) bool {
	return bytes.Equal(s.addr.Address, addr.Address)
}

// Hex is a wrapper for []byte
type Hex []byte

func (h *Hex) UnmarshalText(text []byte) error {
	value, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}

	*h = value
	return nil
}

// MultiSwapAssets is a wrapper for asset
type MultiSwapAssets struct {
	Assets []*MultiSwapAsset
}

// MultiSwapAsset is a wrapper for asset
type MultiSwapAsset struct {
	Group  string `json:"group,omitempty"`
	Amount string `json:"amount,omitempty"`
}

// ConvertToAsset converts MultiSwapAsset to Asset
func ConvertToAsset(in []*MultiSwapAsset) ([]*pb.Asset, error) {
	if in == nil {
		return nil, errors.New("assets can't be nil")
	}

	assets := make([]*pb.Asset, 0, len(in))
	for _, item := range in {
		value, ok := new(big.Int).SetString(item.Amount, 10) //nolint:gomnd
		if !ok {
			return nil, fmt.Errorf("couldn't convert %s to bigint", item.Amount)
		}
		if value.Cmp(big.NewInt(0)) < 0 {
			return nil, fmt.Errorf("value %s should be positive", item.Amount)
		}

		asset := pb.Asset{}
		asset.Amount = value.Bytes()
		asset.Group = item.Group
		assets = append(assets, &asset)
	}

	return assets, nil
}

// IsValidAddressLen checks if address length is valid
func IsValidAddressLen(val []byte) bool {
	return len(val) == AddressLength
}
