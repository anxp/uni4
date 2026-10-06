package revert

type knownError struct {
	signature string
	// nestedArg is the index of the "bytes" argument holding a nested revert, or -1.
	nestedArg int
}

const (
	sigError = "Error(string)"
	sigPanic = "Panic(uint256)"
)

var known = map[[4]byte]knownError{}

// register adds a known error and returns its sentinel for errors.Is matching.
func register(signature string, nestedArg int) *Error {
	sel := selectorOf(signature)
	known[sel] = knownError{signature: signature, nestedArg: nestedArg}

	return &Error{Selector: sel, Signature: signature}
}

// ErrNoData matches a revert without any data (e.g. plain revert() or a failed call to a non-contract).
var ErrNoData = &Error{}

// Signatures below were checked against error declarations in the official Uniswap repositories
// (main branches as of 2026-10-06).

// Standard Solidity errors.
var (
	ErrString = register(sigError, -1) // require(cond, "message") / revert("message")
	ErrPanic  = register(sigPanic, -1) // assert, arithmetic overflow, division by zero...
)

// PoolManager (v4-core), https://github.com/Uniswap/v4-core
//   - src/interfaces/IPoolManager.sol: PoolNotInitialized, CurrenciesOutOfOrderOrEqual, TickSpacingTooLarge,
//     TickSpacingTooSmall, SwapAmountCannotBeZero, CurrencyNotSettled, ManagerLocked
//   - src/libraries/Pool.sol: PoolAlreadyInitialized, PriceLimitAlreadyExceeded, PriceLimitOutOfBounds,
//     NoLiquidityToReceiveFees, InvalidFeeForExactOut
//   - src/libraries/LPFeeLibrary.sol: LPFeeTooLarge
//   - src/libraries/SqrtPriceMath.sol: NotEnoughLiquidity
//   - src/libraries/SafeCast.sol: SafeCastOverflow
//   - src/libraries/CustomRevert.sol: WrappedError
var (
	ErrPoolNotInitialized          = register("PoolNotInitialized()", -1)
	ErrPoolAlreadyInitialized      = register("PoolAlreadyInitialized()", -1)
	ErrCurrenciesOutOfOrderOrEqual = register("CurrenciesOutOfOrderOrEqual(address,address)", -1)
	ErrTickSpacingTooLarge         = register("TickSpacingTooLarge(int24)", -1)
	ErrTickSpacingTooSmall         = register("TickSpacingTooSmall(int24)", -1)
	ErrLPFeeTooLarge               = register("LPFeeTooLarge(uint24)", -1)
	ErrPriceLimitAlreadyExceeded   = register("PriceLimitAlreadyExceeded(uint160,uint160)", -1)
	ErrPriceLimitOutOfBounds       = register("PriceLimitOutOfBounds(uint160)", -1)
	ErrNoLiquidityToReceiveFees    = register("NoLiquidityToReceiveFees()", -1)
	ErrInvalidFeeForExactOut       = register("InvalidFeeForExactOut()", -1)
	ErrSwapAmountCannotBeZero      = register("SwapAmountCannotBeZero()", -1)
	ErrCurrencyNotSettled          = register("CurrencyNotSettled()", -1)
	ErrManagerLocked               = register("ManagerLocked()", -1)
	ErrSafeCastOverflow            = register("SafeCastOverflow()", -1)
	// ErrNotEnoughLiquidity comes from PoolManager price math (SqrtPriceMath) when a swap step asks for more
	// output than the current liquidity can give. Not to be confused with ErrQuoterNotEnoughLiquidity
	// (different selector, has a poolId argument).
	ErrNotEnoughLiquidity = register("NotEnoughLiquidity()", -1)
	// ErrHookWrappedError wraps a revert thrown by a pool's hook; the hook's own revert is nested.
	ErrHookWrappedError = register("WrappedError(address,bytes4,bytes,bytes)", 2)
)

// V4Quoter (v4-periphery), https://github.com/Uniswap/v4-periphery
//   - src/libraries/QuoterRevert.sol: UnexpectedRevertBytes
//   - src/base/BaseV4Quoter.sol: NotEnoughLiquidity(PoolId), NotSelf, UnexpectedCallSuccess
var (
	// ErrQuoterUnexpectedRevert wraps the actual revert of the simulated swap (nested).
	ErrQuoterUnexpectedRevert = register("UnexpectedRevertBytes(bytes)", 0)
	// ErrQuoterNotEnoughLiquidity the simulated swap was only partially filled (the pool ran out of liquidity
	// before reaching the requested amount). PoolId is bytes32 in the ABI.
	ErrQuoterNotEnoughLiquidity = register("NotEnoughLiquidity(bytes32)", -1)
	ErrNotSelf                  = register("NotSelf()", -1)
	ErrUnexpectedCallSuccess    = register("UnexpectedCallSuccess()", -1)
)

// V4Router (v4-periphery) - slippage protection, https://github.com/Uniswap/v4-periphery
//   - src/interfaces/IV4Router.sol: V4TooLittleReceived, V4TooMuchRequested
var (
	ErrTooLittleReceived = register("V4TooLittleReceived(uint256,uint256)", -1)
	ErrTooMuchRequested  = register("V4TooMuchRequested(uint256,uint256)", -1)
)

// UniversalRouter, https://github.com/Uniswap/universal-router
//   - contracts/interfaces/IUniversalRouter.sol: TransactionDeadlinePassed, ExecutionFailed, LengthMismatch
//   - contracts/base/Dispatcher.sol: InvalidCommandType
var (
	ErrTransactionDeadlinePassed = register("TransactionDeadlinePassed()", -1)
	// ErrExecutionFailed wraps the revert of a failed command (nested).
	ErrExecutionFailed    = register("ExecutionFailed(uint256,bytes)", 1)
	ErrLengthMismatch     = register("LengthMismatch()", -1)
	ErrInvalidCommandType = register("InvalidCommandType(uint256)", -1)
)

// Permit2, https://github.com/Uniswap/permit2
//   - src/interfaces/IAllowanceTransfer.sol: AllowanceExpired, InsufficientAllowance
var (
	ErrAllowanceExpired      = register("AllowanceExpired(uint256)", -1)
	ErrInsufficientAllowance = register("InsufficientAllowance(uint256)", -1)
)
