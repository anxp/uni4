package pool

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/anxp/evmtc/abi"
	"github.com/anxp/evmtc/com"
)

// PoolKey uniquely identifies a Uniswap v4 pool.
//
// The evm tag reflects the actual order of fields and their types in the
// Uniswap v4 smart contract: "<order>,<abi-type>".
type PoolKey struct {
	// The lower currency of the pool, sorted numerically.
	// For native ETH, Currency currency0 = Currency.wrap(address(0)).
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

var (
	ErrSameCurrencies      = errors.New("pool currencies must be different")
	ErrCurrenciesNotSorted = errors.New("pool currencies are not sorted: currency0 must be lower than currency1")
	ErrInvalidFee          = errors.New("invalid pool fee")
	ErrInvalidTickSpacing  = errors.New("invalid pool tick spacing")
	ErrCurrencyNotInPool   = errors.New("currency is not part of the pool")
)

// NewPoolKey builds a PoolKey, sorting currencies as required by Uniswap v4.
// Use NativeCurrency (address(0)) for the chain's native coin.
func NewPoolKey(currencyX, currencyY com.Address, fee uint32, tickSpacing int32, hooks com.Address) PoolKey {
	currency0, currency1 := SortCurrencies(currencyX, currencyY)

	return PoolKey{
		Currency0:   currency0,
		Currency1:   currency1,
		Fee:         fee,
		TickSpacing: tickSpacing,
		Hooks:       hooks,
	}
}

// Validate checks the same invariants PoolManager checks on pool initialization.
// It does not check whether the pool actually exists on-chain.
func (p PoolKey) Validate() error {
	cmp := bytes.Compare(p.Currency0[:], p.Currency1[:])
	if cmp == 0 {
		return ErrSameCurrencies
	}
	if cmp > 0 {
		return ErrCurrenciesNotSorted
	}

	if p.Fee > MaxLPFee && !IsDynamicFee(p.Fee) {
		return fmt.Errorf("%w: %d (max %d or dynamic flag 0x%x)", ErrInvalidFee, p.Fee, MaxLPFee, DynamicFeeFlag)
	}

	if p.TickSpacing < MinTickSpacing || p.TickSpacing > MaxTickSpacing {
		return fmt.Errorf("%w: %d (allowed %d..%d)", ErrInvalidTickSpacing, p.TickSpacing, MinTickSpacing, MaxTickSpacing)
	}

	return nil
}

// ID returns PoolId = keccak256(abi.encode(poolKey)), the identifier used by PoolManager and StateView.
func (p PoolKey) ID() (com.Hash, error) {
	encoded, err := p.Encode()
	if err != nil {
		return com.Hash{}, err
	}

	return com.Keccak256Hash(encoded), nil
}

// Encode returns abi.encode(poolKey): 5 static 32-byte words.
// Since PoolKey is a static tuple, this is also its in-place encoding when nested in other structs.
func (p PoolKey) Encode() ([]byte, error) {
	encoded, err := abi.EncodeTuple(p)
	if err != nil {
		return nil, fmt.Errorf("failed to ABI encode PoolKey: %w", err)
	}

	return encoded, nil
}

// HasCurrency reports whether the currency is one of the pool's currencies.
func (p PoolKey) HasCurrency(currency com.Address) bool {
	return currency == p.Currency0 || currency == p.Currency1
}

// ZeroForOne returns the swap direction for the given input currency:
// true if currencyIn is Currency0 (swap currency0 -> currency1), false if it is Currency1.
func (p PoolKey) ZeroForOne(currencyIn com.Address) (bool, error) {
	switch currencyIn {
	case p.Currency0:
		return true, nil
	case p.Currency1:
		return false, nil
	default:
		return false, fmt.Errorf("%w: %s", ErrCurrencyNotInPool, currencyIn.Hex())
	}
}

// OtherCurrency returns the pool's currency opposite to the given one.
func (p PoolKey) OtherCurrency(currency com.Address) (com.Address, error) {
	zeroForOne, err := p.ZeroForOne(currency)
	if err != nil {
		return com.Address{}, err
	}

	if zeroForOne {
		return p.Currency1, nil
	}

	return p.Currency0, nil
}
