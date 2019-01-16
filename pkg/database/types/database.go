package types

import (
	"bytes"
	"fmt"

	"github.com/threefoldfoundation/rexplorer/pkg/types"

	"github.com/threefoldtech/rivine/pkg/encoding/siabin"
)

type (
	// CoinOutput is used to store all spent/unspent coin outputs in the custom CSV format (used internally only)
	CoinOutput struct {
		State       CoinOutputState
		UnlockHash  types.UnlockHash
		CoinValue   types.Currency
		LockType    LockType
		LockValue   types.LockValue
		Description string
	}
	// CoinOutputLock is used to store the lock value and a reference to its parent CoinOutput,
	// as to store the lock in a scoped bucket.
	CoinOutputLock struct {
		CoinOutputID types.CoinOutputID
		LockValue    types.LockValue
	}
)

const csvSeperator = ","

// String implements Stringer.String
func (co CoinOutput) String() string {
	str := FormatStringers(csvSeperator, co.State, co.UnlockHash, co.CoinValue, co.LockType, co.LockValue, co.Description)
	return str
}

// LoadString implements StringLoader.LoadString
func (co *CoinOutput) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &co.State, &co.UnlockHash, &co.CoinValue, &co.LockType, &co.LockValue, &co.Description)
}

// Bytes returns a binary representation of a CoinOutput,
// using Rivine's binary encoding package (github.com/threefoldtech/rivine/pkg/encoding/siabin).
func (co CoinOutput) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	encoder := siabin.NewEncoder(buf)
	encoder.EncodeAll(
		co.State,
		co.UnlockHash,
		co.CoinValue,
		co.LockType,
		co.LockValue,
		co.Description,
	)
	return buf.Bytes()
}

// LoadBytes decodes the given bytes using the binary representation of a CoinOutput,
// making use of Rivine's binary encoding package (github.com/threefoldtech/rivine/pkg/encoding/siabin) to decode,
// the previously encoded CoinOutput.
func (co *CoinOutput) LoadBytes(b []byte) error {
	decoder := siabin.NewDecoder(bytes.NewReader(b))
	err := decoder.DecodeAll(
		&co.State,
		&co.UnlockHash,
		&co.CoinValue,
		&co.LockType,
		&co.LockValue,
		&co.Description,
	)
	if err != nil {
		return fmt.Errorf("failed to decode CoinOutput: %v", err)
	}
	return nil
}

// String implements Stringer.String
func (col CoinOutputLock) String() string {
	return FormatStringers(csvSeperator, col.CoinOutputID, col.LockValue)
}

// LoadString implements StringLoader.LoadString
func (col *CoinOutputLock) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &col.CoinOutputID, &col.LockValue)
}
