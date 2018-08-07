package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rivine/rivine/types"

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

	var multiSigAddressesKey string
	switch networkName {
	case "standard", "testnet":
		multiSigAddressesKey = fmt.Sprintf("tfchain:%s:address:%s:multisig.addresses", networkName, uh.String())
	default:
		panic("invalid network name: " + networkName)
	}

	// get all addresses
	ss, err := redis.Strings(conn.Do("SMEMBERS", multiSigAddressesKey))
	if err != nil {
		if err != redis.ErrNil {
			panic("failed to get multisig addresses " + err.Error())
		}
		fmt.Println("no multisig addresses found")
		return
	}

	// parse all unlock hashes first, so we can panic prior to printing, should an unlock hash be invalid
	uhs := make([]types.UnlockHash, len(ss))
	for i, s := range ss {
		err = uhs[i].LoadString(s)
		if err != nil {
			panic(fmt.Sprintf("received invalid multisig address %q: %v", s, err))
		}
	}
	// print all unlock hashes
	for _, uh := range uhs {
		fmt.Println("* " + uh.String())
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
