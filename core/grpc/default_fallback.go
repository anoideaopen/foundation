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
//
// Returns:
// - contract.Router: The newly created contract.Router instance.
func DefaultReflectxFallback(base contract.Base) contract.Router {
	router, err := reflectx.NewRouter(base)
	if err != nil {
		panic(err)
	}

	return router
}
