package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/rivine/rivine/types"

	"github.com/gomodule/redigo/redis"
)

// Database represents the interface of a Database (client) as used by the Explorer module of this binary.
type Database interface {
	GetExplorerState() (ExplorerState, error)
	SetExplorerState(state ExplorerState) error

	GetNetworkStats() (NetworkStats, error)
	SetNetworkStats(stats NetworkStats) error

	AddCoinOutput(id types.CoinOutputID, co CoinOutput) error
	AddLockedCoinOutput(id types.CoinOutputID, co CoinOutput, lt LockType, lockValue LockValue) error
	SpendCoinOutput(id types.CoinOutputID) error
	RevertCoinInput(id types.CoinOutputID) error
	RevertCoinOutput(id types.CoinOutputID) (oldState CoinOutputState, err error)

	ApplyCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error)
	RevertCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error)

	SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash, signaturesRequired uint64) error

	Close() error
}

// public function parameter data structures
type (
	// CoinOutput redefines a regular Rivine CoinOutput, adding a description field to it.
	// The description field is usually taken directly from the ArbitraryData field,
	// it is however hardcoded for tx fees and block creator rewards.
	CoinOutput struct {
		Value       types.Currency             `json:"value"`
		Condition   types.UnlockConditionProxy `json:"condition"`
		Description ByteSlice                  `json:"description"`
	}
)

// internal data structures
type (
	// NetworkInfo defines the info of the chain network data is dumped from,
	// used as to prevent name colissions.
	NetworkInfo struct {
		ChainName   string `json:"chainName"`
		NetworkName string `json:"networkName"`
	}
)

// public data structures
type (
	// Wallet collects all data for an address in a simple format,
	// focussing on its balance and multisign properties.
	Wallet struct {
		// Balance is optional and defines the balance the wallet currently has.
		Balance WalletBalance `json:"balance"`
		// MultiSignAddresses is optional and is only defined if the wallet is part of
		// one or multiple multisign wallets.
		MultiSignAddresses []types.UnlockHash `json:"multisignaddresses"`
		// MultiSignData is optional and is only defined if the wallet is a multisign wallet.
		MultiSignData WalletMultiSignData `json:"multisign"`
	}
	// WalletBalance contains the unlocked and/or locked balance of a wallet.
	WalletBalance struct {
		Unlocked types.Currency      `json:"unlocked,omitemtpy"`
		Locked   WalletLockedBalance `json:"locked,omitemtpy"`
	}
	// WalletLockedBalance contains the locked balance of a wallet,
	// defining the total amount of coins as well as all the outputs that are locked.
	WalletLockedBalance struct {
		Total   types.Currency        `json:"total"`
		Outputs WalletLockedOutputMap `json:"outputs"`
	}
	// WalletLockedOutputMap defines the mapping between a coin output ID and its walletLockedOutput data
	WalletLockedOutputMap map[types.CoinOutputID]WalletLockedOutput
	// WalletLockedOutput defines a locked output targetted at a wallet.
	WalletLockedOutput struct {
		Amount      types.Currency `json:"amount"`
		LockedUntil LockValue      `json:"lockedUntil"`
		Description []byte         `json:"description,omitemtpy"`
	}
	// WalletMultiSignData defines the extra data defined for a MultiSignWallet.
	WalletMultiSignData struct {
		Owners             []types.UnlockHash `json:"owners"`
		SignaturesRequired uint64             `json:"signaturesRequired"`
	}
)

// Specialised Wallet Structures to prevent the decoding of data which isn't required
type (
	// WalletFocusBalance decodes only the balance property
	//
	// See Wallet for more information about all properties.
	WalletFocusBalance struct {
		Balance            WalletBalance   `json:"balance"`
		MultiSignAddresses json.RawMessage `json:"multisignaddresses,omitemtpy"`
		MultiSignData      json.RawMessage `json:"multisign,omitemtpy"`
	}
	// WalletFocusUnlockedBalance decodes only the unlocked balance property
	//
	// See Wallet for more information about all properties.
	WalletFocusUnlockedBalance struct {
		Balance            WalletBalanceFocusUnlocked `json:"balance"`
		MultiSignAddresses json.RawMessage            `json:"multisignaddresses,omitemtpy"`
		MultiSignData      json.RawMessage            `json:"multisign,omitemtpy"`
	}
	// WalletBalanceFocusUnlocked decodes only the unlocked property
	//
	// See WalletBalance for more information about all properties.
	WalletBalanceFocusUnlocked struct {
		Unlocked types.Currency  `json:"unlocked"`
		Locked   json.RawMessage `json:"locked,omitemtpy"`
	}
	// WalletFocusMultiSignAddresses decodes only the MultiSignAddresses property
	//
	// See Wallet for more information  about all properties.
	WalletFocusMultiSignAddresses struct {
		Balance            json.RawMessage    `json:"balance,omitemtpy"`
		MultiSignAddresses []types.UnlockHash `json:"multisignaddresses"`
		MultiSignData      json.RawMessage    `json:"multisign,omitemtpy"`
	}
	// WalletFocusMultiSignData decodes only the MultiSignData property
	//
	// See Wallet for more information  about all properties.
	WalletFocusMultiSignData struct {
		Balance            json.RawMessage     `json:"balance,omitemtpy"`
		MultiSignAddresses json.RawMessage     `json:"multisignaddresses,omitemtpy"`
		MultiSignData      WalletMultiSignData `json:"multisign"`
	}
)

// MarshalJSON implements json.Marshaller.MarshalJSON
func (w Wallet) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	if !w.Balance.IsZero() {
		b, err := json.Marshal(w.Balance)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal balance: %v", err)
		}
		m["balance"] = json.RawMessage(b)
	}
	if len(w.MultiSignAddresses) > 0 {
		b, err := json.Marshal(w.MultiSignAddresses)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal multisign addresses: %v", err)
		}
		m["multisignaddresses"] = json.RawMessage(b)
	}
	if len(w.MultiSignData.Owners) > 0 {
		b, err := json.Marshal(w.MultiSignData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal multisign data: %v", err)
		}
		m["multisign"] = json.RawMessage(b)
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (w WalletFocusBalance) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	b, err := json.Marshal(w.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal balance: %v", err)
	}
	m["balance"] = json.RawMessage(b)
	if len(w.MultiSignAddresses) > 0 {
		m["multisignaddresses"] = w.MultiSignAddresses
	}
	if len(w.MultiSignData) > 0 {
		m["multisign"] = w.MultiSignData
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (w WalletFocusUnlockedBalance) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	b, err := json.Marshal(w.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal balance: %v", err)
	}
	m["balance"] = json.RawMessage(b)
	if len(w.MultiSignAddresses) > 0 {
		m["multisignaddresses"] = w.MultiSignAddresses
	}
	if len(w.MultiSignData) > 0 {
		m["multisign"] = w.MultiSignData
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (wb WalletBalanceFocusUnlocked) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	b, err := json.Marshal(wb.Unlocked)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unlocked balance: %v", err)
	}
	m["unlocked"] = json.RawMessage(b)
	if len(wb.Locked) > 0 {
		m["locked"] = wb.Locked
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (w WalletFocusMultiSignAddresses) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	if len(w.Balance) > 0 {
		m["balance"] = w.Balance
	}
	if len(w.MultiSignAddresses) > 0 {
		b, err := json.Marshal(w.MultiSignAddresses)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal multisign addresses: %v", err)
		}
		m["multisignaddresses"] = json.RawMessage(b)
	}
	if len(w.MultiSignData) > 0 {
		m["multisign"] = w.MultiSignData
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (w WalletFocusMultiSignData) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	if len(w.Balance) > 0 {
		m["balance"] = w.Balance
	}
	if len(w.MultiSignAddresses) > 0 {
		m["multisignaddresses"] = w.MultiSignAddresses
	}
	if len(w.MultiSignData.Owners) > 0 {
		b, err := json.Marshal(w.MultiSignData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal multisign data: %v", err)
		}
		m["multisign"] = json.RawMessage(b)
	}
	return json.Marshal(m)
}

// AddUniqueMultisignAddress adds the given multisign address to the wallet's list of
// multisign addresses which reference this wallet's address.
// It only adds it however if the given multisign address is not known yet.
func (w *WalletFocusMultiSignAddresses) AddUniqueMultisignAddress(address types.UnlockHash) bool {
	for _, uh := range w.MultiSignAddresses {
		if uh.Cmp(address) == 0 {
			return false // nothing to do
		}
	}
	w.MultiSignAddresses = append(w.MultiSignAddresses, address)
	return true
}

// IsZero returns true if this wallet is Zero
func (wb *WalletBalance) IsZero() bool {
	return wb.Unlocked.IsZero() && wb.Locked.Total.IsZero()
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (wb WalletBalance) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage)
	if !wb.Unlocked.IsZero() {
		b, err := json.Marshal(wb.Unlocked)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal unlocked balance: %v", err)
		}
		m["unlocked"] = json.RawMessage(b)
	}
	if !wb.Locked.Total.IsZero() {
		b, err := json.Marshal(wb.Locked)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal locked balance and outputs: %v", err)
		}
		m["locked"] = json.RawMessage(b)
	}
	return json.Marshal(m)
}

// AddLockedCoinOutput adds the unique locked coin output to the wallet's map of locked outputs
// as well as adds the coin output's value to the total amount of locked coins registered for this wallet.
func (wlb *WalletLockedBalance) AddLockedCoinOutput(id types.CoinOutputID, co WalletLockedOutput) error {
	if len(wlb.Outputs) == 0 {
		wlb.Outputs = make(WalletLockedOutputMap)
	} else if _, exists := wlb.Outputs[id]; exists {
		return fmt.Errorf("trying to add existing locked coin output %s", id.String())
	}
	wlb.Outputs[id] = co
	wlb.Total = wlb.Total.Add(co.Amount)
	return nil
}

// SubLockedCoinOutput removes the unique existing locked coin output from the wallet's map of locked outputs,
// as well as subtract the coin output's value from the total amount of locked coins registered for this wallet.
func (wlb *WalletLockedBalance) SubLockedCoinOutput(id types.CoinOutputID) error {
	if len(wlb.Outputs) == 0 {
		return fmt.Errorf("trying to remove non-existing locked coin output %s", id.String())
	}
	co, exists := wlb.Outputs[id]
	if !exists {
		return fmt.Errorf("trying to remove non-existing locked coin output %s", id.String())
	}
	delete(wlb.Outputs, id)
	wlb.Total = wlb.Total.Sub(co.Amount)
	return nil
}

// MarshalJSON implements json.Marshaller.MarshalJSON
func (wlom WalletLockedOutputMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]WalletLockedOutput, len(wlom))
	for k, v := range wlom {
		m[k.String()] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaller.UnmarshalJSON
func (wlom *WalletLockedOutputMap) UnmarshalJSON(b []byte) error {
	var m map[string]WalletLockedOutput
	err := json.Unmarshal(b, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal raw WalletLockedOutputMap: %v", err)
	}
	*wlom = make(WalletLockedOutputMap, len(m))
	for k, v := range m {
		var id types.CoinOutputID
		err = id.LoadString(k)
		if err != nil {
			return fmt.Errorf("failed to locked output %s: %v",
				k, err)
		}
		(*wlom)[id] = v
	}
	return nil
}

// StringLoader loads a string and uses it as the (parsed) value.
type StringLoader interface {
	LoadString(string) error
}

// FormatStringers formats the given stringers into one string using the given seperator
func FormatStringers(seperator string, stringers ...fmt.Stringer) string {
	n := len(stringers)
	if n == 0 {
		return ""
	}
	ss := make([]string, n)
	for i, stringer := range stringers {
		ss[i] = stringer.String()
	}
	return strings.Join(ss, seperator)
}

// ParseStringLoaders splits the given string into the given seperator
// and loads each part into a given string loader.
func ParseStringLoaders(csv, seperator string, stringLoaders ...StringLoader) (err error) {
	n := len(stringLoaders)
	parts := strings.SplitN(csv, seperator, n)
	if m := len(parts); n != m {
		return fmt.Errorf("CSV record has incorrect amount of records, expected %d but received %d", n, m)
	}
	for i, sl := range stringLoaders {
		err = sl.LoadString(parts[i])
		if err != nil {
			return
		}
	}
	return
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

// ByteSlice can be loaded from a base64-encoded string,
// and encodes to one as well when turned into a string.
type ByteSlice []byte

// String implements fmt.Stringer.String
func (bs ByteSlice) String() string {
	return base64.StdEncoding.EncodeToString([]byte(bs))
}

// LoadString implements StringLoader.LoadString
func (bs *ByteSlice) LoadString(str string) (err error) {
	*bs, err = base64.StdEncoding.DecodeString(str)
	return
}

type (
	// RedisDatabase is a Database (client) implementation for Redis, using github.com/gomodule/redigo.
	//
	// Note that the state as stored in this redis database will be corrupt when an error (due to a code bug) occurs.
	// This is however not a problem, as that bug should be fixed, and the redis database can be repopulated aftwards.
	// Because of this it is probably wise to reserve a Redis database (slot), used only by this explorer.
	//
	// Note as well that this implementation will break as soon as you have multiple clients writing to the database.
	// Many clients are allowed to read from the redis database, only this explorer module (and only as one instance) should write to
	// the redis database (slot) used. Multiple writers are NOT supported! You've been warned.
	//
	// Following key (templates) are reserved by this Redis database implementation:
	//
	//	  internal keys:
	//	  <chainName>:<networkName>:state												(JSON) used for internal state of this explorer, in JSON format
	//	  <chainName>:<networkName>:cos													(custom) all coin outputs
	//	  <chainName>:<networkName>:lcos.height:<height>								(custom) all locked coin outputs on a given height
	//	  <chainName>:<networkName>:lcos.time:<timestamp-(timestamp%7200)>				(custom) all locked coin outputs for a given timestmap range
	//
	//	  public keys:
	//	  <chainName>:<networkName>:stats												(JSON) used for global network statistics
	//	  <chainName>:<networkName>:addresses											(SET) set of unique wallet addresses used (even if reverted) in the network
	//    <chainName>:<networkName>:address:<unlockHashHex>:balance						(JSON) used by all wallet addresses
	//    <chainName>:<networkName>:address:<unlockHashHex>:outputs.locked				(mapping id->JSON(output))
	//																					used to store locked (by time or blockHeight) outputs destined for an address
	//    <chainName>:<networkName>:address:<unlockHashHex>:multisig.addresses			(SET) used in both directions for multisig (wallet) addresses
	//
	// Rivine Value Encodings:
	//	 + addresses are Hex-encoded and the exact format (and how it is created) is described in:
	//     https://github.com/rivine/rivine/blob/master/doc/transactions/unlockhash.md#textstring-encoding
	//   + currencies are encoded as described in https://godoc.org/math/big#Int.Text
	//     using base 10, and using the smallest coin unit as value (e.g. 10^-9 TFT)
	//   + coin outputs are stored in the Rivine-defined JSON format, described in:
	//     https://github.com/rivine/rivine/blob/master/doc/transactions/transaction.md#json-encoding-of-outputs-in-v0-transactions (v0 tx) and
	//     https://github.com/rivine/rivine/blob/master/doc/transactions/transaction.md#json-encoding-of-outputs-in-v1-transactions (v1 tx)
	//
	// JSON formats of value types defined by this module:
	//
	//  example of global stats (stored under <chainName>:<networkName>:stats):
	//    { // see: NetworkStats
	//    	"timestamp": 1533714154,
	//    	"blockHeight": 77185,
	//    	"txCount": 77501,
	//    	"valueTxCount": 317,
	//    	"coinOutputCount": 78637,
	//    	"lockedCoinOutputCount": 743,
	//    	"coinInputCount": 356,
	//    	"minerPayoutCount": 77424,
	//    	"minerPayouts": "77216500000001",
	//    	"coins": "695176216500000001",
	//    	"lockedCoins": "4899281850000000"
	//    }
	//
	//  example of wallet balance (stored under <chainName>:<networkName>:address:<unlockHashHex>:balance)
	//    { // see: AddressBalance
	//        "locked": "0",
	//        "unlocked": "250000000000"
	//    }
	RedisDatabase struct {
		// The redis connection, no time out
		conn redis.Conn

		blockFrequency LockValue

		// cached version of the chain stats
		networkBlockHeight types.BlockHeight
		networkTime        types.Timestamp

		// All Lua scripts used by this redis client implementation, for advanced features.
		// Loaded when creating the client, and using the script's SHA1 (EVALSHA) afterwards.
		coinOutputDropScript                           *redis.Script
		lockByTimeScript, unlockByTimeScript           *redis.Script
		lockByHeightScript, unlockByHeightScript       *redis.Script
		spendCoinOutputScript, unspendCoinOutputScript *redis.Script
	}
)

var (
	_ Database = (*RedisDatabase)(nil)
)

type (
	// AddressBalance is used to store the locked/unlocked balance of a wallet (linked to an address)
	AddressBalance struct {
		Locked   types.Currency `json:"locked,omitempty"`
		Unlocked types.Currency `json:"unlocked,omitempty"`
	}
	// DatabaseCoinOutput is used to store all spent/unspent coin outputs in the custom CSV format (used internally only)
	DatabaseCoinOutput struct {
		UnlockHash  types.UnlockHash
		CoinValue   types.Currency
		State       CoinOutputState
		LockType    LockType
		LockValue   LockValue
		Description ByteSlice
	}
	// DatabaseCoinOutputLock is used to store the lock value and a reference to its parent CoinOutput,
	// as to store the lock in a scoped bucket.
	DatabaseCoinOutputLock struct {
		CoinOutputID types.CoinOutputID
		LockValue    LockValue
	}
	// DatabaseCoinOutputResult is returned by a Lua scripts which updates/marks a CoinOutput.
	DatabaseCoinOutputResult struct {
		CoinOutputID types.CoinOutputID
		UnlockHash   types.UnlockHash
		CoinValue    types.Currency
		LockType     LockType
		LockValue    LockValue
		Description  ByteSlice
	}
)

const csvSeperator = ","

// String implements Stringer.String
func (co DatabaseCoinOutput) String() string {
	str := FormatStringers(csvSeperator, co.State, co.UnlockHash, co.CoinValue, co.LockType, co.LockValue, co.Description)
	return str
}

// LoadString implements StringLoader.LoadString
func (co *DatabaseCoinOutput) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &co.State, &co.UnlockHash, &co.CoinValue, &co.LockType, &co.LockValue, &co.Description)
}

// String implements Stringer.String
func (col DatabaseCoinOutputLock) String() string {
	return FormatStringers(csvSeperator, col.CoinOutputID, col.LockValue)
}

// LoadString implements StringLoader.LoadString
func (col *DatabaseCoinOutputLock) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &col.CoinOutputID, &col.LockValue)
}

// String implements Stringer.String
func (cor DatabaseCoinOutputResult) String() string {
	return FormatStringers(csvSeperator, cor.CoinOutputID, cor.UnlockHash, cor.CoinValue, cor.LockType, cor.LockValue, cor.Description)
}

// LoadString implements StringLoader.LoadString
func (cor *DatabaseCoinOutputResult) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &cor.CoinOutputID, &cor.UnlockHash, &cor.CoinValue, &cor.LockType, &cor.LockValue, &cor.Description)
}

const (
	internalKey          = "internal"
	internalFieldState   = "state"
	internalFieldNetwork = "network"

	statsKey = "stats"

	addressesKey = "addresses"

	lockedByHeightOutputsKey    = "lcos.height"
	lockedByTimestampOutputsKey = "lcos.time"
)

// NewRedisDatabase creates a new Redis Database client, used by the internal explorer module,
// see RedisDatabase for more information.
func NewRedisDatabase(address string, db int, bcInfo types.BlockchainInfo, chainCts types.ChainConstants) (*RedisDatabase, error) {
	// dial a TCP connection
	conn, err := redis.Dial("tcp", address, redis.DialDatabase(db))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to dial a Redis connection to tcp://%s@%d: %v", address, db, err)
	}
	// compute all keys and return the RedisDatabase instance
	rdb := RedisDatabase{
		conn:           conn,
		blockFrequency: LockValue(chainCts.BlockFrequency),
	}
	// ensure the network info is as expected (or register if this is a fresh db)
	err = rdb.registerOrValidateNetworkInfo(bcInfo)
	if err != nil {
		return nil, err
	}
	// create and load scripts
	err = rdb.createAndLoadScripts()
	if err != nil {
		return nil, fmt.Errorf("failed to create/load a lua script: %v", err)
	}
	return &rdb, nil
}

// Close implements Database.Close
//
// closes the internal redis db client connection
func (rdb *RedisDatabase) Close() error {
	err := rdb.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close redis db client connection: %v", err)
	}
	return nil
}

// internal logic to create and load scripts usd for advanced lua-script-driven logic
func (rdb *RedisDatabase) createAndLoadScripts() (err error) {
	rdb.coinOutputDropScript, err = rdb.createAndLoadScript(hashDropScriptSource)
	if err != nil {
		return
	}

	rdb.unlockByTimeScript, err = rdb.createAndLoadScript(
		updateTimeLocksScriptSource,
		CoinOutputStateLocked.String(), CoinOutputStateLiquid.String(), ">=")
	if err != nil {
		return
	}
	rdb.lockByTimeScript, err = rdb.createAndLoadScript(
		updateTimeLocksScriptSource,
		CoinOutputStateLiquid.String(), CoinOutputStateLocked.String(), "<")
	if err != nil {
		return
	}

	rdb.unlockByHeightScript, err = rdb.createAndLoadScript(
		updateHeightLocksScriptSource,
		CoinOutputStateLocked.String(), CoinOutputStateLiquid.String())
	if err != nil {
		return
	}
	rdb.lockByHeightScript, err = rdb.createAndLoadScript(
		updateHeightLocksScriptSource,
		CoinOutputStateLiquid.String(), CoinOutputStateLocked.String())
	if err != nil {
		return
	}

	rdb.spendCoinOutputScript, err = rdb.createAndLoadScript(
		updateCoinOutputScriptSource,
		CoinOutputStateLiquid.String(), CoinOutputStateSpent.String())
	if err != nil {
		return
	}
	rdb.unspendCoinOutputScript, err = rdb.createAndLoadScript(
		updateCoinOutputScriptSource,
		CoinOutputStateSpent.String(), CoinOutputStateLiquid.String())
	if err != nil {
		return
	}

	// all scripts loaded successfully
	return nil
}
func (rdb *RedisDatabase) createAndLoadScript(src string, argv ...interface{}) (*redis.Script, error) {
	src = fmt.Sprintf(src, argv...)
	script := redis.NewScript(0, src)
	err := script.Load(rdb.conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua-Script: %v", err)
	}
	return script, nil
}

const (
	hashDropScriptSource = `
local coinOutputID = ARGV[1]
local key = 'c:' .. coinOutputID:sub(1,4)
local field = coinOutputID:sub(5)
local value = redis.call("HGET", key, field)
redis.call("HDEL", key, field)
return value
`
	updateCoinOutputsSnippetSource = `
local results = {}
for i = 1 , #outputsToUpdate do
	local outputID = outputsToUpdate[i]
	local key = 'c:' .. outputID:sub(1,4)
	local field = outputID:sub(5)
	local output = redis.call("HGET", key, field)
	if output:sub(1,1) == "%[1]s" then
		output = "%[2]s" .. output:sub(2)
		redis.call("HSET", key, field, output)
		results[#results+1] = outputID .. output:sub(2)
	end
end
return results
`
	updateTimeLocksScriptSource = `
local bucketKey = ARGV[1]
local timenow = tonumber(ARGV[2])

local outputsToUpdate = {}
local bucketLength = tonumber(redis.call('LLEN', bucketKey))
for i = 1 , bucketLength do
	local str = redis.call('LINDEX', bucketKey, i-1)
	local timelock = tonumber(str:sub(66))
	if timenow %[3]s timelock then
		outputsToUpdate[#outputsToUpdate+1] = str:sub(1,64)
	end
end
` + updateCoinOutputsSnippetSource
	updateHeightLocksScriptSource = `
local bucketKey = ARGV[1]

local outputsToUpdate = {}
local bucketLength = tonumber(redis.call('LLEN', bucketKey))
for i = 1 , bucketLength do
	outputsToUpdate[#outputsToUpdate+1] = redis.call('LINDEX', bucketKey, i-1)
end
` + updateCoinOutputsSnippetSource
	updateCoinOutputScriptSource = `
local coinOutputID = ARGV[1]
local key = 'c:' .. coinOutputID:sub(1,4)
local field = coinOutputID:sub(5)

local output = redis.call("HGET", key, field)
if output:sub(1,1) ~= "%[1]s" then
	return nil
end
output = "%[2]s" .. output:sub(2)
redis.call("HSET", key, field, output)
return coinOutputID .. output:sub(2)
`
)

// registerOrValidateNetworkInfo registeres the network name and chain name if it doesn't exist yet,
// otherwise it ensures that the returned network info matches the expected network info.
func (rdb *RedisDatabase) registerOrValidateNetworkInfo(bcInfo types.BlockchainInfo) error {
	networkInfo := NetworkInfo{
		ChainName:   bcInfo.Name,
		NetworkName: bcInfo.NetworkName,
	}
	rdb.conn.Send("HSETNX", internalKey, internalFieldNetwork, JSONMarshal(networkInfo))
	rdb.conn.Send("HGET", internalKey, internalFieldNetwork)
	replies, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return fmt.Errorf("failed to register/validate network info: %v", err)
	}
	if len(replies) != 2 {
		return errors.New("failed to register/validate network info: unexpected amount of replies received")
	}
	var receivedNetworkInfo NetworkInfo
	err = RedisJSONValue(&receivedNetworkInfo)(replies[1], err)
	if err != nil {
		return fmt.Errorf("failed to validate network info: %v", err)
	}
	if receivedNetworkInfo != networkInfo {
		return fmt.Errorf("cannot store data for chain %s/%s: db has already data for chain %s/%s stored",
			networkInfo.ChainName, networkInfo.NetworkName,
			receivedNetworkInfo.ChainName, receivedNetworkInfo.NetworkName)
	}
	return nil
}

// GetExplorerState implements Database.GetExplorerState
func (rdb *RedisDatabase) GetExplorerState() (ExplorerState, error) {
	var state ExplorerState
	switch err := RedisJSONValue(&state)(rdb.conn.Do("HGET", internalKey, internalFieldState)); err {
	case nil:
		return state, nil
	case redis.ErrNil:
		// default to fresh explorer state if not stored yet
		return NewExplorerState(), nil
	default:
		return ExplorerState{}, err
	}
}

// SetExplorerState implements Database.SetExplorerState
func (rdb *RedisDatabase) SetExplorerState(state ExplorerState) error {
	return RedisError(rdb.conn.Do("HSET", internalKey, internalFieldState, JSONMarshal(state)))
}

// GetNetworkStats implements Database.GetNetworkStats
func (rdb *RedisDatabase) GetNetworkStats() (NetworkStats, error) {
	var stats NetworkStats
	switch err := RedisJSONValue(&stats)(rdb.conn.Do("GET", statsKey)); err {
	case nil:
		rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
		return stats, nil
	case redis.ErrNil:
		// default to fresh network stats if not stored yet
		stats = NewNetworkStats()
		rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
		return stats, nil
	default:
		return NetworkStats{}, err
	}
}

// SetNetworkStats implements Database.SetNetworkStats
func (rdb *RedisDatabase) SetNetworkStats(stats NetworkStats) error {
	err := RedisError(rdb.conn.Do("SET", statsKey, JSONMarshal(stats)))
	if err != nil {
		return err
	}
	rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
	return nil
}

// AddCoinOutput implements Database.AddCoinOutput
func (rdb *RedisDatabase) AddCoinOutput(id types.CoinOutputID, co CoinOutput) error {
	uh := co.Condition.UnlockHash()

	addressKey, addressField := getAddressKeyAndField(uh)
	// get initial values
	wallet, err := RedisWalletFocusUnlockedBalance(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", uh.String(), addressKey, addressField, err)
	}

	// increase coin count
	wallet.Balance.Unlocked = wallet.Balance.Unlocked.Add(co.Value)

	coinOutputKey, coinOutputField := getCoinOutputKeyAndField(id)

	// set all values pipelined
	// store address, an address never gets deleted
	rdb.conn.Send("SADD", addressesKey, uh.String())
	// store output
	rdb.conn.Send("HSET", coinOutputKey, coinOutputField, DatabaseCoinOutput{
		UnlockHash:  uh,
		CoinValue:   co.Value,
		State:       CoinOutputStateLiquid,
		LockType:    LockTypeNone,
		LockValue:   0,
		Description: co.Description,
	}.String())
	rdb.conn.Send("HSET", addressKey, addressField, JSONMarshal(wallet))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 3))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// AddLockedCoinOutput implements Database.AddLockedCoinOutput
func (rdb *RedisDatabase) AddLockedCoinOutput(id types.CoinOutputID, co CoinOutput, lt LockType, lockValue LockValue) error {
	uh := co.Condition.UnlockHash()

	addressKey, addressField := getAddressKeyAndField(uh)
	// get initial values
	wallet, err := RedisWalletFocusBalance(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", uh.String(), addressKey, addressField, err)
	}

	err = wallet.Balance.Locked.AddLockedCoinOutput(id, WalletLockedOutput{
		Amount:      co.Value,
		LockedUntil: rdb.lockValueAsLockTime(lt, lockValue),
		Description: co.Description,
	})
	if err != nil {
		return fmt.Errorf(
			"redis: failed to add locked coinoutput %s to wallet for %s: %v",
			id.String(), uh.String(), err)
	}

	// set all values pipeline

	// store address, an address never gets deleted
	rdb.conn.Send("SADD", addressesKey, uh.String())
	// store coinoutput in list of locked coins for wallet
	// keep track of locked output
	switch lt {
	case LockTypeHeight:
		rdb.conn.Send("RPUSH", getLockHeightBucketKey(lockValue), id.String())
	case LockTypeTime:
		rdb.conn.Send("RPUSH", getLockTimeBucketKey(lockValue), DatabaseCoinOutputLock{
			CoinOutputID: id,
			LockValue:    lockValue,
		}.String())
	}
	// store output
	coinOutputKey, coinOutputField := getCoinOutputKeyAndField(id)
	rdb.conn.Send("HSET", coinOutputKey, coinOutputField, DatabaseCoinOutput{
		UnlockHash:  uh,
		CoinValue:   co.Value,
		State:       CoinOutputStateLocked,
		LockType:    lt,
		LockValue:   lockValue,
		Description: co.Description,
	}.String())
	rdb.conn.Send("HSET", addressKey, addressField, JSONMarshal(wallet))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 4))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// SpendCoinOutput implements Database.SpendCoinOutput
func (rdb *RedisDatabase) SpendCoinOutput(id types.CoinOutputID) error {
	var result DatabaseCoinOutputResult
	err := RedisStringLoader(&result)(rdb.spendCoinOutputScript.Do(rdb.conn, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to spend coin output: cannot update coin output %s: %v",
			id.String(), err)
	}

	// get wallet, so its balance can be updated
	addressKey, addressField := getAddressKeyAndField(result.UnlockHash)
	wallet, err := RedisWalletFocusUnlockedBalance(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", result.UnlockHash.String(), addressKey, addressField, err)
	}

	// update unlocked coins
	wallet.Balance.Unlocked = wallet.Balance.Unlocked.Sub(result.CoinValue)

	// update balance
	err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, JSONMarshal(wallet)))
	if err != nil {
		return fmt.Errorf("redis: failed to spend coin output: failed to update coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// RevertCoinInput implements Database.RevertCoinInput
// more or less a reverse process of SpendCoinOutput
func (rdb *RedisDatabase) RevertCoinInput(id types.CoinOutputID) error {
	var result DatabaseCoinOutputResult
	err := RedisStringLoader(&result)(rdb.unspendCoinOutputScript.Do(rdb.conn, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to revert coin input: cannot update coin output %s: %v",
			id.String(), err)
	}

	// get wallet, so its balance can be updated
	addressKey, addressField := getAddressKeyAndField(result.UnlockHash)
	wallet, err := RedisWalletFocusUnlockedBalance(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", result.UnlockHash.String(), addressKey, addressField, err)
	}

	// update coin count
	wallet.Balance.Unlocked = wallet.Balance.Unlocked.Add(result.CoinValue)

	// update balance
	err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, JSONMarshal(wallet)))
	if err != nil {
		return fmt.Errorf("redis: failed to revert coin input: failed to update coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// RevertCoinOutput implements Database.RevertCoinOutput
func (rdb *RedisDatabase) RevertCoinOutput(id types.CoinOutputID) (CoinOutputState, error) {
	var co DatabaseCoinOutput
	err := RedisStringLoader(&co)(rdb.coinOutputDropScript.Do(rdb.conn, id.String()))
	if err != nil {
		return CoinOutputStateNil, fmt.Errorf(
			"redis: failed to revert coin output: cannot drop coin output %s: %v",
			id.String(), err)
	}
	if co.State == CoinOutputStateNil {
		return CoinOutputStateNil, fmt.Errorf(
			"redis: failed to revert coin output: nil coin output state %s",
			id.String())
	}

	var sendCount int
	if co.State != CoinOutputStateSpent {
		// update all data for this unspent coin output
		sendCount++

		// get wallet, so its balance can be updated
		addressKey, addressField := getAddressKeyAndField(co.UnlockHash)
		wallet, err := RedisWalletFocusBalance(rdb.conn.Do("HGET", addressKey, addressField))
		if err != nil {
			return CoinOutputStateNil, fmt.Errorf(
				"redis: failed to get wallet for %s at %s#%s: %v", co.UnlockHash.String(), addressKey, addressField, err)
		}

		// update correct balanace
		switch co.State {
		case CoinOutputStateLiquid:
			// update unlocked balance of address wallet
			wallet.Balance.Unlocked = wallet.Balance.Unlocked.Sub(co.CoinValue)
		case CoinOutputStateLocked:
			// update locked ouput map and balance of address wallet
			err = wallet.Balance.Locked.SubLockedCoinOutput(id)
			if err != nil {
				return CoinOutputStateNil, fmt.Errorf(
					"redis: failed to revert coin output %s: %v",
					id.String(), err)
			}
		}

		// update balance
		rdb.conn.Send("HSET", addressKey, addressField, JSONMarshal(wallet))
	}

	// always remove lock properties if a lock is used, no matter the state
	if co.LockType != LockTypeNone {
		sendCount++
		// remove locked coin output lock
		switch co.LockType {
		case LockTypeHeight:
			rdb.conn.Send("LREM", getLockHeightBucketKey(co.LockValue), 1, id.String())
		case LockTypeTime:
			rdb.conn.Send("LREM", getLockTimeBucketKey(co.LockValue), 1, DatabaseCoinOutputLock{
				CoinOutputID: id,
				LockValue:    co.LockValue,
			}.String())
		}
	}

	if sendCount > 0 {
		err = RedisError(RedisFlushAndReceive(rdb.conn, sendCount))
		if err != nil {
			return CoinOutputStateNil, fmt.Errorf(
				"redis: failed to revert coin output %s: failed to submit %d changes: %v",
				id.String(), sendCount, err)
		}
	}

	return co.State, nil
}

// ApplyCoinOutputLocks implements Database.ApplyCoinOutputLocks
func (rdb *RedisDatabase) ApplyCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error) {
	rdb.networkTime, rdb.networkBlockHeight = time, height
	rdb.unlockByHeightScript.SendHash(rdb.conn, getLockHeightBucketKey(LockValue(height)))
	rdb.unlockByTimeScript.SendHash(rdb.conn, getLockTimeBucketKey(LockValue(time)), LockValue(time).String())
	values, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to unlock outputs: %v", err)
	}
	lockedByHeightCoinOutputResults, err := RedisCoinOutputResults(values[0], nil)
	if err != nil && err != redis.ErrNil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-height output results: %v", err)
	}
	lockedByTimeCoinOutputResults, err := RedisCoinOutputResults(values[1], nil)
	if err != nil && err != redis.ErrNil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-time output results: %v", err)
	}
	lockedCoinOutputResults := append(lockedByHeightCoinOutputResults, lockedByTimeCoinOutputResults...)
	for _, lcor := range lockedCoinOutputResults {
		addressKey, addressField := getAddressKeyAndField(lcor.UnlockHash)
		// get initial values
		wallet, err := RedisWalletFocusBalance(rdb.conn.Do("HGET", addressKey, addressField))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to get wallet for %s at %s#%s: %v", lcor.UnlockHash.String(), addressKey, addressField, err)
		}

		// locked -> unlocked
		err = wallet.Balance.Locked.SubLockedCoinOutput(lcor.CoinOutputID)
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to unlock output %s in wallet for %s at %s#%s: %v",
				lcor.CoinOutputID.String(), lcor.UnlockHash.String(), addressKey, addressField, err)
		}
		coins = coins.Add(lcor.CoinValue)
		n++
		wallet.Balance.Unlocked = wallet.Balance.Unlocked.Add(lcor.CoinValue)
		// update balance
		err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, JSONMarshal(wallet)))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"failed to update balance of %q and update unlocked coin outputs: %v",
				lcor.UnlockHash.String(), err)
		}
	}
	return n, coins, nil
}

// RevertCoinOutputLocks implements Database.ApplyCoinOutputLocks
func (rdb *RedisDatabase) RevertCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error) {
	rdb.networkTime, rdb.networkBlockHeight = time, height
	rdb.lockByHeightScript.SendHash(rdb.conn, getLockHeightBucketKey(LockValue(height)))
	rdb.lockByTimeScript.SendHash(rdb.conn, getLockTimeBucketKey(LockValue(time)), LockValue(time).String())
	values, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to lock outputs: %v", err)
	}
	unlockedByHeightCoinOutputResults, err := RedisCoinOutputResults(values[0], nil)
	if err != nil && err != redis.ErrNil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-height output results: %v", err)
	}
	unlockedByTimeCoinOutputResults, err := RedisCoinOutputResults(values[1], nil)
	if err != nil && err != redis.ErrNil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-time output results: %v", err)
	}
	unlockedCoinOutputResults := append(unlockedByHeightCoinOutputResults, unlockedByTimeCoinOutputResults...)
	for _, ulcor := range unlockedCoinOutputResults {
		addressKey, addressField := getAddressKeyAndField(ulcor.UnlockHash)
		// get initial values
		wallet, err := RedisWalletFocusBalance(rdb.conn.Do("HGET", addressKey, addressField))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to get wallet for %s at %s#%s: %v", ulcor.UnlockHash.String(), addressKey, addressField, err)
		}

		// unlocked -> locked
		err = wallet.Balance.Locked.AddLockedCoinOutput(ulcor.CoinOutputID, WalletLockedOutput{
			Amount:      ulcor.CoinValue,
			LockedUntil: rdb.lockValueAsLockTime(ulcor.LockType, ulcor.LockValue),
			Description: ulcor.Description,
		})
		coins = coins.Add(ulcor.CoinValue)
		n++
		wallet.Balance.Unlocked = wallet.Balance.Unlocked.Sub(ulcor.CoinValue)
		// update balance
		err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, JSONMarshal(wallet)))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"failed to update balance of %q and update locked coin outputs: %v",
				ulcor.UnlockHash.String(), err)
		}
	}
	return n, coins, nil
}

// SetMultisigAddresses implements Database.SetMultisigAddresses
func (rdb *RedisDatabase) SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash, signaturesRequired uint64) error {
	// store multisig wallet first, as that will indicate if the owners (should) have the address or not
	addressKey, addressField := getAddressKeyAndField(address)
	// get initial values
	wallet, err := RedisWalletFocusMultiSignData(rdb.conn.Do("HGET", addressesKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get multisig wallet for %s at %s#%s: %v", address.String(), addressesKey, addressField, err)
	}
	if len(wallet.MultiSignData.Owners) > 0 {
		return nil // nothing to do
	}
	// add owners and signatures required
	wallet.MultiSignData.SignaturesRequired = signaturesRequired
	wallet.MultiSignData.Owners = make([]types.UnlockHash, len(owners))
	copy(wallet.MultiSignData.Owners[:], owners[:])
	err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, JSONMarshal(wallet)))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to set multisig wallet for %s at %s#%s: %v", address.String(), addressKey, addressField, err)
	}

	// now get the wallet for all owners, and set the multisig address,
	// this step is a bit expensive, as the entire wallet has to be loaded,
	// but luckily it only has to be done once per wallet appearance coin output
	for _, owner := range owners {
		// store multisig wallet first, as that will indicate if the owners (should) have the address or not
		addressKey, addressField := getAddressKeyAndField(owner)
		// get initial values
		wallet, err := RedisWalletFocusMultiSignAddresses(rdb.conn.Do("HGET", addressKey, addressField))
		if err != nil {
			return fmt.Errorf(
				"redis: failed to get wallet for %s at %s#%s: %v", owner.String(), addressKey, addressField, err)
		}
		// add multisig address
		if !wallet.AddUniqueMultisignAddress(address) {
			log.Printf("[ERROR] wallet %s already knows multisig wallet %s while it is expected to be new",
				owner.String(), address.String())
			continue
		}
		err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, JSONMarshal(wallet)))
		if err != nil {
			return fmt.Errorf(
				"redis: failed to set wallet for %s at %s#%s: %v", address.String(), addressKey, addressField, err)
		}
	}
	return nil
}

func (rdb *RedisDatabase) lockValueAsLockTime(lt LockType, value LockValue) LockValue {
	switch lt {
	case LockTypeTime:
		return value
	case LockTypeHeight:
		return LockValue(rdb.networkTime) + (value-LockValue(rdb.networkBlockHeight))*rdb.blockFrequency
	default:
		panic(fmt.Sprintf("invalid lock type %d", lt))
	}
}

func getAddressKeyAndField(uh types.UnlockHash) (key, field string) {
	str := uh.String()
	key, field = "a:"+str[:6], str[6:]
	return
}

func getCoinOutputKeyAndField(id types.CoinOutputID) (key, field string) {
	str := id.String()
	key, field = "c:"+str[:4], str[4:]
	return
}

// getLockTimeBucketKey is an internal util function,
// used to create the timelocked bucket keys, grouping timelocked outputs within a given time range together.
func getLockTimeBucketKey(lockValue LockValue) string {
	return lockedByTimestampOutputsKey + ":" + (lockValue - lockValue%7200).String()
}

// getLockHeightBucketKey is an internal util function,
// used to create the heightlocked bucket keys, grouping all heightlocked outputs with the same lock-height value.
func getLockHeightBucketKey(lockValue LockValue) string {
	return lockedByHeightOutputsKey + ":" + lockValue.String()
}

// JSON Helper Functions

// JSONMarshal marshals the given value and panics if that fails.
func JSONMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Redis Helper Functions

// RedisError ignores the reply from a Redis server,
// and returns only the returned error.
func RedisError(_ interface{}, err error) error {
	return err
}

// RedisSumInt64s interprets the reply as an int64 slice,
// and sums all those values into a single int64.
func RedisSumInt64s(reply interface{}, err error) (int64, error) {
	xs, err := redis.Int64s(reply, err)
	if err != nil {
		return 0, err
	}
	var sum int64
	for _, x := range xs {
		sum += x
	}
	return sum, nil
}

// RedisJSONValue creates a function that can be used to unmarshal a string/byte-slice
// as a JSON value into the given (reference) value (v).
func RedisJSONValue(v interface{}) func(interface{}, error) error {
	return func(reply interface{}, err error) error {
		b, err := redis.Bytes(reply, err)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, v)
	}
}

// RedisStringLoader creates a function that can be used to unmarshal a string value
// as a (custom) StringLoader value into the given (reference) value (v).
func RedisStringLoader(sl StringLoader) func(interface{}, error) error {
	return func(reply interface{}, err error) error {
		s, err := redis.String(reply, err)
		if err != nil {
			return err
		}
		return sl.LoadString(s)
	}
}

// RedisWallet unmarshals a JSON-encoded address (wallet) value,
// but creates a fresh wallet if no wallet was created yet for that address.
func RedisWallet(r interface{}, e error) (wallet Wallet, err error) {
	err = RedisJSONValue(&wallet)(r, e)
	if err == redis.ErrNil {
		err = nil
		wallet = Wallet{}
	}
	return
}

// RedisWalletFocusUnlockedBalance unmarshals a JSON-encoded address (wallet) value,
// but creates a fresh wallet if no wallet was created yet for that address.
func RedisWalletFocusUnlockedBalance(r interface{}, e error) (wallet WalletFocusUnlockedBalance, err error) {
	err = RedisJSONValue(&wallet)(r, e)
	if err == redis.ErrNil {
		err = nil
		wallet = WalletFocusUnlockedBalance{}
	}
	return
}

// RedisWalletFocusBalance unmarshals a JSON-encoded address (wallet) value,
// but creates a fresh wallet if no wallet was created yet for that address.
func RedisWalletFocusBalance(r interface{}, e error) (wallet WalletFocusBalance, err error) {
	err = RedisJSONValue(&wallet)(r, e)
	if err == redis.ErrNil {
		err = nil
		wallet = WalletFocusBalance{}
	}
	return
}

// RedisWalletFocusMultiSignAddresses unmarshals a JSON-encoded address (wallet) value,
// but creates a fresh wallet if no wallet was created yet for that address.
func RedisWalletFocusMultiSignAddresses(r interface{}, e error) (wallet WalletFocusMultiSignAddresses, err error) {
	err = RedisJSONValue(&wallet)(r, e)
	if err == redis.ErrNil {
		err = nil
		wallet = WalletFocusMultiSignAddresses{}
	}
	return
}

// RedisWalletFocusMultiSignData unmarshals a JSON-encoded address (wallet) value,
// but creates a fresh wallet if no wallet was created yet for that address.
func RedisWalletFocusMultiSignData(r interface{}, e error) (wallet WalletFocusMultiSignData, err error) {
	err = RedisJSONValue(&wallet)(r, e)
	if err == redis.ErrNil {
		err = nil
		wallet = WalletFocusMultiSignData{}
	}
	return
}

// RedisCoinOutputResults returns all CoinOutputResults found for a given []string redis reply,
// only used in combination with `(*RedisDatabase).UpdateLockedCoinOutputs`, see that method for more information.
func RedisCoinOutputResults(reply interface{}, err error) ([]DatabaseCoinOutputResult, error) {
	strings, err := redis.Strings(reply, err)
	if err != nil {
		return nil, err
	}
	results := make([]DatabaseCoinOutputResult, len(strings))
	for i, str := range strings {
		err = results[i].LoadString(str)
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

// RedisFlushAndReceive is used to flush all buffered commands (using SEND),
// and receiving all exepcted replies.
func RedisFlushAndReceive(conn redis.Conn, n int) (interface{}, error) {
	err := conn.Flush()
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, n)
	for i := 0; i < n; i++ {
		values[i], err = conn.Receive()
		if err != nil {
			return values, err
		}
	}
	return values, nil
}
