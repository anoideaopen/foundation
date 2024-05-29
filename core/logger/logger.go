package logger

import (
	"os"

	"github.com/op/go-logging"
)

const defaultFormatStr = "%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}"

var lg *logging.Logger

// Logger returns the logger for chaincode
func Logger() *logging.Logger {
	if lg == nil {
		lg = logging.MustGetLogger("chaincode")
		formatStr := os.Getenv("CORE_CHAINCODE_LOGGING_FORMAT")
		format, err := logging.NewStringFormatter(formatStr)
		if err != nil {
			format = defaultChaincodeLoggingFormat()
		}
		stderr := logging.NewLogBackend(os.Stderr, "", 0)
		formatted := logging.NewBackendFormatter(stderr, format)
		levelStr := os.Getenv("CORE_CHAINCODE_LOGGING_LEVEL")
		if levelStr == "" {
			levelStr = "warning"
		}
		level, err := logging.LogLevel(levelStr)
		if err != nil {
			panic(err)
		}
		leveled := logging.AddModuleLevel(formatted)
		leveled.SetLevel(level, "")
		lg.SetBackend(leveled)
	}
	return lg
}

func defaultChaincodeLoggingFormat() logging.Formatter {
	format, err := logging.NewStringFormatter(defaultFormatStr)
	if err != nil {
		format = logging.DefaultFormatter
	}
	return format
}
