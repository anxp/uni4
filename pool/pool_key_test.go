package pool

import (
	"errors"
	"testing"

	"github.com/anxp/evmtc/com"
)

var (
	usdcMainnet = com.MustAddressFromHex("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	usdtMainnet = com.MustAddressFromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	wbtcMainnet = com.MustAddressFromHex("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599")
)

// Expected IDs were cross-checked on Ethereum mainnet: StateView.getSlot0(id) returns
// a non-zero sqrtPriceX96 and the matching lpFee for each of these pools.
func TestPoolKeyID(t *testing.T) {
	tests := []struct {
		name string
		key  PoolKey
		want string
	}{
		{"ETH/USDC 0.05%", NewPoolKey(NativeCurrency, usdcMainnet, 500, 10, com.Address{}), "0x21c67e77068de97969ba93d4aab21826d33ca12bb9f565d8496e8fda8a82ca27"},
		{"ETH/USDC 0.30%", NewPoolKey(usdcMainnet, NativeCurrency, 3000, 60, com.Address{}), "0xdce6394339af00981949f5f3baf27e3610c76326a700af57e4b3e3ae4977f78d"},
		{"ETH/USDT 0.05%", NewPoolKey(NativeCurrency, usdtMainnet, 500, 10, com.Address{}), "0x72331fcb696b0151904c03584b66dc8365bc63f8a144d89a773384e3a579ca73"},
		{"USDC/USDT 0.0007%", NewPoolKey(usdcMainnet, usdtMainnet, Fee00007, 1, com.Address{}), "0x0fb0e40cec3bb23e13abc585958a93c796fbea56955e19a23727a716a0423239"},
		{"USDC/USDT 0.0008%", NewPoolKey(usdcMainnet, usdtMainnet, Fee00008, 1, com.Address{}), "0x395f91b34aa34a477ce3bc6505639a821b286a62b1a164fc1887fa3a5ef713a5"},
		{"USDC/USDT 0.001%", NewPoolKey(usdtMainnet, usdcMainnet, Fee0001, 1, com.Address{}), "0x8aa4e11cbdf30eedc92100f4c8a31ff748e201d44712cc8c90d189edaa8e4e47"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := tt.key.ID()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.Hex() != tt.want {
				t.Errorf("got %s, want %s", id.Hex(), tt.want)
			}
		})
	}
}

func TestNewPoolKeySortsCurrencies(t *testing.T) {
	a := NewPoolKey(usdtMainnet, usdcMainnet, 500, 10, com.Address{})
	b := NewPoolKey(usdcMainnet, usdtMainnet, 500, 10, com.Address{})

	if a != b {
		t.Fatalf("pool keys differ depending on argument order: %+v vs %+v", a, b)
	}
	if a.Currency0 != usdcMainnet || a.Currency1 != usdtMainnet {
		t.Errorf("unexpected order: currency0=%s currency1=%s", a.Currency0.Hex(), a.Currency1.Hex())
	}

	n := NewPoolKey(usdcMainnet, NativeCurrency, 500, 10, com.Address{})
	if !IsNative(n.Currency0) {
		t.Error("native currency must always be currency0")
	}
}

func TestPoolKeyValidate(t *testing.T) {
	valid := NewPoolKey(NativeCurrency, usdcMainnet, 500, 10, com.Address{})
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dynamic := valid
	dynamic.Fee = DynamicFeeFlag
	if err := dynamic.Validate(); err != nil {
		t.Errorf("dynamic fee must be valid: %v", err)
	}

	tests := []struct {
		name   string
		modify func(*PoolKey)
		want   error
	}{
		{"same currencies", func(p *PoolKey) { p.Currency1 = p.Currency0 }, ErrSameCurrencies},
		{"not sorted", func(p *PoolKey) { p.Currency0, p.Currency1 = p.Currency1, p.Currency0 }, ErrCurrenciesNotSorted},
		{"fee too high", func(p *PoolKey) { p.Fee = MaxLPFee + 1 }, ErrInvalidFee},
		{"zero tick spacing", func(p *PoolKey) { p.TickSpacing = 0 }, ErrInvalidTickSpacing},
		{"negative tick spacing", func(p *PoolKey) { p.TickSpacing = -10 }, ErrInvalidTickSpacing},
		{"tick spacing too high", func(p *PoolKey) { p.TickSpacing = MaxTickSpacing + 1 }, ErrInvalidTickSpacing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid
			tt.modify(&p)
			if err := p.Validate(); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestZeroForOne(t *testing.T) {
	p := NewPoolKey(NativeCurrency, usdcMainnet, 500, 10, com.Address{})

	if z, err := p.ZeroForOne(NativeCurrency); err != nil || !z {
		t.Errorf("ETH -> USDC: got %v, %v; want true", z, err)
	}
	if z, err := p.ZeroForOne(usdcMainnet); err != nil || z {
		t.Errorf("USDC -> ETH: got %v, %v; want false", z, err)
	}
	if _, err := p.ZeroForOne(usdtMainnet); !errors.Is(err, ErrCurrencyNotInPool) {
		t.Errorf("expected ErrCurrencyNotInPool, got %v", err)
	}
}
