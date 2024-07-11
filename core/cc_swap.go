package core

import (
	"github.com/anoideaopen/foundation/core/swap"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

const (
	userSideTimeout = 10800 // 3 hours
)

// swapDoneHandler processes a request to mark a swap as done.
// If the ChainCode is configured to disable swaps, it will immediately return an error.
//
// It loads initial arguments and then proceeds to execute the swap user done logic.
//
// Returns a shim.Success response if the swap done logic executes successfully.
// Otherwise, it returns a shim.Error response.
func (cc *Chaincode) swapDoneHandler(
	stub shim.ChaincodeStubInterface,
	args []string,
) peer.Response {
	if cc.contract.ContractConfig().GetOptions().GetDisableSwaps() {
		return shim.Error("handling swap done failed, " + ErrSwapDisabled.Error())
	}

	return swap.UserDone(cc.contract, stub, args[0], args[1])
}
