package pool

const (
	// MaxLPFee is the maximum static LP fee (100%) in hundredths of a bip: 1_000_000 == 100%.
	MaxLPFee uint32 = 1_000_000

	// DynamicFeeFlag is the PoolKey.Fee value that marks a pool with a dynamic fee managed by its hook.
	DynamicFeeFlag uint32 = 0x800000

	// MinTickSpacing and MaxTickSpacing are the tick spacing bounds enforced by PoolManager.
	MinTickSpacing int32 = 1
	MaxTickSpacing int32 = 32767
)

// Common fee tiers in hundredths of a bip (e.g. 3000 == 0.30%).
// Unlike v3, v4 allows any fee, these are just the most widely used values.
const (
	Fee00007 uint32 = 7      // 0.0007%, used by stablecoin pairs (e.g. USDC/USDT)
	Fee00008 uint32 = 8      // 0.0008%, used by stablecoin pairs (e.g. USDC/USDT)
	Fee0001  uint32 = 10     // 0.001%, used by stablecoin pairs (e.g. USDC/USDT)
	Fee001   uint32 = 100    // 0.01%
	Fee005   uint32 = 500    // 0.05%
	Fee030   uint32 = 3000   // 0.30%
	Fee100   uint32 = 10_000 // 1.00%
)

// DefaultTickSpacing returns the tick spacing conventionally paired with the given fee tier
// (the same pairing as in Uniswap v3 and the Uniswap interface; stablecoin tiers below 0.01%
// use tick spacing 1, as the existing USDC/USDT pools on Ethereum do).
// The second return value is false if the fee is not one of the common tiers;
// in that case the tick spacing must be known from elsewhere (e.g. the pool's Initialize event).
func DefaultTickSpacing(fee uint32) (int32, bool) {
	switch fee {
	case Fee00007, Fee00008, Fee0001, Fee001:
		return 1, true
	case Fee005:
		return 10, true
	case Fee030:
		return 60, true
	case Fee100:
		return 200, true
	default:
		return 0, false
	}
}

// IsDynamicFee reports whether the fee value marks a dynamic-fee pool.
func IsDynamicFee(fee uint32) bool {
	return fee == DynamicFeeFlag
}
