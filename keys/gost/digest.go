package gost

import (
	"github.com/ddulesov/gogost/gost34112012256"
)

// Sum256 calculates and returns a hash sum of 256 bits for data,
// using the algorithm of GOST R 34.11-2012.
func Sum256(data []byte) (digest [32]byte) {
	// Create a new hash instance for GOST R 34.11-2012 with a hash length of 256 bits.
	hasher := gost34112012256.New()

	// Write the data to the hash. If the writing is error-free, then calculate the hash.
	if _, err := hasher.Write(data); err == nil {
		// We get the hash and copy it into the digest array.
		copy(digest[:], hasher.Sum(nil))
	}

	// Return the calculated hash.
	return
}
