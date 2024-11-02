package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/ledger"
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
