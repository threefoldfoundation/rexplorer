# rexplorer

[![Build Status](https://travis-ci.org/threefoldfoundation/rexplorer.svg?branch=master)](https://travis-ci.org/threefoldfoundation/rexplorer)
[![GoDoc](https://godoc.org/github.com/threefoldfoundation/rexplorer?status.svg)](https://godoc.org/github.com/threefoldfoundation/rexplorer)
[![Go Report Card](https://goreportcard.com/badge/github.com/threefoldfoundation/rexplorer)](https://goreportcard.com/report/github.com/threefoldfoundation/rexplorer)

`rexplorer` is a small explorer binary, that can aid in exploring a [tfchain][tfchain] network.
It applies/reverts data —received from an embedded consensus module— into a Redis db of choice,
such that the tfchain network data can be consumed/used in a meaningful way.

The public data (data meant for consumption) stored into Redis is stored using a public encoding format.
The available encoding formats are [MessagePack][encoding-msgp], [JSON][encoding-json] and [Protocol Buffers][encoding-pb].
Consequently you should be able to consume the public data using any (programming) language which
supports the desired format.

> For [Golang][golang] you can use the
> [`github.com/threefoldfoundation/rexplorer/pkg/database`](https://godoc.org/github.com/threefoldfoundation/rexplorer/pkg/database),
> [`github.com/threefoldfoundation/rexplorer/pkg/encoding`](https://godoc.org/github.com/threefoldfoundation/rexplorer/pkg/encoding) and
> [`github.com/threefoldfoundation/rexplorer/pkg/types`](https://godoc.org/github.com/threefoldfoundation/rexplorer/pkg/types) in order to easily consume the data.

> [Python 3][python] is supported as well, in the form of an integration test, where it is tested for all available
> encoding formats. You can find the source code for
> that integration test at
> [./tests/integration/sumcoins/main.py](./tests/integration/sumcoins/main.py).
>
> For this integration test we used the following [pip](http://pip.pypa.io) packages:
> [msgpack@0.5.6](https://pypi.org/project/msgpack/), [redis@2.10.6](https://pypi.org/project/redis/)
> and [protobuf@3.6.1](https://pypi.org/project/protobuf/).

[golang]: http://golang.org
[python]: http://python.org

## Index

* [Install](#install): how to install `rexplorer`;
* [Usage](#usage): how to use `rexplorer`;
* [Reserved Redis Keys](#reserved-redis-keys): explains which keys are used to store values in a Redis Database by a `rexplorer` instance;
* [Encoding](#encoding): explains about the different encoding protocols supported by `rexplorer` (make sure to also read the last part of the [Reserved Redis Keys](#reserved-redis-keys) chapter as it explains the structure of the encoded values);
  * [Statistics](#statistics): statistics to put the characteristics of the supported encoding protocols into context (explains also how these statistics are gathered);
* [Examples](#examples): shows some example use cases of `rexplorer` using Go as well as directly with the official `redis-cli` tool;
* [Testing](#testing): explains how the `rexplorer` codebase and tool is tested, as well as how you can run these tests yourself.

## Install

```
$ go get -u github.com/threefoldfoundation/rexplorer && rexplorer version
Tool version            v0.2.1
TFChain Daemon version  v1.1.0
Rivine protocol version v1.0.7

Go Version   v1.11
GOOS         darwin
GOARCH       amd64
```

> `rexplorer` supports Go 1.9 and above. Older versions of Golang may work but aren't supported.

## Usage

To start a `rexplorer` instance for the standard network,
storing all persistent non-redis data into a sub directory of the current root directory,
you can do so as simple as:

```
$ rexplorer
2018/08/30 22:07:02 starting rexplorer v0.1.2-c7471a3...
2018/08/30 22:07:02 loading network config, registering types and loading rivine transaction db (0/3)...
2018/08/30 22:07:02 loading rivine gateway module (1/3)...
2018/08/30 22:07:02 loading rivine consensus module (2/3)...
2018/08/30 22:07:02 loading internal explorer module (3/3)...
2018/08/30 22:10:36 rexplorer is up and running...
```

The persistent dir (used for some local BoltDB consensus/gateway data) can be changed
using the `-d`/`--persistent-directory` flag.

Should you want to explore `testnet` instead of the `standard` net you can use the `--network testnet` flag.

By default [MessagePack][encoding-msgp] is used to encode all public data in the Redis Database,
should you want to use [JSON][encoding-json] instead you can use the `--encoding json` flag.
[Protocol Buffers][encoding-pb] is available as well and can be used using the `--encoding protobuf` flag.

For more information use the `--help` flag:

```
$ rexplorer --help
start the rexplorer daemon

(*) Extra Information for the '-f/--filter' flag:
  The following kind of glob expressions can
  be made for the optional description filters:
  - Multiple Characters Wildcard: *
        example: 'bar:*' // matches 'bar:', 'bar: foo', 'bar:ok', ...

  - Single Character Wildcard: ?
        example: 'h?' // matches 'ho', 'ha', 'hi', 'hh', ...

  - Allow any character from a given character list: [xy]
        example: 'b[aoi]t' // matches 'bot', 'bit', 'bat'

  - Allow any character from a given character range: [x-y]
        example: 'a[a-c]a' // matches 'aaa', 'aba', 'aca'

  - Allow any character except a given character list: [!xy]
        example: 'a[!ac]ba' // does not match 'acba',
                            // but does match 'abba', 'adba', ...

  - Allow any character except a character within a given character range: [!x-y]
        example: '[!a-f]oo' // does not match 'foo', 'boo', ...
                            // but does match 'woo', 'zoo', 5oo, ...

Usage:
  rexplorer [flags]
  rexplorer [command]

Available Commands:
  help        Help about any command
  version     show versions of this tool

Flags:
  -e, --encoding EncodingType             which encoding protocol to use, one of {json,msgp,protobuf} (default msgp)
  -f, --filter DescriptionFilterSetFlag   list unlocked outputs in wallets if the description of output matches any of the unique glob (*) filters
  -h, --help                              help for rexplorer
  -n, --network string                    the name of the network to which the daemon connects, one of {standard,testnet} (default "standard")
  -d, --persistent-directory string       location of the root diretory used to store persistent data of the daemon of tfchain
      --profile-addr string               enables profiling of this rexplorer instance as an http service
      --redis-addr string                 which (tcp) address the redis server listens on (default ":6379")
      --redis-db int                      which redis database slot to use
      --rpc-addr string                   which port the gateway listens on (default ":23112")

Use "rexplorer [command] --help" for more information about a command.
```

## Reserved Redis Keys

Ideally you use a Redis database (slot) just for the `rexplorer` instance.
However should you not be able to allocate an entire database (slot) just for the `rexplorer instance`,
please do not ever touch the reserved keys. You'll break your own explored data should you write/delete any values
stored directly or indirectly of a reserved key.

There are two types of keys:

* internal keys: these are keys which are meant for internals of the `rexplorer` instance, and are not meant for public consumption;
* public keys: these are keys meant for public consumption and have a well-defined format;

Following _internal_ keys are reserved:

* `internal`:
    * used for internal state of this explorer
    * format value: [Redis HASHMAP][redistypes], where the keys have different types and are not to be touched
* `c:<4_random_hex_chars>`:
    * all coin outputs, and for each coin output only the info which is required for the inner workings of the `rexplorer`
    * format value: custom
    * example key: `c:6986`
* `lcos.height:<height>`:
    * all locked coin outputs on a given height
    * format value: custom
    * example key: `lcos.height:42`
* `lcos.time:<timestamp-(timestamp%7200)>`:
    * all locked coin outputs for a given timestamp range
    * format value: custom
    * example key: `lcos.time:1526335200`

Following _public_ keys are reserved:

* `stats`:
    * used for global network statistics
    * format value: JSON/MessagePack
* `coincreators`:
    * used to store the unique wallet addresses of the current coin creators
    * format value: [Redis SET][redistypes], where each value is a [Rivine][rivine]-defined hex-encoded UnlockHash
* `addresses`:
    * set of unique wallet addresses used (even if reverted) in the network
    * format value: [Redis SET][redistypes], where each value is a [Rivine][rivine]-defined hex-encoded UnlockHash
* `a:01<4_random_hex_chars>`:
    * used by all wallet addresses, contains unlocked balance, locked balance and coin outputs as well as all multisig wallets jointly owned by this wallet
    * format value: [Redis HASHMAP][redistypes], where each field's value is a JSON/MessagePack/Protobuf
    * example key: `a:012b61`
    * fields have the format `<72_random_hex_chars>`, an example: `389e7f103288371830c632439fe709044c3ab5c374947ab4eca68ee987d3f736b360e530`
* `a:03<4_random_hex_chars>`:
    * used by all multisig wallet addresses, contains unlocked balance, locked balance and coin outputs as well as owner addresses and signatures required
    * format value: [Redis HASHMAP][redistypes], where each field's value is a JSON/MessagePack/ProtoBuf
    * example key: `a:032b61`
    * fields have the format `<72_random_hex_chars>`, an example: `389e7f103288371830c632439fe709044c3ab5c374947ab4eca68ee987d3f736b360e530`
* `e:<6_random_hex_chars>`:
    * used by all registered ERC20 Address, containing the mapped TFT wallet address as value
    * format value: [Redis HASHMAP][redistypes], where each field's value is a JSON/MessagePack/ProtoBuf
    * example key: `e:512b61`
    * fields have the format `<34_random_hex_chars>`, an example: `389e7f103288371830c632439fe709044c`
* `b:<1+_random_digits>`:
    * used by all 3Bots, containing a hashmap of maximum 100 3bot records;
    * format value: [Redis HASHMAP][redistypes], where each field's value is a JSON/MessagePack/Protobuf of a 3Bot record
    * example keys: `b:0`, `b12`, `b:1234`
    * fields have the format `<1_or_2_random_digits>`, examples: `0`, `1`, `99`

Rivine (Primitive) Value Encodings:

String/Text encodings used for [JSON][encoding-json] encoding:
* addresses are Hex-encoded and the exact format (and how it is created) is described in:
  <https://github.com/threefoldtech/rivine/blob/master/doc/transactions/unlockhash.md#textstring-encoding>
* currencies are encoded as described in <https://godoc.org/math/big#Int.Text>
  using base 10, and using the smallest coin unit as value (e.g. 10^-9 TFT)

String/Text encodings used for all encodings:
* coin output identifiers are hex-encoded versions of the 32 byte hash, forming a 64 byte string as a result;

Binary encodings used for [MessagePack][encoding-msgp] and [Protocol Buffers][encoding-pb]:
* addresses are binary encoded and the exact format (and how it is created) is described in:
  <https://github.com/threefoldtech/rivine/blob/master/doc/transactions/unlockhash.md#binary-encoding>
* currencies are encoded as described in <https://godoc.org/math/big#Int.Bytes>
  using base 10, in Big-Endian order, and using the smallest coin unit as value (e.g. 10^-9 TFT)

TFChain (Primitive) Value Encodings (applies to 3Bot-related content only):

Binary encodings used for [MessagePack][encoding-msgp] and [Protocol Buffers][encoding-pb]:
* sorted sets of network addresses are encoded in a tfchain-defined encoding format. It is encoded as a variable-sized slice
  with each element being a binary-encoded network address:
  * You can read about how variable-sized slices are encoded at: <https://github.com/threefoldfoundation/tfchain/blob/master/doc/binary_encoding.md#standard-encoding>;
  * You can read about how network addresses are encoded at:
  <https://github.com/threefoldfoundation/tfchain/blob/master/doc/binary_encoding.md#Network-Address>;
* sorted sets of bot names are encoded in a tfchain-defined encoding format. It is encoded as a variable-sized slice
  with each element being a binary-encoded bot name:
  * You can read about how bot names are encoded at:
  <https://github.com/threefoldfoundation/tfchain/blob/master/doc/3bot.md#bot-name>;
* bot identifiers and compact time stamps (expiration time) are encoded as uint32, using the standard encoding protocol
  ([MessagePack][encoding-msgp] or [Protocol Buffers][encoding-pb]);
* public keys are encoded using a 1-byte prefix (read more about it
  <https://github.com/threefoldfoundation/tfchain/blob/master/doc/binary_encoding.md#Public-Key>),
  with the prefix indicating the signature algorithm, and the rest of the bytes being the actual public key as a raw byte slice.


[JSON][encoding-json] formats of value types defined by this module:

* example of global stats (stored under `stats`):

```json
{
	"timestamp": 1535661244,
	"blockHeight": 103481,
	"txCount": 103830,
	"coinCreationTxCount": 2,
	"coinBurnTxCount": 2,
	"coinCreatorDefinitionTxCount": 1,
	"botRegistrationTxCount": 3402,
	"botUpdateTxCount": 100,
	"valueTxCount": 348,
	"coinOutputCount": 104414,
	"lockedCoinOutputCount": 736,
	"coinInputCount": 1884,
	"minerPayoutCount": 103481,
	"txFeeCount": 306,
    "foundationFeeCount": 500,
	"minerPayouts": "1034810000000000",
	"txFees": "36100000071",
    "foundationFees": "5230100000000",
	"coins": "101054810300000000",
	"lockedCoins": "8045200000000"
}
```

> When using MessagePack (the default encoding type), the keys are the same as when encoding as JSON,
> and the values are encoded in the exact same way, except that the resulting values
> follow the [MessagePack spec][msgp-spec].

* example of a wallet (stored under `a:01<4_random_hex_chars>`):

```javascript
{
    "balance": {
        "unlocked": {
            // NOTE that the total unlocked balance does not have to
            // match the sum of the listed unlocked outputs, due to the fact
            // that unlocked outputs are only shown if their description
            // matches any of the (CLI) specified description filters
            "total": "10005000",
            "outputs": [
                {
                    "amount": "9999999",
                    "description": "for:you"
                },
                {
                    "amount": "1",
                    "description": "Surprise!"
                }
            ],
        },
        "locked": {
            // the total locked balance will always match the sum
            // of all listed locked outputs
            "total": "5000",
            "outputs": [
                {
                    "amount": "2000",
                    "lockedUntil": 1534105468
                },
                {
                    "amount": "3000",
                    "lockedUntil": 1534105468,
                    "description": "SGVsbG8=",
                }
            ]
        }
    },
    "multisignaddresses": [
        "0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"
    ]
}
```

Null fields will not be encoded in JSON encoding.

* example of a multisig wallet (stored under `a:03<4_random_hex_chars>`):

```json
{
    "balance": {
        "unlocked": {
            "total": "10000000"
        }
    },
    "multisign": {
        "owners": [
            "01b650391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa",
            "0114df42a3bb8303a745d23c47062a1333246b3adac446e6d62f4de74f5223faf4c2da465e76af"
        ],
        "signaturesRequired": 1
    }
}
```

* example of a 3Bot record (stored under `b:<1+_random_digits>` `<1_or_2_random_digits>`):

```json
{
    "id": 1,
    "addresses":["example.com","91.198.174.192"],
    "names": ["thisis.mybot", "voicebot.example", "voicebot.example.myorg"],
    "publickey": "ed25519:00bde9571b30e1742c41fcca8c730183402d967df5b17b5f4ced22c677806614",
	"expiration": 1542815220
}
```

[MessagePack][encoding-msgp]:

When using `MessagePack` encoding the Layout is pretty much the same,
except that the keys are different and the values are encoded following the [MessagePack spec][msgp-spec].

Here is how in a JSON Layout the keys are of the different [MessagePack][encoding-msgp] structured values:

```javascript
{
	"cts": 1535661244, // chain timestamp
	"cbh": 103481, // chain blockheight
	"txc": 103830, // transaction count
	"cctxc": 2, // coin creation transaction count
	"ccdtxc": 1, // coin creator definition transaction count
	"tbrtxc": 201, // three bot registration transaction count
	"tbutxc": 10, // three bot update transaction count
	"vtxc": 348, // value transaction count
	"coc": 104414, // coin output count
	"lcoc": 736, // locked coin output count
	"cic": 1884, // coin input count
	"mpc": 103481, // miner payout count
	"txfc": 306, // transaction fee count
	"mpt": "1034810000000000", // miner payouts total
	"txft": "36100000071", // transaction fees total
	"ct": "101054810300000000", // coins total
	"lct": "8045200000000" // locked coins total
}
```

```javascript
{
    "b": { // balance
        "u": { // unlocked
            // NOTE that the total unlocked balance does not have to
            // match the sum of the listed unlocked outputs, due to the fact
            // that unlocked outputs are only shown if their description
            // matches any of the (CLI) specified description filters
            "t": "10005000", // total
            "o": [ // outputs
                {
                    "a": "9999999", // amount
                    "d": "for:you" // description
                },
                {
                    "a": "1", // amount
                    "d": "Surprise!" // description
                }
            ],
        },
        "l": { // locked
            // the total locked balance will always match the sum
            // of all listed locked outputs
            "t": "5000", // total
            "o": [ // outputs
                {
                    "a": "2000", // amount
                    "lu": 1534105468 // locked until
                },
                {
                    "a": "3000", // amount
                    "lu": 1534105468, // locked until
                    "d": "SGVsbG8=", // description
                }
            ]
        }
    },
    "ma": [ // MultiSignAddresses
        "0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"
    ]
}
```

```javascript
{
    "b": { // balance
        "u": { // unlocked
            "t": "10000000" // total
        }
    },
    "m": { // multisign
        "o": [ // owners
            "01b650391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa",
            "0114df42a3bb8303a745d23c47062a1333246b3adac446e6d62f4de74f5223faf4c2da465e76af"
        ],
        "sr": 1 // signatures required
    }
}
```

```javascript
{
    "i": 1,
    "a":["example.com","91.198.174.192"],
    "n": ["thisis.mybot", "voicebot.example", "voicebot.example.myorg"],
    "k": "ed25519:00bde9571b30e1742c41fcca8c730183402d967df5b17b5f4ced22c677806614",
    "e": 1542815220
}
```

[Protocol Buffers][encoding-pb]:

> When using ProtocolBuffer (an optional encoding type, less widely supported but faster),
> you'll have to decode using the `types.proto` Protocol Buffer scheme of public types found at
> [./pkg/types/types.proto](./pkg/types/types.proto).

## Encoding

The default encoding protocol for rexplorer is [MessagePack][encoding-msgp], used for all public structured values,
more specifically wallets and network statistics, as well as some internal structured values. [JSON][encoding-json] are
and [Protocol Buffers][encoding-pb] are supported as well however.

> The `rexplorer` tool as well as most Go examples allow you to specify the encoding protocol used
> using the `--encoding (msgp|protobuf|json)` flag:
>
> * `--encoding msgp` to use [MessagePack][encoding-msgp];
> * `--encoding json` to use [JSON][encoding-json];
> * `--encoding protobuf` to use [Protocol Buffers][encoding-pb].

As a single Redis database is to be reserved for a single (tfchain) network,
you can also only use one encoding protocol per Redis database as a consequence.
Should you for example try to start `rexplorer` using the `--encoding json` flag,
when it was previously always started on that Redis Database with the `--encoding msgp` (default) flag,
it will immediately exit with an error complaining about the fact that you are trying to mix encoding protocols.

### Statistics

The default encoding protocol for rexplorer is [MessagePack][encoding-msgp],
it is chosen  as the default protocol for following reasons:

* it is a format which is well supported across (programming) languages and environments;
* it is reasonably fast (even though it gets on the slow side for big values);
* it is much more compact than for example `JSON`;

You might however have other preferences. [JSON][encoding-json] is even more widely supported
and has the advantage that it can be read by humans without need to decode it prior to consumption.
[Protocol Buffers][encoding-pb] is —in the current implementation—
slightly slower than the [MessagePack][encoding-msgp] implementation but is more compact in terms of byte size.
[Protocol Buffers][encoding-pb] has however not as much support across (programming) languages as [MessagePack][encoding-msgp].
It should also be noted that if the current [MessagePack][encoding-msgp] implementation would not encode null values,
it would be a bit more compact as well.

Here are some numbers to put the available encoding protocols into context.

||[MessagePack][encoding-msgp]|[JSON][encoding-json]|[Protocol Buffers][encoding-pb]|
|---|---|---|---|
|time to sync from disk (ms/block)|1.28775|7.94517|1.60229|
|byte size of global `stats` value|142|372|72|}
|byte size of individual wallets
|minimum|28|46|8|
|maximum|33 986|50 156|33 227|
|average|324|487|298|
|total|80 904|121 296|74 421|
|byte size of multi-signature wallets|||
|minimum|88|211|74|
|maximum|158|373|145|
|average|112|268|96|
|total|1124|2687|968|

#### Information about the last statistics update

blockchain information:
* network: [tfchain testnet](http://explorer.testnet.threefoldtoken.com);
* block height: `108 762,`
* network time: `2018-09-07 17:13:44 +0200 CEST`

`rexplorer` version:
```
$ rexplorer version
Tool version            v0.1.2-2c7b54b
TFChain Daemon version  v1.1.0
Rivine protocol version v1.0.7

Go Version   v1.11
GOOS         darwin
GOARCH       amd64
```

Hardware Information:
```
$ system_profiler SPHardwareDataType | grep -v UUID | grep -v SerialHardware:

    Hardware Overview:

      Model Name: MacBook Pro
      Model Identifier: MacBookPro14,2
      Processor Name: Intel Core i5
      Processor Speed: 3,1 GHz
      Number of Processors: 1
      Total Number of Cores: 2
      L2 Cache (per Core): 256 KB
      L3 Cache: 4 MB
      Memory: 16 GB
      Boot ROM Version: MBP142.0178.B00
      SMC Version (system): 2.44f1
```

#### How are these statistics gathered

First of all you should apply the following diff
to the `rexplorer` codebase (using `git apply` or manually):

```diff
diff --git a/commands.go b/commands.go
index 4498f38..b85d24b 100644
--- a/commands.go
+++ b/commands.go
@@ -189,7 +189,8 @@ func (cmd *Commands) Root(_ *cobra.Command, args []string) (cmdErr error) {
 		log.Println("rexplorer is up and running...")
 
 		// wait until done
-		<-ctx.Done()
+		//<-ctx.Done()
+		cancel()
 	}()
 
 	// stop the server if a kill signal is caught
@@ -205,7 +206,7 @@ func (cmd *Commands) Root(_ *cobra.Command, args []string) (cmdErr error) {
 	}
 
 	cancel()
-	wg.Wait()
+	//wg.Wait()
 
 	log.Println("Goodbye!")
 	return

```

This diff will make sure that `rexplorer` exists as soon the
`Explorer` internal module is done syncing blocks from disk,
as received from the `Consensus` module.

> Prior to doing all this, make sure that you already
> have a reasonable amount of blocks on disk, gathered using
> an unmodified `rexplorer` version.
>
> The more blocks, the more useful the statistics will be in general.

Make sure to flush all the Redis databases that you'll use,
using `redis-cli -n <N> flushdb` or `redis-cli flushall`.

Once you met all these pre-conditions you can populate your redis databases, one by one, ideally in an isolated environment:

* `time rexplorer --redis-db 1 --network testnet --encoding msgp`;
* `time rexplorer --redis-db 2 --network testnet --encoding json`;
* `time rexplorer --redis-db 3 --network testnet --encoding protobuf`.

This will give you the total time it took to execute these commands, dividing the total time by the block height should give you the average time it took to process and store a block. These commands will also have populated your Redis databases, allowing you to automatically gather the other statistics as well.

Using [./tools/statsbytesize](./tools/statsbytesize) you can gather the byte size of the global `stats` value for a given Redis database (no decoding is required for this tool):

```
$ go run ./tools/statsbytesize/main.go --redis-db 3
byte size of stats value:  104
```

Using [./tools/walletbytesize](./tools/walletbytesize) you can gather the minimum, maximum and average byte size of individual and multi-signature wallets for a given Redis database (make sure to specify the correct `--encoding` flag for each Redis database slot):

```
$ go run ./tools/walletbytesize/main.go --redis-db 3 --encoding protobuf
2018/08/31 11:06:15 scanned wallet addresses in 51 SSCAN cycles
                       average       min          max
Individual Wallets     165           10           33426
MultiSig Wallets       106           84           155
```

## Examples

These examples assume you have a `rexplorer` instance running (and synced!!!),
using the default redis address (`:6379`) and default db slot (`0`).

### Get coins

There is a Go example that you can checkout at [/examples/getcoins/main.go](/examples/getcoins/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getcoins/main.go 01de096b60b4ece712409b8dc1ea1f9247b89953774503efdb689284a4ff06412c82223f867f2f
unlocked: 10000 TFT
locked:   0 TFT
--------------------
total: 10000 TFT
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli HGET a:01de09 6b60b4ece712409b8dc1ea1f9247b89953774503efdb689284a4ff06412c82223f867f2f
{"balance":{"unlocked":{"total":"10000000000000"}}}
```

As you can see for yourself, the balance of an address is stored as a JSON object (if you use the `--encoding json` flag).,
and the total balance is something which you have to compute yourself.

> Note that your redis-cli` output will look like binary gibberish in case you are using [MessagePack][encoding-msgp]
> or [Protocol Buffers][encoding-pb] as the encoding type of your rexplorer.
> If so, you'll first have to decode prior to being able to consume it as human reader.
> The Golang example does this automatically for you.

### Get 3Bot

There is a Go example that you can checkout at [/examples/getbot/main.go](/examples/getbot/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getbot/main.go 2
{
  "id": 2,
  "addresses": [
    "bot.threefold.io"
  ],
  "names": [
    "chatbot.example"
  ],
  "publickey": "ed25519:ddc61d4b8e70a9d02a7c99e28f58fdaae93645fa376aa75db23411567ec9b7df",
  "expiration": 1602780240
}

```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli HGET b:0 2
{"id":2,"addresses":["bot.threefold.io"],"names":["chatbot.example"],"publickey":"ed25519:ddc61d4b8e70a9d02a7c99e28f58fdaae93645fa376aa75db23411567ec9b7df","expiration":1602780240}
```

As you can see for yourself, the record of an 3Bot is stored directly as an object (if you use the `--encoding json` flag).

> Note that your redis-cli` output will look like binary gibberish in case you are using [MessagePack][encoding-msgp]
> or [Protocol Buffers][encoding-pb] as the encoding type of your rexplorer.
> If so, you'll first have to decode prior to being able to consume it as human reader.
> The Golang example does this automatically for you.

### Get outputs

In order to fetch the logic for a given wallet, we have to apply the same logic as already
covered in the [Get coins](#get-coins) example. This is because the outputs for a wallet
are stored together with the balance totals and MultiSig info for a given wallet in a single structured value.

There is a Go example that you can checkout at [/examples/getoutputs/main.go](/examples/getoutputs/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getoutputs/main.go 01547499996a673c394f7c1f229a20c6e75262b64f44d0fb45cf30f497da6c35710457a561f102
Unlocked Outputs (1159):

37b30b768bad83e47d388d49d24d22e2d05ffea5470bc9e5a4ea64cd54538ea0  1 TFT            reward:block
...
ed3d1b313ff30ee8be26f0b65a71eff0dce6ee788f67622ea8383ad778cbe937  1 TFT            reward:block

Locked Outputs (19):

2e5f286c7c28b647ddedf15a9dfb50225ca864cd5b3f9912136ee3323b40d7fb  1 TFT            1537719085  reward:block
...
5ed251edcf144b26b003e7201b52cd968a57bcb976142e1f5e734759599e791b  1 TFT            1537689491  reward:block
```

Again, you can get the data directly from redis using `redis-cli`, but you will still have
to decode the data and filter the desired data before you can use it.
The Golang example does this all for you.

> Note that this wallet has unlocked outputs only because these outputs have descriptions
> which match the filters applied to the used `rexplorer` instance.
> The filters applied to the instance in this example was `-f 'reward:*` on the standard network.

### Get all unique addresses used

Get all the unique addresses used within a network.
Even if an address is only used in a reverted block, it is still tracked and kept:

```
$ redis-cli smembers addresses
1) 01fea3ae2854f6e497c92a1cdd603a0bc92ada717200e74f64731e86a923479883519804b18d9d
2) 01fef1037d0e51042838e4265a1af4f753b8f69de5a7be85a5f3a3c6bd1fbcb8f20986b4aae3a5
3) 0148d275cffe21a79a865d78529682e347d56615e0033ff114731014349b970c033acae5fbf3a3
4) 01cc1872da1c5b2f6bc02fead5f660992477b7c3d7133c75746b7adeec72bdda5c9149cb36e34a
5) 01cc6173a28b18ce172466c2b7aca93465ff8c2ccebbad27c4c54bada8a80e8de667c0a0ae0e5f
...
630) 017c69af16f91cfe3360e39205411b15f9a0f6cb7502e2d4cbf7c428d44595b9f3a4b377740bfe
631) 01806f23a376c216ca96a2dc0b65f74ad47bcd13ae4d65b8af65211fa6540cc7ccd270c647d443
632) 0142c3d75a2ce6052e316e0d61f290cdc2e974de9107e26185703722ab2c6c0f6d203d0f8341ca
633) 01060d90351de9bf9892711c713e42622cf4c8743e08e5ee800da4a393446ded584cb8bf8250d8
634) 018f199d998e248936eb4ac7a78f6084b6613d7611b7b0f2838dd245c4217ff2413b10d54d070f
635) 01fdd6c673686ccf1aa2caa5c0ea08bcb25e8fe3cbbc1079b350500314ef8defa4991cdf27bc1f
```

If you pipe the command from this example into the command of [the Get Coins example](#get-coins),
you'll be able to get the balance of each of the wallets in existence of a network.

Following this example we can see how to get the amount of unique addresses used in a network:

```
$ redis-cli scard addresses
(integer) 517
```

### Get the unique wallet addresses of the current Coin Creators

Get all the unique wallet addresses of the current coin creators.
There is a Go example that you can checkout at [/examples/getcoincreators/main.go](/examples/getcoincreators/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getcoincreators/main.go
1) 016148ac9b17828e0933796eaca94418a376f2aa3fefa15685cea5fa462093f0150e09067f7512
2) 013a787bf6248c518aee3a040a14b0dd3a029bc8e9b19a1823faf5bcdde397f4201ad01aace4c9
3) 01d553fab496f3fd6092e25ce60e6f72e24b57950bffc0d372d659e38e5a95e89fb117b4eb3481
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli SMEMBERS coincreators
1) "016148ac9b17828e0933796eaca94418a376f2aa3fefa15685cea5fa462093f0150e09067f7512"
2) "013a787bf6248c518aee3a040a14b0dd3a029bc8e9b19a1823faf5bcdde397f4201ad01aace4c9"
3) "01d553fab496f3fd6092e25ce60e6f72e24b57950bffc0d372d659e38e5a95e89fb117b4eb3481"
```

As you can see for yourself, the unique wallet addresses are stored in a Redis Set,
in a hex-encoded string format, so getting each coin creator is pretty easy.
For each coin creator you can than also gather information such as the wallet of each coin creator.

### Get global statistics

There is a Go example that you can checkout at [/examples/getstats/main.go](/examples/getstats/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getstats/main.go --redis-db 1
tfchain network has:
  * a total of 99885898.21 TFT, of which 99885756.21 TFT is liquid,
    142 TFT is locked, 35700 TFT is paid out as miner payouts,
    21 TFT is paid out as tx fees and 321 TFT is paid out as foundation fees
  * 99.99986% liquid coins of a total of 99885898.21 TFT coins
  * 00.00014% locked coins of a total of 99885898.21 TFT coins
  * a total of 3591 transactions, of which 16 wallet-value transactions,
    3 coin creation transactions, 0 coin creator definition transactions,
    1 3Bot registration transactions, 1 3Bot update transactions
    and 3570 are pure block creation transactions
  * a block height of 3570, with the time of the highest block
    being 2019-01-12 04:20:13 +0100 CET (1547263213)
  * a total of 3571 blocks, 18 value transactions and 18 coin inputs
  * a total of 3623 coin outputs, of which 3612 are liquid, 11 are locked,
    32 transfer value, 3570 are miner payouts, 21 are tx fees
    and 6 are foundation fees
  * a total of 15 unique addresses that have been used
  * an average of 01.77778% value coin outputs per value transaction
  * an average of 00.00504% value transactions per block
  * 99.69638% liquid outputs of a total of 3623 coin outputs
  * 00.30362% locked outputs of a total of 3623 coin outputs
  * 00.50125% value transactions of a total of 3591 transactions
  * 00.08354% coin creation transactions of a total of 3591 transactions
  * 00.19493% coin burn transactions of a total of 3591 transactions
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli get stats
{"timestamp":1540577367,"blockHeight":299,"txCount":316,"coinCreationTxCount":0,"coinCreatorDefinitionTxCount":0,"threeBotRegistrationTransactionCount":2,"threeBotUpdateTransactionCount":14,"valueTxCount":16,"coinOutputCount":348,"lockedCoinOutputCount":10,"coinInputCount":16,"minerPayoutCount":299,"txFeeCount":16,"minerPayouts":"2990000000000","txFees":"16000000000","coins":"100002990000000000","lockedCoins":"100000000000"}
```

As you can see for yourself, the balance of an address is stored as a JSON object (if you use the `--encoding json` flag).
In the Golang example we added some extra logic to showcase some examples of
some statistics you can compute based on the tracked global statistical values.

> Note that your redis-cli` output will look like binary gibberish in case you are using [MessagePack][encoding-msgp]
> or [Protocol Buffers][encoding-pb] as the encoding type of your rexplorer.
> If so, you'll first have to decode prior to being able to consume it as human reader.
> The Golang example does this automatically for you.

### Get multi-signature Addresses

There is a Go example that you can checkout at [/examples/getmultisigaddresses/main.go](/examples/getmultisigaddresses/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getmultisigaddresses/main.go --redis-db 1 01b650391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa
* 0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli HGET a:01b650 391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa
{"balance":{"unlocked":{"total":"0","outputs":null},"locked":{"total":"0","outputs":null}},"multisignAddresses":["0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"],"multisign":{"owners":null,"signaturesRequired":0}}
```

This example also works in the opposite direction, where the multisig address will return all owner addresses:

```
$ redis-cli HGET a:0359aa aa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37
{"multisignAddresses":["0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"]}
```

> Note that your `redis-cli` output will look like binary gibberish in case you are using [MessagePack][encoding-msgp]
> or [Protocol Buffers][encoding-pb] as the encoding type of your rexplorer.
> If so, you'll first have to decode prior to being able to consume it as human reader.
> The Golang example does this automatically for you.

## Testing

The quality and correctness of `rexplorer` is tested using
both [unit tests](#unit-tests) and [integration tests](#integration-tests).

### Unit Tests

You can run all unit tests in one command as follows:

```
$ make unit-tests
go test -race -tags "debug testing" . ./pkg/database/types ./pkg/encoding ./pkg/rflag ./pkg/types
?       github.com/threefoldfoundation/rexplorer        [no test files]
ok      github.com/threefoldfoundation/rexplorer/pkg/database/types     (cached)
?       github.com/threefoldfoundation/rexplorer/pkg/encoding   [no test files]
ok      github.com/threefoldfoundation/rexplorer/pkg/rflag      (cached)
ok      github.com/threefoldfoundation/rexplorer/pkg/types      (cached)
```

Even better is if you run the `test` Make target instead, as that will also detects ineffectual assignments:

```
$ make test
ineffassign . ./pkg/database/types ./pkg/encoding ./pkg/rflag ./pkg/types
go test -race -tags "debug testing" . ./pkg/database/types ./pkg/encoding ./pkg/rflag ./pkg/types
?       github.com/threefoldfoundation/rexplorer        [no test files]
ok      github.com/threefoldfoundation/rexplorer/pkg/database/types     (cached)
?       github.com/threefoldfoundation/rexplorer/pkg/encoding   [no test files]
ok      github.com/threefoldfoundation/rexplorer/pkg/rflag      (cached)
ok      github.com/threefoldfoundation/rexplorer/pkg/types      (cached)
```

> the Make `test` target (and thus also its depending Make `ineffassign` target)
> does require you to install _ineffassign_, which can be done using the following command:
> `go get -u github.com/gordonklaus/ineffassign`

Should you want to run the unit tests without having to use Make, you can do so as follows:

```
$ go test -race -tags "debug testing" . $(go list ./pkg/...)
?       github.com/threefoldfoundation/rexplorer        [no test files]
ok      github.com/threefoldfoundation/rexplorer/pkg/database/types     (cached)
?       github.com/threefoldfoundation/rexplorer/pkg/encoding   [no test files]
ok      github.com/threefoldfoundation/rexplorer/pkg/rflag      (cached)
ok      github.com/threefoldfoundation/rexplorer/pkg/types      (cached)
```

### Integration Tests

The integration tests have the following conditions:

+ (1) You have the [tfchain][tfchain] network standard synced —using `rexplorer`— to the desired height (be it the network height or not);
+ (2) You have the [tfchain][tfchain] network testnet synced —using `rexplorer`— to the desired height (be it the network height or not);
+ (3) You have no `rexplorer` running when running the integration tests;
+ (4) You still have the Redis server(s) running in the background which contain the aggregated data by the `rexplorer` as mentioned in (1) and (2);

If you meet all conditions listed above you can run the integration tests as follows:

```
$ make integration-tests
go run tests/integration/sumcoins/main.go \
                --redis-addr ":6379" --redis-db "1" \
                --encoding "msgp"
sumcoins test —using encoding msgp— on block height 113868 passed for 541 wallets :)
go run tests/integration/sumcoins/main.go \
                --redis-addr ":6379" --redis-db "0" \
                --encoding "msgp"
sumcoins test —using encoding msgp— on block height 103625 passed for 729 wallets :)
python3 tests/integration/sumcoins/main.py \
                --redis-port "6379" --redis-db "1" \
                --encoding "msgp"
sumcoins test --using encoding msgp-- on block height 113868 passed for 541 wallets :)
python3 tests/integration/sumcoins/main.py \
                --redis-port "6379" --redis-db "0" \
                --encoding "msgp"
sumcoins test --using encoding msgp-- on block height 103625 passed for 729 wallets :)
go run tests/integration/sumoutputs/main.go \
                --redis-addr ":6379" --redis-db "1" \
                --encoding "msgp"
coin output scanner is now at coin output #5000 with id ffec585ef3118b3b5707b8dd1cabe2cd2e183340863eb1720edacd739973e418...
...
coin output scanner is now at coin output #110000 with id dada46d17ace12b869c160f4a2396170e90cf827665246d93c4737a370c9c5ab...
found 114896 coin outputs spread over 54103 buckets
sumoutputs test —using encoding msgp— on block height 113868 passed for 114896 outputs :)
go run tests/integration/sumoutputs/main.go \
                --redis-addr ":6379" --redis-db "0" \
                --encoding "msgp"
coin output scanner is now at coin output #5000 with id 514b6678bd22451cee9abbdd21aa5672be30e22d00dae25a9945343d0307a5ce...
...
coin output scanner is now at coin output #105000 with id 3bafa8e01ce132d3d777c965b1e0042fab444e5b2a555ed90da06fea8b4fb415...
found 105410 coin outputs spread over 52562 buckets
sumoutputs test —using encoding msgp— on block height 103625 passed for 105410 outputs :)
go run tests/integration/validatevalues/main.go \
                --redis-addr ":6379" --redis-db "1" \
                --encoding "msgp"
Global stats are valid :)
Internal keys are valid :)
height-locked output scanner is now at output #5000 with id 15ede59eac6544af28c3eae72e3900dd729048f220e610110f03a3b623f6cf65...
...
height-locked output scanner is now at output #110000 with id aee1836a8773e38bdeb2a39d6ef0d1856558843574552aef001c9eef9b2ddc93...
Height-Locked Output entries are valid :)
All 3 coin creators are known and tracked :)
validatevalues test —using encoding msgp— on block height 113868 passed :)
go run tests/integration/validatevalues/main.go \
                --redis-addr ":6379" --redis-db "0" \
                --encoding "msgp"
Global stats are valid :)
Internal keys are valid :)
time-locked output scanner is now at output #100 with id 85eb2a127cebe70c4fa560bd268fa4d4d8f915a75eff7494b8e367e53c6923dd...
Time-Locked Output entries are valid :)
height-locked output scanner is now at output #5000 with id 75318b55d6ece13b30703a4f8f1eb077a23092a9b7888efe2b5fdee14b830d14...
...
height-locked output scanner is now at output #100000 with id bdf3c443f75c8d8190a1e5939a5838b0d6ac61321a222aa86ba54fcb59ed317c...
Height-Locked Output entries are valid :)
All 3 coin creators are known and tracked :)
validatevalues test —using encoding msgp— on block height 103625 passed :)
```

> In order to be able to run the integration tests,
> you'll need `Make`, [Golang][golang] (1.9 or above) and
> [Python 3][python]. Prior to executing the 
> `integration-tests` Make target you'll also have
> to install all external [Python 3][python]
> dependencies which can be be done using the command:
> ```
> pip3 install -r ./tests/integration/sumcoins/requirements.txt
> ```

## Repository Owners

* Rob Van Mieghem ([@robvanmieghem](https://github.com/robvanmieghem))
* Lee Smet ([@leesmet](https://github.com/leesmet))
* Glen De Cauwsemaecker ([@glendc](https://github.com/glendc))

[tfchain]: https://github.com/threefoldfoundation/tfchain
[rivine]: https://github.com/threefoldtech/rivine
[redistypes]: https://redis.io/topics/data-types
[msgp-spec]: https://github.com/msgpack/msgpack/blob/master/spec.md#int-format-family

[encoding-msgp]: https://msgpack.org
[encoding-json]: https://json.org
[encoding-pb]: https://developers.google.com/protocol-buffers/
