package core

import (
	"testing"
	"time"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/assert"
)

var (
	etlTime = time.Date(2022, 8, 9, 17, 24, 10, 163589, time.Local)
	etlMili = etlTime.UnixMilli()
)

func TestIncorrectNonce(t *testing.T) {
	var err error
	lastNonce := new(pb.Nonce)

	lastNonce.Nonce, err = setNonce(1, lastNonce.Nonce, defaultNonceTTL)
	assert.EqualError(t, err, "incorrect nonce format")
}

func TestCorrectNonce(t *testing.T) {
	var err error
	lastNonce := new(pb.Nonce)

	lastNonce.Nonce, err = setNonce(uint64(etlMili), lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
}

func TestNonceOldOk(t *testing.T) {
	var err error

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = setNonce(1660055050000, lastNonce.Nonce, 0)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050010, lastNonce.Nonce, 0)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050010}, lastNonce.Nonce)
}

func TestNonceOldFail(t *testing.T) {
	var err error

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = setNonce(1660055050010, lastNonce.Nonce, 0)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050010}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050000, lastNonce.Nonce, 0)
	assert.Error(t, err)
	assert.Equal(t, []uint64{1660055050010}, lastNonce.Nonce)
}

func TestNonceOk(t *testing.T) {
	var err error

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = setNonce(1660055050000, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050020, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000, 1660055050020}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050010, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000, 1660055050010, 1660055050020}, lastNonce.Nonce)
}

func TestNonceOkCut(t *testing.T) {
	var err error

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = setNonce(1660055050000, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050020, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000, 1660055050020}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055100010, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050020, 1660055100010}, lastNonce.Nonce)
}

func TestNonceFailTTL(t *testing.T) {
	var err error

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = setNonce(1660055050010, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050010}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055000009, lastNonce.Nonce, defaultNonceTTL)
	assert.EqualError(t, err, "incorrect nonce 1660055000009, less than 1660055050010")
	assert.Equal(t, []uint64{1660055050010}, lastNonce.Nonce)
}

func TestNonceFailRepeat(t *testing.T) {
	var err error

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = setNonce(1660055050000, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050020, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000, 1660055050020}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050010, lastNonce.Nonce, defaultNonceTTL)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1660055050000, 1660055050010, 1660055050020}, lastNonce.Nonce)

	// repeat nonce
	lastNonce.Nonce, err = setNonce(1660055050000, lastNonce.Nonce, defaultNonceTTL)
	assert.EqualError(t, err, "nonce 1660055050000 already exists")
	assert.Equal(t, []uint64{1660055050000, 1660055050010, 1660055050020}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050010, lastNonce.Nonce, defaultNonceTTL)
	assert.EqualError(t, err, "nonce 1660055050010 already exists")
	assert.Equal(t, []uint64{1660055050000, 1660055050010, 1660055050020}, lastNonce.Nonce)

	lastNonce.Nonce, err = setNonce(1660055050020, lastNonce.Nonce, defaultNonceTTL)
	assert.EqualError(t, err, "nonce 1660055050020 already exists")
	assert.Equal(t, []uint64{1660055050000, 1660055050010, 1660055050020}, lastNonce.Nonce)
}
