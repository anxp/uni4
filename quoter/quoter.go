// Package quoter wraps the Uniswap v4 V4Quoter contract (v4-periphery/src/lens/V4Quoter.sol).
//
// V4Quoter answers "how much will I get / how much do I need to pay" for a swap, without swapping.
//
// How it works: the quote function actually executes the swap against the real pool state, reads
// the resulting amount, and then reverts, so all state changes are rolled back. Because of that
// the quote functions are not declared as read-only ("view") in the contract ABI, even though
// they never change anything.
//
// They must be called via eth_call (a simulated call that is never sent to the network), not as a
// transaction: it's free, and a real transaction would only waste gas. This package does exactly that.
package quoter

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anxp/evmtc/abi"
	"github.com/anxp/evmtc/com"
	"github.com/anxp/evmtc/contract"
	"github.com/anxp/evmtc/interfaces"

	"github.com/anxp/uni4/pool"
	"github.com/anxp/uni4/revert"
)

// Method signatures as they are hashed into selectors. PoolKey and PathKey are expanded into tuples:
//
//	QuoteExactSingleParams = (PoolKey poolKey, bool zeroForOne, uint128 exactAmount, bytes hookData)
//	QuoteExactParams       = (Currency exactCurrency, PathKey[] path, uint128 exactAmount)
//
// The four V4Quoter methods cover two questions ("exact input" / "exact output") for two route kinds
// (one pool / multi-hop path):
const (
	// quoteExactInputSingle: "I give exactly X of token A, how much of token B will I get?" - swap through ONE pool.
	// Example: sell exactly 1 ETH in the ETH/USDC pool -> returns the amount of USDC received.
	sigQuoteExactInputSingle = "quoteExactInputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))"

	// quoteExactOutputSingle: "I want to get exactly Y of token B, how much of token A must I pay?" - swap through ONE pool.
	// Example: buy exactly 1000 USDC in the ETH/USDC pool -> returns the amount of ETH required.
	sigQuoteExactOutputSingle = "quoteExactOutputSingle(((address,address,uint24,int24,address),bool,uint128,bytes))"

	// quoteExactInput: same question as quoteExactInputSingle, but the swap goes through SEVERAL pools (path).
	// Example: sell exactly 1 ETH via ETH -> USDC -> USDT -> returns the amount of USDT received.
	sigQuoteExactInput = "quoteExactInput((address,(address,uint24,int24,address,bytes)[],uint128))"

	// quoteExactOutput: same question as quoteExactOutputSingle, but the swap goes through SEVERAL pools (path).
	// Example: buy exactly 1000 USDT via ETH -> USDC -> USDT -> returns the amount of ETH required.
	sigQuoteExactOutput = "quoteExactOutput((address,(address,uint24,int24,address,bytes)[],uint128))"
)

var maxUint128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

var ErrInvalidAmount = errors.New("exact amount must be positive and fit into uint128")

// ExactSingleParams mirrors IV4Quoter.QuoteExactSingleParams (single-pool swap).
type ExactSingleParams struct {
	PoolKey pool.PoolKey
	// ZeroForOne is the swap direction: true for currency0 -> currency1.
	// Use PoolKey.ZeroForOne(currencyIn) to get it from the input currency.
	ZeroForOne bool
	// ExactAmount is the amount in (for exact input) or amount out (for exact output), in token base units.
	ExactAmount *big.Int
	HookData    []byte
}

// ExactParams mirrors IV4Quoter.QuoteExactParams (multi-hop swap).
type ExactParams struct {
	// ExactCurrency is the input currency (for exact input) or the output currency (for exact output).
	ExactCurrency com.Address
	// Path is built with pool.NewExactInputPath / pool.NewExactOutputPath accordingly.
	Path []pool.PathKey
	// ExactAmount is the amount in (for exact input) or amount out (for exact output), in token base units.
	ExactAmount *big.Int
}

// Quote is the result of a quote call.
type Quote struct {
	// Amount is the amount out (for exact input quotes) or amount in (for exact output quotes).
	Amount *big.Int
	// GasEstimate is the gas used by the simulated swap, as reported by V4Quoter.
	// It's only an approximation of the real swap transaction cost.
	GasEstimate *big.Int
}

// Quoter calls V4Quoter on a specific network.
type Quoter struct {
	client  interfaces.EthCaller
	address com.Address
}

// New creates a Quoter for the V4Quoter contract at the given address (see deployments.Deployment.V4Quoter).
func New(client interfaces.EthCaller, quoterAddress com.Address) *Quoter {
	return &Quoter{client: client, address: quoterAddress}
}

// QuoteExactInputSingle returns how much of the output currency a single-pool swap of ExactAmount gives.
func (q *Quoter) QuoteExactInputSingle(params ExactSingleParams) (Quote, error) {
	return q.quoteSingle(sigQuoteExactInputSingle, params)
}

// QuoteExactOutputSingle returns how much of the input currency is needed to receive ExactAmount from a single pool.
func (q *Quoter) QuoteExactOutputSingle(params ExactSingleParams) (Quote, error) {
	return q.quoteSingle(sigQuoteExactOutputSingle, params)
}

// QuoteExactInput returns how much of the output currency a multi-hop swap of ExactAmount gives.
func (q *Quoter) QuoteExactInput(params ExactParams) (Quote, error) {
	return q.quotePath(sigQuoteExactInput, params)
}

// QuoteExactOutput returns how much of the input currency is needed to receive ExactAmount via a multi-hop path.
func (q *Quoter) QuoteExactOutput(params ExactParams) (Quote, error) {
	return q.quotePath(sigQuoteExactOutput, params)
}

func (q *Quoter) quoteSingle(signature string, params ExactSingleParams) (Quote, error) {
	calldata, err := encodeSingle(signature, params)
	if err != nil {
		return Quote{}, err
	}

	return q.call(calldata)
}

func (q *Quoter) quotePath(signature string, params ExactParams) (Quote, error) {
	calldata, err := encodePath(signature, params)
	if err != nil {
		return Quote{}, err
	}

	return q.call(calldata)
}

// quoteResponse is the (uint256 amount, uint256 gasEstimate) return value of all V4Quoter quote functions.
type quoteResponse struct {
	Amount      *big.Int `evm:"uint256"`
	GasEstimate *big.Int `evm:"uint256"`
}

func (q *Quoter) call(calldata []byte) (Quote, error) {
	resp, err := contract.ContractCallWithCalldata[quoteResponse](q.client, q.address.Hex(), calldata, abi.ABIMixed)
	if err != nil {
		// Attach decoded revert reason, e.g. "UnexpectedRevertBytes(bytes) <- PoolNotInitialized()".
		return Quote{}, fmt.Errorf("quote failed: %w", revert.FromRPCError(err))
	}

	return Quote{Amount: resp.Amount, GasEstimate: resp.GasEstimate}, nil
}

func encodeSingle(signature string, params ExactSingleParams) ([]byte, error) {
	if err := validateAmount(params.ExactAmount); err != nil {
		return nil, err
	}

	if err := params.PoolKey.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pool key: %w", err)
	}

	poolKey, err := params.PoolKey.Encode()
	if err != nil {
		return nil, err
	}

	zeroForOne, err := abi.EncodeData(params.ZeroForOne, abi.ABIBool)
	if err != nil {
		return nil, err
	}

	amount, err := abi.EncodeData(params.ExactAmount, abi.ABIUint128)
	if err != nil {
		return nil, err
	}

	// PoolKey is a static tuple, so it's encoded in place (5 words) within the params tuple head.
	tuple, err := abi.EncodeArguments(
		abi.Static(poolKey),
		abi.Static(zeroForOne[:]),
		abi.Static(amount[:]),
		abi.Dynamic(abi.EncodeDynamicBytes(params.HookData)),
	)
	if err != nil {
		return nil, err
	}

	// The params tuple has a dynamic field (hookData), so it's passed as a dynamic argument (by offset).
	return abi.EncodeContractPayloadFromArgs(signature, abi.Dynamic(tuple))
}

func encodePath(signature string, params ExactParams) ([]byte, error) {
	if err := validateAmount(params.ExactAmount); err != nil {
		return nil, err
	}

	if len(params.Path) == 0 {
		return nil, pool.ErrEmptyPath
	}

	currency, err := abi.EncodeData(params.ExactCurrency, abi.ABIAddress)
	if err != nil {
		return nil, err
	}

	path, err := pool.EncodePath(params.Path)
	if err != nil {
		return nil, err
	}

	amount, err := abi.EncodeData(params.ExactAmount, abi.ABIUint128)
	if err != nil {
		return nil, err
	}

	tuple, err := abi.EncodeArguments(
		abi.Static(currency[:]),
		abi.Dynamic(path),
		abi.Static(amount[:]),
	)
	if err != nil {
		return nil, err
	}

	return abi.EncodeContractPayloadFromArgs(signature, abi.Dynamic(tuple))
}

func validateAmount(amount *big.Int) error {
	if amount == nil || amount.Sign() <= 0 || amount.Cmp(maxUint128) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidAmount, amount)
	}

	return nil
}
