package revert

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/anxp/evmtc/rpc"
)

func word(b ...byte) []byte {
	w := make([]byte, 32)
	copy(w[32-len(b):], b)
	return w
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// encodeBytesError builds selector + abi.encode(bytes) for a single-bytes-argument error.
func encodeBytesError(sel [4]byte, payload []byte) []byte {
	padded := make([]byte, (len(payload)+31)/32*32)
	copy(padded, payload)
	return bytes.Join([][]byte{sel[:], word(0x20), word(byte(len(payload))), padded}, nil)
}

func TestDecodeRealQuoterRevert(t *testing.T) {
	// Real revert data returned by V4Quoter on Ethereum mainnet for a non-existent pool.
	data := mustHex("6190b2b0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"486aa30700000000000000000000000000000000000000000000000000000000")

	e := Decode(data)
	if e.Signature != "UnexpectedRevertBytes(bytes)" {
		t.Fatalf("unexpected signature %q", e.Signature)
	}
	if e.Inner == nil || e.Inner.Signature != "PoolNotInitialized()" {
		t.Fatalf("unexpected inner error: %+v", e.Inner)
	}

	err := fmt.Errorf("quote failed: %w", e)
	if !errors.Is(err, ErrPoolNotInitialized) {
		t.Error("errors.Is must find nested PoolNotInitialized")
	}
	if !errors.Is(err, ErrQuoterUnexpectedRevert) {
		t.Error("errors.Is must match the wrapper error too")
	}
	if errors.Is(err, ErrQuoterNotEnoughLiquidity) {
		t.Error("errors.Is must not match unrelated errors")
	}

	want := "UnexpectedRevertBytes(bytes) <- PoolNotInitialized()"
	if e.Error() != want {
		t.Errorf("got %q, want %q", e.Error(), want)
	}
}

func TestNotEnoughLiquiditySelectorsDiffer(t *testing.T) {
	if ErrNotEnoughLiquidity.Selector == ErrQuoterNotEnoughLiquidity.Selector {
		t.Fatal("core and quoter NotEnoughLiquidity must have different selectors")
	}

	// Selector hardcoded in v4-core SqrtPriceMath.sol assembly: mstore(0, 0x4323a555).
	if hex.EncodeToString(ErrNotEnoughLiquidity.Selector[:]) != "4323a555" {
		t.Errorf("unexpected selector %x", ErrNotEnoughLiquidity.Selector)
	}

	e := Decode(ErrNotEnoughLiquidity.Selector[:])
	if !errors.Is(e, ErrNotEnoughLiquidity) || errors.Is(e, ErrQuoterNotEnoughLiquidity) {
		t.Errorf("unexpected match for %v", e)
	}
}

func TestDecodeErrorString(t *testing.T) {
	data := encodeBytesError(selectorOf(sigError), []byte("STF"))

	e := Decode(data)
	if e.Reason != "STF" || !errors.Is(e, ErrString) {
		t.Fatalf("unexpected decode: %+v", e)
	}
	if e.Error() != `reverted: "STF"` {
		t.Errorf("unexpected message %q", e.Error())
	}
}

func TestDecodePanic(t *testing.T) {
	sel := selectorOf(sigPanic)
	e := Decode(append(sel[:], word(0x11)...))

	if e.PanicCode == nil || e.PanicCode.Int64() != 0x11 || !errors.Is(e, ErrPanic) {
		t.Fatalf("unexpected decode: %+v", e)
	}
}

func TestDecodeExecutionFailed(t *testing.T) {
	// ExecutionFailed(uint256 commandIndex, bytes message) wrapping V4TooLittleReceived(100, 99).
	innerSel := selectorOf("V4TooLittleReceived(uint256,uint256)")
	inner := bytes.Join([][]byte{innerSel[:], word(100), word(99)}, nil) // 68 bytes

	outerSel := selectorOf("ExecutionFailed(uint256,bytes)")
	padded := make([]byte, 96)
	copy(padded, inner)
	data := bytes.Join([][]byte{outerSel[:], word(0), word(0x40), word(byte(len(inner))), padded}, nil)

	e := Decode(data)
	if !errors.Is(e, ErrExecutionFailed) || !errors.Is(e, ErrTooLittleReceived) {
		t.Fatalf("unexpected decode: %v", e)
	}
	if !bytes.Equal(e.Inner.Data, inner) {
		t.Errorf("inner data mismatch: %x", e.Inner.Data)
	}
}

func TestDecodeEdgeCases(t *testing.T) {
	if e := Decode(nil); !errors.Is(e, ErrNoData) || e.Error() != "reverted without data" {
		t.Errorf("empty data: %v", e)
	}

	unknown := Decode(mustHex("deadbeef01"))
	if unknown.Signature != "" || errors.Is(unknown, ErrNoData) || errors.Is(unknown, ErrPoolNotInitialized) {
		t.Errorf("unknown selector decoded as %+v", unknown)
	}

	// Wrapper with a truncated/garbage nested payload must not panic and must have no Inner.
	sel := selectorOf("UnexpectedRevertBytes(bytes)")
	broken := Decode(append(sel[:], word(0xff)...))
	if broken.Inner != nil || broken.Signature != "UnexpectedRevertBytes(bytes)" {
		t.Errorf("broken wrapper decoded as %+v", broken)
	}
}

func TestFromRPCError(t *testing.T) {
	rpcErr := &rpc.RpcError{
		Code:    3,
		Message: "execution reverted",
		Data:    json.RawMessage(`"0x486aa307"`),
	}
	wrapped := fmt.Errorf("failed to call eth_call: %w", rpcErr)

	err := FromRPCError(wrapped)
	if !errors.Is(err, ErrPoolNotInitialized) {
		t.Errorf("expected ErrPoolNotInitialized, got %v", err)
	}

	var gotRPC *rpc.RpcError
	if !errors.As(err, &gotRPC) || gotRPC.Code != 3 {
		t.Error("original RpcError must stay reachable via errors.As")
	}

	want := "failed to call eth_call: RPC error #3: execution reverted: PoolNotInitialized()"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}

	// Errors without revert data are returned unchanged.
	plain := errors.New("connection refused")
	if FromRPCError(plain) != plain {
		t.Error("error without RpcError must be returned as is")
	}

	noData := &rpc.RpcError{Code: -32000, Message: "header not found"}
	if FromRPCError(noData) != error(noData) {
		t.Error("RpcError without data must be returned as is")
	}
}
