package revert

import (
	"errors"
	"fmt"

	"github.com/anxp/evmtc/rpc"
)

// FromRPCError decodes revert data carried by an evmtc *rpc.RpcError anywhere in err's chain
// (e.g. an error returned by contract.ContractCallWithCalldata) and attaches it to err.
//
// The result matches both the original error and the decoded *Error with errors.Is / errors.As,
// so errors.Is(err, revert.ErrPoolNotInitialized) works. If err carries no revert data, it's returned as is.
func FromRPCError(err error) error {
	var rpcErr *rpc.RpcError
	if !errors.As(err, &rpcErr) {
		return err
	}

	data, ok := rpcErr.RevertData()
	if !ok {
		return err
	}

	return fmt.Errorf("%w: %w", err, Decode(data))
}
