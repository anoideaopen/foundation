package client

import (
	"strconv"
	"time"

	"github.com/anoideaopen/foundation/core"
)

type Nonce struct {
	val uint64
}

func NewNonceByTime() *Nonce {
	return &Nonce{val: uint64(time.Now().UnixMilli())}
}

func NewNonceByUint64(val uint64) *Nonce {
	if val < core.LeftBorderNonce || val >= core.RightBorderNonce {
		return &Nonce{}
	}

	return &Nonce{val: val}
}

func (n *Nonce) Get() string {
	if n.val < core.LeftBorderNonce || n.val >= core.RightBorderNonce {
		return ""
	}

	return strconv.FormatUint(n.val, 10)
}

func (n *Nonce) Next() {
	n.val++
}

func (n *Nonce) Add(v uint64) {
	n.val += v
}
