package core

import (
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/op/go-logging"
)

// Deprecated: added only for backward compatibility.
// This method was used by customers in chaincodes implementation.
// After customers change to the new logger from "github.com/anoideaopen/foundation/core/logger", this method will be deleted.
func Logger() *logging.Logger {
	return logger.Logger()
}
