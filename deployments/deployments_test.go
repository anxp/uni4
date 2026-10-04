package deployments

import (
	"errors"
	"math/big"
	"testing"
)

func TestPresetsAreValid(t *testing.T) {
	if len(presets) != 7 {
		t.Errorf("expected 7 presets, got %d", len(presets))
	}

	for id, d := range presets {
		if d.ChainID.String() != id {
			t.Errorf("preset %q is registered under chain #%s but has ChainID %s", d.Name, id, d.ChainID)
		}

		if err := d.ValidateForSwap(); err != nil {
			t.Errorf("preset %q: %v", d.Name, err)
		}

		if d.PoolManager.IsZero() || d.PositionManager.IsZero() || d.StateView.IsZero() {
			t.Errorf("preset %q: PoolManager, PositionManager and StateView must be set", d.Name)
		}

		if d.Permit2 != Permit2Address {
			t.Errorf("preset %q: unexpected Permit2 address %s", d.Name, d.Permit2.Hex())
		}
	}
}

func TestGetDeploymentByChainID(t *testing.T) {
	d, err := GetDeploymentByChainID(big.NewInt(8453))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.UniversalRouter != Base.UniversalRouter {
		t.Errorf("got %s, want %s", d.UniversalRouter.Hex(), Base.UniversalRouter.Hex())
	}

	// Mutating the returned ChainID must not affect the preset.
	d.ChainID.SetInt64(1)
	if Base.ChainID.Int64() != 8453 {
		t.Error("preset ChainID was mutated through returned Deployment")
	}

	if _, err := GetDeploymentByChainID(big.NewInt(999999)); !errors.Is(err, ErrUnsupportedChain) {
		t.Errorf("expected ErrUnsupportedChain, got %v", err)
	}

	if _, err := GetDeploymentByChainID(nil); err == nil {
		t.Error("expected error for nil chain ID")
	}
}

func TestValidateForSwap(t *testing.T) {
	d := Base
	d.V4Quoter = [20]byte{}
	if err := d.ValidateForSwap(); err == nil {
		t.Error("expected error for missing V4Quoter")
	}

	d = Base
	d.ChainID = nil
	if err := d.ValidateForSwap(); err == nil {
		t.Error("expected error for missing ChainID")
	}
}
