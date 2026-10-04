package main

import (
	"bytes"

	"github.com/anxp/evmtc/abi"
	"github.com/anxp/evmtc/com"
)

import "fmt"

// PoolKey
// evm tag reflects actual order of fields and field type in UniswapV4 smart contract: "<order>,<abi-type>"
type PoolKey struct {
	// The lower currency of the pool, sorted numerically.
	// For native ETH, Currency currency0 = Currency.wrap(address(0));
	Currency0 com.Address `evm:"0,address"`
	// The higher currency of the pool, sorted numerically.
	Currency1 com.Address `evm:"1,address"`
	// The pool LP fee, capped at 1_000_000.
	// If the highest bit is 1, the pool has a dynamic fee and must be exactly equal to 0x800000.
	// Original EVM type -> uint24.
	Fee uint32 `evm:"2,uint24"`
	// Ticks that involve positions must be a multiple of tick spacing.
	// Original EVM type -> int24.
	TickSpacing int32 `evm:"3,int24"`
	// Address of the hooks contract.
	Hooks com.Address `evm:"4,address"`
}

func NewPoolKey(tokenX, tokenY com.Address, fee uint32, tickSpacing int32, hooks com.Address) PoolKey {

	var currency0, currency1 com.Address

	if bytes.Compare(tokenX.Bytes(), tokenY.Bytes()) < 0 {
		currency0 = tokenX
		currency1 = tokenY
	} else {
		currency0 = tokenY
		currency1 = tokenX
	}

	return PoolKey{
		Currency0:   currency0,
		Currency1:   currency1,
		Fee:         fee,
		TickSpacing: tickSpacing,
		Hooks:       hooks,
	}
}

func (p PoolKey) ID() (com.Hash, error) {
	encoded, err := abi.EncodeTuple(p)
	if err != nil {
		return com.Hash{}, fmt.Errorf("failed to ABI encode PoolKey: %w", err)
	}

	id := com.Keccak256Hash(encoded)

	return id, nil
}

func main() {
	fmt.Print("Hello World")
}
