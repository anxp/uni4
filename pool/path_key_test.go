package pool

import (
	"bytes"
	"errors"
	"testing"

	"github.com/anxp/evmtc/com"
)

func word(b ...byte) []byte {
	w := make([]byte, 32)
	copy(w[32-len(b):], b)
	return w
}

func TestNewExactInputPath(t *testing.T) {
	ethUsdc := NewPoolKey(NativeCurrency, usdcMainnet, 500, 10, com.Address{})
	usdcWbtc := NewPoolKey(usdcMainnet, wbtcMainnet, 3000, 60, com.Address{})

	path, err := NewExactInputPath(NativeCurrency, []PoolKey{ethUsdc, usdcWbtc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(path) != 2 || path[0].IntermediateCurrency != usdcMainnet || path[1].IntermediateCurrency != wbtcMainnet {
		t.Fatalf("unexpected path: %+v", path)
	}
	if path[0].Fee != 500 || path[0].TickSpacing != 10 || path[1].Fee != 3000 || path[1].TickSpacing != 60 {
		t.Errorf("fee/tick spacing not copied correctly: %+v", path)
	}

	// Reconstructing pools from the path must give the original pool keys.
	if got := path[0].PoolKey(NativeCurrency); got != ethUsdc {
		t.Errorf("hop #0 pool key mismatch: %+v", got)
	}
	if got := path[1].PoolKey(path[0].IntermediateCurrency); got != usdcWbtc {
		t.Errorf("hop #1 pool key mismatch: %+v", got)
	}

	if _, err := NewExactInputPath(usdtMainnet, []PoolKey{ethUsdc}); !errors.Is(err, ErrCurrencyNotInPool) {
		t.Errorf("expected ErrCurrencyNotInPool, got %v", err)
	}
	if _, err := NewExactInputPath(NativeCurrency, nil); !errors.Is(err, ErrEmptyPath) {
		t.Errorf("expected ErrEmptyPath, got %v", err)
	}
}

func TestNewExactOutputPath(t *testing.T) {
	ethUsdc := NewPoolKey(NativeCurrency, usdcMainnet, 500, 10, com.Address{})
	usdcWbtc := NewPoolKey(usdcMainnet, wbtcMainnet, 3000, 60, com.Address{})

	// ETH -> USDC -> WBTC, exact WBTC out.
	path, err := NewExactOutputPath(wbtcMainnet, []PoolKey{ethUsdc, usdcWbtc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// V4Router walks backwards from currencyOut: path[i].IntermediateCurrency is the input of hop i.
	if len(path) != 2 || path[0].IntermediateCurrency != NativeCurrency || path[1].IntermediateCurrency != usdcMainnet {
		t.Fatalf("unexpected path: %+v", path)
	}
	if got := path[1].PoolKey(wbtcMainnet); got != usdcWbtc {
		t.Errorf("hop #1 pool key mismatch: %+v", got)
	}
	if got := path[0].PoolKey(path[1].IntermediateCurrency); got != ethUsdc {
		t.Errorf("hop #0 pool key mismatch: %+v", got)
	}
}

func TestEncodePath(t *testing.T) {
	path := []PathKey{{
		IntermediateCurrency: usdcMainnet,
		Fee:                  500,
		TickSpacing:          -1, // not a valid spacing, used only to check int24 sign extension
		Hooks:                com.Address{},
		HookData:             []byte{0xab, 0xcd},
	}}

	got, err := EncodePath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	minusOne := bytes.Repeat([]byte{0xff}, 32)
	hookData := make([]byte, 32)
	hookData[0], hookData[1] = 0xab, 0xcd

	want := bytes.Join([][]byte{
		word(1),    // array length
		word(0x20), // offset of element #0 (relative to the start of the elements area)
		word(usdcMainnet[:]...),
		word(0x01, 0xf4), // fee 500
		minusOne,         // tickSpacing -1
		word(),           // hooks
		word(0xa0),       // offset of hookData within the tuple (5 head words)
		word(2),          // hookData length
		hookData,         // hookData, right-padded
	}, nil)

	if !bytes.Equal(got, want) {
		t.Errorf("encoding mismatch\n got: %x\nwant: %x", got, want)
	}
}
