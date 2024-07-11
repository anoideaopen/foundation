package core

import (
	"github.com/anoideaopen/foundation/core/multiswap"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// multiSwapDoneHandler processes a request to mark multiple swaps as done.
// If the ChainCode is configured to disable multi swaps, it will immediately return an error.
//
// It loads initial arguments and then proceeds to execute the multi-swap user done logic.
//
// Returns a shim.Success response if the multi-swap done logic executes successfully.
// Otherwise, it returns a shim.Error response.
func (cc *Chaincode) multiSwapDoneHandler(
	stub shim.ChaincodeStubInterface,
	symbol string,
	args []string,
) peer.Response {
	if cc.contract.ContractConfig().GetOptions().GetDisableMultiSwaps() {
		return shim.Error("handling multi-swap done failed, " + ErrMultiSwapDisabled.Error())
	}

	return multiswap.UserDone(cc.contract, stub, symbol, args[0], args[1])
}
