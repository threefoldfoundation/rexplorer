package main

import (
	"flag"
	"fmt"
	"os"

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

	addressKey, addressField := getAddressKeyAndField(uh)
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

	// print all unlock hashes
	for _, uh := range wallet.MultiSignAddresses {
		fmt.Println("* " + uh.String())
	}
}

func getAddressKeyAndField(uh types.UnlockHash) (key, field string) {
	str := uh.String()
	key, field = "a:"+str[:6], str[6:]
	return
}

var (
	dbAddress    string
	dbSlot       int
	encodingType encoding.Type
)

func init() {
	flag.StringVar(&dbAddress, "db-address", ":6379", "(tcp) address of the redis db")
	flag.IntVar(&dbSlot, "db-slot", 0, "slot/index of the redis db")
	flag.Var(&encodingType, "encoding",
		"which encoding protocol to use, one of {json,msgp} (default: "+encodingType.String()+")")
}
