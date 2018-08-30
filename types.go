package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	rivineencoding "github.com/rivine/rivine/encoding"
	"github.com/rivine/rivine/modules"
)

// message pack
//go:generate msgp -marshal=false -io=true

// protobuf (using gogo/protobuf)
//   requires protoc (https://github.com/protocolbuffers/protobuf/releases/tag/v3.6.1) and
// gogofaster (go get -u github.com/gogo/protobuf/proto github.com/gogo/protobuf/gogoproto github.com/gogo/protobuf/protoc-gen-gogofaster)
//go:generate protoc -I=. -I=$GOPATH/src -I=vendor/github.com/gogo/protobuf/protobuf --gogofaster_out=. types.proto

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

// as to ensure we implement the interfaces we think we implement
var (
	_ encoding.ProtocolBufferMarshaler   = ExplorerState{}
	_ encoding.ProtocolBufferUnmarshaler = (*ExplorerState)(nil)

	_ encoding.ProtocolBufferMarshaler   = NetworkInfo{}
	_ encoding.ProtocolBufferUnmarshaler = (*NetworkInfo)(nil)
)

// NewExplorerState creates a nil (fresh) explorer state.
func NewExplorerState() ExplorerState {
	return ExplorerState{
		CurrentChangeID: types.AsConsensusChangeID(modules.ConsensusChangeBeginning),
	}
}

// ProtocolBufferMarshal implements encoding.ProtocolBufferMarshaler.ProtocolBufferMarshal
// using the generated code based on the PBExplorerState Message defined in ./types.proto
func (state ExplorerState) ProtocolBufferMarshal(w encoding.ProtocolBufferWriter) error {
	err := w.Marshal(&PBExplorerState{
		CurrentConsensusChangeId: rivineencoding.Marshal(state.CurrentChangeID),
	})
	if err != nil {
		return fmt.Errorf("ExplorerState: %v", err)
	}
	return nil
}

// ProtocolBufferUnmarshal implements encoding.ProtocolBufferUnmarshaler.ProtocolBufferUnmarshal
// using the generated code based on the PBExplorerState Message defined in ./types.proto
func (state *ExplorerState) ProtocolBufferUnmarshal(r encoding.ProtocolBufferReader) error {
	var pb PBExplorerState
	err := r.Unmarshal(&pb)
	if err != nil {
		return fmt.Errorf("ExplorerState: %v", err)
	}
	err = rivineencoding.Unmarshal(pb.CurrentConsensusChangeId, &state.CurrentChangeID)
	if err != nil {
		return fmt.Errorf("ExplorerState: CurrentChangeID: %v", err)
	}
	return nil
}

// ProtocolBufferMarshal implements encoding.ProtocolBufferMarshaler.ProtocolBufferMarshal
// using the generated code based on the PBExplorerState Message defined in ./types.proto
func (info NetworkInfo) ProtocolBufferMarshal(w encoding.ProtocolBufferWriter) error {
	err := w.Marshal(&PBNetworkInfo{
		ChainName:   info.ChainName,
		NetworkName: info.NetworkName,
	})
	if err != nil {
		return fmt.Errorf("NetworkInfo: %v", err)
	}
	return nil
}

// ProtocolBufferUnmarshal implements encoding.ProtocolBufferUnmarshaler.ProtocolBufferUnmarshal
// using the generated code based on the PBExplorerState Message defined in ./types.proto
func (info *NetworkInfo) ProtocolBufferUnmarshal(r encoding.ProtocolBufferReader) error {
	var pb PBNetworkInfo
	err := r.Unmarshal(&pb)
	if err != nil {
		return fmt.Errorf("NetworkInfo: %v", err)
	}
	info.ChainName, info.NetworkName = pb.ChainName, pb.NetworkName
	return nil
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

// MarshalSia implements rivine/encoding.MarshalSia
func (lt LockType) MarshalSia(w io.Writer) error {
	_, err := w.Write([]byte{byte(lt)})
	return err
}

// UnmarshalSia implements rivine/encoding.UnmarshalSia
func (lt *LockType) UnmarshalSia(r io.Reader) error {
	var b [1]byte
	_, err := r.Read(b[:])
	if err != nil {
		return err
	}
	nlt := LockType(b[0])
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

// BytesLoader loads a byte slice and uses it as the (parsed) value.
type BytesLoader interface {
	LoadBytes([]byte) error
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

// Byte returns this CoinOutputState as a byte
func (cos CoinOutputState) Byte() byte {
	return byte(cos)
}

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

// MarshalSia implements rivine/encoding.MarshalSia
func (cos CoinOutputState) MarshalSia(w io.Writer) error {
	_, err := w.Write([]byte{cos.Byte()})
	return err
}

// UnmarshalSia implements rivine/encoding.UnmarshalSia
func (cos *CoinOutputState) UnmarshalSia(r io.Reader) error {
	var b [1]byte
	_, err := r.Read(b[:])
	if err != nil {
		return err
	}
	ncos := CoinOutputState(b[0])
	if ncos == CoinOutputStateNil || ncos > CoinOutputStateSpent {
		return fmt.Errorf("invalid coin output state %d", ncos)
	}
	*cos = ncos
	return nil
}
