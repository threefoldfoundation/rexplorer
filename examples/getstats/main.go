package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/big"

	"github.com/rivine/rivine/pkg/client"
	"github.com/rivine/rivine/types"
	"github.com/threefoldfoundation/tfchain/pkg/config"

	"github.com/gomodule/redigo/redis"
)

func main() {
	flag.Parse()

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(err)
	}

	var (
		statsKey     string
		addressesKey string
	)
	switch networkName {
	case "standard", "testnet":
		statsKey = fmt.Sprintf("tfchain:%s:stats", networkName)
		addressesKey = fmt.Sprintf("tfchain:%s:addresses", networkName)
	default:
		panic("invalid network name: " + networkName)
	}

	b, err := redis.Bytes(conn.Do("GET", statsKey))
	if err != nil {
		panic("failed to get network stats: " + err.Error())
	}
	var stats struct {
		Timestamp              types.Timestamp   `json:"timestamp"`
		BlockHeight            types.BlockHeight `json:"blockHeight"`
		TransactionCount       uint64            `json:"txCount"`
		ValueTransactionCount  uint64            `json:"valueTxCount"`
		CointOutputCount       uint64            `json:"coinOutputCount"`
		LockedCointOutputCount uint64            `json:"lockedCoinOutputCount"`
		CointInputCount        uint64            `json:"coinInputCount"`
		MinerPayoutCount       uint64            `json:"minerPayoutCount"`
		MinerPayouts           types.Currency    `json:"minerPayouts"`
		Coins                  types.Currency    `json:"coins"`
		LockedCoins            types.Currency    `json:"lockedCoins"`
	}
	err = json.Unmarshal(b, &stats)
	if err != nil {
		panic("failed to json-unmarshal network stats: " + err.Error())
	}

	uniqueAddressCount, err := redis.Uint64(conn.Do("SCARD", addressesKey))
	if err != nil {
		panic("failed to get length of unique addresses: " + err.Error())
	}

	cfg := config.GetBlockchainInfo()
	cc := client.NewCurrencyConvertor(config.GetCurrencyUnits(), cfg.CoinUnit)

	fmt.Printf("tfchain/%s has:\n", networkName)
	liquidCoins := stats.Coins.Sub(stats.LockedCoins)
	fmt.Printf("  * a total of %s, of which %s is liquid,\n    %s is locked and %s is paid out as fees and miner rewards\n",
		cc.ToCoinStringWithUnit(stats.Coins), cc.ToCoinStringWithUnit(liquidCoins),
		cc.ToCoinStringWithUnit(stats.LockedCoins), cc.ToCoinStringWithUnit(stats.MinerPayouts))
	if !liquidCoins.IsZero() {
		lcpb := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(liquidCoins.Big()), big.NewFloat(0).SetInt(stats.Coins.Big()))
		lcpb = lcpb.Mul(lcpb, big.NewFloat(100))
		lcp, _ := lcpb.Float64()
		fmt.Printf("  * %08.5f%% liquid coins of a total of %s coins\n", lcp, cc.ToCoinStringWithUnit(stats.Coins))
	}
	if !stats.LockedCoins.IsZero() {
		lcpb := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(stats.LockedCoins.Big()), big.NewFloat(0).SetInt(stats.Coins.Big()))
		lcpb = lcpb.Mul(lcpb, big.NewFloat(100))
		lcp, _ := lcpb.Float64()
		fmt.Printf("  * %08.5f%% locked coins of a total of %s coins\n", lcp, cc.ToCoinStringWithUnit(stats.Coins))
	}
	fmt.Printf("  * a block height of %d, with the time of the highest block being %s (%d)\n",
		stats.BlockHeight, stats.Timestamp.String(), stats.Timestamp)
	fmt.Printf("  * a total of %d blocks, %d value transactions and %d coin inputs\n",
		stats.BlockHeight+1, stats.ValueTransactionCount, stats.CointInputCount)
	liquidCoinOutputCount := stats.CointOutputCount - stats.LockedCointOutputCount
	fmt.Printf("  * a total of %d coin outputs, of which %d are liquid, %d are locked and %d are payouts/fees\n",
		stats.CointOutputCount, liquidCoinOutputCount, stats.LockedCointOutputCount, stats.MinerPayoutCount)
	fmt.Printf("  * a total of %d unique wallet addresses that have been used\n", uniqueAddressCount)
	walletCoinOutputs := stats.CointOutputCount - stats.MinerPayoutCount
	fmt.Printf("  * an average of %08.5f%% wallet coin outputs per value transaction\n",
		float64(walletCoinOutputs)/float64(stats.ValueTransactionCount))
	fmt.Printf("  * an average of %08.5f%% value transactions per block\n",
		float64(stats.ValueTransactionCount)/float64(stats.BlockHeight+1))
	if liquidCoinOutputCount > 0 {
		fmt.Printf("  * %08.5f%% liquid outputs of a total of %d coin outputs\n",
			float64(liquidCoinOutputCount)/float64(stats.CointOutputCount)*100, stats.CointOutputCount)
	}
	if stats.LockedCointOutputCount > 0 {
		fmt.Printf("  * %08.5f%% locked outputs of a total of %d coin outputs\n",
			float64(stats.LockedCointOutputCount)/float64(stats.CointOutputCount)*100, stats.CointOutputCount)
	}
	if stats.ValueTransactionCount > 0 {
		fmt.Printf("  * %08.5f%% value transactions of a total of %d transactions\n",
			float64(stats.ValueTransactionCount)/float64(stats.TransactionCount)*100, stats.TransactionCount)
	}
}

var (
	dbAddress   string
	dbSlot      int
	networkName string
)

func init() {
	flag.StringVar(&dbAddress, "db-address", ":6379", "(tcp) address of the redis db")
	flag.IntVar(&dbSlot, "db-slot", 0, "slot/index of the redis db")
	flag.StringVar(&networkName, "network", "standard", "network name, one of {standard,testnet}")
}
