package big

import "errors"

// Validate checks if the Int value is negative and returns an error if it is.
func (z *Int) Validate() error {
	if z.Int.Sign() < 0 {
		return errors.New("negative number")
	}

	return nil
}
