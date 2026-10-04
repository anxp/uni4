package pool

import (
	"bytes"

	"github.com/anxp/evmtc/com"
)

// NativeCurrency represents the chain's native coin (ETH, BNB, POL...) in Uniswap v4.
// In v4 contracts it is Currency.wrap(address(0)).
var NativeCurrency = com.Address{}

// IsNative reports whether the currency is the chain's native coin.
func IsNative(currency com.Address) bool {
	return currency.IsZero()
}

// SortCurrencies returns the two currencies in the order Uniswap v4 expects in PoolKey:
// numerically lower address first. Native currency (address(0)) is therefore always currency0.
func SortCurrencies(currencyX, currencyY com.Address) (currency0, currency1 com.Address) {
	if bytes.Compare(currencyX[:], currencyY[:]) < 0 {
		return currencyX, currencyY
	}

	return currencyY, currencyX
}
