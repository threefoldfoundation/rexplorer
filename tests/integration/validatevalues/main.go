package main

import (
	"flag"
	"fmt"
	"strings"

	dtypes "github.com/threefoldfoundation/rexplorer/pkg/database/types"
	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	"github.com/gomodule/redigo/redis"
)

func main() {
	flag.Parse()

	encoder, err := encoding.NewEncoder(encodingType)
	if err != nil {
		panic(err)
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(err)
	}

	// validate stats

	b, err := redis.Bytes(conn.Do("GET", "stats"))
	if err != nil {
		panic("failed to get network stats: " + err.Error())
	}
	var stats types.NetworkStats
	err = encoder.Unmarshal(b, &stats)
	if err != nil {
		panic("failed to unmarshal network stats: " + err.Error())
	}

	if stats.Timestamp.Timestamp <= 1511996400 {
		panic(fmt.Sprintf("global stats error: timestamp %d is invalid", stats.Timestamp))
	}
	if stats.BlockHeight.BlockHeight == 0 {
		panic(fmt.Sprintf("global stats error: block height %d is invalid", stats.BlockHeight))
	}
	if stats.CoinCreationTransactionCount+stats.CoinCreatorDefinitionTransactionCount > stats.TransactionCount {
		panic(fmt.Sprintf("global stats error: tx count %d is smaller than special tx count (%d+%d)",
			stats.TransactionCount, stats.CoinCreationTransactionCount, stats.CoinCreatorDefinitionTransactionCount))
	}
	if stats.LockedCoinOutputCount > stats.CoinOutputCount {
		panic(fmt.Sprintf("global stats error: locked coin outputcount %d is bigger than total coin output count %d ",
			stats.LockedCoinOutputCount, stats.CoinOutputCount))
	}

	if stats.LockedCoins.Cmp(stats.Coins) == 1 {
		panic(fmt.Sprintf("global stats error: locked coins (%s) > total coins (%s)",
			stats.LockedCoins.String(), stats.Coins.String()))
	}

	var totalCoins types.Currency
	totalCoins = totalCoins.Add(stats.TransactionFees)
	totalCoins = totalCoins.Add(stats.MinerPayouts)
	if totalCoins.Cmp(stats.Coins) == 1 {
		panic(fmt.Sprintf("global stats error: txFees(%s)+minerPayouts(%s) > total coins (%s)",
			stats.TransactionFees.String(), stats.MinerPayouts.String(),
			stats.Coins.String()))
	}

	fmt.Println("Global stats are valid :)")

	// validate internal hashmap values

	internalMapCursor := "0"
	internalKeysSeen := make(map[string]interface{})
	for {
		results, err := redis.Values(conn.Do("HSCAN", "internal", internalMapCursor))
		if err != nil {
			panic(fmt.Sprintf("unexpected error while scanning through internal map with cursor %q: %v", internalMapCursor, err.Error()))
		}
		if n := len(results); n != 2 {
			panic(fmt.Sprintf("unexpected to receive 2 results from a HSCAN call, but received %d result(s)", n))
		}

		internalMap, err := redis.StringMap(results[1], nil)
		if err != nil {
			panic(fmt.Sprintf("error while scanning through internal hashmap with cursor %q: invalid values: %v", internalMapCursor, err.Error()))
		}
		for key, value := range internalMap {
			internalKeysSeen[key] = struct{}{}
			switch key {
			case "state":
				var state dtypes.ExplorerState
				err := encoder.Unmarshal([]byte(value), &state)
				if err != nil {
					panic(fmt.Sprintf("error while decoding internal explorer state: %v", err))
				}
			case "network":
				var networkInfo dtypes.NetworkInfo
				err := encoder.Unmarshal([]byte(value), &networkInfo)
				if err != nil {
					panic(fmt.Sprintf("error while decoding internal network info: %v", err))
				}
				if networkInfo.ChainName != "tfchain" {
					panic("unexpected chain name: " + networkInfo.ChainName)
				}
				if networkInfo.NetworkName != "testnet" && networkInfo.NetworkName != "standard" && networkInfo.NetworkName != "devnet" {
					panic("unexpected network name: " + networkInfo.NetworkName)
				}
			case "encoding":
				var internalEncodingType encoding.Type
				err := internalEncodingType.LoadString(value)
				if err != nil {
					panic(fmt.Sprintf("error while decoding encoding type %s: %v", value, err))
				}
				if internalEncodingType != encodingType {
					panic(fmt.Sprintf(
						"internal encoding type (%d) does not match flag-given encoding type (%d)",
						internalEncodingType, encodingType))
				}
			case "desc.filters":
				var receivedFilters types.DescriptionFilterSet
				err := receivedFilters.LoadString(value)
				if err != nil {
					panic(fmt.Sprintf("error while decoding received filters %s: %v", value, err))
				}
			default:
				panic("unexpected internal hashmap key: " + key)
			}
		}

		internalMapCursor, err = redis.String(results[0], nil)
		if err != nil {
			panic(fmt.Sprintf("failed to interpret cursor received from last internal HSCAN call: %v", err))
		}
		if internalMapCursor == "0" {
			break
		}
	}
	if len(internalKeysSeen) != 4 {
		var allKeys []string
		for key := range internalKeysSeen {
			allKeys = append(allKeys, key)
		}
		panic("some internal keys were missing, the following internal keys have been found: " + strings.Join(allKeys, ","))
	}

	fmt.Println("Internal keys are valid :)")

	// go through all time-locked coin outputs to ensure they all exist,
	// in the `sumoutputs` integration test we already check the other direction
	tlcursor := "0"
	var tlcCounter int
	for {
		results, err := redis.Values(conn.Do("SCAN", tlcursor, "MATCH", "lcos.time:*"))
		if err != nil {
			panic(fmt.Sprintf("unexpected error while scanning through all time-locked entries cursor %q: %v", tlcursor, err.Error()))
		}
		if n := len(results); n != 2 {
			panic(fmt.Sprintf("unexpected to receive 2 results from a SCAN-MATCH call, but received %d result(s)", n))
		}
		tlcursor, err = redis.String(results[0], nil)
		if err != nil {
			panic(fmt.Sprintf("failed to interpret cursor received from last tlocked SCAN-MATCH call: %v", err))
		}

		keys, err := redis.Strings(results[1], nil)
		if err == redis.ErrNil || len(keys) == 0 {
			if tlcursor == "0" {
				break
			}
			continue
		}
		if err != nil {
			panic(fmt.Sprintf("error while getting all time-locked entry keys returned from SCAN-MATCH call with cursor %s: %v", tlcursor, err))
		}
		for _, key := range keys {
			strings, err := redis.Strings(conn.Do("LRANGE", key, 0, 10000))
			if err != redis.ErrNil {
				if err != nil {
					panic(fmt.Sprintf("error to fetch all keys in time-lock-entry bucket %s: %v", key, err))
				}
				for _, str := range strings {

					var col dtypes.CoinOutputLock
					err := col.LoadString(str)
					if err != nil {
						panic(fmt.Sprintf("error to load locked coin output value from str %q: %v", str, err))
					}
					coinOutputID := col.CoinOutputID.String()

					// print progress
					tlcCounter++
					if tlcCounter%100 == 0 {
						fmt.Printf("time-locked output scanner is now at output #%d with id %s...\n", tlcCounter, coinOutputID)
					}

					// get coin output, to ensure it exist, and ensure lock value is the same
					b, err := redis.Bytes(conn.Do("HGET", "c:"+coinOutputID[:4], coinOutputID[4:]))
					if err != nil {
						panic(fmt.Sprintf("error while getting coin output %s: %v", coinOutputID, err))
					}
					var output dtypes.CoinOutput
					err = output.LoadBytes([]byte(b))
					if err != nil {
						panic(fmt.Sprintf("unexpected error while decoding coin output %s: %v", coinOutputID, err))
					}
					if col.LockValue != output.LockValue {
						panic(fmt.Sprintf("lock entry found for coin output %s, but lockvalue found(%d) != expected(%s) ",
							coinOutputID, col.LockValue, output.LockValue))
					}
				}
			}
		}

		if tlcursor == "0" {
			break
		}
	}
	if tlcCounter > 0 {
		fmt.Println("Time-Locked Output entries are valid :)")
	}

	// go through all time-locked coin outputs to ensure they all exist,
	// in the `sumoutputs` integration test we already check the other direction
	hlcursor := "0"
	var hlcCount int
	for {
		results, err := redis.Values(conn.Do("SCAN", hlcursor, "MATCH", "lcos.height:*"))
		if err != nil {
			panic(fmt.Sprintf("unexpected error while scanning through all height-locked entries cursor %q: %v", hlcursor, err.Error()))
		}
		if n := len(results); n != 2 {
			panic(fmt.Sprintf("unexpected to receive 2 results from a SCAN-MATCH call, but received %d result(s)", n))
		}
		hlcursor, err = redis.String(results[0], nil)
		if err != nil {
			panic(fmt.Sprintf("failed to interpret cursor received from last hlocked SCAN-MATCH call: %v", err))
		}

		keys, err := redis.Strings(results[1], nil)
		if err == redis.ErrNil || len(keys) == 0 {
			if hlcursor == "0" {
				break
			}
			continue
		}
		if err != nil {
			panic(fmt.Sprintf("error while getting all time-locked entry keys returned from SCAN-MATCH call with cursor %s: %v", tlcursor, err))
		}
		for _, key := range keys {
			ids, err := redis.Strings(conn.Do("LRANGE", key, 0, 10000))
			if err != redis.ErrNil {
				if err != nil {
					panic(fmt.Sprintf("error to fetch all keys in time-lock-entry bucket %s: %v", key, err))
				}
				strs := strings.SplitN(key, ":", 2)
				if len(strs) != 2 {
					panic(fmt.Sprintf("bad bucket key %s (invalid format)", key))
				}
				var expectedLockValue types.LockValue
				err := expectedLockValue.LoadString(strs[1])
				if err != nil {
					panic(fmt.Sprintf("bad bucket key %s (invalid integer)", key))
				}
				for _, coinOutputID := range ids {
					// print progress
					hlcCount++
					if hlcCount%5000 == 0 {
						fmt.Printf("height-locked output scanner is now at output #%d with id %s...\n", hlcCount, coinOutputID)
					}

					// get coin output, to ensure it exist, and ensure lock value is the same
					b, err := redis.Bytes(conn.Do("HGET", "c:"+coinOutputID[:4], coinOutputID[4:]))
					if err != nil {
						panic(fmt.Sprintf("error while getting coin output %s: %v", coinOutputID, err))
					}
					var output dtypes.CoinOutput
					err = output.LoadBytes([]byte(b))
					if err != nil {
						panic(fmt.Sprintf("unexpected error while decoding coin output %s: %v", coinOutputID, err))
					}
					if expectedLockValue != output.LockValue {
						panic(fmt.Sprintf("lock entry found for coin output %s, but lockvalue found(%d) != expected(%s) ",
							coinOutputID, expectedLockValue, output.LockValue))
					}
				}
			}
		}

		if hlcursor == "0" {
			break
		}
	}
	if tlcCounter > 0 {
		fmt.Println("Height-Locked Output entries are valid :)")
	}

	// validate coin creators

	coinCreators, err := redis.Strings(conn.Do("SMEMBERS", "coincreators"))
	if err != redis.ErrNil {
		if err != nil {
			panic("failed to get coin creators: " + err.Error())
		}
		for _, coinCreator := range coinCreators {
			// ensure each coin creator is known in the set of unique and known addresses
			addressIsKnown, err := redis.Bool(conn.Do("SISMEMBER", "addresses", coinCreator))
			if err != nil {
				panic(fmt.Sprintf(
					"unexpected error while checking if coin creator address %s is part of all known unique addresses: %v",
					coinCreator, err))
			}
			if !addressIsKnown {
				panic(fmt.Sprintf(
					"coin creator %s is not part of the set of unique known addresses while it is expected to be",
					coinCreator))
			}
		}
		fmt.Printf("All %d coin creators are known and tracked :)\n", len(coinCreators))
	}

	fmt.Printf(
		"validatevalues test —using encoding %s— on block height %d passed :)\n",
		encodingType.String(), stats.BlockHeight.BlockHeight)
}

var (
	dbAddress    string
	dbSlot       int
	encodingType encoding.Type
)

func init() {
	flag.StringVar(&dbAddress, "redis-addr", ":6379", "(tcp) address of the redis db")
	flag.IntVar(&dbSlot, "redis-db", 0, "slot/index of the redis db")
	flag.Var(&encodingType, "encoding",
		"which encoding protocol to use, one of {json,msgp,protobuf} (default: "+encodingType.String()+")")
}
