package ledger

import (
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

type Accounting interface {
	AddAccountingRecord(
		token string,
		from *types.Address,
		to *types.Address,
		amount *big.Int,
		senderBalanceType balance.BalanceType,
		recipientBalanceType balance.BalanceType,
		reason string,
	)
}
