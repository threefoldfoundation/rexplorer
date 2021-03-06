package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/threefoldfoundation/rexplorer/pkg/database"
	dtypes "github.com/threefoldfoundation/rexplorer/pkg/database/types"
	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	"github.com/threefoldtech/rivine/crypto"
	"github.com/threefoldtech/rivine/pkg/encoding/siabin"
	rivinetypes "github.com/threefoldtech/rivine/types"

	"github.com/gomodule/redigo/redis"
)

// Database represents the interface of a Database (client) as used by the Explorer module of this binary.
type Database interface {
	GetExplorerState() (dtypes.ExplorerState, error)
	SetExplorerState(state dtypes.ExplorerState) error

	GetNetworkStats() (types.NetworkStats, error)
	SetNetworkStats(stats types.NetworkStats) error

	AddCoinOutput(id types.CoinOutputID, co CoinOutput) error
	AddLockedCoinOutput(id types.CoinOutputID, co CoinOutput, lt dtypes.LockType, lockValue types.LockValue) error
	SpendCoinOutput(id types.CoinOutputID) error
	RevertCoinInput(id types.CoinOutputID) error
	RevertCoinOutput(id types.CoinOutputID) (oldState dtypes.CoinOutputState, err error)

	ApplyCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error)
	RevertCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error)

	SetMultisigAddresses(address types.UnlockHash, owners []types.UnlockHash, signaturesRequired uint64) error
	SetCoinCreators(creators []types.UnlockHash) error

	CreateBotRecord(record types.BotRecord) error
	UpdateBotRecord(id types.BotID, fn func(*types.BotRecord) error) error
	DeleteBotRecord(id types.BotID) error

	AddERC20AddressRegistration(erc20Address types.ERC20Address, tftAddress types.UnlockHash) error
	DeleteERC20AddressRegistration(erc20Address types.ERC20Address) error

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
	//	  stats																			(JSON/MsgPack/Proto) used for global network statistics
	//    coincreators																	(SET) set of unique wallet addresses of the coin creator(s)
	//	  addresses																		(SET) set of unique wallet addresses used (even if reverted) in the network
	//    a:<01|02|03><4_random_hex_chars>												(JSON/MsgPack/Proto) used by all contract and wallet addresses, storing all content of the wallet/contract
	//    e:<6_random_hex_chars>														(JSON/MsgPack/Proto) used by all ERC20 Address, storing the mapping to a TFT address
	//
	// Rivine Value Encodings:
	//	 + addresses are Hex-encoded and the exact format (and how it is created) is described in:
	//     https://github.com/threefoldtech/rivine/blob/master/doc/transactions/unlockhash.md#textstring-encoding
	//   + currencies are encoded as described in https://godoc.org/math/big#Int.Text
	//     using base 10, and using the smallest coin unit as value (e.g. 10^-9 TFT)
	//   + coin outputs are stored in the Rivine-defined JSON format, described in:
	//     https://github.com/threefoldtech/rivine/blob/master/doc/transactions/transaction.md#json-encoding-of-outputs-in-v0-transactions (v0 tx) and
	//     https://github.com/threefoldtech/rivine/blob/master/doc/transactions/transaction.md#json-encoding-of-outputs-in-v1-transactions (v1 tx)
	//
	// JSON formats of value types defined by this module:
	//
	//  example of global stats (stored under <chainName>:<networkName>:stats):
	//    {
	//        "timestamp": 1535661244,
	//        "blockHeight": 103481,
	//        "txCount": 103830,
	//        "coinCreationTxCount": 2,
	//        "coinCreatorDefinitionTxCount": 1,
	//		  "coinBurnTxCount": 1,
	//        "botRegistrationTxCount": 3402,
	//        "botUpdateTxCount": 100,
	//        "valueTxCount": 348,
	//        "coinOutputCount": 104414,
	//        "lockedCoinOutputCount": 736,
	//        "coinInputCount": 1884,
	//        "minerPayoutCount": 103481,
	//        "txFeeCount": 306,
	//        "foundationFeeCount": 10,
	//        "minerPayouts": "1034810000000000",
	//        "txFees": "36100000071",
	//        "foundationFees": "410003200",
	//        "coins": "101054810300000000",
	//        "lockedCoins": "8045200000000"
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
	//
	// example of a 3Bot record (stored under b:<1+_random_digits> <1_or_2_random_digits>)
	//    {
	//        "id": 1,
	//        "addresses":["example.com","91.198.174.192"],
	//        "names": ["thisis.mybot", "voicebot.example", "voicebot.example.myorg"],
	//        "publickey": "ed25519:00bde9571b30e1742c41fcca8c730183402d967df5b17b5f4ced22c677806614",
	//        "expiration": 1542815220
	//    }
	//
	RedisDatabase struct {
		// The redis connection, no time out
		conn redis.Conn

		// used for the encoding/decoding of structured data
		encoder encoding.Encoder

		// cached chain constants
		blockFrequency types.LockValue

		// optional description filter set
		// which is used to know which unlocked outputs to store in a wallet,
		// as only outputs which have descriptions (that also match any of the filters in the filter set)
		// will be stored as part of a structured wallet value
		filters types.DescriptionFilterSet

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
	// DatabaseCoinOutputResult is returned by a Lua scripts which updates/marks a CoinOutput.
	DatabaseCoinOutputResult struct {
		CoinOutputID types.CoinOutputID
		UnlockHash   types.UnlockHash
		CoinValue    types.Currency
		LockType     dtypes.LockType
		LockValue    types.LockValue
		Description  string
	}
)

// LoadBytes implements BytesLoader.LoadBytes
func (cor *DatabaseCoinOutputResult) LoadBytes(b []byte) error {
	// load prefixed coin output ID
	const coinOutputIDStringSize = crypto.HashSize * 2
	if len(b) < coinOutputIDStringSize+1 {
		return fmt.Errorf("failed to load Prefixed CoinOutputID in DatabaseCoinOutputResult from given byte slice: %v", io.EOF)
	}
	err := cor.CoinOutputID.LoadString(string(b[:coinOutputIDStringSize]))
	if err != nil {
		return fmt.Errorf("failed to load Prefixed CoinOutputID in DatabaseCoinOutputResult from given byte slice: %v", err)
	}

	// load returned CoinOutput values
	decoder := siabin.NewDecoder(bytes.NewReader(b[coinOutputIDStringSize:]))
	err = decoder.DecodeAll(
		&cor.UnlockHash,
		&cor.CoinValue,
		&cor.LockType,
		&cor.LockValue,
		&cor.Description,
	)
	if err != nil {
		return fmt.Errorf("failed to decode CoinOutput: %v", err)
	}
	return nil
}

const (
	internalKey                     = "internal"
	internalFieldState              = "state"
	internalFieldNetwork            = "network"
	internalFieldEncoding           = "encoding"
	internalFieldDescriptionFilters = "desc.filters"

	lockedByHeightOutputsKeyPrefix    = "lcos.height:"
	lockedByTimestampOutputsKeyPrefix = "lcos.time:"
)

// NewRedisDatabase creates a new Redis Database client, used by the internal explorer module,
// see RedisDatabase for more information.
func NewRedisDatabase(address string, db int, encodingType encoding.Type, bcInfo rivinetypes.BlockchainInfo, chainCts rivinetypes.ChainConstants, filters types.DescriptionFilterSet, yesToAll bool) (*RedisDatabase, error) {
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
		return nil, err
	}
	// create our encoder, now that we know our encoding type is OK, as we'll need it from here on out
	rdb.encoder, err = encoding.NewEncoder(encodingType)
	if err != nil {
		return nil, err
	}
	// ensure the network info is as expected (or register if this is a fresh db)
	err = rdb.registerOrValidateNetworkInfo(bcInfo)
	if err != nil {
		return nil, err
	}
	// set the description filter set (an empty set is fine too)
	err = rdb.registerFilterSet(filters, yesToAll)
	if err != nil {
		return nil, err
	}
	// create and load scripts
	err = rdb.createAndLoadScripts()
	if err != nil {
		return nil, fmt.Errorf("failed to create/load a lua script(s): %v", err)
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
		dtypes.CoinOutputStateLocked.String(), dtypes.CoinOutputStateLiquid.String(), ">=")
	if err != nil {
		return
	}
	rdb.lockByTimeScript, err = rdb.createAndLoadScript(
		updateTimeLocksScriptSource,
		dtypes.CoinOutputStateLiquid.String(), dtypes.CoinOutputStateLocked.String(), "<")
	if err != nil {
		return
	}

	rdb.unlockByHeightScript, err = rdb.createAndLoadScript(
		updateHeightLocksScriptSource,
		dtypes.CoinOutputStateLocked.Byte(), dtypes.CoinOutputStateLiquid.Byte())
	if err != nil {
		return
	}
	rdb.lockByHeightScript, err = rdb.createAndLoadScript(
		updateHeightLocksScriptSource,
		dtypes.CoinOutputStateLiquid.Byte(), dtypes.CoinOutputStateLocked.Byte())
	if err != nil {
		return
	}

	rdb.spendCoinOutputScript, err = rdb.createAndLoadScript(
		updateCoinOutputScriptSource,
		dtypes.CoinOutputStateLiquid.Byte(), dtypes.CoinOutputStateSpent.Byte())
	if err != nil {
		return
	}
	rdb.unspendCoinOutputScript, err = rdb.createAndLoadScript(
		updateCoinOutputScriptSource,
		dtypes.CoinOutputStateSpent.Byte(), dtypes.CoinOutputStateLiquid.Byte())
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
	if output:byte(1) == %[1]v then
		output = string.char(%[2]v) .. output:sub(2)
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
if output:byte(1) ~= %[1]v then
	return nil
end
output = string.char(%[2]v) .. output:sub(2)
redis.call("HSET", key, field, output)
return coinOutputID .. output:sub(2)
`
)

// registerFilterSet registers the filter set if it doesn't exist yet,
// otherwise it ensures that the returned filter set matches the expected filter set.
func (rdb *RedisDatabase) registerFilterSet(filters types.DescriptionFilterSet, overwriteFilters bool) error {
	// load previous stored, as to be able to delete out references and add new references
	var receivedFilters types.DescriptionFilterSet
	err := RedisStringLoader(&receivedFilters)(rdb.conn.Do("HGET", internalKey, internalFieldDescriptionFilters))
	if err != nil {
		if err == redis.ErrNil {
			// assume a new database, simply register description filter set
			err = RedisError(rdb.conn.Do("HSET", internalKey, internalFieldDescriptionFilters, filters.String()))
			if err != nil {
				return fmt.Errorf("failed to register/validate description filter set: %v", err)
			}
			return nil
		}
		return fmt.Errorf("failed to register/validate description filter set: %v", err)
	}

	// validate returned filter set
	removedFilters := receivedFilters.Difference(filters)
	addedFilters := filters.Difference(receivedFilters)
	filtersChanged := false
	if removedFilters.Len() > 0 || addedFilters.Len() > 0 {
		filtersChanged = true
		if !overwriteFilters {
			var question string
			if removedFilters.Len() > 0 {
				question += "{" + removedFilters.String() + "} will be removed."
			}
			if addedFilters.Len() > 0 {
				if question != "" {
					question += " "
				}
				question += "{" + addedFilters.String() + "} will be added."
			}
			question += " Are you sure that you want to change the description filters to apply " +
				"and modify the stored wallet values as a consequence?"
			overwriteFilters, err = askYesNoQuestion(question)
			if err != nil {
				return fmt.Errorf("failed to register/validate description filter set: %v", err)
			}
			if !overwriteFilters {
				return errors.New("failed to register/validate description filter set: user denied to modify existing filters")
			}
		}
		// apply new filter set
		err = rdb.applyNewFilterSet(addedFilters, removedFilters)
		if err != nil {
			return fmt.Errorf("failed to register/validate description filter set: "+
				"an error occurred while applying new filter set: %v", err)
		}
	}

	rdb.filters = filters

	if filtersChanged {
		// store filters
		err = RedisError(rdb.conn.Do("HSET", internalKey, internalFieldDescriptionFilters, filters.String()))
		if err != nil {
			return fmt.Errorf("failed to register/validate description filter set: %v", err)
		}
	}

	// return successfully
	return nil
}

// for all outputs, check if it has a description that matches an added/removed filter,
// if so, the wallet has to be fetched and updated IFF the output is in the unlocked state.
func (rdb *RedisDatabase) applyNewFilterSet(addedFilters, removedFilters types.DescriptionFilterSet) error {
	// needed to know the total coin outputs, as to be able to report progress within a useful context
	stats, err := rdb.GetNetworkStats()
	if err != nil {
		return fmt.Errorf("failed to applyNewFilters: could not fetch network stats: %v", err)
	}

	// cache wallets, so we do not constantly need to serialize and deserialize
	wallets := make(map[types.UnlockHash]*types.Wallet)

	// used to keep track of output conter, and report progress
	var outputCounter int

	// scan through all outputs (scanning through all outputs of all buckets)
	cursor := "0"
	for {
		results, err := redis.Values(rdb.conn.Do("SCAN", cursor, "MATCH", "c:*"))
		if err != nil {
			return fmt.Errorf("unexpected error while scanning through unique outputs with cursor %q: %v", cursor, err.Error())
		}
		if n := len(results); n != 2 {
			return fmt.Errorf("expected to receive 2 results from a SCAN call, but received %d result(s)", n)
		}
		cursor, err = redis.String(results[0], nil)
		if err != nil {
			return fmt.Errorf("failed to interpret cursor received from last SCAN call: %v", err)
		}

		buckets, err := redis.Strings(results[1], nil)
		if err == redis.ErrNil || len(buckets) == 0 {
			// MATCH is applied only at the end, therefore it is possible
			// that an iteration returns no elements
			if cursor == "0" {
				break
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error while scanning through unique outputs with cursor %q: invalid addresses: %v", cursor, err.Error())
		}

		for _, bucket := range buckets {
			bucketCursor := "0"
			for {
				bucketResults, err := redis.Values(rdb.conn.Do("HSCAN", bucket, bucketCursor))
				if err != nil {
					return fmt.Errorf("unexpected error while scanning through outputs bucket with cursor %q: %v", bucketCursor, err.Error())
				}
				if n := len(bucketResults); n != 2 {
					return fmt.Errorf("unexpected to receive 2 results from a HSCAN call, but received %d result(s)", n)
				}

				outputs, err := redis.StringMap(bucketResults[1], nil)
				if err != nil {
					return fmt.Errorf("error while scanning through output buckets with cursor %q: invalid outputs: %v", bucketCursor, err.Error())
				}

				for key, value := range outputs {
					outputIDStr := bucket[2:] + key
					var outputID types.CoinOutputID
					err := outputID.LoadString(outputIDStr)
					if err != nil {
						return fmt.Errorf("unexpected error while decoding coin output ID %s: %v", outputIDStr, err)
					}
					var output dtypes.CoinOutput
					err = output.LoadBytes([]byte(value))
					if err != nil {
						return fmt.Errorf("unexpected error while decoding coin output %s: %v", outputIDStr, err)
					}

					// print progress
					outputCounter++
					if outputCounter%5000 == 0 {
						log.Printf("[filter update] coin output scanner is now at coin output %d/%d...\n",
							outputCounter, stats.CoinOutputCount)
					}

					// check state, we only care about the ones which are in state Liquid,
					// reason being that only those are expected to be in the unlocked list of a wallet,
					// which is the list populated after using the description filters
					if output.State != dtypes.CoinOutputStateLiquid {
						continue
					}
					// if the description is empty we also do not care
					if output.Description == "" {
						continue
					}

					var added, removed bool
					added = addedFilters.Match(output.Description)
					if !added {
						removed = removedFilters.Match(output.Description)
						if !removed {
							// if not affected by a removed/added filter we can stop as well,
							// as the output will not be affected either
							continue
						}
					}

					// get wallet for given unlock hash
					wallet, ok := wallets[output.UnlockHash]
					if !ok {
						addressKey, addressField := database.GetAddressKeyAndField(output.UnlockHash)
						b, err := redis.Bytes(rdb.conn.Do("HGET", addressKey, addressField))
						if err != nil {
							// not even redis.ErrNil is expected at this point
							return fmt.Errorf("failed to get wallet (uh: %s) for output %s: %v", output.UnlockHash, outputIDStr, err)
						}
						if len(b) > 0 {
							wallet = new(types.Wallet)
							err = rdb.encoder.Unmarshal(b, wallet)
							if err != nil {
								return errors.New("failed to unmarshal wallet: " + err.Error())
							}
						}
						wallets[output.UnlockHash] = wallet
					}

					// add or remove the output
					if added {
						// add the unlocked output
						err = wallet.Balance.Unlocked.AddUnlockedCoinOutput(outputID, types.WalletUnlockedOutput{
							Amount:      output.CoinValue,
							Description: output.Description,
						}, false)
						if err != nil {
							return fmt.Errorf("failed to add unlocked coin output %s to wallet %s: %v",
								outputIDStr, output.UnlockHash.String(), err)
						}
					} else {
						// remove the unlocked output
						err = wallet.Balance.Unlocked.SubUnlockedCoinOutput(outputID, output.CoinValue, false)
						if err != nil {
							return fmt.Errorf("failed to remove unlocked coin output %s from wallet %s: %v",
								outputIDStr, output.UnlockHash.String(), err)
						}
					}
				}

				bucketCursor, err = redis.String(bucketResults[0], nil)
				if err != nil {
					return fmt.Errorf("failed to interpret cursor received from last HSCAN call: %v", err)
				}
				if bucketCursor == "0" {
					break
				}
			}
		}

		if cursor == "0" {
			break
		}
	}

	// store all updated wallets
	for uh, wallet := range wallets {
		rdb.planWalletStorageOrDeletion(*wallet, uh)
	}
	err = RedisError(RedisFlushAndReceive(rdb.conn, len(wallets)))
	if err != nil {
		return fmt.Errorf("failed to update wallets with modifed filterset: %v", err)
	}
	return nil
}

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
	networkInfo := dtypes.NetworkInfo{
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
	var receivedNetworkInfo dtypes.NetworkInfo
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
func (rdb *RedisDatabase) GetExplorerState() (dtypes.ExplorerState, error) {
	var state dtypes.ExplorerState
	switch err := rdb.redisStructuredValue(&state)(rdb.conn.Do("HGET", internalKey, internalFieldState)); err {
	case nil:
		return state, nil
	case redis.ErrNil:
		// default to fresh explorer state if not stored yet
		return dtypes.NewExplorerState(), nil
	default:
		return dtypes.ExplorerState{}, err
	}
}

// SetExplorerState implements Database.SetExplorerState
func (rdb *RedisDatabase) SetExplorerState(state dtypes.ExplorerState) error {
	return RedisError(rdb.conn.Do("HSET", internalKey, internalFieldState, rdb.marshalData(&state)))
}

// GetNetworkStats implements Database.GetNetworkStats
func (rdb *RedisDatabase) GetNetworkStats() (types.NetworkStats, error) {
	var stats types.NetworkStats
	switch err := rdb.redisStructuredValue(&stats)(rdb.conn.Do("GET", database.StatsKey)); err {
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
	err := RedisError(rdb.conn.Do("SET", database.StatsKey, rdb.marshalData(&stats)))
	if err != nil {
		return err
	}
	rdb.networkTime, rdb.networkBlockHeight = stats.Timestamp, stats.BlockHeight
	return nil
}

// AddCoinOutput implements Database.AddCoinOutput
func (rdb *RedisDatabase) AddCoinOutput(id types.CoinOutputID, co CoinOutput) error {
	uh := types.AsUnlockHash(co.Condition.UnlockHash())

	addressKey, addressField := database.GetAddressKeyAndField(uh)
	// get initial values
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", uh.String(), addressKey, addressField, err)
	}

	// increase coin count and optionally add output
	if rdb.filters.Match(co.Description) {
		err = wallet.Balance.Unlocked.AddUnlockedCoinOutput(id, types.WalletUnlockedOutput{
			Amount:      co.Value,
			Description: co.Description,
		}, true)
		if err != nil {
			return fmt.Errorf(
				"redis: failed to add unlocked coinoutput %s to wallet for %s: %v",
				id.String(), uh.String(), err)
		}
	} else {
		wallet.Balance.Unlocked.Total = wallet.Balance.Unlocked.Total.Add(co.Value)
	}

	coinOutputKey, coinOutputField := database.GetCoinOutputKeyAndField(id)

	// set all values pipelined
	// store address, an address never gets deleted
	rdb.conn.Send("SADD", database.AddressesKey, uh.String())
	// store output
	rdb.conn.Send("HSET", coinOutputKey, coinOutputField, dtypes.CoinOutput{
		UnlockHash:  uh,
		CoinValue:   co.Value,
		State:       dtypes.CoinOutputStateLiquid,
		LockType:    dtypes.LockTypeNone,
		LockValue:   0,
		Description: co.Description,
	}.Bytes())
	// store or delete the updated wallet (for now sending it only)
	rdb.planWalletStorageOrDeletion(wallet, uh)
	// submit all changes
	err = RedisError(RedisFlushAndReceive(rdb.conn, 3))
	if err != nil {
		return fmt.Errorf("redis: failed to add coinoutput %s: %v", id.String(), err)
	}
	return nil
}

// AddLockedCoinOutput implements Database.AddLockedCoinOutput
func (rdb *RedisDatabase) AddLockedCoinOutput(id types.CoinOutputID, co CoinOutput, lt dtypes.LockType, lockValue types.LockValue) error {
	uh := types.AsUnlockHash(co.Condition.UnlockHash())

	addressKey, addressField := database.GetAddressKeyAndField(uh)
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
	rdb.conn.Send("SADD", database.AddressesKey, uh.String())
	// store coinoutput in list of locked coins for wallet
	// keep track of locked output
	switch lt {
	case dtypes.LockTypeHeight:
		rdb.conn.Send("RPUSH", getLockHeightBucketKey(lockValue), id.String())
	case dtypes.LockTypeTime:
		rdb.conn.Send("RPUSH", getLockTimeBucketKey(lockValue), dtypes.CoinOutputLock{
			CoinOutputID: id,
			LockValue:    lockValue,
		}.String())
	}
	// store output
	coinOutputKey, coinOutputField := database.GetCoinOutputKeyAndField(id)
	rdb.conn.Send("HSET", coinOutputKey, coinOutputField, dtypes.CoinOutput{
		UnlockHash:  uh,
		CoinValue:   co.Value,
		State:       dtypes.CoinOutputStateLocked,
		LockType:    lt,
		LockValue:   lockValue,
		Description: co.Description,
	}.Bytes())
	// store or delete the updated wallet (for now sending it only)
	rdb.planWalletStorageOrDeletion(wallet, uh)
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
	err := RedisBytesLoader(&result)(rdb.spendCoinOutputScript.Do(rdb.conn, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to spend coin output: cannot update coin output %s: %v",
			id.String(), err)
	}

	// get wallet, so its balance can be updated
	addressKey, addressField := database.GetAddressKeyAndField(result.UnlockHash)
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", result.UnlockHash.String(), addressKey, addressField, err)
	}

	// update unlocked coins (and optionally mapped outputs)
	wallet.Balance.Unlocked.SubUnlockedCoinOutput(id, result.CoinValue, true)

	// store or delete the updated wallet
	err = rdb.storeOrDeleteWallet(wallet, result.UnlockHash)
	if err != nil {
		return fmt.Errorf(
			"redis: failed to spend coin output: failed to store/delete wallet %s as part of coinoutput %s: %v",
			result.UnlockHash.String(), id.String(), err)
	}
	return nil
}

// RevertCoinInput implements Database.RevertCoinInput
// more or less a reverse process of SpendCoinOutput
func (rdb *RedisDatabase) RevertCoinInput(id types.CoinOutputID) error {
	var result DatabaseCoinOutputResult
	err := RedisBytesLoader(&result)(rdb.unspendCoinOutputScript.Do(rdb.conn, id.String()))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to revert coin input: cannot update coin output %s: %v",
			id.String(), err)
	}

	// get wallet, so its balance can be updated
	addressKey, addressField := database.GetAddressKeyAndField(result.UnlockHash)
	wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
	if err != nil {
		return fmt.Errorf(
			"redis: failed to get wallet for %s at %s#%s: %v", result.UnlockHash.String(), addressKey, addressField, err)
	}

	// increase coin count and optionally add output
	if rdb.filters.Match(result.Description) {
		err = wallet.Balance.Unlocked.AddUnlockedCoinOutput(id, types.WalletUnlockedOutput{
			Amount:      result.CoinValue,
			Description: result.Description,
		}, true)
		if err != nil {
			return fmt.Errorf(
				"redis: failed to add unlocked coinoutput %s to wallet for %s: %v",
				id.String(), result.UnlockHash.String(), err)
		}
	} else {
		wallet.Balance.Unlocked.Total = wallet.Balance.Unlocked.Total.Add(result.CoinValue)
	}

	// store or delete the updated wallet
	err = rdb.storeOrDeleteWallet(wallet, result.UnlockHash)
	if err != nil {
		return fmt.Errorf(
			"redis: failed to revert coin input: failed to store/delete wallet %s as part of coinoutput %s: %v",
			result.UnlockHash.String(), id.String(), err)
	}
	return nil
}

// RevertCoinOutput implements Database.RevertCoinOutput
func (rdb *RedisDatabase) RevertCoinOutput(id types.CoinOutputID) (dtypes.CoinOutputState, error) {
	var co dtypes.CoinOutput
	err := RedisBytesLoader(&co)(rdb.coinOutputDropScript.Do(rdb.conn, id.String()))
	if err != nil {
		return dtypes.CoinOutputStateNil, fmt.Errorf(
			"redis: failed to revert coin output: cannot drop coin output %s: %v",
			id.String(), err)
	}
	if co.State == dtypes.CoinOutputStateNil {
		return dtypes.CoinOutputStateNil, fmt.Errorf(
			"redis: failed to revert coin output: nil coin output state %s",
			id.String())
	}

	var sendCount int
	if co.State != dtypes.CoinOutputStateSpent {
		// update all data for this unspent coin output
		sendCount++

		// get wallet, so its balance can be updated
		addressKey, addressField := database.GetAddressKeyAndField(co.UnlockHash)
		wallet, err := rdb.redisWallet(rdb.conn.Do("HGET", addressKey, addressField))
		if err != nil {
			return dtypes.CoinOutputStateNil, fmt.Errorf(
				"redis: failed to get wallet for %s at %s#%s: %v", co.UnlockHash.String(), addressKey, addressField, err)
		}

		// update correct balanace
		switch co.State {
		case dtypes.CoinOutputStateLiquid:
			// update unlocked balance of address wallet (and optionally mapped outputs)
			wallet.Balance.Unlocked.SubUnlockedCoinOutput(id, co.CoinValue, true)
		case dtypes.CoinOutputStateLocked:
			// update locked output map and balance of address wallet
			err = wallet.Balance.Locked.SubLockedCoinOutput(id)
			if err != nil {
				return dtypes.CoinOutputStateNil, fmt.Errorf(
					"redis: failed to revert coin output %s: %v",
					id.String(), err)
			}
		}

		// store or delete the updated wallet (just sending it for now)
		rdb.planWalletStorageOrDeletion(wallet, co.UnlockHash)
	}

	// always remove lock properties if a lock is used, no matter the state
	if co.LockType != dtypes.LockTypeNone {
		sendCount++
		// remove locked coin output lock
		switch co.LockType {
		case dtypes.LockTypeHeight:
			rdb.conn.Send("LREM", getLockHeightBucketKey(co.LockValue), 1, id.String())
		case dtypes.LockTypeTime:
			rdb.conn.Send("LREM", getLockTimeBucketKey(co.LockValue), 1, dtypes.CoinOutputLock{
				CoinOutputID: id,
				LockValue:    co.LockValue,
			}.String())
		}
	}

	if sendCount > 0 {
		err = RedisError(RedisFlushAndReceive(rdb.conn, sendCount))
		if err != nil {
			return dtypes.CoinOutputStateNil, fmt.Errorf(
				"redis: failed to revert coin output %s: failed to submit %d changes: %v",
				id.String(), sendCount, err)
		}
	}

	return co.State, nil
}

// ApplyCoinOutputLocks implements Database.ApplyCoinOutputLocks
func (rdb *RedisDatabase) ApplyCoinOutputLocks(height types.BlockHeight, time types.Timestamp) (n uint64, coins types.Currency, err error) {
	if height.BlockHeight == 0 {
		return
	}
	rdb.networkTime, rdb.networkBlockHeight = time, height
	rdb.unlockByHeightScript.SendHash(rdb.conn, getLockHeightBucketKey(types.LockValue(height.BlockHeight)))
	rdb.unlockByTimeScript.SendHash(rdb.conn, getLockTimeBucketKey(types.LockValue(time.Timestamp)), types.LockValue(time.Timestamp).String())
	values, err := redis.Values(RedisFlushAndReceive(rdb.conn, 2))
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to unlock outputs: %v", err)
	}
	lockedByHeightCoinOutputResults, err := RedisCoinOutputResults(values[0], nil)
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-height output results: %v", err)
	}
	lockedByTimeCoinOutputResults, err := RedisCoinOutputResults(values[1], nil)
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-time output results: %v", err)
	}
	lockedCoinOutputResults := append(lockedByHeightCoinOutputResults, lockedByTimeCoinOutputResults...)
	for _, lcor := range lockedCoinOutputResults {
		addressKey, addressField := database.GetAddressKeyAndField(lcor.UnlockHash)
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
		if rdb.filters.Match(lcor.Description) {
			err = wallet.Balance.Unlocked.AddUnlockedCoinOutput(lcor.CoinOutputID, types.WalletUnlockedOutput{
				Amount:      lcor.CoinValue,
				Description: lcor.Description,
			}, true)
			if err != nil {
				return 0, types.Currency{}, fmt.Errorf(
					"redis: failed to add unlocked coinoutput %s to wallet for %s: %v",
					lcor.CoinOutputID.String(), lcor.UnlockHash.String(), err)
			}
		} else {
			wallet.Balance.Unlocked.Total = wallet.Balance.Unlocked.Total.Add(lcor.CoinValue)
		}
		// store or delete the updated wallet
		err = rdb.storeOrDeleteWallet(wallet, lcor.UnlockHash)
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
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-height output results: %v", err)
	}
	unlockedByTimeCoinOutputResults, err := RedisCoinOutputResults(values[1], nil)
	if err != nil {
		return 0, types.Currency{}, fmt.Errorf("failed to update+parse locked-by-time output results: %v", err)
	}
	unlockedCoinOutputResults := append(unlockedByHeightCoinOutputResults, unlockedByTimeCoinOutputResults...)
	for _, ulcor := range unlockedCoinOutputResults {
		addressKey, addressField := database.GetAddressKeyAndField(ulcor.UnlockHash)
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
		// update unlocked balance (and optionally mapped outputs)
		wallet.Balance.Unlocked.SubUnlockedCoinOutput(ulcor.CoinOutputID, ulcor.CoinValue, true)
		// store or delete the updated wallet
		err = rdb.storeOrDeleteWallet(wallet, ulcor.UnlockHash)
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
	addressKey, addressField := database.GetAddressKeyAndField(address)
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
	// store or delete the updated wallet
	err = rdb.storeOrDeleteWallet(wallet, address)
	if err != nil {
		return fmt.Errorf(
			"redis: failed to store/delete multisig wallet %s: %v", address.String(), err)
	}

	// now get the wallet for all owners, and set the multisig address,
	// this step is a bit expensive, as the entire wallet has to be loaded,
	// but luckily it only has to be done once per wallet appearance coin output
	for _, owner := range owners {
		// store multisig wallet first, as that will indicate if the owners (should) have the address or not
		addressKey, addressField := database.GetAddressKeyAndField(owner)
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
		// store or delete the updated wallet
		err = rdb.storeOrDeleteWallet(wallet, owner)
		if err != nil {
			return fmt.Errorf(
				"redis: SetMultisigAddresses %s for owner %s: %v",
				address.String(), owner.String(), err)
		}
	}
	return nil
}

// SetCoinCreators implements Database.SetCoinCreators
func (rdb *RedisDatabase) SetCoinCreators(creators []types.UnlockHash) error {
	rdb.conn.Send("DEL", database.CoinCreatorsKey)
	for _, creator := range creators {
		rdb.conn.Send("SADD", database.CoinCreatorsKey, creator.String())
		// also track the coin creator address
		rdb.conn.Send("SADD", database.AddressesKey, creator.String())
	}
	_, err := RedisFlushAndReceive(rdb.conn, 1+len(creators))
	if err != nil {
		return fmt.Errorf("failed to set coin creators: %v", err)
	}
	return nil
}

// CreateBotRecord implements Database.CreateBotRecord
func (rdb *RedisDatabase) CreateBotRecord(record types.BotRecord) error {
	key, field := database.GetThreeBotKeyAndField(record.ID)
	err := RedisError(rdb.conn.Do("HSETNX", key, field, rdb.marshalData(&record)))
	if err != nil {
		return fmt.Errorf("failed to create bot record for 3Bot %v: %v", record.ID, err)
	}
	return nil
}

// UpdateBotRecord implements Database.UpdateBotRecord
func (rdb *RedisDatabase) UpdateBotRecord(id types.BotID, fn func(*types.BotRecord) error) error {
	key, field := database.GetThreeBotKeyAndField(id)
	b, err := redis.Bytes(rdb.conn.Do("HGET", key, field))
	if err != nil {
		// not even redis.ErrNil is expected at this point
		return fmt.Errorf("failed to get 3Bot record for 3Bot %v: %v", id, err)
	}
	record := new(types.BotRecord)
	err = rdb.encoder.Unmarshal(b, record)
	if err != nil {
		return fmt.Errorf("failed to unmarshal 3Bot record for 3Bot %v: %v", id, err)
	}
	err = fn(record)
	if err != nil {
		return fmt.Errorf("failed to update 3Bot record for 3Bot %v: %v", id, err)
	}
	err = RedisError(rdb.conn.Do("HSET", key, field, rdb.marshalData(record)))
	if err != nil {
		return fmt.Errorf("failed to update 3bot record for 3Bot %v in redis: %v", record.ID, err)
	}
	return nil
}

// DeleteBotRecord implements Database.DeleteBotRecord
func (rdb *RedisDatabase) DeleteBotRecord(id types.BotID) error {
	key, field := database.GetThreeBotKeyAndField(id)
	err := RedisError(rdb.conn.Do("HDEL", key, field))
	if err != nil {
		return fmt.Errorf("failed to delete bot record from 3Bot %v: %v", id, err)
	}
	return nil
}

// AddERC20AddressRegistration implements Database.AddERC20AddressRegistration
func (rdb *RedisDatabase) AddERC20AddressRegistration(erc20Address types.ERC20Address, tftAddress types.UnlockHash) error {
	key, field := database.GetERC20AddressKeyAndField(erc20Address)
	err := RedisError(rdb.conn.Do("HSETNX", key, field, rdb.marshalData(&tftAddress)))
	if err != nil {
		return fmt.Errorf("failed to create ERC20 Address Registration for address %v: %v", erc20Address, err)
	}
	return nil
}

// DeleteERC20AddressRegistration implements Database.DeleteERC20AddressRegistration
func (rdb *RedisDatabase) DeleteERC20AddressRegistration(erc20Address types.ERC20Address) error {
	key, field := database.GetERC20AddressKeyAndField(erc20Address)
	err := RedisError(rdb.conn.Do("HDEL", key, field))
	if err != nil {
		return fmt.Errorf("failed to delete ERC20 Address Registration for addr %v: %v", erc20Address, err)
	}
	return nil
}

func (rdb *RedisDatabase) storeOrDeleteWallet(wallet types.Wallet, uh types.UnlockHash) error {
	addressKey, addressField := database.GetAddressKeyAndField(uh)
	// either delete or store the wallet, depending on whether or not still contains content
	if wallet.IsNil() {
		// delete the wallet, as it has no content any longer
		err := RedisError(rdb.conn.Do("HDEL", addressKey, addressField))
		if err != nil {
			return fmt.Errorf("failed to delete wallet %s: %v", uh.String(), err)
		}
		return nil
	}
	// store the updated wallet, which still contains content
	err := RedisError(rdb.conn.Do("HSET", addressKey, addressField, rdb.marshalData(&wallet)))
	if err != nil {
		return fmt.Errorf("failed to store updated wallet %s: %v", uh.String(), err)
	}
	return nil
}

func (rdb *RedisDatabase) planWalletStorageOrDeletion(wallet types.Wallet, uh types.UnlockHash) {
	addressKey, addressField := database.GetAddressKeyAndField(uh)
	// either delete or store the wallet, depending on whether or not still contains content
	if wallet.IsNil() {
		// delete the wallet, as it has no content any longer
		rdb.conn.Send("HDEL", addressKey, addressField)
		return
	}
	// store the updated wallet, which still contains content
	rdb.conn.Send("HSET", addressKey, addressField, rdb.marshalData(&wallet))
}

func (rdb *RedisDatabase) lockValueAsLockTime(lt dtypes.LockType, value types.LockValue) types.LockValue {
	switch lt {
	case dtypes.LockTypeTime:
		return value
	case dtypes.LockTypeHeight:
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

// getLockTimeBucketKey is an internal util function,
// used to create the timelocked bucket keys, grouping timelocked outputs within a given time range together.
func getLockTimeBucketKey(lockValue types.LockValue) string {
	return lockedByTimestampOutputsKeyPrefix + (lockValue - lockValue%7200).String()
}

// getLockHeightBucketKey is an internal util function,
// used to create the heightlocked bucket keys, grouping all heightlocked outputs with the same lock-height value.
func getLockHeightBucketKey(lockValue types.LockValue) string {
	return lockedByHeightOutputsKeyPrefix + lockValue.String()
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
func RedisStringLoader(sl dtypes.StringLoader) func(interface{}, error) error {
	return func(reply interface{}, err error) error {
		s, err := redis.String(reply, err)
		if err != nil {
			return err
		}
		return sl.LoadString(s)
	}
}

// RedisBytesLoader creates a function that can be used to unmarshal a slice byte value
// as a (custom) BytesLoader value into the given (reference) value (v).
func RedisBytesLoader(bl dtypes.BytesLoader) func(interface{}, error) error {
	return func(reply interface{}, err error) error {
		b, err := redis.Bytes(reply, err)
		if err != nil {
			return err
		}
		return bl.LoadBytes(b)
	}
}

// RedisCoinOutputResults returns all CoinOutputResults found for a given []string redis reply,
// only used in combination with `(*RedisDatabase).UpdateLockedCoinOutputs`, see that method for more information.
func RedisCoinOutputResults(reply interface{}, err error) ([]DatabaseCoinOutputResult, error) {
	slices, err := redis.ByteSlices(reply, err)
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, err
	}
	results := make([]DatabaseCoinOutputResult, len(slices))
	for i, slice := range slices {
		err = results[i].LoadBytes(slice)
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
