package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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

	AddCoinOutput(id types.CoinOutputID, co types.CoinOutput) error
	AddLockedCoinOutput(id types.CoinOutputID, co types.CoinOutput, lt LockType, lockValue uint64) error
	SpendCoinOutput(id types.CoinOutputID) error
	RevertCoinOutput(id types.CoinOutputID) error
	UpdateLockedCoinOutputs(height types.BlockHeight, time types.Timestamp) error

	SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash) error
}

// LockType represents the type of a lock, used to lock a (coin) output.
type LockType uint8

// The different types of locks used to lock (coin) outputs.
const (
	LockTypeNone LockType = iota
	LockTypeHeight
	LockTypeTime
)

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
	//	  <chainName>:<networkName>:ucos												(custom) all unspent coin outputs
	//	  <chainName>:<networkName>:lcos.height:<height>								(custom) all locked coin outputs on a given height
	//	  <chainName>:<networkName>:lcos.time:<timestamp[:-5]>							(custom) all locked coin outputs for a given timestmap range
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
	//        "timestamp": 1533670089,
	//        "blockHeight": 76824,
	//        "txCount": 77139,
	//        "valueTxCount": 316,
	//        "coinOutputCount": 1211,
	//        "coinInputCount": 355,
	//        "minerPayoutCount": 77062,
	//        "minerPayouts": "76855400000001",
	//        "coins": "695175855400000001"
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

		// All Lua scripts used by this redis client implementation, for advanced features.
		// Loaded when creating the client, and using the script's SHA1 (EVALSHA) afterwards.
		popAllScript            *redis.Script
		popIfNotLessThenScript  *redis.Script
		hashDropScript          *redis.Script
		updateLockTypeUCOScript *redis.Script

		// Most of the keys used by this Redis database.
		stateKey, statsKey          string
		addressKeyPrefix            string
		addressesKey                string
		unspentCoinOutputsKey       string
		lockedByHeightOutputsKey    string
		lockedByTimestampOutputsKey string
	}
)

var (
	_ Database = (*RedisDatabase)(nil)
)

// ColonSeperatedStringValue defines a small marshal/unmarshal interface,
// for marshalling and unmarshalling values as a colon-seperated value list,
// used as a custom format, specialized for making it easy
// to manipulate/extracting properties from within lua (scripts).
type ColonSeperatedStringValue interface {
	MarshalCSV() string
	UnmarshalCSV(string) error
}

type (
	// AddressBalance is used to store the locked/unlocked balance of a wallet (linked to an address)
	AddressBalance struct {
		Locked   types.Currency `json:"locked,omitempty"`
		Unlocked types.Currency `json:"unlocked,omitempty"`
	}
	// UnspentCoinOutput is used to store all unspent coin outputs in the custom CSV format (used internally only)
	UnspentCoinOutput struct {
		UnlockHash types.UnlockHash
		Value      types.Currency
		Lock       LockType
		LockValue  uint64
	}
	// LockedCoinOutputValue is used to index all locked outputs in the custom CSV format (used internally only),
	// making it easy to automatically unlock (previously locked) coin outputs.
	LockedCoinOutputValue struct {
		UnlockHash   types.UnlockHash
		CoinOutputID types.CoinOutputID
		LockValue    uint64
	}
)

// MarshalCSV implements ColonSeperatedStringValue.MarshalCSV
func (uco UnspentCoinOutput) MarshalCSV() string {
	return fmt.Sprintf("%d:%s:%s:%d", uco.Lock, uco.UnlockHash.String(), uco.Value.String(), uco.LockValue)
}

// UnmarshalCSV implements ColonSeperatedStringValue.UnmarshalCSV
//
// Would be easier if we could make use of fmt.Sscanf,
// but in Go the string value (%s) is greedy, and cannot made ungreedy as can be done in C.
func (uco *UnspentCoinOutput) UnmarshalCSV(s string) error {
	parts := strings.SplitN(s, ":", 4)
	if n := len(parts); n != 4 {
		return fmt.Errorf(
			"failed to parse CSV value %q from specified format: insufficient parts (%d, require 4)",
			s, n)
	}
	lockTypeUint, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return fmt.Errorf("failed to parse raw lock type %q: %v", parts[0], err)
	}
	uco.Lock = LockType(lockTypeUint)
	err = uco.UnlockHash.LoadString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to parse raw unlock hash %q: %v", parts[1], err)
	}
	i := big.NewInt(0)
	err = i.UnmarshalText([]byte(parts[2]))
	if err != nil {
		return fmt.Errorf("failed to parse raw (coin) value %q: %v", parts[2], err)
	}
	uco.Value = types.NewCurrency(i)
	uco.LockValue, err = strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse raw lock value %q: %v", parts[3], err)
	}
	return nil
}

// MarshalCSV implements ColonSeperatedStringValue.MarshalCSV
func (locv LockedCoinOutputValue) MarshalCSV() string {
	return fmt.Sprintf("%s:%s:%d",
		locv.UnlockHash.String(), locv.CoinOutputID.String(), locv.LockValue)
}

// UnmarshalCSV implements ColonSeperatedStringValue.UnmarshalCSV
//
// Would be easier if we could make use of fmt.Sscanf,
// but in Go the string value (%s) is greedy, and cannot made ungreedy as can be done in C.
func (locv *LockedCoinOutputValue) UnmarshalCSV(s string) error {
	parts := strings.SplitN(s, ":", 3)
	if n := len(parts); n != 3 {
		return fmt.Errorf(
			"failed to parse CSV value %q from specified format: insufficient parts (%d, require 3)",
			s, n)
	}
	err := locv.UnlockHash.LoadString(parts[0])
	if err != nil {
		return fmt.Errorf("failed to raw unlock hash %q: %v", parts[0], err)
	}
	err = locv.CoinOutputID.LoadString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to raw CoinOutputID %q: %v", parts[1], err)
	}
	locv.LockValue, err = strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse raw lock value %q: %v", parts[2], err)
	}
	return nil
}

const (
	addressKeySuffixBalance           = "balance"
	addressKeySuffixLockedOutputs     = "outputs.locked"
	addressKeySuffixMultiSigAddresses = "multisig.addresses"
)

const (
	// used to pop time-locked outputs
	luaScriptPopByTime = `
local key = ARGV[1]
local lockValue = tonumber(ARGV[2])

local values = {}
local length = tonumber(redis.call('LLEN', key))
for i = 1 , length do
	local str = redis.call('LPOP', key)
	local lockStr = string.sub(str, 145)
	if tonumber(lockStr) >= lockValue then
		values[#values+1] = str
	else
		redis.call('RPUSH', key, str)
	end
end
return values
`

	// used to pop height-locked outputs
	luaScriptPopAll = `
local key = ARGV[1]
local values = {}
local length = tonumber(redis.call('LLEN', key))
for i = 1 , length do
    local val = redis.call('LPOP', key)
    if val then
		values[#values+1] = val
    end
end
return values
`
	// used to implement the hash drop command
	luaScriptHashDrop = `
local key = ARGV[1]
local field = ARGV[2]
local value = redis.call("HGET", key, field)
redis.call("HDEL", key, field)
return value
`

	// used to update lock type of an unspent coin output in place
	luaScriptUpdateLockTypeUnspentCoinOutput = `
local key = ARGV[1]
local field = ARGV[2]
local lockValue = ARGV[3]
-- get value, modify in-place, set it again
local value = redis.call("HGET", key, field)
value = lockValue .. value:sub(2)
redis.call("HSET", key, field, value)
-- return the entire value, with modifications
return value
`
)

// NewRedisDatabase creates a new Redis Database client, used by the internal explorer module,
// see RedisDatabase for more information.
func NewRedisDatabase(address string, db int, bcInfo types.BlockchainInfo) (*RedisDatabase, error) {
	// dial a TCP connection
	conn, err := redis.Dial("tcp", address, redis.DialDatabase(db))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to dial a Redis connection to tcp://%s@%d: %v", address, db, err)
	}
	// load all scripts
	popAllScript := redis.NewScript(0, luaScriptPopAll)
	err = popAllScript.Load(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua-Script PopAll: %v", err)
	}
	popIfNotLessThenScript := redis.NewScript(0, luaScriptPopByTime)
	err = popIfNotLessThenScript.Load(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua-Script PopIfNotLessThen: %v", err)
	}
	hashDropScript := redis.NewScript(0, luaScriptHashDrop)
	err = hashDropScript.Load(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua-Script HashDrop: %v", err)
	}
	updateLockTypeUCOScript := redis.NewScript(0, luaScriptUpdateLockTypeUnspentCoinOutput)
	err = updateLockTypeUCOScript.Load(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua-Script UpdateLockTypeUnspentCoinOutput: %v", err)
	}
	// compute all keys and return the RedisDatabase instance
	prefix := fmt.Sprintf("%s:%s:",
		strings.ToLower(bcInfo.Name), strings.ToLower(bcInfo.NetworkName))
	return &RedisDatabase{
		conn:                        conn,
		popAllScript:                popAllScript,
		popIfNotLessThenScript:      popIfNotLessThenScript,
		hashDropScript:              hashDropScript,
		updateLockTypeUCOScript:     updateLockTypeUCOScript,
		stateKey:                    prefix + "state",
		statsKey:                    prefix + "stats",
		addressKeyPrefix:            prefix + "address:",
		addressesKey:                prefix + "addresses",
		unspentCoinOutputsKey:       prefix + "ucos",
		lockedByHeightOutputsKey:    prefix + "lcos.height",
		lockedByTimestampOutputsKey: prefix + "lcos.time",
	}, nil
}

// GetExplorerState implements Database.GetExplorerState
func (rdb *RedisDatabase) GetExplorerState() (ExplorerState, error) {
	var state ExplorerState
	switch err := RedisJSONValue(&state)(rdb.conn.Do("GET", rdb.stateKey)); err {
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
	return RedisError(rdb.conn.Do("SET", rdb.stateKey, JSONMarshal(state)))
}

// GetNetworkStats implements Database.GetNetworkStats
func (rdb *RedisDatabase) GetNetworkStats() (NetworkStats, error) {
	var stats NetworkStats
	switch err := RedisJSONValue(&stats)(rdb.conn.Do("GET", rdb.statsKey)); err {
	case nil:
		return stats, nil
	case redis.ErrNil:
		// default to fresh network stats if not stored yet
		return NewNetworkStats(), nil
	default:
		return NetworkStats{}, err
	}
}

// SetNetworkStats implements Database.SetNetworkStats
func (rdb *RedisDatabase) SetNetworkStats(stats NetworkStats) error {
	return RedisError(rdb.conn.Do("SET", rdb.statsKey, JSONMarshal(stats)))
}

// AddCoinOutput implements Database.AddCoinOutput
func (rdb *RedisDatabase) AddCoinOutput(id types.CoinOutputID, co types.CoinOutput) error {
	uh := co.Condition.UnlockHash()

	balanceKey := rdb.getAddressKey(uh, addressKeySuffixBalance)
	// get initial values
	balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get balance for %s at %s: %v", uh.String(), balance, err)
	}

	// increase coin count
	balance.Unlocked = balance.Unlocked.Add(co.Value)

	// set all values pipelined
	// store address, an address never gets deleted
	rdb.conn.Send("SADD", rdb.addressesKey, uh.String())
	// store output
	rdb.conn.Send("HSET", rdb.unspentCoinOutputsKey, id.String(), UnspentCoinOutput{
		UnlockHash: uh,
		Value:      co.Value,
		Lock:       LockTypeNone,
		LockValue:  0,
	}.MarshalCSV())
	rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 3))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// AddLockedCoinOutput implements Database.AddLockedCoinOutput
func (rdb *RedisDatabase) AddLockedCoinOutput(id types.CoinOutputID, co types.CoinOutput, lt LockType, lockValue uint64) error {
	uh := co.Condition.UnlockHash()

	balanceKey := rdb.getAddressKey(uh, addressKeySuffixBalance)
	// get initial values
	balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get balance for %s at %s: %v", uh.String(), balanceKey, err)
	}

	// increase coin count
	balance.Locked = balance.Locked.Add(co.Value)

	// set all values pipeline

	// store address, an address never gets deleted
	rdb.conn.Send("SADD", rdb.addressesKey, uh.String())
	// store coinoutput in list of locked coins for wallet
	rdb.conn.Send("HSET", rdb.getAddressKey(uh, addressKeySuffixLockedOutputs), id.String(), JSONMarshal(co))
	// keep track of locked output
	locv := LockedCoinOutputValue{
		UnlockHash:   uh,
		CoinOutputID: id,
		LockValue:    lockValue,
	}.MarshalCSV()
	switch lt {
	case LockTypeHeight:
		rdb.conn.Send("RPUSH", rdb.getLockHeightBucketKey(lockValue), locv)
	case LockTypeTime:
		rdb.conn.Send("RPUSH", rdb.getLockTimeBucketKey(lockValue), locv)
	}
	// store output
	rdb.conn.Send("HSET", rdb.unspentCoinOutputsKey, id.String(), UnspentCoinOutput{
		UnlockHash: uh,
		Value:      co.Value,
		Lock:       lt,
		LockValue:  lockValue,
	}.MarshalCSV())
	rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 5))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// SpendCoinOutput implements Database.SpendCoinOutput
func (rdb *RedisDatabase) SpendCoinOutput(id types.CoinOutputID) error {
	// drop+get coinOutput using coinOutputID
	var co UnspentCoinOutput
	err := RedisCSV(&co)(rdb.hashDropScript.Do(rdb.conn, rdb.unspentCoinOutputsKey, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to spend coin output: cannot drop coin output %s in %s: %v",
			id.String(), rdb.unspentCoinOutputsKey, err)
	}
	if co.Lock != LockTypeNone {
		return fmt.Errorf("trying to spend locked coin output %s: %s", id.String(), co.MarshalCSV())
	}

	// get balance, so it can be updated
	balanceKey := rdb.getAddressKey(co.UnlockHash, addressKeySuffixBalance)
	balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get balance for %s at %s: %v", co.UnlockHash.String(), balanceKey, err)
	}

	// update coin count
	balance.Unlocked = balance.Unlocked.Sub(co.Value)

	// update balance
	_, err = rdb.conn.Do("SET", balanceKey, JSONMarshal(balance))
	if err != nil {
		return fmt.Errorf("redis: failed to spend coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// RevertCoinOutput implements Database.RevertCoinOutput
func (rdb *RedisDatabase) RevertCoinOutput(id types.CoinOutputID) error {
	// drop+get coinOutput using coinOutputID
	var co UnspentCoinOutput
	err := RedisCSV(&co)(rdb.hashDropScript.Do(rdb.conn, rdb.unspentCoinOutputsKey, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to revert coin output: cannot drop coin output %s in %s: %v",
			id.String(), rdb.unspentCoinOutputsKey, err)
	}

	// get balance, so it can be updated
	balanceKey := rdb.getAddressKey(co.UnlockHash, addressKeySuffixBalance)
	balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get balance for %s at %s: %v", co.UnlockHash.String(), balanceKey, err)
	}

	// update all data for this coin output
	sendCount := 1
	if co.Lock == LockTypeNone {
		// update unlocked balance of address wallet
		balance.Unlocked = balance.Unlocked.Sub(co.Value)
	} else {
		sendCount += 2
		// update locked balance of address wallet
		balance.Locked = balance.Locked.Sub(co.Value)
		// remove locked coin output from the locked coin output list linked to the address
		rdb.conn.Send("HDEL", rdb.getAddressKey(co.UnlockHash, addressKeySuffixLockedOutputs), id.String())
		// remove locked coin output value (marker)
		lockMarkerValue := LockedCoinOutputValue{
			UnlockHash:   co.UnlockHash,
			CoinOutputID: id,
			LockValue:    co.LockValue,
		}.MarshalCSV()
		switch co.Lock {
		case LockTypeHeight:
			rdb.conn.Send("LREM", rdb.getLockHeightBucketKey(co.LockValue), 1, lockMarkerValue)
		case LockTypeTime:
			rdb.conn.Send("LREM", rdb.getLockTimeBucketKey(co.LockValue), 1, lockMarkerValue)
		}
	}
	// update balance
	rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, sendCount))
	if err != nil {
		return fmt.Errorf("redis: failed to revert coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// UpdateLockedCoinOutputs implements Database.UpdateLockedCoinOutputs
func (rdb *RedisDatabase) UpdateLockedCoinOutputs(height types.BlockHeight, time types.Timestamp) error {
	// pop all unlocked outputs which were previously locked
	rdb.popAllScript.SendHash(rdb.conn, rdb.getLockHeightBucketKey(uint64(height)))
	rdb.popIfNotLessThenScript.SendHash(rdb.conn, rdb.getLockTimeBucketKey(uint64(time)), uint64(time))
	values, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return fmt.Errorf("failed to pop locked outputs: %v", err)
	}
	lockedByHeightCoinOutputs, err := RedisLockedCoinOutputValues(values[0], nil)
	if err != nil && err != redis.ErrNil {
		return fmt.Errorf("failed to pop+parse locked-by-height outputs: %v", err)
	}
	lockedByTimeCoinOutputs, err := RedisLockedCoinOutputValues(values[1], nil)
	if err != nil && err != redis.ErrNil {
		return fmt.Errorf("failed to pop+parse locked-by-time outputs: %v", err)
	}
	lockedCoinOuts := append(lockedByHeightCoinOutputs, lockedByTimeCoinOutputs...)
	for _, lco := range lockedCoinOuts {
		// update and return unspent coin output
		var uco UnspentCoinOutput
		err = RedisCSV(&uco)(rdb.updateLockTypeUCOScript.Do(rdb.conn,
			rdb.unspentCoinOutputsKey, lco.CoinOutputID.String(), "0"))
		if err != nil {
			return fmt.Errorf("failed to parse returned CSV UnspentCoinOutput value for ID %s: %v", lco.CoinOutputID.String(), err)
		}
		// get balance of user and update it
		balanceKey := rdb.getAddressKey(lco.UnlockHash, addressKeySuffixBalance)
		// get initial values
		balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
		if err != nil {
			return fmt.Errorf(
				"redis: failed to get balance for %s at %s: %v", lco.UnlockHash.String(), balanceKey, err)
		}
		balance.Locked = balance.Locked.Sub(uco.Value) // locked -> unlocked
		balance.Unlocked = balance.Unlocked.Add(uco.Value)
		// update balance and pop unlocked output from address (assume we can pop in order)
		rdb.conn.Send("HDEL", rdb.getAddressKey(lco.UnlockHash, addressKeySuffixLockedOutputs), lco.CoinOutputID.String())
		rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
		err = RedisError(RedisFlushAndReceive(rdb.conn, 2))
		if err != nil {
			return fmt.Errorf(
				"failed to update balance of %q and pop the unlocked locked coin output: %v",
				lco.UnlockHash.String(), err)
		}
	}
	return nil
}

// SetMultisigAddresses implements Database.SetMultisigAddresses
func (rdb *RedisDatabase) SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash) error {
	for _, owner := range owners {
		rdb.conn.Send("SADD", rdb.getAddressKey(owner, addressKeySuffixMultiSigAddresses), address.String())
	}
	n, err := RedisSumInt64s(RedisFlushAndReceive(rdb.conn, len(owners)))
	if err != nil {
		return fmt.Errorf(
			"failed to link the multisig address %q to the owner's address: %v",
			address.String(), err)
	}
	if n == 0 {
		// we'll assume that if owners already tracked multisig, that we do not need to create the multisig wallet
		return nil
	}

	// track the owners within the multisig address namespace as well
	if m := int64(len(owners)); n != m {
		log.Printf(
			"[ERROR] either all owners should have the multisig address linked or none, have %d/%d",
			n, m)
	}
	// we'll assume that multisig address doesn't have the wallet created yet, if this happens
	for _, owner := range owners {
		rdb.conn.Send("SADD", rdb.getAddressKey(address, addressKeySuffixMultiSigAddresses), owner.String())
	}
	err = RedisError(RedisFlushAndReceive(rdb.conn, len(owners)))
	if err != nil {
		return fmt.Errorf(
			"failed to store all owner address for the multisig address %q: %v",
			address.String(), err)
	}
	return nil
}

// getAddressKey is an internal util function,
// used to create an address key using the suffix as property and the UnlockHash (uh) as identifier.
func (rdb *RedisDatabase) getAddressKey(uh types.UnlockHash, suffix string) string {
	return rdb.addressKeyPrefix + uh.String() + ":" + suffix
}

// getLockTimeBucketKey is an internal util function,
// used to create the timelocked bucket keys, grouping timelocked outputs within a given time range together.
func (rdb *RedisDatabase) getLockTimeBucketKey(lockValue uint64) string {
	str := strconv.FormatUint(lockValue, 10)
	return fmt.Sprintf("%s:%s", rdb.lockedByTimestampOutputsKey, str[:len(str)-5])
}

// getLockHeightBucketKey is an internal util function,
// used to create the heightlocked bucket keys, grouping all heightlocked outputs with the same lock-height value.
func (rdb *RedisDatabase) getLockHeightBucketKey(lockValue uint64) string {
	return fmt.Sprintf("%s:%d", rdb.lockedByHeightOutputsKey, lockValue)
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

// RedisCSV creates a function that can be used to unmarshal a string value
// as a (custom) CSV value into the given (reference) value (v).
func RedisCSV(v ColonSeperatedStringValue) func(interface{}, error) error {
	return func(reply interface{}, err error) error {
		s, err := redis.String(reply, err)
		if err != nil {
			return err
		}
		return v.UnmarshalCSV(s)
	}
}

// RedisAddressBalance unmarshals a JSON-encoded address (wallet) balance value,
// but creates a fresh (zero-currencied) balance if no balance was created yet for that wallet.
func RedisAddressBalance(r interface{}, e error) (balance AddressBalance, err error) {
	err = RedisJSONValue(&balance)(r, e)
	if err == redis.ErrNil {
		err = nil
		balance.Locked, balance.Unlocked = types.ZeroCurrency, types.ZeroCurrency
	}
	return
}

// RedisLockedCoinOutputValues returns all LockedCoinOutputValues found for a given []string redis reply,
// only used in combination with `(*RedisDatabase).UpdateLockedCoinOutputs`, see that method for more information.
func RedisLockedCoinOutputValues(reply interface{}, err error) ([]LockedCoinOutputValue, error) {
	lcos, err := redis.Strings(reply, err)
	if err != nil {
		return nil, err
	}
	values := make([]LockedCoinOutputValue, len(lcos))
	for i, lco := range lcos {
		err = values[i].UnmarshalCSV(lco)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
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
