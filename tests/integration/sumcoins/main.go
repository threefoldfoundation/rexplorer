package main

import (
	"flag"
	"fmt"

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

	// get all unique addresses
	addresses, err := redis.Strings(conn.Do("SMEMBERS", "addresses"))
	if err != nil {
		panic("failed to get all unique addresses: " + err.Error())
	}

	// compute total unlocked and locked coins for all addresses
	var unlockedCoins, lockedCoins types.Currency
	for _, addr := range addresses {
		var wallet types.Wallet
		addressKey, addressField := getAddressKeyAndField(addr)
		b, err := redis.Bytes(conn.Do("HGET", addressKey, addressField))
		if err != nil {
			if err != redis.ErrNil {
				panic("failed to get wallet " + err.Error())
			}
			b = nil
		}
		if len(b) > 0 {
			err = encoder.Unmarshal(b, &wallet)
			if err != nil {
				panic("failed to json-unmarshal wallet: " + err.Error())
			}
		}
		// keep track of total locked coins and unlocked coins
		unlockedCoins = unlockedCoins.Add(wallet.Balance.Unlocked.Total)
		lockedCoins = lockedCoins.Add(wallet.Balance.Locked.Total)

		// ensure that the sum of all unlocked outputs is not greater than the total unlocked balance of each wallet
		if !wallet.Balance.Unlocked.Total.IsZero() {
			var unlockedSum types.Currency
			for _, output := range wallet.Balance.Unlocked.Outputs {
				unlockedSum = unlockedSum.Add(output.Amount)
			}
			if c := unlockedSum.Cmp(wallet.Balance.Unlocked.Total); c == 1 {
				diff := unlockedSum.Sub(wallet.Balance.Unlocked.Total)
				panic(fmt.Sprintf("unexpected total unlocked of wallet %s coins: %s > %s (diff: %s)",
					addr, unlockedSum.String(), wallet.Balance.Unlocked.Total.String(), diff.String()))
			}
		}

		// ensure that the sum of all locked outputs equals the total locked balance of each wallet
		if !wallet.Balance.Locked.Total.IsZero() {
			var lockedSum types.Currency
			for _, output := range wallet.Balance.Locked.Outputs {
				lockedSum = lockedSum.Add(output.Amount)
			}
			if c := lockedSum.Cmp(wallet.Balance.Locked.Total); c != 0 {
				var diff types.Currency
				switch c {
				case -1:
					diff = wallet.Balance.Locked.Total.Sub(lockedSum)
				case 1:
					diff = lockedSum.Sub(wallet.Balance.Locked.Total)
				}

				panic(fmt.Sprintf("unexpected total locked of wallet %s coins: %s != %s (diff: %s)",
					addr, lockedSum.String(), wallet.Balance.Locked.Total.String(), diff.String()))
			}
		}

	}
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
		"sumcoins test —using encoding %s— on block height %d passed :)\n",
		encodingType.String(), stats.BlockHeight.BlockHeight)
}

func getAddressKeyAndField(addr string) (key, field string) {
	key, field = "a:"+addr[:6], addr[6:]
	return
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
		"which encoding protocol to use, one of {json,msgp} (default: "+encodingType.String()+")")
}
