package main

import (
	"flag"
	"fmt"
	"math/big"

	"github.com/threefoldfoundation/rexplorer/pkg/database"
	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	"github.com/threefoldfoundation/tfchain/pkg/config"
	"github.com/threefoldtech/rivine/pkg/client"

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

	b, err := redis.Bytes(conn.Do("GET", database.StatsKey))
	if err != nil {
		panic("failed to get network stats: " + err.Error())
	}
	var stats types.NetworkStats
	err = encoder.Unmarshal(b, &stats)
	if err != nil {
		panic("failed to unmarshal network stats: " + err.Error())
	}

	uniqueAddressCount, err := redis.Uint64(conn.Do("SCARD", database.AddressesKey))
	if err != nil {
		panic("failed to get length of unique addresses: " + err.Error())
	}

	cfg := config.GetBlockchainInfo()
	cc := client.NewCurrencyConvertor(config.GetCurrencyUnits(), cfg.CoinUnit)

	fmt.Println("tfchain network has:")
	liquidCoins := stats.Coins.Sub(stats.LockedCoins)
	fmt.Printf("  * a total of %s, of which %s is liquid,\n    %s is locked, %s is paid out as miner payouts\n    and %s is paid out as tx fees\n",
		cc.ToCoinStringWithUnit(stats.Coins.Currency), cc.ToCoinStringWithUnit(liquidCoins.Currency),
		cc.ToCoinStringWithUnit(stats.LockedCoins.Currency), cc.ToCoinStringWithUnit(stats.MinerPayouts.Currency),
		cc.ToCoinStringWithUnit(stats.TransactionFees.Currency))
	if !liquidCoins.IsZero() {
		lcpb := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(liquidCoins.Big()), big.NewFloat(0).SetInt(stats.Coins.Big()))
		lcpb = lcpb.Mul(lcpb, big.NewFloat(100))
		lcp, _ := lcpb.Float64()
		fmt.Printf("  * %08.5f%% liquid coins of a total of %s coins\n", lcp, cc.ToCoinStringWithUnit(stats.Coins.Currency))
	}
	if !stats.LockedCoins.IsZero() {
		lcpb := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(stats.LockedCoins.Big()), big.NewFloat(0).SetInt(stats.Coins.Big()))
		lcpb = lcpb.Mul(lcpb, big.NewFloat(100))
		lcp, _ := lcpb.Float64()
		fmt.Printf("  * %08.5f%% locked coins of a total of %s coins\n", lcp, cc.ToCoinStringWithUnit(stats.Coins.Currency))
	}
	fmt.Printf("  * a total of %d transactions, of which %d wallet-value transactions,\n    %d coin creation transactions, %d coin creator definition transactions,\n    %d 3Bot registration transactions, %d 3Bot update transactions\n    and %d are pure block creation transactions\n",
		stats.TransactionCount,
		stats.ValueTransactionCount-stats.ThreeBotRegistrationTransactionCount-stats.ThreeBotUpdateTransactionCount,
		stats.CoinCreationTransactionCount, stats.CoinCreatorDefinitionTransactionCount,
		stats.ThreeBotRegistrationTransactionCount, stats.ThreeBotUpdateTransactionCount,
		stats.TransactionCount-stats.ValueTransactionCount-stats.CoinCreationTransactionCount-stats.CoinCreatorDefinitionTransactionCount)
	fmt.Printf("  * a block height of %d, with the time of the highest block\n    being %s (%d)\n",
		stats.BlockHeight.BlockHeight, stats.Timestamp.String(), stats.Timestamp.Timestamp)
	fmt.Printf("  * a total of %d blocks, %d value transactions and %d coin inputs\n",
		stats.BlockHeight.BlockHeight+1, stats.ValueTransactionCount, stats.CoinInputCount)
	liquidCoinOutputCount := stats.CoinOutputCount - stats.LockedCoinOutputCount
	valueCoinOutputs := stats.CoinOutputCount - stats.MinerPayoutCount - stats.TransactionFeeCount
	fmt.Printf("  * a total of %d coin outputs, of which %d are liquid, %d are locked,\n    %d transfer value, %d are miner payouts and %d are tx fees\n",
		stats.CoinOutputCount, liquidCoinOutputCount, stats.LockedCoinOutputCount,
		valueCoinOutputs, stats.MinerPayoutCount, stats.TransactionFeeCount)
	fmt.Printf("  * a total of %d unique addresses that have been used\n", uniqueAddressCount)
	if stats.ValueTransactionCount > 0 {
		if valueCoinOutputs > 0 {
			fmt.Printf("  * an average of %08.5f%% value coin outputs per value transaction\n",
				float64(valueCoinOutputs)/float64(stats.ValueTransactionCount))
		}
		fmt.Printf("  * an average of %08.5f%% value transactions per block\n",
			float64(stats.ValueTransactionCount)/float64(stats.BlockHeight.BlockHeight+1))
	}
	if liquidCoinOutputCount > 0 {
		fmt.Printf("  * %08.5f%% liquid outputs of a total of %d coin outputs\n",
			float64(liquidCoinOutputCount)/float64(stats.CoinOutputCount)*100, stats.CoinOutputCount)
	}
	if stats.LockedCoinOutputCount > 0 {
		fmt.Printf("  * %08.5f%% locked outputs of a total of %d coin outputs\n",
			float64(stats.LockedCoinOutputCount)/float64(stats.CoinOutputCount)*100, stats.CoinOutputCount)
	}
	if stats.ValueTransactionCount > 0 {
		fmt.Printf("  * %08.5f%% value transactions of a total of %d transactions\n",
			float64(stats.ValueTransactionCount)/float64(stats.TransactionCount)*100, stats.TransactionCount)
	}
	if stats.CoinCreationTransactionCount > 0 {
		fmt.Printf("  * %08.5f%% coin creation transactions of a total of %d transactions\n",
			float64(stats.CoinCreationTransactionCount)/float64(stats.TransactionCount)*100, stats.TransactionCount)
	}
	if stats.CoinCreatorDefinitionTransactionCount > 0 {
		fmt.Printf("  * %08.5f%% coin creator definition transactions of a total of %d transactions\n",
			float64(stats.CoinCreatorDefinitionTransactionCount)/float64(stats.TransactionCount)*100, stats.TransactionCount)
	}
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
