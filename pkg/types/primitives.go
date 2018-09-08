package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"

	"github.com/rivine/rivine/crypto"
	"github.com/rivine/rivine/encoding"
	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"

	"github.com/tinylib/msgp/msgp"
)

// Type overwrites as to be able to define MessagePack (de)serialization methods for them.
type (
	// Timestamp overwrites the Rivine Timestamp type,
	// encapsulating it internally for practical reasons.
	Timestamp struct {
		types.Timestamp
	}
	// BlockHeight overwrites the Rivine BlockHeight type,
	// encapsulating it internally for practical reasons.
	BlockHeight struct {
		types.BlockHeight
	}
	// Currency overwrites the Rivine Currency type,
	// encapsulating it internally for practical reasons.
	Currency struct {
		types.Currency
	}
	// UnlockHash overwrites the Rivine UnlockHash type,
	// encapsulating it internally for practical reasons.
	UnlockHash struct {
		types.UnlockHash
	}

	// CoinOutputID overwrites the Rivine CoinOutputID type,
	// encapsulating it internally for practical reasons.
	CoinOutputID struct {
		types.CoinOutputID
	}

	// ConsensusChangeID overwrites the Rivine ConsensusChangeID type,
	// encapsulating it internally for practical reasons.
	ConsensusChangeID struct {
		modules.ConsensusChangeID
	}
)

// AsTimestamp turns a Rivine-typed Timestamp into
// the Timestamp overwritten type used in this project.
func AsTimestamp(c types.Timestamp) Timestamp {
	return Timestamp{Timestamp: c}
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(ts.Timestamp)
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &ts.Timestamp)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (ts Timestamp) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteUint64(uint64(ts.Timestamp))
	if err != nil {
		return fmt.Errorf("failed to write timestamp as uint64: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (ts *Timestamp) DecodeMsg(r *msgp.Reader) error {
	u, err := r.ReadUint64()
	if err != nil {
		return fmt.Errorf("failed to read timestamp as uint64: %v", err)
	}
	ts.Timestamp = types.Timestamp(u)
	return nil
}

// Msgsize returns an upper bound estimate of the number
// of bytes occupied by the serialized message
func (ts Timestamp) Msgsize() int {
	return msgp.Uint64Size
}

// LockValue returns this Timstamp as a LockValue
func (ts Timestamp) LockValue() LockValue {
	return LockValue(ts.Timestamp)
}

// AsBlockHeight turns a Rivine-typed BlockHeight into
// the BlockHeight overwritten type used in this project.
func AsBlockHeight(c types.BlockHeight) BlockHeight {
	return BlockHeight{BlockHeight: c}
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (bh BlockHeight) MarshalJSON() ([]byte, error) {
	return json.Marshal(bh.BlockHeight)
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (bh *BlockHeight) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &bh.BlockHeight)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (bh BlockHeight) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteUint64(uint64(bh.BlockHeight))
	if err != nil {
		return fmt.Errorf("failed to write blockheight as uint64: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (bh *BlockHeight) DecodeMsg(r *msgp.Reader) error {
	u, err := r.ReadUint64()
	if err != nil {
		return fmt.Errorf("failed to read blockheight as uint64: %v", err)
	}
	bh.BlockHeight = types.BlockHeight(u)
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (bh BlockHeight) Msgsize() int {
	return msgp.Uint64Size
}

// Increase this block height by 1
func (bh *BlockHeight) Increase() {
	bh.BlockHeight++
}

// Decrease this block height by 1,
// causing a panic in case the block height is 0 already.
func (bh *BlockHeight) Decrease() {
	if bh.BlockHeight == 0 {
		panic("cannot decrease block height 0")
	}
	bh.BlockHeight--
}

// LockValue returns this block height as a LockValue.
func (bh BlockHeight) LockValue() LockValue {
	return LockValue(bh.BlockHeight)
}

// AsCurrency returns a Rivine-typed Currency into
// the Currency overwritten type used in this project.
func AsCurrency(c types.Currency) Currency {
	return Currency{Currency: c}
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (c Currency) MarshalJSON() ([]byte, error) {
	return c.Currency.MarshalJSON()
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (c *Currency) UnmarshalJSON(data []byte) error {
	return c.Currency.UnmarshalJSON(data)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (c Currency) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteBytes(c.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write currency as string: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (c *Currency) DecodeMsg(r *msgp.Reader) error {
	b, err := r.ReadBytes(nil)
	if err != nil {
		return fmt.Errorf("failed to read currency as string: %v", err)
	}
	err = c.LoadBytes(b)
	if err != nil {
		return fmt.Errorf("failed to load currency-string as currency: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (c Currency) Msgsize() int {
	return stringerLength(c)
}

// Bytes returns this currency as a Big-Endian order byte slice.
func (c Currency) Bytes() []byte {
	return c.Big().Bytes()
}

// LoadBytes loads this Currency from a Big-Endian order byte slice.
func (c *Currency) LoadBytes(b []byte) error {
	if len(b) == 0 {
		c.Currency = types.ZeroCurrency
		return nil
	}
	c.Currency = types.NewCurrency(new(big.Int).SetBytes(b))
	return nil
}

// Add adds two currencies together, returning this Currency instance,
// containing the sum of the two currencies as value.
func (c *Currency) Add(o Currency) Currency {
	return AsCurrency(c.Currency.Add(o.Currency))
}

// Sub subtracts two currencies from one another, returning this Currency instance,
// containing the difference of the two currencies as value.
func (c *Currency) Sub(o Currency) Currency {
	return AsCurrency(c.Currency.Sub(o.Currency))
}

// Cmp compares two currencies, returning
//
//   -1 if this currency is less than the other
//    0 if the two currencies are equal
//    1 if this currency is greater than the other
func (c *Currency) Cmp(o Currency) int {
	return c.Currency.Cmp(o.Currency)
}

// AsUnlockHash returns a Rivine-typed UnlockHash into
// the UnlockHash overwritten type used in this project.
func AsUnlockHash(uh types.UnlockHash) UnlockHash {
	return UnlockHash{UnlockHash: uh}
}

// String implements fmt.Stringer.String
func (uh UnlockHash) String() string {
	if uh.Type == types.UnlockTypeNil {
		// rivine returns an empty string for the nil hash,
		// this is fine for its purposes, but in our case we know a nil-type unlock hash
		// MUST mean that it's actually meant as an UnlockHash and valid in the situation it appears,
		// we know because we get our data from the ConsensusSet,
		// hence we want to return it as an actual NilUnlockHash in string format.
		return strings.Repeat("0", 78)
	}
	return uh.UnlockHash.String()
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (uh UnlockHash) MarshalJSON() ([]byte, error) {
	return uh.UnlockHash.MarshalJSON()
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (uh *UnlockHash) UnmarshalJSON(data []byte) error {
	return uh.UnlockHash.UnmarshalJSON(data)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (uh UnlockHash) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteBytes(encoding.Marshal(uh))
	if err != nil {
		return fmt.Errorf("failed to write UnlockHash as string: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (uh *UnlockHash) DecodeMsg(r *msgp.Reader) error {
	b, err := r.ReadBytes(nil)
	if err != nil {
		return fmt.Errorf("failed to read UnlockHash as string: %v", err)
	}
	err = encoding.Unmarshal(b, uh)
	if err != nil {
		return fmt.Errorf("failed to load UnlockHash-string as UnlockHash: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (uh UnlockHash) Msgsize() int {
	return stringerLength(uh)
}

// AsCoinOutputID returns a Rivine-typed CoinOutputID into
// the CoinOutputID overwritten type used in this project.
func AsCoinOutputID(id types.CoinOutputID) CoinOutputID {
	return CoinOutputID{CoinOutputID: id}
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (id CoinOutputID) MarshalJSON() ([]byte, error) {
	return id.CoinOutputID.MarshalJSON()
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (id *CoinOutputID) UnmarshalJSON(data []byte) error {
	return id.CoinOutputID.UnmarshalJSON(data)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (id CoinOutputID) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteString(id.String())
	if err != nil {
		return fmt.Errorf("failed to write CoinOutputID as string: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (id *CoinOutputID) DecodeMsg(r *msgp.Reader) error {
	str, err := r.ReadString()
	if err != nil {
		return fmt.Errorf("failed to read CoinOutputID as string: %v", err)
	}
	err = id.LoadString(str)
	if err != nil {
		return fmt.Errorf("failed to load CoinOutputID-string as CoinOutputID: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (id CoinOutputID) Msgsize() int {
	return stringerLength(id)
}

// AsConsensusChangeID returns a Rivine-typed ConsensusChangeID into
// the ConsensusChangeID overwritten type used in this project.
func AsConsensusChangeID(id modules.ConsensusChangeID) ConsensusChangeID {
	return ConsensusChangeID{ConsensusChangeID: id}
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (id ConsensusChangeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.ConsensusChangeID)
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (id *ConsensusChangeID) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &id.ConsensusChangeID)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (id ConsensusChangeID) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteString(hex.EncodeToString(id.ConsensusChangeID[:]))
	if err != nil {
		return fmt.Errorf("failed to write ConsensusChangeID as a hex-encoded string: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (id *ConsensusChangeID) DecodeMsg(r *msgp.Reader) error {
	str, err := r.ReadString()
	if err != nil {
		return fmt.Errorf("failed to read ConsensusChangeID as a hex-encoded string: %v", err)
	}
	// *2 because there are 2 hex characters per byte.
	if len(str) != crypto.HashSize*2 {
		return fmt.Errorf("failed to decode ConsensusChangeID: %v", crypto.ErrHashWrongLen)
	}
	hBytes, err := hex.DecodeString(str)
	if err != nil {
		return fmt.Errorf("failed to decode ConsensusChangeID: could not unmarshal hash: %v", err)
	}
	copy(id.ConsensusChangeID[:], hBytes)
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (id ConsensusChangeID) Msgsize() int {
	return 66 // type byte (1), length byte (<256 = 1) and string length (64)
}

// stringerLength is a small utility function to return
// the MessagePack value length of a Stringer value
func stringerLength(stringer fmt.Stringer) int {
	switch l := len(stringer.String()); {
	case l <= 31:
		return 1 + l
	case l < 256:
		return 2 + l
	case l < 65536:
		return 3 + l
	case l < 4294967296:
		return 5 + l
	default:
		panic("unsupported msgp string length " + strconv.Itoa(l))
	}
}

// LockValue represents a LockValue,
// representing either a timestamp or a block height
type LockValue uint64

// String implements Stringer.String
func (lv LockValue) String() string {
	return strconv.FormatUint(uint64(lv), 10)
}

// LoadString implements StringLoader.LoadString
func (lv *LockValue) LoadString(str string) error {
	v, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*lv = LockValue(v)
	return nil
}

// MarshalSia implements rivine/encoding.MarshalSia
func (lv LockValue) MarshalSia(w io.Writer) error {
	return encoding.NewEncoder(w).Encode(uint64(lv))
}

// UnmarshalSia implements rivine/encoding.UnmarshalSia
func (lv *LockValue) UnmarshalSia(r io.Reader) error {
	var raw uint64
	err := encoding.NewDecoder(r).Decode(&raw)
	if err != nil {
		return err
	}
	*lv = LockValue(raw)
	return nil
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (lv LockValue) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteUint64(uint64(lv))
	if err != nil {
		return fmt.Errorf("failed to write LockValue as uint64: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (lv *LockValue) DecodeMsg(r *msgp.Reader) error {
	u, err := r.ReadUint64()
	if err != nil {
		return fmt.Errorf("failed to read LockValue as uint64: %v", err)
	}
	*lv = LockValue(u)
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (lv LockValue) Msgsize() int {
	return msgp.Uint64Size
}
