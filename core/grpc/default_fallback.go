package grpc

import (
	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/reflectx"
)

// DefaultReflectxFallback creates a new contract.Router instance using
// the reflectx.NewRouter function.
//
// Parameters:
// - base: The contract.Base instance to be used by the router.
// - cfg: The reflectx.RouterConfig instance containing configuration options for the router.
//
// Returns:
// - contract.Router: The newly created contract.Router instance.
func DefaultReflectxFallback(
	base contract.Base,
	cfg reflectx.RouterConfig,
) contract.Router {
	router, err := reflectx.NewRouter(base, cfg)
	if err != nil {
		panic(err)
	}

	return router
}
