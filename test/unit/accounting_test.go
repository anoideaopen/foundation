package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/ledger"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/stretchr/testify/require"
)

func TestAccountingInterfaceMatchWithTxCacheStubImplementation(t *testing.T) {
	var stub interface{} = &cachestub.TxCacheStub{}

	t.Run("Check if TxCacheStub implements Accounting interface", func(t *testing.T) {
		if _, ok := stub.(ledger.Accounting); !ok {
			require.True(t, ok)
		}
	})
}

func TestAccountingInterfaceMatchWithMockStub(t *testing.T) {
	var s interface{} = &stub.Stub{}

	t.Run("Check if mock stub implements Accounting interface", func(t *testing.T) {
		if _, ok := s.(ledger.Accounting); !ok {
			require.True(t, ok)
		}
	})
}
