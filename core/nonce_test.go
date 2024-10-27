package core

import (
	"errors"
	"testing"
	"time"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
)

var (
	etlTime = time.Date(2022, 8, 9, 17, 24, 10, 163589, time.Local)
	etlMili = etlTime.UnixMilli()
)

func TestIncorrectNonce(t *testing.T) {
	n := &Nonce{
		guardF: func(nnc uint64) error {
			if nnc < LeftBorderNonce || nnc >= RightBorderNonce {
				return errors.New("incorrect nonce format")
			}
			return nil
		},
	}

	require.EqualError(t, n.guardF(1), "incorrect nonce format")
}

func TestCorrectNonce(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(uint64(etlMili)*multiKoeffForGeneralNonce, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
}

func TestNonceOldOk(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(1660055050000000000, lastNonce.Nonce, 0)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050010000000, lastNonce.Nonce, 0)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050010000000}, lastNonce.Nonce)
}

func TestNonceOldFail(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(1660055050010000000, lastNonce.Nonce, 0)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050010000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050000000000, lastNonce.Nonce, 0)
	require.Error(t, err)
	require.Equal(t, []uint64{1660055050010000000}, lastNonce.Nonce)
}

func TestNonceOk(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(1660055050000000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050020000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000, 1660055050020000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050010000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000, 1660055050010000000, 1660055050020000000}, lastNonce.Nonce)
}

func TestNonceOkCut(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(1660055050000000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050020000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000, 1660055050020000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055100010000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050020000000, 1660055100010000000}, lastNonce.Nonce)
}

func TestNonceFailTTL(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(1660055050010000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050010000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055000009000000, lastNonce.Nonce, defaultNonceTTL)
	require.EqualError(t, err, "incorrect nonce 1660055000009000000, less than 1660055050010000000")
	require.Equal(t, []uint64{1660055050010000000}, lastNonce.Nonce)
}

func TestNonceFailRepeat(t *testing.T) {
	var err error
	n := new(Nonce)

	lastNonce := new(pb.Nonce)
	lastNonce.Nonce, err = n.set(1660055050000000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050020000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000, 1660055050020000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050010000000, lastNonce.Nonce, defaultNonceTTL)
	require.NoError(t, err)
	require.Equal(t, []uint64{1660055050000000000, 1660055050010000000, 1660055050020000000}, lastNonce.Nonce)

	// repeat nonce
	lastNonce.Nonce, err = n.set(1660055050000000000, lastNonce.Nonce, defaultNonceTTL)
	require.EqualError(t, err, "nonce 1660055050000000000 already exists")
	require.Equal(t, []uint64{1660055050000000000, 1660055050010000000, 1660055050020000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050010000000, lastNonce.Nonce, defaultNonceTTL)
	require.EqualError(t, err, "nonce 1660055050010000000 already exists")
	require.Equal(t, []uint64{1660055050000000000, 1660055050010000000, 1660055050020000000}, lastNonce.Nonce)

	lastNonce.Nonce, err = n.set(1660055050020000000, lastNonce.Nonce, defaultNonceTTL)
	require.EqualError(t, err, "nonce 1660055050020000000 already exists")
	require.Equal(t, []uint64{1660055050000000000, 1660055050010000000, 1660055050020000000}, lastNonce.Nonce)
}
