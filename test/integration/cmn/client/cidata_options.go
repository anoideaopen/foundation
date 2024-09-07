package client

import (
	"github.com/anoideaopen/robot/helpers/ntesting"
)

// CiDataOption specifies some ciDataOptions parameter
type CiDataOption func(ciData ntesting.CiTestData) error

// CiData values

// WithCiDataRedisAddress specifies CiData redis address
func WithCiDataRedisAddress(address string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.RedisAddr = address
		return nil
	}
}

// WithCiDataRedisPassword specifies CiData redis password
func WithCiDataRedisPassword(password string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.RedisPass = password
		return nil
	}
}

// WithCiDataHlfProfilePath specifies CiData HLF profile path
func WithCiDataHlfProfilePath(profilePath string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfProfilePath = profilePath
		return nil
	}
}

// WithCiDataFiatChannel specifies CiData Fiat channel name
func WithCiDataFiatChannel(channelName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfFiatChannel = channelName
		return nil
	}
}

// WithCiDataCcChannel specifies CiData CC channel name
func WithCiDataCcChannel(channelName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfCcChannel = channelName
		return nil
	}
}

// WithCiDataIndustrialChannel specifies CiData CC channel name
func WithCiDataIndustrialChannel(channelName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfIndustrialChannel = channelName
		return nil
	}
}

// WithCiDataNoCcChannel specifies CiDats channel without chaincode name
func WithCiDataNoCcChannel(channelName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfNoCcChannel = channelName
		return nil
	}
}

// WithCiDataHlfUserName specifies CiData HLF userName
func WithCiDataHlfUserName(userName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfUserName = userName
		return nil
	}
}

// WithCiDataHlfCertPath specifies CiData HLF cert path
func WithCiDataHlfCertPath(path string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfCert = path
		return nil
	}
}

// WithCiDataFiatOwner specifies CiData Fiat owner private key in base58
func WithCiDataFiatOwner(publicKeyBase58 string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfFiatOwnerKey = publicKeyBase58
		return nil
	}
}

// WithCiDataCcOwner specifies CiData CC owner private key in base58
func WithCiDataCcOwner(publicKeyBase58 string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfCcOwnerKey = publicKeyBase58
		return nil
	}
}

// WithCiDataIndustrialOwner specifies CiData Industrial owner private key in base58
func WithCiDataIndustrialOwner(publicKeyBase58 string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfIndustrialOwnerKey = publicKeyBase58
		return nil
	}
}

// WithCiDataSk specifies CiData Sk path
func WithCiDataSk(path string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfSk = path
		return nil
	}
}

// WithCiDataIndustrialGroup1 specifies CiData industrial token group 1 name
func WithCiDataIndustrialGroup1(groupName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfIndustrialGroup1 = groupName
		return nil
	}
}

// WithCiDataIndustrialGroup2 specifies CiData industrial token group 2 name
func WithCiDataIndustrialGroup2(groupName string) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfIndustrialGroup2 = groupName
		return nil
	}
}

// WithCiDataDoSwapTests specifies CiData doSwap value
func WithCiDataDoSwapTests(doSwap bool) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfDoSwapTests = doSwap
		return nil
	}
}

// WithCiDataDoMultiSwapTests specifies CiData doMultiSwap value
func WithCiDataDoMultiSwapTests(doMultiSwap bool) CiDataOption {
	return func(ciData ntesting.CiTestData) error {
		ciData.HlfDoMultiSwapTests = doMultiSwap
		return nil
	}
}
