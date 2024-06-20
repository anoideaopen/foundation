package token

import (
	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/test/unit/service"
)

type Balance struct {
	core.BaseContract
	service.UnimplementedBalanceServiceServer
}
