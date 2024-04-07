//go:build tools
// +build tools

package tools

import (
	_ "github.com/IBM/idemix/tools/idemixgen"
	_ "github.com/hyperledger/fabric/cmd/configtxgen"
	_ "github.com/hyperledger/fabric/cmd/cryptogen"
	_ "github.com/hyperledger/fabric/cmd/discover"
	_ "github.com/hyperledger/fabric/cmd/orderer"
	_ "github.com/hyperledger/fabric/cmd/osnadmin"
	_ "github.com/hyperledger/fabric/cmd/peer"
	_ "github.com/hyperledger/fabric/integration/chaincode/simple/cmd"
)
