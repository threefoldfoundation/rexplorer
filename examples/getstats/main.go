package main

import (
	"encoding/json"
	"flag"
	"fmt"

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
		Timestamp             types.Timestamp   `json:"timestamp"`
		BlockHeight           types.BlockHeight `json:"blockHeight"`
		TransactionCount      uint64            `json:"txCount"`
		ValueTransactionCount uint64            `json:"valueTxCount"`
		CointOutputCount      uint64            `json:"coinOutputCount"`
		CointInputCount       uint64            `json:"coinInputCount"`
		MinerPayoutCount      uint64            `json:"minerPayoutCount"`
		MinerPayouts          types.Currency    `json:"minerPayouts"`
		Coins                 types.Currency    `json:"coins"`
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
	fmt.Printf("  * a total of %s, of which %s are payed out as fees and miner rewards\n",
		cc.ToCoinStringWithUnit(stats.Coins), cc.ToCoinStringWithUnit(stats.MinerPayouts))
	fmt.Printf("  * a block height of %d, with the time of the highest block being %s (%d)\n",
		stats.BlockHeight, stats.Timestamp.String(), stats.Timestamp)
	fmt.Printf("  * a total of %d blocks, %d value transactions, %d coin outputs, %d miner payouts and %d coin inputs\n",
		stats.BlockHeight+1, stats.ValueTransactionCount,
		stats.CointOutputCount, stats.MinerPayoutCount, stats.CointInputCount)
	fmt.Printf("  * a total of %d unique wallet addresses that have been used\n", uniqueAddressCount)
	fmt.Printf("  * an average of %f coin outputs per value transaction\n",
		float64(stats.CointOutputCount)/float64(stats.ValueTransactionCount))
	fmt.Printf("  * an average of %f value transactions per block\n",
		float64(stats.ValueTransactionCount)/float64(stats.BlockHeight+1))
	fmt.Printf("  * %f%% value transactions of a total of %d transactions\n",
		float64(stats.ValueTransactionCount)/float64(stats.TransactionCount)*100, stats.TransactionCount)

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
