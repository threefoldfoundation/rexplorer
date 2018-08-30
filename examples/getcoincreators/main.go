package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gomodule/redigo/redis"
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 0 {
		panic("usage: " + os.Args[0])
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(err)
	}

	addresses, err := redis.Strings(conn.Do("SMEMBERS", "coincreators"))
	if err != nil {
		if err != redis.ErrNil {
			panic("failed to get coin creators " + err.Error())
		}
		addresses = nil
	}
	if len(addresses) == 0 {
		panic("no coin creators found")
	}

	// print all unlock hashes
	for idx, address := range addresses {
		fmt.Printf("%d) %s\r\n", idx+1, address)
	}
}

var (
	dbAddress string
	dbSlot    int
)

func init() {
	flag.StringVar(&dbAddress, "db-address", ":6379", "(tcp) address of the redis db")
	flag.IntVar(&dbSlot, "db-slot", 0, "slot/index of the redis db")
}
