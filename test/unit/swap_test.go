package unit

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"golang.org/x/crypto/sha3"
)

func TestAtomicSwap(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, nil, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, nil, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	txID := user1.SignedInvoke("cc", "swapBegin", "CC", "VT", "450", swapHash)
	user1.BalanceShouldBe("cc", 550)
	m.WaitSwapAnswer("vt", txID, time.Second*5)

	user1.Invoke("vt", "swapDone", txID, swapKey)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)

	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 0)
}

func TestAtomicSwapBack(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, nil, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, nil, nil, owner.Address())

	user1 := m.NewWallet()

	user1.AddAllowedBalance("vt", "CC", 1000)
	user1.AddGivenBalance("cc", "VT", 1000)
	user1.AllowedBalanceShouldBe("vt", "CC", 1000)

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	txID := user1.SignedInvoke("vt", "swapBegin", "CC", "CC", "450", swapHash)
	m.WaitSwapAnswer("cc", txID, time.Second*5)

	user1.Invoke("cc", "swapDone", txID, swapKey)
	user1.AllowedBalanceShouldBe("cc", "CC", 0)
	user1.AllowedBalanceShouldBe("vt", "CC", 550)
	user1.BalanceShouldBe("cc", 450)
	user1.BalanceShouldBe("vt", 0)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 550)
}
