package pool

import (
	"errors"
	"fmt"

	"github.com/anxp/evmtc/abi"
	"github.com/anxp/evmtc/com"
)

// PathKey is one hop of a multi-hop swap path, as defined in v4-periphery (libraries/PathKey.sol).
//
// Together with the swap's starting (or, for exact output, ending) currency, a PathKey fully
// describes a pool: the other currency of the pool is the one the swap is currently holding.
type PathKey struct {
	// The currency to swap into on this hop (for exact input) / swap from (for exact output).
	IntermediateCurrency com.Address `evm:"0,address"`
	// Original EVM type -> uint24.
	Fee uint32 `evm:"1,uint24"`
	// Original EVM type -> int24.
	TickSpacing int32       `evm:"2,int24"`
	Hooks       com.Address `evm:"3,address"`
	// Arbitrary data passed to the hooks of this hop's pool. Usually empty.
	HookData []byte `evm:"4,bytes"`
}

var ErrEmptyPath = errors.New("swap path must contain at least one pool")

// PoolKey reconstructs the pool of this hop, given the other currency of the pool.
func (k PathKey) PoolKey(otherCurrency com.Address) PoolKey {
	return NewPoolKey(otherCurrency, k.IntermediateCurrency, k.Fee, k.TickSpacing, k.Hooks)
}

// NewExactInputPath builds a path for an exact-input swap from currencyIn through the given pools,
// in swap order. Each PathKey's IntermediateCurrency is the currency received on that hop,
// so the last PathKey's IntermediateCurrency is the swap's output currency.
func NewExactInputPath(currencyIn com.Address, pools []PoolKey) ([]PathKey, error) {
	if len(pools) == 0 {
		return nil, ErrEmptyPath
	}

	path := make([]PathKey, 0, len(pools))
	current := currencyIn

	for i, p := range pools {
		next, err := p.OtherCurrency(current)
		if err != nil {
			return nil, fmt.Errorf("hop #%d: %w", i, err)
		}

		path = append(path, PathKey{
			IntermediateCurrency: next,
			Fee:                  p.Fee,
			TickSpacing:          p.TickSpacing,
			Hooks:                p.Hooks,
		})

		current = next
	}

	return path, nil
}

// NewExactOutputPath builds a path for an exact-output swap that ends in currencyOut, through the
// given pools in swap order (first pool takes the input currency, last pool returns currencyOut).
//
// V4Router walks exact-output paths backwards: each PathKey's IntermediateCurrency is the currency
// paid INTO that hop, so the first PathKey's IntermediateCurrency is the swap's input currency.
func NewExactOutputPath(currencyOut com.Address, pools []PoolKey) ([]PathKey, error) {
	if len(pools) == 0 {
		return nil, ErrEmptyPath
	}

	path := make([]PathKey, len(pools))
	current := currencyOut

	for i := len(pools) - 1; i >= 0; i-- {
		p := pools[i]

		prev, err := p.OtherCurrency(current)
		if err != nil {
			return nil, fmt.Errorf("hop #%d: %w", i, err)
		}

		path[i] = PathKey{
			IntermediateCurrency: prev,
			Fee:                  p.Fee,
			TickSpacing:          p.TickSpacing,
			Hooks:                p.Hooks,
		}

		current = prev
	}

	return path, nil
}

// EncodePath ABI-encodes a PathKey[] value (the "tail" representation of a dynamic array),
// ready to be embedded with abi.Dynamic(...) into swap params.
func EncodePath(path []PathKey) ([]byte, error) {
	elements := make([][]byte, len(path))

	for i, k := range path {
		encoded, err := abi.EncodeTuple(k)
		if err != nil {
			return nil, fmt.Errorf("failed to ABI encode PathKey #%d: %w", i, err)
		}
		elements[i] = encoded
	}

	// PathKey contains "bytes hookData", so it's a dynamic tuple and array elements are referenced by offsets.
	return abi.EncodeArray(elements, true)
}
