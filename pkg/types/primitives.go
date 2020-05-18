package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	threebottypes "github.com/threefoldfoundation/tfchain/extensions/threebot/types"
	erc20types "github.com/threefoldtech/rivine-extension-erc20/types"
	"github.com/threefoldtech/rivine/pkg/encoding/rivbin"

	"github.com/threefoldtech/rivine/crypto"
	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/pkg/encoding/siabin"
	"github.com/threefoldtech/rivine/types"

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
	// ERC20Address overwrites the TFChain ERC20Address type,
	// encapsulating it internally for practical reasons.
	ERC20Address struct {
		erc20types.ERC20Address
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
	binaryUnlockHash, err := siabin.Marshal(uh)
	if err != nil {
		return fmt.Errorf("failed to marshall UnlockHash as string: %v", err)
	}
	err = w.WriteBytes(binaryUnlockHash)
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
	err = siabin.Unmarshal(b, uh)
	if err != nil {
		return fmt.Errorf("failed to load UnlockHash-string as UnlockHash: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (uh UnlockHash) Msgsize() int {
	const uhs = 33 // type (1) + hash (32)
	return uhs + binaryMsgpPrefixSize(uhs)
}

// AsERC20Address returns a TFChain-typed ERC20Address into
// the ERC20Address overwritten type used in this project.
func AsERC20Address(uh erc20types.ERC20Address) ERC20Address {
	return ERC20Address{ERC20Address: uh}
}

// String implements fmt.Stringer.String
func (addr ERC20Address) String() string {
	return addr.ERC20Address.String()
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (addr ERC20Address) MarshalJSON() ([]byte, error) {
	return addr.ERC20Address.MarshalJSON()
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (addr ERC20Address) UnmarshalJSON(data []byte) error {
	return addr.ERC20Address.UnmarshalJSON(data)
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (addr ERC20Address) EncodeMsg(w *msgp.Writer) error {
	encodedERC20Address, err := rivbin.Marshal(addr)
	if err != nil {
		return fmt.Errorf("failed to encode ERC20Address as bytes: %v", err)
	}
	err = w.WriteBytes(encodedERC20Address)
	if err != nil {
		return fmt.Errorf("failed to write ERC20Address as bytes: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (addr *ERC20Address) DecodeMsg(r *msgp.Reader) error {
	b, err := r.ReadBytes(nil)
	if err != nil {
		return fmt.Errorf("failed to read ERC20Address as bytes: %v", err)
	}
	err = rivbin.Unmarshal(b, addr)
	if err != nil {
		return fmt.Errorf("failed to load ERC20Address-bytes as ERC20Address: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (addr ERC20Address) Msgsize() int {
	length := erc20types.ERC20AddressLength
	return length + binaryMsgpPrefixSize(length)
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

func binaryMsgpPrefixSize(length int) int {
	if length <= math.MaxUint8 {
		return 2
	}
	if length <= math.MaxUint16 {
		return 3
	}
	if length <= math.MaxUint32 {
		return 5
	}
	panic("unsupported msgp binary length: " + strconv.Itoa(length))
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

// MarshalSia implements rivine/siabin.MarshalSia
func (lv LockValue) MarshalSia(w io.Writer) error {
	return siabin.NewEncoder(w).Encode(uint64(lv))
}

// UnmarshalSia implements rivine/siabin.UnmarshalSia
func (lv *LockValue) UnmarshalSia(r io.Reader) error {
	var raw uint64
	err := siabin.NewDecoder(r).Decode(&raw)
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

// Type overwrites as to be able to define MessagePack (de)serialization methods for tfchain types.
type (
	// NetworkAddressSortedSet overwrites the Tfchain NetworkAddressSortedSet type,
	// encapsulating it internally for practical reasons.
	NetworkAddressSortedSet struct {
		threebottypes.NetworkAddressSortedSet
	}
	// CompactTimestamp overwrites the Tfchain CompactTimestamp type,
	// encapsulating it internally for practical reasons.
	CompactTimestamp struct {
		threebottypes.CompactTimestamp
	}
	// BotID overwrites the Tfchain BotID type,
	// encapsulating it internally for practical reasons.
	BotID struct {
		threebottypes.BotID
	}
	// BotNameSortedSet overwrites the Tfchain BotNameSortedSet type,
	// encapsulating it internally for practical reasons.
	BotNameSortedSet struct {
		threebottypes.BotNameSortedSet
	}
	// PublicKey overwrites the Tfchain PublicKey type,
	// encapsulating it internally for practical reasons.
	PublicKey struct {
		types.PublicKey
	}
)

// NewNetworkAddressSortedSetFromTfchainNetworkAddressSortedSet returns a tfchain-typed NetworkAddressSortedSet into
// the NetworkAddressSortedSet overwritten type used in this project.
func NewNetworkAddressSortedSetFromTfchainNetworkAddressSortedSet(nass threebottypes.NetworkAddressSortedSet) NetworkAddressSortedSet {
	return NetworkAddressSortedSet{NetworkAddressSortedSet: nass}
}

// TfchainNetworkAddressSortedSet returns the tfchain-typed NetworkAddressSortedSet, embedded by this type.
func (nass NetworkAddressSortedSet) TfchainNetworkAddressSortedSet() threebottypes.NetworkAddressSortedSet {
	return nass.NetworkAddressSortedSet
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (nass NetworkAddressSortedSet) EncodeMsg(w *msgp.Writer) error {
	encoded, err := rivbin.Marshal(nass.NetworkAddressSortedSet)
	if err != nil {
		return fmt.Errorf("failed to encode network address sorted set as bytes: %v", err)
	}
	err = w.WriteBytes(encoded)
	if err != nil {
		return fmt.Errorf("failed to write network address sorted set as bytes: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (nass *NetworkAddressSortedSet) DecodeMsg(r *msgp.Reader) error {
	b, err := r.ReadBytes(nil)
	if err != nil {
		return fmt.Errorf("failed to read network address sorted set as bytes: %v", err)
	}
	err = rivbin.Unmarshal(b, &nass.NetworkAddressSortedSet)
	if err != nil {
		return fmt.Errorf("failed to byte-decode network address sorted set: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (nass NetworkAddressSortedSet) Msgsize() int {
	l := nass.Len() * 63
	return l + binaryMsgpPrefixSize(l)
}

// NewCompactTimeStampFromTfchainCompactTimestamp returns a tfchain-typed CompactTimestamp into
// the CompactTimestamp overwritten type used in this project.
func NewCompactTimeStampFromTfchainCompactTimestamp(cts threebottypes.CompactTimestamp) CompactTimestamp {
	return CompactTimestamp{CompactTimestamp: cts}
}

// TfchainCompactTimestamp returns the tfchain-typed CompactTimestamp, embedded by this type.
func (cts CompactTimestamp) TfchainCompactTimestamp() threebottypes.CompactTimestamp {
	return cts.CompactTimestamp
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (cts CompactTimestamp) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteUint32(cts.UInt32())
	if err != nil {
		return fmt.Errorf("failed to write compact time stamp as uint32: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (cts *CompactTimestamp) DecodeMsg(r *msgp.Reader) error {
	x, err := r.ReadUint32()
	if err != nil {
		return fmt.Errorf("failed to read compact time stamp as uint32: %v", err)
	}
	cts.SetUInt32(x)
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (cts CompactTimestamp) Msgsize() int {
	return msgp.Uint32Size
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (cts CompactTimestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(cts.CompactTimestamp)
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (cts *CompactTimestamp) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &cts.CompactTimestamp)
}

// NewBotIDFromTfchainBotID returns a tfchain-typed BotID into
// the BotID overwritten type used in this project.
func NewBotIDFromTfchainBotID(id threebottypes.BotID) BotID {
	return BotID{BotID: id}
}

// TfchainBotID returns the tfchain-typed BotID, embedded by this type.
func (id BotID) TfchainBotID() threebottypes.BotID {
	return id.BotID
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (id BotID) EncodeMsg(w *msgp.Writer) error {
	err := w.WriteUint32(uint32(id.BotID))
	if err != nil {
		return fmt.Errorf("failed to write bot ID as uint32: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (id *BotID) DecodeMsg(r *msgp.Reader) error {
	u, err := r.ReadUint32()
	if err != nil {
		return fmt.Errorf("failed to read Bot ID as uint32: %v", err)
	}
	id.BotID = threebottypes.BotID(u)
	return nil
}

// MarshalJSON implements json.Marshaler.MarshalJSON
func (id BotID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.BotID)
}

// UnmarshalJSON implements json.Marshaler.UnmarshalJSON
func (id *BotID) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &id.BotID)
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (id BotID) Msgsize() int {
	return msgp.Uint32Size
}

// UInt32 returns this BotID as an uint32-typed value.
func (id BotID) UInt32() uint32 {
	return uint32(id.BotID)
}

// SetUInt32 sets the internal value of this BotID
// equal to the given uint32-typed value.
func (id *BotID) SetUInt32(x uint32) {
	id.BotID = threebottypes.BotID(x)
}

// Increment the BotID by one.
func (id *BotID) Increment() {
	id.BotID++
}

// Decrement the BotID by one.
func (id *BotID) Decrement() {
	id.BotID--
}

// NewBotNameSortedSetFromTfchainBotNameSortedSet returns a tfchain-typed BotNameSortedSet into
// the BotNameSortedSet overwritten type used in this project.
func NewBotNameSortedSetFromTfchainBotNameSortedSet(bnss threebottypes.BotNameSortedSet) BotNameSortedSet {
	return BotNameSortedSet{BotNameSortedSet: bnss}
}

// TfchainBotNameSortedSet returns the tfchain-typed BotNameSortedSet, embedded by this type.
func (bnss BotNameSortedSet) TfchainBotNameSortedSet() threebottypes.BotNameSortedSet {
	return bnss.BotNameSortedSet
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (bnss BotNameSortedSet) EncodeMsg(w *msgp.Writer) error {
	encoded, err := rivbin.Marshal(bnss.BotNameSortedSet)
	if err != nil {
		return fmt.Errorf("failed to emcode bot name sorted set as bytes: %v", err)
	}
	err = w.WriteBytes(encoded)
	if err != nil {
		return fmt.Errorf("failed to write bot name sorted set as bytes: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (bnss *BotNameSortedSet) DecodeMsg(r *msgp.Reader) error {
	b, err := r.ReadBytes(nil)
	if err != nil {
		return fmt.Errorf("failed to read bot name sorted set as bytes: %v", err)
	}
	err = rivbin.Unmarshal(b, &bnss.BotNameSortedSet)
	if err != nil {
		return fmt.Errorf("failed to byte-decode bot name sorted set: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (bnss BotNameSortedSet) Msgsize() int {
	l := bnss.Len() * 63
	return l + binaryMsgpPrefixSize(l)
}

// NewPublicKeyFromTfchainPublicKey returns a tfchain-typed PublicKey into
// the PublicKey overwritten type used in this project.
func NewPublicKeyFromTfchainPublicKey(pk types.PublicKey) PublicKey {
	return PublicKey{PublicKey: pk}
}

// TfchainPublicKey returns the tfchain-typed PublicKey, embedded by this type.
func (pk PublicKey) TfchainPublicKey() types.PublicKey {
	return pk.PublicKey
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
func (pk PublicKey) EncodeMsg(w *msgp.Writer) error {
	encoded, err := rivbin.Marshal(pk.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to encode public key as bytes: %v", err)
	}
	err = w.WriteBytes(encoded)
	if err != nil {
		return fmt.Errorf("failed to write public key as bytes: %v", err)
	}
	return nil
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
func (pk *PublicKey) DecodeMsg(r *msgp.Reader) error {
	b, err := r.ReadBytes(nil)
	if err != nil {
		return fmt.Errorf("failed to read public key as bytes: %v", err)
	}
	err = rivbin.Unmarshal(b, &pk.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to byte-decode public key: %v", err)
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (pk PublicKey) Msgsize() int {
	l := 1 // 1 byte for the algorithm prefix
	switch pk.Algorithm {
	case types.SignatureAlgoEd25519:
		l += crypto.PublicKeySize
	case types.SignatureAlgoNil:
		// add nothing
	default:
		l += 64 // should 32 bytes ever stop being sufficient, 64 is a likely choice
	}
	return l + binaryMsgpPrefixSize(l)
}
