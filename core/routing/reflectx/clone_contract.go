package reflectx

import (
	"sync"

	"github.com/lmlat/go-clone"
)

var mu = sync.Mutex{}

// Clone creates a copy of the given contract.
func Clone(contract any) any {
	mu.Lock()
	contractCopy := clone.Shallow(contract)
	mu.Unlock()

	return contractCopy
}
