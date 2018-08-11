package main

import (
	"encoding/json"
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

	AddCoinOutput(id types.CoinOutputID, co types.CoinOutput) error
	AddLockedCoinOutput(id types.CoinOutputID, co types.CoinOutput, lt LockType, lockValue LockValue) error
	SpendCoinOutput(id types.CoinOutputID) error
	RevertCoinInput(id types.CoinOutputID) error
	RevertCoinOutput(id types.CoinOutputID) (oldState CoinOutputState, err error)

	ApplyCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error)
	RevertCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error)

	SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash) error
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
	// CoinOutput is used to store all spent/unspent coin outputs in the custom CSV format (used internally only)
	CoinOutput struct {
		UnlockHash types.UnlockHash
		CoinValue  types.Currency
		State      CoinOutputState
		LockType   LockType
		LockValue  LockValue
	}
	// CoinOutputLock is used to store the lock value and a reference to its parent CoinOutput,
	// as to store the lock in a scoped bucket.
	CoinOutputLock struct {
		CoinOutputID types.CoinOutputID
		LockValue    LockValue
	}
	// CoinOutputResult is returned by a Lua scripts which updates/marks a CoinOutput.
	CoinOutputResult struct {
		CoinOutputID types.CoinOutputID
		UnlockHash   types.UnlockHash
		CoinValue    types.Currency
		LockType     LockType
		LockValue    LockValue
	}
)

const csvSeperator = ","

// String implements Stringer.String
func (co CoinOutput) String() string {
	str := FormatStringers(csvSeperator, co.State, co.UnlockHash, co.CoinValue, co.LockType, co.LockValue)
	return str
}

// LoadString implements StringLoader.LoadString
func (co *CoinOutput) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &co.State, &co.UnlockHash, &co.CoinValue, &co.LockType, &co.LockValue)
}

// String implements Stringer.String
func (col CoinOutputLock) String() string {
	return FormatStringers(csvSeperator, col.CoinOutputID, col.LockValue)
}

// LoadString implements StringLoader.LoadString
func (col *CoinOutputLock) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &col.CoinOutputID, &col.LockValue)
}

// String implements Stringer.String
func (cor CoinOutputResult) String() string {
	return FormatStringers(csvSeperator, cor.CoinOutputID, cor.UnlockHash, cor.CoinValue, cor.LockType, cor.LockValue)
}

// LoadString implements StringLoader.LoadString
func (cor *CoinOutputResult) LoadString(str string) error {
	return ParseStringLoaders(str, csvSeperator, &cor.CoinOutputID, &cor.UnlockHash, &cor.CoinValue, &cor.LockType, &cor.LockValue)
}

const (
	internalKey          = "internal"
	internalFieldState   = "state"
	internalFieldNetwork = "network"

	statsKey = "stats"

	addressKeyPrefix                  = "address:"
	addressKeySuffixBalance           = "balance"
	addressKeySuffixLockedOutputs     = "outputs.locked"
	addressKeySuffixMultiSigAddresses = "multisig.addresses"

	addressesKey = "addresses"

	coinOutputsKey = "cos"

	lockedByHeightOutputsKey    = "lcos.height"
	lockedByTimestampOutputsKey = "lcos.time"
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
	// compute all keys and return the RedisDatabase instance
	rdb := RedisDatabase{
		conn: conn,
	}
	// create and load scripts
	err = rdb.createAndLoadScripts()
	if err != nil {
		return nil, fmt.Errorf("failed to create/load a lua script: %v", err)
	}
	return &rdb, nil
}

// internal logic to create and load scripts usd for advanced lua-script-driven logic
func (rdb *RedisDatabase) createAndLoadScripts() (err error) {
	rdb.coinOutputDropScript, err = rdb.createAndLoadScript(hashDropScriptSource, coinOutputsKey)
	if err != nil {
		return
	}

	rdb.unlockByTimeScript, err = rdb.createAndLoadScript(
		updateTimeLocksScriptSource,
		coinOutputsKey, CoinOutputStateLocked.String(), CoinOutputStateLiquid.String(), ">=")
	if err != nil {
		return
	}
	rdb.lockByTimeScript, err = rdb.createAndLoadScript(
		updateTimeLocksScriptSource,
		coinOutputsKey, CoinOutputStateLiquid.String(), CoinOutputStateLocked.String(), "<")
	if err != nil {
		return
	}

	rdb.unlockByHeightScript, err = rdb.createAndLoadScript(
		updateHeightLocksScriptSource,
		coinOutputsKey, CoinOutputStateLocked.String(), CoinOutputStateLiquid.String())
	if err != nil {
		return
	}
	rdb.lockByHeightScript, err = rdb.createAndLoadScript(
		updateHeightLocksScriptSource,
		coinOutputsKey, CoinOutputStateLiquid.String(), CoinOutputStateLocked.String())
	if err != nil {
		return
	}

	rdb.spendCoinOutputScript, err = rdb.createAndLoadScript(
		updateCoinOutputScriptSource,
		coinOutputsKey, CoinOutputStateLiquid.String(), CoinOutputStateSpent.String())
	if err != nil {
		return
	}
	rdb.unspendCoinOutputScript, err = rdb.createAndLoadScript(
		updateCoinOutputScriptSource,
		coinOutputsKey, CoinOutputStateSpent.String(), CoinOutputStateLiquid.String())
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
local value = redis.call("HGET", "%[1]s", coinOutputID)
redis.call("HDEL", "%[1]s", coinOutputID)
return value
`
	updateCoinOutputsSnippetSource = `
local results = {}
for i = 1 , #outputsToUpdate do
	local output = redis.call("HGET", "%[1]s", outputsToUpdate[i])
	if output:sub(1,1) == "%[2]s" then
		output = "%[3]s" .. output:sub(2)
		redis.call("HSET", "%[1]s", outputsToUpdate[i], output)
		results[#results+1] = outputsToUpdate[i] .. output:sub(2)
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
	if timenow %[4]s timelock then
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

local output = redis.call("HGET", "%[1]s", coinOutputID)
if output:sub(1,1) ~= "%[2]s" then
	return nil
end
output = "%[3]s" .. output:sub(2)
redis.call("HSET", "%[1]s", coinOutputID, output)
return coinOutputID .. output:sub(2)
`
)

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
	return RedisError(rdb.conn.Do("SET", statsKey, JSONMarshal(stats)))
}

// AddCoinOutput implements Database.AddCoinOutput
func (rdb *RedisDatabase) AddCoinOutput(id types.CoinOutputID, co types.CoinOutput) error {
	uh := co.Condition.UnlockHash()

	balanceKey := getAddressKey(uh, addressKeySuffixBalance)
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
	rdb.conn.Send("SADD", addressesKey, uh.String())
	// store output
	rdb.conn.Send("HSET", coinOutputsKey, id.String(), CoinOutput{
		UnlockHash: uh,
		CoinValue:  co.Value,
		State:      CoinOutputStateLiquid,
		LockType:   LockTypeNone,
		LockValue:  0,
	}.String())
	rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 3))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// AddLockedCoinOutput implements Database.AddLockedCoinOutput
func (rdb *RedisDatabase) AddLockedCoinOutput(id types.CoinOutputID, co types.CoinOutput, lt LockType, lockValue LockValue) error {
	uh := co.Condition.UnlockHash()

	balanceKey := getAddressKey(uh, addressKeySuffixBalance)
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
	rdb.conn.Send("SADD", addressesKey, uh.String())
	// store coinoutput in list of locked coins for wallet
	rdb.conn.Send("HSET", getAddressKey(uh, addressKeySuffixLockedOutputs), id.String(), JSONMarshal(co))
	// keep track of locked output
	switch lt {
	case LockTypeHeight:
		rdb.conn.Send("RPUSH", getLockHeightBucketKey(lockValue), id.String())
	case LockTypeTime:
		rdb.conn.Send("RPUSH", getLockTimeBucketKey(lockValue), CoinOutputLock{
			CoinOutputID: id,
			LockValue:    lockValue,
		}.String())
	}
	// store output
	rdb.conn.Send("HSET", coinOutputsKey, id.String(), CoinOutput{
		UnlockHash: uh,
		CoinValue:  co.Value,
		State:      CoinOutputStateLocked,
		LockType:   lt,
		LockValue:  lockValue,
	}.String())
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
	var result CoinOutputResult
	err := RedisStringLoader(&result)(rdb.spendCoinOutputScript.Do(rdb.conn, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to spend coin output: cannot update coin output %s in %s: %v",
			id.String(), coinOutputsKey, err)
	}

	// get balance, so it can be updated
	balanceKey := getAddressKey(result.UnlockHash, addressKeySuffixBalance)
	balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to spend coin output: failed to get balance for %s at %s: %v", result.UnlockHash.String(), balanceKey, err)
	}

	// update coin count
	balance.Unlocked = balance.Unlocked.Sub(result.CoinValue)

	// update balance
	err = RedisError(rdb.conn.Do("SET", balanceKey, JSONMarshal(balance)))
	if err != nil {
		return fmt.Errorf("redis: failed to spend coin output: failed to update coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// RevertCoinInput implements Database.RevertCoinInput
// more or less a reverse process of SpendCoinOutput
func (rdb *RedisDatabase) RevertCoinInput(id types.CoinOutputID) error {
	var result CoinOutputResult
	err := RedisStringLoader(&result)(rdb.unspendCoinOutputScript.Do(rdb.conn, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to revert coin input: cannot update coin output %s in %s: %v",
			id.String(), coinOutputsKey, err)
	}

	// get balance, so it can be updated
	balanceKey := getAddressKey(result.UnlockHash, addressKeySuffixBalance)
	balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to revert coin inpt: failed to get balance for %s at %s: %v", result.UnlockHash.String(), balanceKey, err)
	}

	// update coin count
	balance.Unlocked = balance.Unlocked.Add(result.CoinValue)

	// update balance
	err = RedisError(rdb.conn.Do("SET", balanceKey, JSONMarshal(balance)))
	if err != nil {
		return fmt.Errorf("redis: failed to revert coin input: failed to update coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// RevertCoinOutput implements Database.RevertCoinOutput
func (rdb *RedisDatabase) RevertCoinOutput(id types.CoinOutputID) (CoinOutputState, error) {
	var co CoinOutput
	err := RedisStringLoader(&co)(rdb.coinOutputDropScript.Do(rdb.conn, id.String()))
	if err != nil {
		return CoinOutputStateNil, fmt.Errorf(
			"redis: failed to revert coin output: cannot drop coin output %s in %s: %v",
			id.String(), coinOutputsKey, err)
	}
	if co.State == CoinOutputStateNil {
		return CoinOutputStateNil, fmt.Errorf(
			"redis: failed to revert coin output: nil coin output state %s in %s: %v",
			id.String(), coinOutputsKey, err)
	}

	var sendCount int
	if co.State != CoinOutputStateSpent {
		// update all data for this unspent coin output
		sendCount++

		// get balance, so it can be updated
		balanceKey := getAddressKey(co.UnlockHash, addressKeySuffixBalance)
		balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
		if err != nil {
			return CoinOutputStateNil, fmt.Errorf(
				"redis: failed to get balance for %s at %s: %v", co.UnlockHash.String(), balanceKey, err)
		}

		// update correct balanace
		switch co.State {
		case CoinOutputStateLiquid:
			// update unlocked balance of address wallet
			balance.Unlocked = balance.Unlocked.Sub(co.CoinValue)
		case CoinOutputStateLocked:
			// update locked balance of address wallet
			balance.Locked = balance.Locked.Sub(co.CoinValue)
		}

		// update balance
		rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
	}

	// always remove lock properties if a lock is used, no matter the state
	if co.LockType != LockTypeNone {
		sendCount += 2
		// remove locked coin output from the locked coin output list linked to the address
		rdb.conn.Send("HDEL", getAddressKey(co.UnlockHash, addressKeySuffixLockedOutputs), id.String())
		// remove locked coin output lock
		switch co.LockType {
		case LockTypeHeight:
			rdb.conn.Send("LREM", getLockHeightBucketKey(co.LockValue), 1, id.String())
		case LockTypeTime:
			rdb.conn.Send("LREM", getLockTimeBucketKey(co.LockValue), 1, CoinOutputLock{
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
		// get balance of user and update it
		balanceKey := getAddressKey(lcor.UnlockHash, addressKeySuffixBalance)
		// get initial values
		balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to get balance for %s at %s: %v", lcor.UnlockHash.String(), balanceKey, err)
		}
		balance.Locked = balance.Locked.Sub(lcor.CoinValue) // locked -> unlocked
		coins = coins.Add(lcor.CoinValue)
		n++
		balance.Unlocked = balance.Unlocked.Add(lcor.CoinValue)
		// update balance and pop unlocked output from address
		rdb.conn.Send("HDEL", getAddressKey(lcor.UnlockHash, addressKeySuffixLockedOutputs), lcor.CoinOutputID.String())
		rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
		err = RedisError(RedisFlushAndReceive(rdb.conn, 2))
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
		// get balance of user and update it
		balanceKey := getAddressKey(ulcor.UnlockHash, addressKeySuffixBalance)
		// get initial values
		balance, err := RedisAddressBalance(rdb.conn.Do("GET", balanceKey))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to get balance for %s at %s: %v", ulcor.UnlockHash.String(), balanceKey, err)
		}
		balance.Locked = balance.Locked.Add(ulcor.CoinValue) // unlocked -> locked
		coins = coins.Add(ulcor.CoinValue)
		n++
		balance.Unlocked = balance.Unlocked.Sub(ulcor.CoinValue)
		// update balance and pop unlocked output from address
		rdb.conn.Send("HDEL", getAddressKey(ulcor.UnlockHash, addressKeySuffixLockedOutputs), ulcor.CoinOutputID.String())
		rdb.conn.Send("SET", balanceKey, JSONMarshal(balance))
		err = RedisError(RedisFlushAndReceive(rdb.conn, 2))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"failed to update balance of %q and update locked coin outputs: %v",
				ulcor.UnlockHash.String(), err)
		}
	}
	return n, coins, nil
}

// SetMultisigAddresses implements Database.SetMultisigAddresses
func (rdb *RedisDatabase) SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash) error {
	for _, owner := range owners {
		rdb.conn.Send("SADD", getAddressKey(owner, addressKeySuffixMultiSigAddresses), address.String())
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
			"[ERROR] either all owners should have the multisig address %q linked or none, have %d/%d",
			address.String(), n, m)
	}
	// we'll assume that multisig address doesn't have the wallet created yet, if this happens
	for _, owner := range owners {
		rdb.conn.Send("SADD", getAddressKey(address, addressKeySuffixMultiSigAddresses), owner.String())
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
func getAddressKey(uh types.UnlockHash, suffix string) string {
	return addressKeyPrefix + uh.String() + ":" + suffix
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

// RedisCoinOutputResults returns all CoinOutputResults found for a given []string redis reply,
// only used in combination with `(*RedisDatabase).UpdateLockedCoinOutputs`, see that method for more information.
func RedisCoinOutputResults(reply interface{}, err error) ([]CoinOutputResult, error) {
	strings, err := redis.Strings(reply, err)
	if err != nil {
		return nil, err
	}
	results := make([]CoinOutputResult, len(strings))
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
