package cctransfer

import (
	"errors"
)

// ErrEmptyIDTransfer CCTransfer errors.
var (
	ErrEmptyIDTransfer           = errors.New("id transfer is empty")
	ErrSaveNilTransfer           = errors.New("save nil transfer")
	ErrNotFound                  = errors.New("transfer not found")
	ErrInvalidIDUser             = errors.New("invalid argument id user")
	ErrInvalidToken              = errors.New("invalid argument token")
	ErrInvalidChannel            = errors.New("invalid argument channel to")
	ErrIDTransferExist           = errors.New("id transfer already exists")
	ErrTransferCommit            = errors.New("transfer already commit")
	ErrTransferNotCommit         = errors.New("transfer not commit")
	ErrUnauthorizedOperation     = errors.New("unauthorized operation")
	ErrInvalidBookmark           = errors.New("invalid bookmark")
	ErrPageSizeLessOrEqZero      = errors.New("page size is less or equal to zero")
	ErrAdminNotSet               = errors.New("admin is not set in base config")
	ErrUnauthorisedNotAdmin      = errors.New("unauthorised, sender is not an admin")
	ErrInvalidTransferItemsCount = errors.New("invalid argument transfer items count")
	ErrInvalidTokenAlreadyExists = errors.New("invalid argument token already exists")
)
