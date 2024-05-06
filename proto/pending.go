package proto

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
)

type addressDump struct {
	UserID       string `json:"user_id,omitempty"` //nolint:tagliatelle
	Address      string `json:"address,omitempty"`
	IsIndustrial bool   `json:"is_industrial,omitempty"` //nolint:tagliatelle
	IsMultisig   bool   `json:"is_multisig,omitempty"`   //nolint:tagliatelle
}

type pendingTxDump struct {
	Method    string       `json:"method"`
	Sender    *addressDump `json:"sender"`
	Args      []string     `json:"args"`
	Timestamp int64
	Nonce     uint64
}

// DumpJSON returns the JSON representation of the pending transaction
func (x *PendingTx) DumpJSON() []byte {
	var sender *addressDump
	if x.GetSender() != nil {
		sender = &addressDump{
			UserID:       x.GetSender().GetUserID(),
			Address:      base58.CheckEncode(x.GetSender().GetAddress()[1:], x.GetSender().GetAddress()[0]),
			IsIndustrial: x.GetSender().GetIsIndustrial(),
			IsMultisig:   x.GetSender().GetIsMultisig(),
		}
	}

	data, err := json.MarshalIndent(&pendingTxDump{
		Method:    x.GetMethod(),
		Sender:    sender,
		Args:      x.GetArgs(),
		Timestamp: x.GetTimestamp(),
		Nonce:     x.GetNonce(),
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	return data
}
