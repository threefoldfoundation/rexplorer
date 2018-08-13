package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	rivinetypes "github.com/rivine/rivine/types"

	"github.com/gomodule/redigo/redis"
)

// Database represents the interface of a Database (client) as used by the Explorer module of this binary.
type Database interface {
	GetExplorerState() (ExplorerState, error)
	SetExplorerState(state ExplorerState) error

	GetNetworkStats() (types.NetworkStats, error)
	SetNetworkStats(stats types.NetworkStats) error

	AddCoinOutput(id types.CoinOutputID, co CoinOutput) error
	AddLockedCoinOutput(id types.CoinOutputID, co CoinOutput, lt LockType, lockValue types.LockValue) error
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
		Value       types.Currency
		Condition   rivinetypes.UnlockConditionProxy
		Description string
	}
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
	//	  internal																		(Redis Hashmap) used for internal state of this explorer
	//	  c:<4_random_hex_chars>														(custom) all coin outputs
	//	  lcos.height:<height>															(custom) all locked coin outputs on a given height
	//	  lcos.time:<timestamp-(timestamp%7200)>										(custom) all locked coin outputs for a given timestamp range
	//
	//	  public keys:
	//	  stats																			(JSON/MsgPack) used for global network statistics
	//	  addresses																		(SET) set of unique wallet addresses used (even if reverted) in the network
	//    a:<01|02|03><4_random_hex_chars>												(JSON/MsgPack) used by all contract and wallet addresses, storing all content of the wallet/contract
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
	//  example of a wallet (stored under a:01<4_random_hex_chars>)
	//    {
	//        "balance": {
	//            "unlocked": "10000000",
	//            "locked": {
	//                "total": "5000",
	//                "outputs": [
	//                    {
	//                        "amount": "2000",
	//                        "lockedUntil": 1534105468
	//                    },
	//                    {
	//                        "amount": "100",
	//                        "lockedUntil": 1534105468,
	//                        "description": "SGVsbG8=",
	//                    }
	//                ]
	//            }
	//        },
	//        "multisignaddresses": [
	//            "0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"
	//        ]
	//    }
	//
	//  example of a multisig wallet (stored under a:03<4_random_hex_chars>)
	//    {
	//        "balance": {
	//            "unlocked": "10000000"
	//        },
	//        "multisign": {
	//            "owners": [
	//                "01b650391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa",
	//                "0114df42a3bb8303a745d23c47062a1333246b3adac446e6d62f4de74f5223faf4c2da465e76af"
	//            ],
	//            "signaturesRequired": 1
	//        }
	//    }
	RedisDatabase struct {
		// The redis connection, no time out
		conn redis.Conn

		// used for the encoding/decoding of structured data
		encoder encoding.Encoder

		// cached chain constants
		blockFrequency types.LockValue

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
		LockValue   types.LockValue
		Description string
	}
	// DatabaseCoinOutputLock is used to store the lock value and a reference to its parent CoinOutput,
	// as to store the lock in a scoped bucket.
	DatabaseCoinOutputLock struct {
		CoinOutputID types.CoinOutputID
		LockValue    types.LockValue
	}
	// DatabaseCoinOutputResult is returned by a Lua scripts which updates/marks a CoinOutput.
	DatabaseCoinOutputResult struct {
		CoinOutputID types.CoinOutputID
		UnlockHash   types.UnlockHash
		CoinValue    types.Currency
		LockType     LockType
		LockValue    types.LockValue
		Description  string
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
	internalKey           = "internal"
	internalFieldState    = "state"
	internalFieldNetwork  = "network"
	internalFieldEncoding = "encoding"

	statsKey = "stats"

	addressesKey = "addresses"

	lockedByHeightOutputsKey    = "lcos.height"
	lockedByTimestampOutputsKey = "lcos.time"
)

// NewRedisDatabase creates a new Redis Database client, used by the internal explorer module,
// see RedisDatabase for more information.
func NewRedisDatabase(address string, db int, encodingType encoding.Type, bcInfo rivinetypes.BlockchainInfo, chainCts rivinetypes.ChainConstants) (*RedisDatabase, error) {
	// dial a TCP connection
	conn, err := redis.Dial("tcp", address, redis.DialDatabase(db))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to dial a Redis connection to tcp://%s@%d: %v", address, db, err)
	}
	// compute all keys and return the RedisDatabase instance
	rdb := RedisDatabase{
		conn:           conn,
		blockFrequency: types.LockValue(chainCts.BlockFrequency),
	}
	// ensure the encoding type is as expected (or register if this is a fresh db)
	err = rdb.registerOrValidateEncodingType(encodingType)
	if err != nil {
		return nil, fmt.Errorf("failed to create RedisDatabase client instance: %v", err)
	}
	// create our encoder, now that we know our encoding type is OK, as we'll need it from here on out
	rdb.encoder, err = encoding.NewEncoder(encodingType)
	if err != nil {
		return nil, fmt.Errorf("failed to create RedisDatabase client instance: %v", err)
	}
	// ensure the network info is as expected (or register if this is a fresh db)
	err = rdb.registerOrValidateNetworkInfo(bcInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RedisDatabase client instance: %v", err)
	}
	// create and load scripts
	err = rdb.createAndLoadScripts()
	if err != nil {
		return nil, fmt.Errorf("failed to create RedisDatabase client instance: failed to create/load a lua script: %v", err)
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

// registerOrValidateEncodingType registers the encoding type if it doesn't exist yet,
// otherwise it ensures that the returned encoding type matches the expected encoding type.
func (rdb *RedisDatabase) registerOrValidateEncodingType(encodingType encoding.Type) error {
	rdb.conn.Send("HSETNX", internalKey, internalFieldEncoding, encodingType.String())
	rdb.conn.Send("HGET", internalKey, internalFieldEncoding)
	replies, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return fmt.Errorf("failed to register/validate encoding type: %v", err)
	}
	if len(replies) != 2 {
		return errors.New("failed to register/validate encoding type: unexpected amount of replies received")
	}
	var receivedEncodingType encoding.Type
	err = RedisStringLoader(&receivedEncodingType)(replies[1], err)
	if err != nil {
		return fmt.Errorf("failed to validate encoding type: %v", err)
	}
	if receivedEncodingType != encodingType {
		return fmt.Errorf("cannot encode data using encoding type %s: db already uses encoding type %s",
			encodingType.String(), receivedEncodingType.String())
	}
	return nil
}

// registerOrValidateNetworkInfo registers the network name and chain name if it doesn't exist yet,
// otherwise it ensures that the returned network info matches the expected network info.
func (rdb *RedisDatabase) registerOrValidateNetworkInfo(bcInfo rivinetypes.BlockchainInfo) error {
	networkInfo := NetworkInfo{
		ChainName:   bcInfo.Name,
		NetworkName: bcInfo.NetworkName,
	}
	rdb.conn.Send("HSETNX", internalKey, internalFieldNetwork, rdb.marshalData(&networkInfo))
	rdb.conn.Send("HGET", internalKey, internalFieldNetwork)
	replies, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return fmt.Errorf("failed to register/validate network info: %v", err)
	}
	if len(replies) != 2 {
		return errors.New("failed to register/validate network info: unexpected amount of replies received")
	}
	var receivedNetworkInfo NetworkInfo
	err = rdb.redisStructuredValue(&receivedNetworkInfo)(replies[1], err)
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
	switch err := rdb.redisStructuredValue(&state)(rdb.conn.Do("HGET", internalKey, internalFieldState)); err {
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
	return RedisError(rdb.conn.Do("HSET", internalKey, internalFieldState, rdb.marshalData(&state)))
}

// GetNetworkStats implements Database.GetNetworkStats
func (rdb *RedisDatabase) GetNetworkStats() (types.NetworkStats, error) {
	var stats types.NetworkStats
	switch err := rdb.redisStructuredValue(&stats)(rdb.conn.Do("GET", statsKey)); err {
	case nil:
		rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
		return stats, nil
	case redis.ErrNil:
		// default to fresh network stats if not stored yet
		stats = types.NewNetworkStats()
		rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
		return stats, nil
	default:
		return types.NetworkStats{}, err
	}
}

// SetNetworkStats implements Database.SetNetworkStats
func (rdb *RedisDatabase) SetNetworkStats(stats types.NetworkStats) error {
	err := RedisError(rdb.conn.Do("SET", statsKey, rdb.marshalData(&stats)))
	if err != nil {
		return err
	}
	rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
	return nil
}

// AddCoinOutput implements Database.AddCoinOutput
func (rdb *RedisDatabase) AddCoinOutput(id types.CoinOutputID, co CoinOutput) error {
	uh := types.AsUnlockHash(co.Condition.UnlockHash())

	addressKey, addressField := getAddressKeyAndField(uh)
	// get initial values
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
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
	rdb.conn.Send("HSET", addressKey, addressField, rdb.marshalData(&wallet))
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 3))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// AddLockedCoinOutput implements Database.AddLockedCoinOutput
func (rdb *RedisDatabase) AddLockedCoinOutput(id types.CoinOutputID, co CoinOutput, lt LockType, lockValue types.LockValue) error {
	uh := types.AsUnlockHash(co.Condition.UnlockHash())

	addressKey, addressField := getAddressKeyAndField(uh)
	// get initial values
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", uh.String(), addressKey, addressField, err)
	}

	err = wallet.Balance.Locked.AddLockedCoinOutput(id, types.WalletLockedOutput{
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
	rdb.conn.Send("HSET", addressKey, addressField, rdb.marshalData(&wallet))
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
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", result.UnlockHash.String(), addressKey, addressField, err)
	}

	// update unlocked coins
	wallet.Balance.Unlocked = wallet.Balance.Unlocked.Sub(result.CoinValue)

	// update balance
	err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
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
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", result.UnlockHash.String(), addressKey, addressField, err)
	}

	// update coin count
	wallet.Balance.Unlocked = wallet.Balance.Unlocked.Add(result.CoinValue)

	// update balance
	err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
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
		wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
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
			// update locked output map and balance of address wallet
			err = wallet.Balance.Locked.SubLockedCoinOutput(id)
			if err != nil {
				return CoinOutputStateNil, fmt.Errorf(
					"redis: failed to revert coin output %s: %v",
					id.String(), err)
			}
		}

		// update balance
		rdb.conn.Send("HSET", addressKey, addressField, rdb.marshalData(&wallet))
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
	rdb.unlockByHeightScript.SendHash(rdb.conn, getLockHeightBucketKey(types.LockValue(height.BlockHeight)))
	rdb.unlockByTimeScript.SendHash(rdb.conn, getLockTimeBucketKey(types.LockValue(time.Timestamp)), types.LockValue(time.Timestamp).String())
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
		wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
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
		err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
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
	rdb.lockByHeightScript.SendHash(rdb.conn, getLockHeightBucketKey(height.LockValue()))
	rdb.lockByTimeScript.SendHash(rdb.conn, getLockTimeBucketKey(time.LockValue()), time.LockValue().String())
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
		wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to get wallet for %s at %s#%s: %v", ulcor.UnlockHash.String(), addressKey, addressField, err)
		}

		// unlocked -> locked
		err = wallet.Balance.Locked.AddLockedCoinOutput(ulcor.CoinOutputID, types.WalletLockedOutput{
			Amount:      ulcor.CoinValue,
			LockedUntil: rdb.lockValueAsLockTime(ulcor.LockType, ulcor.LockValue),
			Description: ulcor.Description,
		})
		if err != nil {
			return 0, types.Currency{}, fmt.Errorf(
				"redis: failed to add locked coin output%s for %s: %v",
				ulcor.UnlockHash.String(), ulcor.CoinOutputID.String(), err)
		}
		coins = coins.Add(ulcor.CoinValue)
		n++
		wallet.Balance.Unlocked = wallet.Balance.Unlocked.Sub(ulcor.CoinValue)
		// update balance
		err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
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
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get multisig wallet for %s at %s#%s: %v", address.String(), addressKey, addressField, err)
	}
	if len(wallet.MultiSignData.Owners) > 0 {
		return nil // nothing to do
	}
	// add owners and signatures required
	wallet.MultiSignData.SignaturesRequired = signaturesRequired
	wallet.MultiSignData.Owners = make([]types.UnlockHash, len(owners))
	copy(wallet.MultiSignData.Owners[:], owners[:])
	err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
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
		wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
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
		err = RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
		if err != nil {
			return fmt.Errorf(
				"redis: failed to set wallet for %s at %s#%s: %v", address.String(), addressKey, addressField, err)
		}
	}
	return nil
}

func (rdb *RedisDatabase) lockValueAsLockTime(lt LockType, value types.LockValue) types.LockValue {
	switch lt {
	case LockTypeTime:
		return value
	case LockTypeHeight:
		return rdb.networkTime.LockValue() + (value-rdb.networkBlockHeight.LockValue())*rdb.blockFrequency
	default:
		panic(fmt.Sprintf("invalid lock type %d", lt))
	}
}

func (rdb *RedisDatabase) marshalData(v interface{}) []byte {
	b, err := rdb.encoder.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// redisStructuredValue creates a function that can be used to unmarshal a byte-slice
// as a structured value into the given (reference) value (v).
func (rdb *RedisDatabase) redisStructuredValue(v interface{}) func(interface{}, error) error {
	return func(reply interface{}, err error) error {
		b, err := redis.Bytes(reply, err)
		if err != nil {
			return err
		}
		return rdb.encoder.Unmarshal(b, v)
	}
}

// redisWallet unmarshals an encoded address (wallet) value,
// but creates a fresh wallet if no wallet was created yet for that address.
func (rdb *RedisDatabase) redisWallet(r interface{}, e error) (wallet types.Wallet, err error) {
	err = rdb.redisStructuredValue(&wallet)(r, e)
	if err == redis.ErrNil {
		err = nil
		wallet = types.Wallet{}
	}
	return
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
func getLockTimeBucketKey(lockValue types.LockValue) string {
	return lockedByTimestampOutputsKey + ":" + (lockValue - lockValue%7200).String()
}

// getLockHeightBucketKey is an internal util function,
// used to create the heightlocked bucket keys, grouping all heightlocked outputs with the same lock-height value.
func getLockHeightBucketKey(lockValue types.LockValue) string {
	return lockedByHeightOutputsKey + ":" + lockValue.String()
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
