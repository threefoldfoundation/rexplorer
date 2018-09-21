package main

import (
	"flag"
	"fmt"

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
	// get stats, so we know what are the to be expected total coins and total locked coins
	b, err := redis.Bytes(conn.Do("GET", "stats"))
	if err != nil {
		panic("failed to get network stats: " + err.Error())
	}
	var stats types.NetworkStats
	err = encoder.Unmarshal(b, &stats)
	if err != nil {
		panic("failed to unmarshal network stats: " + err.Error())
	}

	// compute total unlocked and locked coins for all outputs
	var unlockedCoins, lockedCoins types.Currency

	// scan through all outputs (scanning through all outputs of all buckets)
	outputCounter := 0
	bucketCounter := 0
	cursor := "0"
	for {
		results, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", "c:*"))
		if err != nil {
			panic(fmt.Sprintf("unexpected error while scanning through unique outputs with cursor %q: %v", cursor, err.Error()))
		}
		if n := len(results); n != 2 {
			panic(fmt.Sprintf("expected to receive 2 results from a SCAN call, but received %d result(s)", n))
		}
		cursor, err = redis.String(results[0], nil)
		if err != nil {
			panic(fmt.Sprintf("failed to interpret cursor received from last SCAN call: %v", err))
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
			panic(fmt.Sprintf("unexpected error while scanning through unique outputs with cursor %q: invalid addresses: %v", cursor, err.Error()))
		}
		bucketCounter += len(buckets)

		for _, bucket := range buckets {
			bucketCursor := "0"
			for {
				bucketResults, err := redis.Values(conn.Do("HSCAN", bucket, bucketCursor))
				if err != nil {
					panic(fmt.Sprintf("unexpected error while scanning through outputs bucket with cursor %q: %v", cursor, err.Error()))
				}
				if n := len(bucketResults); n != 2 {
					panic(fmt.Sprintf("unexpected to receive 2 results from a HSCAN call, but received %d result(s)", n))
				}

				outputs, err := redis.StringMap(bucketResults[1], nil)
				if err != nil {
					panic(fmt.Sprintf("error while scanning through output buckets with cursor %q: invalid outputs: %v", cursor, err.Error()))
				}

				for key, value := range outputs {
					outputCounter++
					outputID := bucket[2:] + key

					// print progress
					if outputCounter%5000 == 0 {
						fmt.Printf("coin output scanner is now at coin output #%d with id %s...\n", outputCounter, outputID)
					}

					var output dtypes.CoinOutput
					err := output.LoadBytes([]byte(value))
					if err != nil {
						panic(fmt.Sprintf("unexpected error while decoding coin output %s: %v", outputID, err))
					}

					// ensure unlock hash is part of the set of all unique addresses
					addressIsKnown, err := redis.Bool(conn.Do("SISMEMBER", "addresses", output.UnlockHash.String()))
					if err != nil {
						panic(fmt.Sprintf(
							"unexpected error while checking if address %s is part of all known unique addresses: %v",
							output.UnlockHash.String(), err))
					}
					if !addressIsKnown {
						panic(fmt.Sprintf(
							"wallet %s is not part of the set of unique known addresses while it is expected to be",
							output.UnlockHash.String()))
					}

					// get wallet for given unlock hash
					var wallet types.Wallet
					addressKey, addressField := getAddressKeyAndField(output.UnlockHash)
					b, err := redis.Bytes(conn.Do("HGET", addressKey, addressField))
					if err != nil {
						if err != redis.ErrNil {
							panic(fmt.Sprintf("failed to get wallet (uh: %s) for output %s: %v", output.UnlockHash, outputID, err))
						}
						b = nil
					}
					if len(b) > 0 {
						err = encoder.Unmarshal(b, &wallet)
						if err != nil {
							panic("failed to unmarshal wallet: " + err.Error())
						}
					}

					// if the coin lock type is not nil, we want to make sure it also exists and matches our coin output
					switch output.LockType {
					case dtypes.LockTypeNone:
						// ignore
					case dtypes.LockTypeTime:
						bucketKey := getLockTimeBucketKey(output.LockValue)
						strings, err := redis.Strings(conn.Do("LRANGE", bucketKey, 0, 10000))
						var outputFound bool
						if err != redis.ErrNil {
							if err != nil {
								panic(fmt.Sprintf("error to fetch all keys in bucket %s: %v", bucketKey, err))
							}
							for _, str := range strings {
								var col dtypes.CoinOutputLock
								err := col.LoadString(str)
								if err != nil {
									panic(fmt.Sprintf("error to load locked coin output value from str %q: %v", str, err))
								}
								if col.CoinOutputID.String() == outputID {
									outputFound = true
									if col.LockValue != output.LockValue {
										panic(fmt.Sprintf("lock entry found for coin output %s, but lockvalue found(%d) != expected(%s) ",
											outputID, col.LockValue, output.LockValue))
									}
									break
								}
							}
						}
						if !outputFound {
							panic(fmt.Sprintf("no time-lock entry found for coin output %s", outputID))
						}

					case dtypes.LockTypeHeight:
						bucketKey := getLockHeightBucketKey(output.LockValue)
						strings, err := redis.Strings(conn.Do("LRANGE", bucketKey, 0, 10000))
						var outputFound bool
						if err != redis.ErrNil {
							if err != nil {
								panic(fmt.Sprintf("error to fetch all keys in bucket %s: %v", bucketKey, err))
							}
							for _, str := range strings {
								if str == outputID {
									outputFound = true
									break
								}
							}
						}
						if !outputFound {
							panic(fmt.Sprintf("no height-lock entry found for coin output %s", outputID))
						}
					}

					switch output.State {
					case dtypes.CoinOutputStateLiquid:
						// ensure output isn't listed in wallet as a locked output, it is spend and should exist
						for id := range wallet.Balance.Locked.Outputs {
							if id == outputID {
								panic(fmt.Sprintf("liquid coin output (id: %s, lockType: %d) was unexpectedly "+
									"found as locked output in wallet %s", outputID, output.LockType, output.UnlockHash))
							}
						}
						// unlocked outputs aren't checked as the listing of unlocked outputs is optional,
						// we do however want to ensure that the coin value of this doesn't exceed the total amount
						// of unlocked coins as defined in the balance of the wallet
						if wallet.Balance.Unlocked.Total.Cmp(output.CoinValue) == -1 {
							panic(fmt.Sprintf("liquid coin output (id: %s, lockType: %d) value %s exceeded unexpectedly "+
								"the total unlocked balance (%s) of wallet %s", outputID, output.LockType,
								output.CoinValue.String(), wallet.Balance.Unlocked.Total.String(), output.UnlockHash))
						}
						// add coin value of the output to the total amount unlocked coins
						unlockedCoins = unlockedCoins.Add(output.CoinValue)
					case dtypes.CoinOutputStateLocked:
						// ensure output is listed as locked in the wallet
						found := false
						for id := range wallet.Balance.Locked.Outputs {
							if id == outputID {
								found = true
								break
							}
						}
						if !found {
							panic(fmt.Sprintf("locked coin output (id: %s, lockType: %d) was unexpectedly "+
								"not found as locked output in wallet %s", outputID, output.LockType, output.UnlockHash))
						}
						// ensure the locked output does not exceed the total locked balance of the wallet
						if wallet.Balance.Locked.Total.Cmp(output.CoinValue) == -1 {
							panic(fmt.Sprintf("locked coin output (id: %s, lockType: %d) value %s exceeded unexpectedly "+
								"the total locked balance (%s) of wallet %s", outputID, output.LockType,
								output.CoinValue.String(), wallet.Balance.Unlocked.Total.String(), output.UnlockHash))
						}
						// ensure output isn't listed in wallet as an unlocked output, it is locked and should not be listed as unlocked
						for id := range wallet.Balance.Unlocked.Outputs {
							if id == outputID {
								panic(fmt.Sprintf("locked coin output (id: %s, lockType: %d) was unexpectedly "+
									"found as unlocked output in wallet %s", outputID, output.LockType, output.UnlockHash))
							}
						}
						// add coin value of the output to the total amount locked coins
						lockedCoins = lockedCoins.Add(output.CoinValue)
					case dtypes.CoinOutputStateSpent:
						// ensure output isn't listed in wallet as a locked output, it is spend and should exist
						for id := range wallet.Balance.Locked.Outputs {
							if id == outputID {
								panic(fmt.Sprintf("spent coin output (id: %s, lockType: %d) was unexpectedly "+
									"found as locked output in wallet %s", outputID, output.LockType, output.UnlockHash))
							}
						}
						// ensure output isn't listed in wallet as an unlocked output, it is spend and should exist
						for id := range wallet.Balance.Unlocked.Outputs {
							if id == outputID {
								panic(fmt.Sprintf("spent coin output (id: %s, lockType: %d) was unexpectedly "+
									"found as unlocked output in wallet %s", outputID, output.LockType, output.UnlockHash))
							}
						}
						// ignore coin value, as it is spent
					default:
						panic(fmt.Sprintf("unexpected coin output (id: %s) state for: %d", outputID, output.State))
					}
				}

				bucketCursor, err = redis.String(bucketResults[0], nil)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret cursor received from last HSCAN call: %v", err))
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

	fmt.Printf("found %d coin outputs spread over %d buckets\n", outputCounter, bucketCounter)

	totalCoins := unlockedCoins.Add(lockedCoins)

	// ensure our total coin count is as expected
	if c := lockedCoins.Cmp(stats.LockedCoins); c != 0 {
		var diff types.Currency
		switch c {
		case -1:
			diff = stats.LockedCoins.Sub(lockedCoins)
		case 1:
			diff = lockedCoins.Sub(stats.LockedCoins)
		}

		panic(fmt.Sprintf("unexpected locked coins: %s != %s (diff: %s)",
			lockedCoins.String(), stats.LockedCoins.String(), diff.String()))
	}
	if c := totalCoins.Cmp(stats.Coins); c != 0 {
		var diff types.Currency
		switch c {
		case -1:
			diff = stats.Coins.Sub(totalCoins)
		case 1:
			diff = totalCoins.Sub(stats.Coins)
		}

		panic(fmt.Sprintf("unexpected total coins: %s != %s (diff: %s)",
			totalCoins.String(), stats.Coins.String(), diff.String()))
	}

	fmt.Printf(
		"sumoutputs test —using encoding %s— on block height %d passed for %d outputs :)\n",
		encodingType.String(), stats.BlockHeight.BlockHeight, outputCounter)
}

func getAddressKeyAndField(uh types.UnlockHash) (key, field string) {
	addr := uh.String()
	key, field = "a:"+addr[:6], addr[6:]
	return
}

// getLockTimeBucketKey is an internal util function,
// used to create the timelocked bucket keys, grouping timelocked outputs within a given time range together.
func getLockTimeBucketKey(lockValue types.LockValue) string {
	return "lcos.time:" + (lockValue - lockValue%7200).String()
}

// getLockHeightBucketKey is an internal util function,
// used to create the heightlocked bucket keys, grouping all heightlocked outputs with the same lock-height value.
func getLockHeightBucketKey(lockValue types.LockValue) string {
	return "lcos.height:" + lockValue.String()
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
