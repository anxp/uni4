package quoter

import (
	"errors"
	"math/big"
	"testing"

	"github.com/anxp/evmtc/client"
	"github.com/anxp/evmtc/com"

	"github.com/anxp/uni4/deployments"
	"github.com/anxp/uni4/pool"
	"github.com/anxp/uni4/revert"
)

// ethereumRPC is a free public Ethereum mainnet RPC used by live tests.
const ethereumRPC = "https://ethereum-rpc.publicnode.com"

// Live tests run against Ethereum mainnet via ethereumRPC (network access required):
//
//	go test ./quoter/ -run Live -v
//
// To skip them (e.g. offline), run tests with -short.
func newLiveQuoter(t *testing.T) *Quoter {
	if testing.Short() {
		t.Skip("skipping live test in -short mode")
	}

	c, err := client.NewClient(ethereumRPC)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return New(c, deployments.Ethereum.V4Quoter)
}

var (
	usdcMainnet = com.MustAddressFromHex("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	usdtMainnet = com.MustAddressFromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	oneETH      = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

func TestLiveQuoteSingleAndPath(t *testing.T) {
	q := newLiveQuoter(t)
	ethUsdc := pool.NewPoolKey(pool.NativeCurrency, usdcMainnet, pool.Fee005, 10, com.Address{})

	single, err := q.QuoteExactInputSingle(ExactSingleParams{PoolKey: ethUsdc, ZeroForOne: true, ExactAmount: oneETH})
	if err != nil {
		t.Fatalf("QuoteExactInputSingle: %v", err)
	}
	t.Logf("1 ETH -> %s USDC base units (gas estimate %s)", single.Amount, single.GasEstimate)

	// Sanity range for ETH price: 100..100000 USDC.
	if single.Amount.Cmp(big.NewInt(100e6)) < 0 || single.Amount.Cmp(big.NewInt(100000e6)) > 0 {
		t.Errorf("implausible quote: %s", single.Amount)
	}

	// A one-hop path must give exactly the same result as the single-pool quote.
	path, _ := pool.NewExactInputPath(pool.NativeCurrency, []pool.PoolKey{ethUsdc})
	viaPath, err := q.QuoteExactInput(ExactParams{ExactCurrency: pool.NativeCurrency, Path: path, ExactAmount: oneETH})
	if err != nil {
		t.Fatalf("QuoteExactInput: %v", err)
	}
	if viaPath.Amount.Cmp(single.Amount) != 0 {
		t.Errorf("one-hop path quote %s != single quote %s", viaPath.Amount, single.Amount)
	}

	// Exact output for the amount we've just been quoted must require ~1 ETH back.
	out, err := q.QuoteExactOutputSingle(ExactSingleParams{PoolKey: ethUsdc, ZeroForOne: true, ExactAmount: single.Amount})
	if err != nil {
		t.Fatalf("QuoteExactOutputSingle: %v", err)
	}
	diff := new(big.Int).Abs(new(big.Int).Sub(out.Amount, oneETH))
	if diff.Cmp(big.NewInt(1e12)) > 0 { // 0.000001 ETH tolerance for rounding
		t.Errorf("exact output quote %s is too far from 1 ETH", out.Amount)
	}
}

func TestLiveQuoteMultiHop(t *testing.T) {
	q := newLiveQuoter(t)
	ethUsdc := pool.NewPoolKey(pool.NativeCurrency, usdcMainnet, pool.Fee005, 10, com.Address{})
	usdcUsdt := pool.NewPoolKey(usdcMainnet, usdtMainnet, pool.Fee0001, 1, com.Address{})
	pools := []pool.PoolKey{ethUsdc, usdcUsdt}

	inPath, _ := pool.NewExactInputPath(pool.NativeCurrency, pools)
	in, err := q.QuoteExactInput(ExactParams{ExactCurrency: pool.NativeCurrency, Path: inPath, ExactAmount: oneETH})
	if err != nil {
		t.Fatalf("QuoteExactInput: %v", err)
	}
	t.Logf("1 ETH -> USDC -> %s USDT base units", in.Amount)

	outPath, _ := pool.NewExactOutputPath(usdtMainnet, pools)
	out, err := q.QuoteExactOutput(ExactParams{ExactCurrency: usdtMainnet, Path: outPath, ExactAmount: in.Amount})
	if err != nil {
		t.Fatalf("QuoteExactOutput: %v", err)
	}
	t.Logf("%s USDT <- %s wei", in.Amount, out.Amount)

	diff := new(big.Int).Abs(new(big.Int).Sub(out.Amount, oneETH))
	if diff.Cmp(big.NewInt(1e12)) > 0 {
		t.Errorf("exact output multi-hop quote %s is too far from 1 ETH", out.Amount)
	}
}

func TestLiveQuoteNonExistentPool(t *testing.T) {
	q := newLiveQuoter(t)
	// There is no ETH/USDC pool with fee 0.0007% and tick spacing 2.
	key := pool.NewPoolKey(pool.NativeCurrency, usdcMainnet, pool.Fee00007, 2, com.Address{})

	_, err := q.QuoteExactInputSingle(ExactSingleParams{PoolKey: key, ZeroForOne: true, ExactAmount: oneETH})
	if !errors.Is(err, revert.ErrPoolNotInitialized) {
		t.Fatalf("expected ErrPoolNotInitialized, got %v", err)
	}
	t.Logf("error: %v", err)
}
