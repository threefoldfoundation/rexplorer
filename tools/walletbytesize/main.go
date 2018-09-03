package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"

	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	"github.com/gomodule/redigo/redis"
)

func main() {
	flag.Parse()

	encoder, err := encoding.NewEncoder(encodingType)
	onError(err)

	args := flag.Args()
	if len(args) != 0 {
		panic("usage: " + os.Args[0])
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	onError(err)

	individualWalletStats := newSizeStatCollector()
	multisigWalletStats := newSizeStatCollector()

	// go through all unique wallet addresses
	cursor := "0"
	var cycles uint64
	for {
		cycles++

		// get results of current cursor
		pair, err := redis.Values(conn.Do("SSCAN", "addresses", cursor))
		onRedisErr(err)

		// get all addresses
		addresses, err := redis.Strings(pair[1], nil)
		onRedisErr(err)
		for _, addr := range addresses {
			addressKey, addressField := getAddressKeyAndField(addr)
			var wallet types.Wallet
			b, err := redis.Bytes(conn.Do("HGET", addressKey, addressField))
			if err != nil {
				if err != redis.ErrNil {
					panic("failed to get wallet " + err.Error())
				}
				continue // filter out nil bytes
			}

			err = encoder.Unmarshal(b, &wallet)
			onError(err)

			byteSize := int64(len(b))
			if wallet.MultiSignData.SignaturesRequired == 0 {
				individualWalletStats.Track(byteSize)
			} else {
				multisigWalletStats.Track(byteSize)
			}
		}

		// store returned cursor, and exit if we're at "0" again
		cursor, err = redis.String(pair[0], nil)
		onError(err)
		if cursor == "0" {
			log.Printf("scanned wallet addresses in %d SSCAN cycles", cycles)
			break // done
		}
	}

	fmt.Println("                       average       min          max          total")
	avg, min, max, total := individualWalletStats.Stats()
	fmt.Printf("Individual Wallets     %-10d    %-10d   %-10d   %-10d\r\n", avg, min, max, total)
	avg, min, max, total = multisigWalletStats.Stats()
	fmt.Printf("MultiSig Wallets       %-10d    %-10d   %-10d   %-10d\r\n", avg, min, max, total)
}

type sizeStatCollector struct {
	total    *big.Int
	count    int64
	min, max int64
}

func newSizeStatCollector() *sizeStatCollector {
	return &sizeStatCollector{
		total: new(big.Int),
		count: 0,
		min:   math.MaxInt64,
		max:   -1,
	}
}

func (sc *sizeStatCollector) Track(size int64) {
	sc.count++
	if size < sc.min {
		sc.min = size
	}
	if size > sc.max {
		sc.max = size
	}
	sc.total.Add(sc.total, big.NewInt(size))
}

func (sc *sizeStatCollector) Stats() (int64, int64, int64, int64) {
	if sc.count == 0 {
		return 0, 0, 0, 0
	}
	avg, _ := new(big.Float).Quo(new(big.Float).SetInt(sc.total), new(big.Float).SetInt64(sc.count)).Int64()
	return avg, sc.min, sc.max, sc.total.Int64()
}

func getAddressKeyAndField(addr string) (key, field string) {
	key, field = "a:"+addr[:6], addr[6:]
	return
}

func onRedisErr(err error) {
	if err == redis.ErrNil {
		return
	}
	onError(err)
}
func onError(err error) {
	if err != nil {
		panic(err)
	}
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
		"which encoding protocol to use, one of {json,msgp,protobuf} (default: "+encodingType.String()+")")
}
