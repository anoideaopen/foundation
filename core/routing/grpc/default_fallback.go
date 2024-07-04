package grpc

import (
	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/routing/reflectx"
)

// DefaultReflectxFallback creates a new routing.Router instance using
// the reflectx.NewRouter function.
//
// Parameters:
// - base: The contract instance to be used by the router.
//
// Returns:
// - routing.Router: The newly created routing.Router instance.
func DefaultReflectxFallback(base any) routing.Router {
	router, err := reflectx.NewRouter(base)
	if err != nil {
		panic(err)
	}

	return router
}
