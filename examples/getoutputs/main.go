package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/threefoldfoundation/tfchain/pkg/config"
	"github.com/threefoldtech/rivine/pkg/client"

	"github.com/threefoldfoundation/rexplorer/pkg/database"
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

	args := flag.Args()
	if len(args) != 1 {
		panic("usage: " + os.Args[0] + " <unlockhash>")
	}
	var uh types.UnlockHash
	err = uh.LoadString(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "usage: "+os.Args[0]+" <unlockhash>")
		panic(fmt.Sprintf("invalid uh %q: %v", args[0], err))
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(err)
	}

	addressKey, addressField := database.GetAddressKeyAndField(uh)

	var wallet types.Wallet
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
			panic("failed to unmarshal wallet: " + err.Error())
		}
	}

	cfg := config.GetBlockchainInfo()
	cc := client.NewCurrencyConvertor(config.GetCurrencyUnits(), cfg.CoinUnit)

	haveOutputs := false

	if n := len(wallet.Balance.Unlocked.Outputs); n > 0 {
		haveOutputs = true
		fmt.Printf("Unlocked Outputs (%d):\n", n)
		fmt.Println()
		for id, output := range wallet.Balance.Unlocked.Outputs {
			fmt.Printf("%s  %- 15s  %s\n",
				id, cc.ToCoinStringWithUnit(output.Amount.Currency),
				output.Description)
		}
		fmt.Println()
	}
	if n := len(wallet.Balance.Locked.Outputs); n > 0 {
		haveOutputs = true
		fmt.Printf("Locked Outputs (%d):\n", n)
		fmt.Println()
		for id, output := range wallet.Balance.Locked.Outputs {
			fmt.Printf("%s  %- 15s  %- 10s  %s\n",
				id, cc.ToCoinStringWithUnit(output.Amount.Currency),
				output.LockedUntil.String(), output.Description)
		}
		fmt.Println()
	}
	if !haveOutputs {
		fmt.Println("no outputs could be found for the given wallet")
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
