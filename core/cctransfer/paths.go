package cctransfer

import (
	"path"
)

// Data store pathes.
const (
	pathCrossChannelTransfer = "/transfer/"                       // transfer - cross channel transfer
	pathTransferFrom         = pathCrossChannelTransfer + "from/" // f - From + ID
	pathTransferTo           = pathCrossChannelTransfer + "to/"   // t - To + ID
)

// Base returns the last element of path.
// Trailing slashes are removed before extracting the last element.
func Base(fullPath string) string {
	return path.Base(fullPath)
}

// CCFromTransfers returns path to store key.
func CCFromTransfers() string {
	return pathTransferFrom
}

// CCFromTransfer returns path to store key.
func CCFromTransfer(id string) string {
	return path.Join(CCFromTransfers(), id)
}

// CCToTransfers returns path to store key.
func CCToTransfers() string {
	return pathTransferTo
}

// CCToTransfer returns path to store key.
func CCToTransfer(id string) string {
	return path.Join(CCToTransfers(), id)
}
