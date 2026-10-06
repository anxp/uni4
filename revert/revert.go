// Package revert turns raw revert data from Uniswap contracts into Go errors.
//
// A reverted call returns bytes: a 4-byte error selector + ABI-encoded arguments.
// Decode finds the error by selector (e.g. PoolNotInitialized()), extracts Error(string) / Panic(uint256)
// details and unwraps nested reverts (V4Quoter's UnexpectedRevertBytes, UniversalRouter's ExecutionFailed,
// hooks' WrappedError). The result works with errors.Is / errors.As through all wrappers:
//
//	if errors.Is(err, revert.ErrPoolNotInitialized) { ... }
package revert

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/anxp/evmtc/com"
)

// Error is a decoded revert.
type Error struct {
	// Data is the raw revert data (selector + ABI-encoded arguments).
	Data []byte
	// Selector is the first 4 bytes of Data (zero if the revert has no data).
	Selector [4]byte
	// Signature is the error signature, e.g. "PoolNotInitialized()". Empty if the selector is unknown.
	Signature string
	// Reason is the message of a standard Error(string) revert (require/revert with a string).
	Reason string
	// PanicCode is the code of a standard Panic(uint256) revert (e.g. 0x11 for arithmetic overflow).
	PanicCode *big.Int
	// Inner is the decoded nested revert for wrapper errors, nil otherwise.
	Inner *Error
}

func (e *Error) Error() string {
	var s string

	switch {
	case len(e.Data) == 0:
		s = "reverted without data"
	case e.Signature == sigError:
		s = fmt.Sprintf("reverted: %q", e.Reason)
	case e.Signature == sigPanic:
		s = fmt.Sprintf("panic: code 0x%x", e.PanicCode)
	case e.Signature != "":
		s = e.Signature
	default:
		s = fmt.Sprintf("unknown error 0x%s (data 0x%s)", hex.EncodeToString(e.Selector[:]), hex.EncodeToString(e.Data))
	}

	if e.Inner != nil {
		s += " <- " + e.Inner.Error()
	}

	return s
}

// Is matches errors by selector, so that errors.Is(err, ErrPoolNotInitialized) works
// regardless of the error's arguments.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}

	if len(t.Data) == 0 && t.Signature == "" {
		return len(e.Data) == 0
	}

	return e.Selector == t.Selector
}

// Unwrap exposes the nested revert to errors.Is / errors.As.
func (e *Error) Unwrap() error {
	if e.Inner == nil {
		return nil
	}

	return e.Inner
}

// Decode parses revert data. It never fails: undecodable data results in an Error with unknown selector.
func Decode(data []byte) *Error {
	e := &Error{Data: data}
	if len(data) < 4 {
		return e
	}

	copy(e.Selector[:], data[:4])
	args := data[4:]

	k, ok := known[e.Selector]
	if !ok {
		return e
	}
	e.Signature = k.signature

	switch k.signature {
	case sigError:
		if b, ok := bytesArg(args, 0); ok {
			e.Reason = string(b)
		}
	case sigPanic:
		if len(args) >= 32 {
			e.PanicCode = new(big.Int).SetBytes(args[:32])
		}
	}

	if k.nestedArg >= 0 {
		if inner, ok := bytesArg(args, k.nestedArg); ok {
			e.Inner = Decode(inner)
		}
	}

	return e
}

// bytesArg reads a dynamic "bytes" (or "string") argument #index from ABI-encoded arguments.
func bytesArg(args []byte, index int) ([]byte, bool) {
	offset, ok := wordAsOffset(args, index*32)
	if !ok {
		return nil, false
	}

	length, ok := wordAsOffset(args, offset)
	if !ok {
		return nil, false
	}

	start := offset + 32
	if start+length > len(args) {
		return nil, false
	}

	return args[start : start+length], true
}

// wordAsOffset reads the 32-byte word at data[pos:] as an offset or length of a dynamic ABI value.
//
// Such values point inside data itself, so a valid one is never larger than len(data).
// The whole 256-bit word is read; if it is bigger than len(data) (malformed or truncated revert data),
// false is returned instead of a truncated number, so callers never slice out of bounds.
func wordAsOffset(data []byte, pos int) (int, bool) {
	if pos < 0 || pos+32 > len(data) {
		return 0, false
	}

	v := new(big.Int).SetBytes(data[pos : pos+32])
	if !v.IsInt64() || v.Int64() > int64(len(data)) {
		return 0, false
	}

	return int(v.Int64()), true
}

func selectorOf(signature string) [4]byte {
	var s [4]byte
	copy(s[:], com.Keccak256([]byte(signature))[:4])
	return s
}
