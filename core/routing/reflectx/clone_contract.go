package reflectx

import (
	"github.com/lmlat/go-clone"
)

// Clone creates a copy of the given contract.
func Clone(contract any) any {
	return clone.Shallow(contract)
}
