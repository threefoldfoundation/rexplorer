package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/threefoldfoundation/rexplorer/pkg/types"

	"github.com/rivine/rivine/modules"
)

//go:generate msgp -marshal=false -io=true

// private types
type (
	// ExplorerState collects the (internal) state for the explorer.
	ExplorerState struct {
		CurrentChangeID types.ConsensusChangeID `json:"currentchangeid" msg:"currentchangeid"`
	}

	// NetworkInfo defines the info of the chain network data is dumped from,
	// used as to prevent name colissions.
	NetworkInfo struct {
		ChainName   string `json:"chainName" msg:"chainName"`
		NetworkName string `json:"networkName" msg:"networkName"`
	}
)

// NewExplorerState creates a nil (fresh) explorer state.
func NewExplorerState() ExplorerState {
	return ExplorerState{
		CurrentChangeID: types.AsConsensusChangeID(modules.ConsensusChangeBeginning),
	}
}

// LockType represents the type of a lock, used to lock a (coin) output.
type LockType uint8

// The different types of locks used to lock (coin) outputs.
const (
	LockTypeNone LockType = iota
	LockTypeHeight
	LockTypeTime
)

// String implements Stringer.String
func (lt LockType) String() string {
	return strconv.FormatUint(uint64(lt), 10)
}

// LoadString implements StringLoader.LoadString
func (lt *LockType) LoadString(str string) error {
	v, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		return err
	}
	nlt := LockType(v)
	if nlt > LockTypeTime {
		return fmt.Errorf("invalid lock type %d", nlt)
	}
	*lt = nlt
	return nil
}

// StringLoader loads a string and uses it as the (parsed) value.
type StringLoader interface {
	LoadString(string) error
}

// FormatStringers formats the given stringers into one string using the given separator
func FormatStringers(separator string, stringers ...interface{}) string {
	n := len(stringers)
	if n == 0 {
		return ""
	}
	ss := make([]string, n)
	for i, stringer := range stringers {
		switch v := stringer.(type) {
		case string:
			ss[i] = v
		case fmt.Stringer:
			ss[i] = v.String()
		default:
			panic(fmt.Sprintf("unsuported value %[1]v (T: %[1]T)", stringer))
		}
	}
	return strings.Join(ss, separator)
}

// ParseStringLoaders splits the given string into the given separator
// and loads each part into a given string loader.
func ParseStringLoaders(csv, separator string, stringLoaders ...interface{}) (err error) {
	n := len(stringLoaders)
	parts := strings.SplitN(csv, separator, n)
	if m := len(parts); n != m {
		return fmt.Errorf("CSV record has incorrect amount of records, expected %d but received %d", n, m)
	}
	for i, sl := range stringLoaders {
		switch v := sl.(type) {
		case *string:
			*v = parts[i]
		case StringLoader:
			err = v.LoadString(parts[i])
			if err != nil {
				return
			}
		default:
			panic(fmt.Sprintf("unsuported value %[1]v (T: %[1]T)", sl))
		}
	}
	return
}

// CoinOutputState represents the state of a coin output.
type CoinOutputState uint8

// The different states a coin output can be in.
const (
	CoinOutputStateNil CoinOutputState = iota
	CoinOutputStateLiquid
	CoinOutputStateLocked
	CoinOutputStateSpent
)

// String implements Stringer.String
func (cos CoinOutputState) String() string {
	return strconv.FormatUint(uint64(cos), 10)
}

// LoadString implements StringLoader.LoadString
func (cos *CoinOutputState) LoadString(str string) error {
	v, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		return err
	}
	ncos := CoinOutputState(v)
	if ncos == CoinOutputStateNil || ncos > CoinOutputStateSpent {
		return fmt.Errorf("invalid coin output state %d", ncos)
	}
	*cos = ncos
	return nil
}
