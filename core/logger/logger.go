package logger

import (
	"os"

	"github.com/hyperledger/fabric-lib-go/common/flogging"
	"github.com/hyperledger/fabric-lib-go/common/flogging/fabenc"
)

const defaultFormatStr = "%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}"

var lg *flogging.FabricLogger

// Logger returns the logger for chaincode
func Logger() *flogging.FabricLogger {
	if lg == nil {
		formatStr := os.Getenv("CORE_CHAINCODE_LOGGING_FORMAT")
		_, err := fabenc.ParseFormat(formatStr)
		if err != nil {
			formatStr = defaultFormatStr
		}

		levelStr := os.Getenv("CORE_CHAINCODE_LOGGING_LEVEL")
		if levelStr == "" {
			levelStr = "warning"
		}

		flogging.Init(flogging.Config{
			Format:  formatStr,
			LogSpec: levelStr,
			Writer:  os.Stderr,
		})

		lg = flogging.MustGetLogger("chaincode")
	}
	return lg
}
