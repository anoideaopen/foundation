package proto

import "github.com/btcsuite/btcutil/base58"

// AddrString returns the address string
func (x *Address) AddrString() string {
	return base58.CheckEncode(x.GetAddress()[1:], x.GetAddress()[0])
}

// Addr returns the address
func (x *AclResponse) Addr() (out [32]byte) {
	if x.GetAddress() == nil {
		return [32]byte{}
	}
	copy(out[:], x.GetAddress().GetAddress().GetAddress()[:32])
	return out
}
