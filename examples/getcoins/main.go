package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rivine/rivine/pkg/client"
	"github.com/rivine/rivine/types"
	"github.com/threefoldfoundation/tfchain/pkg/config"

	"github.com/gomodule/redigo/redis"
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		panic("usage: " + os.Args[0] + " <unlockhash>")
	}
	var uh types.UnlockHash
	err := uh.LoadString(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "usage: "+os.Args[0]+" <unlockhash>")
		panic(fmt.Sprintf("invalid uh %q: %v", args[0], err))
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(err)
	}

	balanceKey := fmt.Sprintf("address:%s:balance", uh.String())

	var balance struct {
		Locked   types.Currency `json:"locked"`
		Unlocked types.Currency `json:"unlocked"`
	}
	b, err := redis.Bytes(conn.Do("GET", balanceKey))
	if err != nil {
		if err != redis.ErrNil {
			panic("failed to get balance " + err.Error())
		}
		b = []byte("{}")
	}
	err = json.Unmarshal(b, &balance)
	if err != nil {
		panic("failed to json-unmarshal network stats: " + err.Error())
	}

	cfg := config.GetBlockchainInfo()
	cc := client.NewCurrencyConvertor(config.GetCurrencyUnits(), cfg.CoinUnit)
	fmt.Println("unlocked: " + cc.ToCoinStringWithUnit(balance.Unlocked))
	fmt.Println("locked:   " + cc.ToCoinStringWithUnit(balance.Locked))
	fmt.Println("--------------------")
	fmt.Println("total: " + cc.ToCoinStringWithUnit(balance.Locked.Add(balance.Unlocked)))

}

var (
	dbAddress string
	dbSlot    int
)

func init() {
	flag.StringVar(&dbAddress, "db-address", ":6379", "(tcp) address of the redis db")
	flag.IntVar(&dbSlot, "db-slot", 0, "slot/index of the redis db")
}
