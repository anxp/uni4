// Package deployments holds Uniswap v4 contract addresses for supported networks.
//
// Presets are taken from the official Uniswap deployments list:
// https://developers.uniswap.org/docs/protocols/v4/deployments
// (machine-readable feed: https://developers.uniswap.org/deployments.json).
//
// Networks without a preset (testnets, forks, new chains) are supported by constructing
// a Deployment manually and passing it to the client.
package deployments

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anxp/evmtc/com"
)

// Permit2Address is the canonical Permit2 address, identical on all networks.
var Permit2Address = com.MustAddressFromHex("0x000000000022D473030F116dDEE9F6B43aC78BA3")

// Deployment is a set of Uniswap v4 contract addresses on a single network.
type Deployment struct {
	// Name is a human-readable network name, informational only.
	Name    string
	ChainID *big.Int

	// UniversalRouter is the swap entry point. Presets use Universal Router v2.1.2.
	UniversalRouter com.Address
	// Permit2 is the token approval contract used by UniversalRouter and PositionManager.
	Permit2 com.Address
	// PoolManager is the v4 singleton holding the state of all pools.
	PoolManager com.Address
	// PositionManager is the ERC-721 contract for LP positions.
	PositionManager com.Address
	// V4Quoter simulates swaps off-chain (via eth_call) to get expected amounts.
	V4Quoter com.Address
	// StateView is a read-only lens over PoolManager state (slot0, liquidity, positions).
	StateView com.Address
}

var (
	Ethereum = Deployment{
		Name:            "Ethereum",
		ChainID:         big.NewInt(1),
		UniversalRouter: com.MustAddressFromHex("0x23617e59A5925b2A4Bf75d73ff6711cD0b29De85"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x000000000004444c5dc75cB358380D2e3dE08A90"),
		PositionManager: com.MustAddressFromHex("0xbD216513d74C8cf14cf4747E6AaA6420FF64ee9e"),
		V4Quoter:        com.MustAddressFromHex("0x52F0E24D1c21C8A0cB1e5a5dD6198556BD9E1203"),
		StateView:       com.MustAddressFromHex("0x7fFE42C4a5DEeA5b0feC41C94C136Cf115597227"),
	}

	Optimism = Deployment{
		Name:            "Optimism",
		ChainID:         big.NewInt(10),
		UniversalRouter: com.MustAddressFromHex("0xC09255D86DB563cBc11C2fCf4a0C512e160111B4"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x9a13F98Cb987694C9F086b1F5eB990EeA8264Ec3"),
		PositionManager: com.MustAddressFromHex("0x3C3Ea4B57a46241e54610e5f022E5c45859A1017"),
		V4Quoter:        com.MustAddressFromHex("0x1f3131A13296FB91C90870043742C3CDBFF1A8d7"),
		StateView:       com.MustAddressFromHex("0xc18a3169788F4F75A170290584ECA6395C75Ecdb"),
	}

	BSC = Deployment{
		Name:            "BNB Smart Chain",
		ChainID:         big.NewInt(56),
		UniversalRouter: com.MustAddressFromHex("0xDc264714F68d84CF29BC605589405E78bDBE7C9f"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x28e2Ea090877bF75740558f6BFB36A5ffeE9e9dF"),
		PositionManager: com.MustAddressFromHex("0x7A4a5c919aE2541AeD11041A1AEeE68f1287f95b"),
		V4Quoter:        com.MustAddressFromHex("0x9F75dD27D6664c475B90e105573E550ff69437B0"),
		StateView:       com.MustAddressFromHex("0xd13Dd3D6E93f276FAfc9Db9E6BB47C1180aeE0c4"),
	}

	Unichain = Deployment{
		Name:            "Unichain",
		ChainID:         big.NewInt(130),
		UniversalRouter: com.MustAddressFromHex("0xD1b797D92d87B688193A2B976eFc8D577D204343"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x1F98400000000000000000000000000000000004"),
		PositionManager: com.MustAddressFromHex("0x4529A01c7A0410167c5740C487A8DE60232617bf"),
		V4Quoter:        com.MustAddressFromHex("0x333E3C607B141b18fF6de9f258db6e77fE7491E0"),
		StateView:       com.MustAddressFromHex("0x86e8631A016F9068C3f085fAF484Ee3F5fDee8f2"),
	}

	Polygon = Deployment{
		Name:            "Polygon",
		ChainID:         big.NewInt(137),
		UniversalRouter: com.MustAddressFromHex("0xDc264714F68d84CF29BC605589405E78bDBE7C9f"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x67366782805870060151383F4BbFF9daB53e5cD6"),
		PositionManager: com.MustAddressFromHex("0x1Ec2eBf4F37E7363FDfe3551602425af0B3ceef9"),
		V4Quoter:        com.MustAddressFromHex("0xb3d5c3Dfc3a7aEbFF71895A7191796BFFc2c81b9"),
		StateView:       com.MustAddressFromHex("0x5eA1bD7974c8A611cBAB0bDCAFcB1D9CC9b3BA5a"),
	}

	Base = Deployment{
		Name:            "Base",
		ChainID:         big.NewInt(8453),
		UniversalRouter: com.MustAddressFromHex("0xd6145b2D3F379919E8CdEda7B97e37c4b2Ca9c40"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x498581fF718922c3f8e6A244956aF099B2652b2b"),
		PositionManager: com.MustAddressFromHex("0x7C5f5A4bBd8fD63184577525326123B519429bDc"),
		V4Quoter:        com.MustAddressFromHex("0x0d5e0F971ED27FBfF6c2837bf31316121532048D"),
		StateView:       com.MustAddressFromHex("0xA3c0c9b65baD0b08107Aa264b0f3dB444b867A71"),
	}

	Arbitrum = Deployment{
		Name:            "Arbitrum One",
		ChainID:         big.NewInt(42161),
		UniversalRouter: com.MustAddressFromHex("0x2d01411773c8C24805306E89A41F7855C3c4Fe65"),
		Permit2:         Permit2Address,
		PoolManager:     com.MustAddressFromHex("0x360E68faCcca8cA495c1B759Fd9EEe466db9FB32"),
		PositionManager: com.MustAddressFromHex("0xd88F38F930b7952f2DB2432Cb002E7abbF3dD869"),
		V4Quoter:        com.MustAddressFromHex("0x3972C00f7ed4885e145823eb7C655375d275A1C5"),
		StateView:       com.MustAddressFromHex("0x76Fd297e2D437cd7f76d50F01AfE6160f86e9990"),
	}
)

// presets is keyed by decimal chain ID string (same approach as evmtc/chain registry).
var presets = func() map[string]Deployment {
	m := make(map[string]Deployment)
	for _, d := range []Deployment{Ethereum, Optimism, BSC, Unichain, Polygon, Base, Arbitrum} {
		m[d.ChainID.String()] = d
	}
	return m
}()

var ErrUnsupportedChain = errors.New("no built-in Uniswap v4 deployment for this chain")

// GetDeploymentByChainID returns the built-in preset for the given chain ID.
// For networks without a preset, construct a Deployment manually.
func GetDeploymentByChainID(chainID *big.Int) (Deployment, error) {
	if chainID == nil {
		return Deployment{}, errors.New("chain ID is nil")
	}

	d, ok := presets[chainID.String()]
	if !ok {
		return Deployment{}, fmt.Errorf("%w: chain #%s", ErrUnsupportedChain, chainID.String())
	}

	// Copy ChainID so that the caller can't mutate the preset through the shared pointer.
	d.ChainID = new(big.Int).Set(d.ChainID)

	return d, nil
}

// SupportedChainIDs returns chain IDs of all built-in presets.
func SupportedChainIDs() []*big.Int {
	ids := make([]*big.Int, 0, len(presets))
	for _, d := range presets {
		ids = append(ids, new(big.Int).Set(d.ChainID))
	}

	return ids
}

// ValidateForSwap checks that all addresses required for swapping via UniversalRouter are set.
func (d Deployment) ValidateForSwap() error {
	required := []struct {
		name string
		addr com.Address
	}{
		{"UniversalRouter", d.UniversalRouter},
		{"Permit2", d.Permit2},
		{"V4Quoter", d.V4Quoter},
	}

	if d.ChainID == nil || d.ChainID.Sign() <= 0 {
		return errors.New("deployment: ChainID is not set")
	}

	for _, r := range required {
		if r.addr.IsZero() {
			return fmt.Errorf("deployment for chain #%s: %s address is not set", d.ChainID.String(), r.name)
		}
	}

	return nil
}
