package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
		panic("usage: " + os.Args[0] + " <botID>")
	}
	var id types.BotID
	err = id.LoadString(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "usage: "+os.Args[0]+" <BotID>")
		panic(fmt.Sprintf("invalid botID %q: %v", args[0], err))
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(err)
	}

	recordKey, recordField := database.GetThreeBotKeyAndField(id)

	b, err := redis.Bytes(conn.Do("HGET", recordKey, recordField))
	if err != nil {
		if err != redis.ErrNil {
			panic("failed to get bot record " + err.Error())
		}
		b = nil
	}
	var record types.BotRecord
	err = encoder.Unmarshal(b, &record)
	if err != nil {
		panic("failed to unmarshal bot record: " + err.Error())
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	err = e.Encode(record)
	if err != nil {
		panic("failed to print bot record: " + err.Error())
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
