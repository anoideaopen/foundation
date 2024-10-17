// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: foundation_config.proto

package proto

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on Config with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Config) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Config with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in ConfigMultiError, or nil if none found.
func (m *Config) ValidateAll() error {
	return m.validate(true)
}

func (m *Config) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetContract() == nil {
		err := ConfigValidationError{
			field:  "Contract",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetContract()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ConfigValidationError{
					field:  "Contract",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ConfigValidationError{
					field:  "Contract",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetContract()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ConfigValidationError{
				field:  "Contract",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetToken()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ConfigValidationError{
					field:  "Token",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ConfigValidationError{
					field:  "Token",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetToken()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ConfigValidationError{
				field:  "Token",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetExtConfig()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ConfigValidationError{
					field:  "ExtConfig",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ConfigValidationError{
					field:  "ExtConfig",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetExtConfig()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ConfigValidationError{
				field:  "ExtConfig",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return ConfigMultiError(errors)
	}

	return nil
}

// ConfigMultiError is an error wrapping multiple validation errors returned by
// Config.ValidateAll() if the designated constraints aren't met.
type ConfigMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ConfigMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ConfigMultiError) AllErrors() []error { return m }

// ConfigValidationError is the validation error returned by Config.Validate if
// the designated constraints aren't met.
type ConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ConfigValidationError) ErrorName() string { return "ConfigValidationError" }

// Error satisfies the builtin error interface
func (e ConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ConfigValidationError{}

// Validate checks the field values on ContractConfig with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *ContractConfig) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ContractConfig with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ContractConfigMultiError,
// or nil if none found.
func (m *ContractConfig) ValidateAll() error {
	return m.validate(true)
}

func (m *ContractConfig) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if !_ContractConfig_Symbol_Pattern.MatchString(m.GetSymbol()) {
		err := ContractConfigValidationError{
			field:  "Symbol",
			reason: "value does not match regex pattern \"^[A-Z]+[A-Z0-9]+(-[A-Z0-9.]+)?$\"",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetOptions()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ContractConfigValidationError{
					field:  "Options",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ContractConfigValidationError{
					field:  "Options",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetOptions()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ContractConfigValidationError{
				field:  "Options",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if !_ContractConfig_RobotSKI_Pattern.MatchString(m.GetRobotSKI()) {
		err := ContractConfigValidationError{
			field:  "RobotSKI",
			reason: "value does not match regex pattern \"^[0-9a-f]+$\"",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetAdmin()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ContractConfigValidationError{
					field:  "Admin",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ContractConfigValidationError{
					field:  "Admin",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAdmin()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ContractConfigValidationError{
				field:  "Admin",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetTracingCollectorEndpoint()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ContractConfigValidationError{
					field:  "TracingCollectorEndpoint",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ContractConfigValidationError{
					field:  "TracingCollectorEndpoint",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetTracingCollectorEndpoint()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ContractConfigValidationError{
				field:  "TracingCollectorEndpoint",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for MaxChannelTransferItems

	if len(errors) > 0 {
		return ContractConfigMultiError(errors)
	}

	return nil
}

// ContractConfigMultiError is an error wrapping multiple validation errors
// returned by ContractConfig.ValidateAll() if the designated constraints
// aren't met.
type ContractConfigMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ContractConfigMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ContractConfigMultiError) AllErrors() []error { return m }

// ContractConfigValidationError is the validation error returned by
// ContractConfig.Validate if the designated constraints aren't met.
type ContractConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ContractConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ContractConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ContractConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ContractConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ContractConfigValidationError) ErrorName() string { return "ContractConfigValidationError" }

// Error satisfies the builtin error interface
func (e ContractConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sContractConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ContractConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ContractConfigValidationError{}

var _ContractConfig_Symbol_Pattern = regexp.MustCompile("^[A-Z]+[A-Z0-9]+(-[A-Z0-9.]+)?$")

var _ContractConfig_RobotSKI_Pattern = regexp.MustCompile("^[0-9a-f]+$")

// Validate checks the field values on CollectorEndpoint with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *CollectorEndpoint) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on CollectorEndpoint with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// CollectorEndpointMultiError, or nil if none found.
func (m *CollectorEndpoint) ValidateAll() error {
	return m.validate(true)
}

func (m *CollectorEndpoint) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Endpoint

	// no validation rules for AuthorizationHeaderKey

	// no validation rules for AuthorizationHeaderValue

	// no validation rules for TlsCa

	if len(errors) > 0 {
		return CollectorEndpointMultiError(errors)
	}

	return nil
}

// CollectorEndpointMultiError is an error wrapping multiple validation errors
// returned by CollectorEndpoint.ValidateAll() if the designated constraints
// aren't met.
type CollectorEndpointMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m CollectorEndpointMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m CollectorEndpointMultiError) AllErrors() []error { return m }

// CollectorEndpointValidationError is the validation error returned by
// CollectorEndpoint.Validate if the designated constraints aren't met.
type CollectorEndpointValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CollectorEndpointValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CollectorEndpointValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CollectorEndpointValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CollectorEndpointValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CollectorEndpointValidationError) ErrorName() string {
	return "CollectorEndpointValidationError"
}

// Error satisfies the builtin error interface
func (e CollectorEndpointValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCollectorEndpoint.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CollectorEndpointValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CollectorEndpointValidationError{}

// Validate checks the field values on ChaincodeOptions with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *ChaincodeOptions) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ChaincodeOptions with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ChaincodeOptionsMultiError, or nil if none found.
func (m *ChaincodeOptions) ValidateAll() error {
	return m.validate(true)
}

func (m *ChaincodeOptions) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for DisableSwaps

	// no validation rules for DisableMultiSwaps

	if len(errors) > 0 {
		return ChaincodeOptionsMultiError(errors)
	}

	return nil
}

// ChaincodeOptionsMultiError is an error wrapping multiple validation errors
// returned by ChaincodeOptions.ValidateAll() if the designated constraints
// aren't met.
type ChaincodeOptionsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ChaincodeOptionsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ChaincodeOptionsMultiError) AllErrors() []error { return m }

// ChaincodeOptionsValidationError is the validation error returned by
// ChaincodeOptions.Validate if the designated constraints aren't met.
type ChaincodeOptionsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ChaincodeOptionsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ChaincodeOptionsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ChaincodeOptionsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ChaincodeOptionsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ChaincodeOptionsValidationError) ErrorName() string { return "ChaincodeOptionsValidationError" }

// Error satisfies the builtin error interface
func (e ChaincodeOptionsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sChaincodeOptions.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ChaincodeOptionsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ChaincodeOptionsValidationError{}

// Validate checks the field values on Wallet with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Wallet) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Wallet with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in WalletMultiError, or nil if none found.
func (m *Wallet) ValidateAll() error {
	return m.validate(true)
}

func (m *Wallet) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if !_Wallet_Address_Pattern.MatchString(m.GetAddress()) {
		err := WalletValidationError{
			field:  "Address",
			reason: "value does not match regex pattern \"^[1-9A-HJ-NP-Za-km-z]+$\"",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return WalletMultiError(errors)
	}

	return nil
}

// WalletMultiError is an error wrapping multiple validation errors returned by
// Wallet.ValidateAll() if the designated constraints aren't met.
type WalletMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m WalletMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m WalletMultiError) AllErrors() []error { return m }

// WalletValidationError is the validation error returned by Wallet.Validate if
// the designated constraints aren't met.
type WalletValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e WalletValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e WalletValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e WalletValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e WalletValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e WalletValidationError) ErrorName() string { return "WalletValidationError" }

// Error satisfies the builtin error interface
func (e WalletValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sWallet.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = WalletValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = WalletValidationError{}

var _Wallet_Address_Pattern = regexp.MustCompile("^[1-9A-HJ-NP-Za-km-z]+$")

// Validate checks the field values on TokenConfig with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *TokenConfig) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on TokenConfig with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in TokenConfigMultiError, or
// nil if none found.
func (m *TokenConfig) ValidateAll() error {
	return m.validate(true)
}

func (m *TokenConfig) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Name

	// no validation rules for Decimals

	// no validation rules for UnderlyingAsset

	if m.GetIssuer() == nil {
		err := TokenConfigValidationError{
			field:  "Issuer",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetIssuer()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "Issuer",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "Issuer",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetIssuer()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TokenConfigValidationError{
				field:  "Issuer",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetFeeSetter()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "FeeSetter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "FeeSetter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetFeeSetter()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TokenConfigValidationError{
				field:  "FeeSetter",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetFeeAddressSetter()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "FeeAddressSetter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "FeeAddressSetter",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetFeeAddressSetter()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TokenConfigValidationError{
				field:  "FeeAddressSetter",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetRedeemer()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "Redeemer",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, TokenConfigValidationError{
					field:  "Redeemer",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetRedeemer()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TokenConfigValidationError{
				field:  "Redeemer",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return TokenConfigMultiError(errors)
	}

	return nil
}

// TokenConfigMultiError is an error wrapping multiple validation errors
// returned by TokenConfig.ValidateAll() if the designated constraints aren't met.
type TokenConfigMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m TokenConfigMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m TokenConfigMultiError) AllErrors() []error { return m }

// TokenConfigValidationError is the validation error returned by
// TokenConfig.Validate if the designated constraints aren't met.
type TokenConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e TokenConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e TokenConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e TokenConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e TokenConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e TokenConfigValidationError) ErrorName() string { return "TokenConfigValidationError" }

// Error satisfies the builtin error interface
func (e TokenConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sTokenConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = TokenConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = TokenConfigValidationError{}
