package quoter

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/anxp/evmtc/com"

	"github.com/anxp/uni4/pool"
)

func word(b ...byte) []byte {
	w := make([]byte, 32)
	copy(w[32-len(b):], b)
	return w
}

func TestEncodeSingleLayout(t *testing.T) {
	usdc := com.MustAddressFromHex("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	key := pool.NewPoolKey(pool.NativeCurrency, usdc, 500, 10, com.Address{})

	got, err := encodeSingle(sigQuoteExactInputSingle, ExactSingleParams{
		PoolKey:     key,
		ZeroForOne:  true,
		ExactAmount: big.NewInt(1000),
		HookData:    []byte{0x01},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	poolKey, _ := key.Encode()
	hookData := make([]byte, 32)
	hookData[0] = 0x01

	want := bytes.Join([][]byte{
		com.Keccak256([]byte(sigQuoteExactInputSingle))[:4],
		word(0x20),       // offset of the params tuple
		poolKey,          // 5 words, in place
		word(1),          // zeroForOne
		word(0x03, 0xe8), // exactAmount 1000
		word(0x01, 0x00), // offset of hookData within the tuple (8 head words)
		word(1),          // hookData length
		hookData,
	}, nil)

	if !bytes.Equal(got, want) {
		t.Errorf("calldata mismatch\n got: %x\nwant: %x", got, want)
	}
}

func TestEncodePathLayout(t *testing.T) {
	usdc := com.MustAddressFromHex("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	path := []pool.PathKey{{IntermediateCurrency: usdc, Fee: 500, TickSpacing: 10}}

	got, err := encodePath(sigQuoteExactInput, ExactParams{
		ExactCurrency: pool.NativeCurrency,
		Path:          path,
		ExactAmount:   big.NewInt(1000),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	encodedPath, _ := pool.EncodePath(path)

	want := bytes.Join([][]byte{
		com.Keccak256([]byte(sigQuoteExactInput))[:4],
		word(0x20),       // offset of the params tuple
		word(),           // exactCurrency (native)
		word(0x60),       // offset of path within the tuple (3 head words)
		word(0x03, 0xe8), // exactAmount 1000
		encodedPath,
	}, nil)

	if !bytes.Equal(got, want) {
		t.Errorf("calldata mismatch\n got: %x\nwant: %x", got, want)
	}
}

func TestValidateAmount(t *testing.T) {
	tooBig := new(big.Int).Lsh(big.NewInt(1), 128)

	for _, a := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1), tooBig} {
		if err := validateAmount(a); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("amount %v: expected ErrInvalidAmount, got %v", a, err)
		}
	}

	if err := validateAmount(maxUint128); err != nil {
		t.Errorf("max uint128 must be valid: %v", err)
	}
}
