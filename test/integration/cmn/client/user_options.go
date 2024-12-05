package client

import pbfound "github.com/anoideaopen/foundation/proto"

// userOptions is a struct for users key types
type userOptions struct {
	AdminKeyType            pbfound.KeyType
	FeeSetterKeyType        pbfound.KeyType
	FeeAddressSetterKeyType pbfound.KeyType
}

// UserOption specifies some userOptions parameter
type UserOption func(opts *userOptions) error

// WithAdminKeyType specifies userOptions admin key type
func WithAdminKeyType(keyType pbfound.KeyType) UserOption {
	return func(opts *userOptions) error {
		opts.AdminKeyType = keyType
		return nil
	}
}

// WithFeeSetterKeyType specifies userOptions fee setter key type
func WithFeeSetterKeyType(keyType pbfound.KeyType) UserOption {
	return func(opts *userOptions) error {
		opts.FeeSetterKeyType = keyType
		return nil
	}
}

// WithFeeAddressSetterKeyType specifies userOptions fee address setter key type
func WithFeeAddressSetterKeyType(keyType pbfound.KeyType) UserOption {
	return func(opts *userOptions) error {
		opts.FeeAddressSetterKeyType = keyType
		return nil
	}
}
