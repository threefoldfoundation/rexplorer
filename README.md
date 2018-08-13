# rexplorer

rexplorer is a small explorer binary, that can aid in explorering a [tfchain][tfchain] network.
It applies/reverts data —received from an embedded consensus module— into a redis db of choice,
such that the tfchain network data can be consumed/used in a meaningful way.

## Install

```
$ go get -u github.com/threefoldfoundation/rexplorer && rexplorer version
Tool version            v0.1.1
TFChain Daemon version  v1.0.7
Rivine protocol version v1.0.7

Go Version   v1.10.3
GOOS         darwin
GOARCH       amd64
```

## Usage

To start a rexplorer instance for the standard network,
storing all persistent non-redis data into a subdir of the current root directory,
you can do so as simple as:

```
$ rexplorer
2018/08/07 23:58:48 starting rexplorer v0.1.1...
2018/08/07 23:58:48 loading rivine gateway module (1/3)...
2018/08/07 23:58:48 loading rivine consensus module (2/3)...
2018/08/07 23:58:48 loading internal explorer module (3/3)...
```

The persistent dir (used for some local boltdb consensus/gateway data) can be changed
using the `-d`/`--persistent-directory` flag.

Should you want to explore `testnet` instead of the `standard` net you can use the `--network testnet` flag.

By default [MessagePack](https://msgpack.org) is used to encode all public data in the Redis Database,
should you want to use [JSON](http://json.org) instead you can use the `--encoding json` flag.

For more information use the `--help` flag:

```
$ rexplorer --help
start the rexplorer daemon
Usage:
  rexplorer [flags]
  rexplorer [command]
Available Commands:
  help        Help about any command
  version     show versions of this tool
Flags:
  -h, --help                          help for rexplorer
  -n, --network string                the name of the network to which the daemon connects, one of {standard,testnet} (default "standard")
  -d, --persistent-directory string   location of the root diretory used to store persistent data of the daemon of tfchain
      --redis-addr string             which (tcp) address the redis server listens on (default ":6379")
      --redis-db int                  which redis database slot to use
      --rpc-addr string               which port the gateway listens on (default ":23112")
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
    * all locked coin outputs for a given timestmap range
    * format value: custom
    * example key: `lcos.time:1526335200`

Following _public_ keys are reserved:

* `stats`:
    * used for global network statistics
    * format value: JSON/MessagePack
    * example key: `stats`
* `addresses`:
    * set of unique wallet addresses used (even if reverted) in the network
    * format value: [Redis SET][redistypes], where each value is a [Rivine][rivine]-defined hex-encoded UnlockHash
    * example key: `addresses`
* `a:01<4_random_hex_chars>`:
    * used by all wallet addresses, contains unlocked balance, locked balance and coin outputs as well as all multisig wallets jointly owned by this wallet
    * format value: [Redis HASHMAP][redistypes], where each field's value is a JSON/MessagePack
    * example key: `a:012b61`
    * fields have the format `<72_random_hex_chars>`, an example: `389e7f103288371830c632439fe709044c3ab5c374947ab4eca68ee987d3f736b360e530`
* `a:03<4_random_hex_chars>`:
    * used by all multisig wallet addresses, contains unlocked balance, locked balance and coin outputs as well as owner addresses and signatures required
    * format value: [Redis HASHMAP][redistypes], where each field's value is a JSON/MessagePack
    * example key: `a:032b61`
    * fields have the format `<72_random_hex_chars>`, an example: `389e7f103288371830c632439fe709044c3ab5c374947ab4eca68ee987d3f736b360e530`

Rivine Value Encodings:

* addresses are Hex-encoded and the exact format (and how it is created) is described in:
  <https://github.com/rivine/rivine/blob/master/doc/transactions/unlockhash.md#textstring-encoding>
* currencies are encoded as described in <https://godoc.org/math/big#Int.Text>
  using base 10, and using the smallest coin unit as value (e.g. 10^-9 TFT)
* coin outputs are stored in the Rivine-defined JSON format, described in:
  <https://github.com/rivine/rivine/blob/master/doc/transactions/transaction.md#json-encoding-of-outputs-in-v0-transactions> (`v0` tx) and
  <https://github.com/rivine/rivine/blob/master/doc/transactions/transaction.md#json-encoding-of-outputs-in-v1-transactions> (`v1` tx)

JSON formats of value types defined by this module:

* example of global stats (stored under `stats`):

```json
{
	"timestamp": 1533795799,
	"blockHeight": 77892,
	"txCount": 78209,
	"valueTxCount": 318,
	"coinOutputCount": 79368,
	"lockedCoinOutputCount": 742,
	"coinInputCount": 357,
	"minerPayoutCount": 77892,
	"txFeeCount": 240,
	"minerPayouts": "77892000000000",
	"txFees": "31600000001",
	"coins": "695176892000000000",
	"lockedCoins": "4852167650000000"
}
```

> When using MessagePack (the default encoding type), the keys are the same as when encoding as JSON,
> and the values are encoded in the exact same way, except that the resulting values
> follow the [MessagePack spec][msgp-spec].

* example of a wallet (stored under a:01<4_random_hex_chars>):

```json
{
    "balance": {
        "unlocked": "10000000",
        "locked": {
            "total": "5000",
            "outputs": [
                {
                    "amount": "2000",
                    "lockedUntil": 1534105468
                },
                {
                    "amount": "100",
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

> When using MessagePack (the default encoding type), the keys are the same as when encoding as JSON,
> and the values are encoded in the exact same way, except that the resulting values
> follow the [MessagePack spec][msgp-spec].

* example of a multisig wallet (stored under a:03<4_random_hex_chars>):

```json
{
    "balance": {
        "unlocked": "10000000"
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

> When using MessagePack (the default encoding type), the keys are the same as when encoding as JSON,
> and the values are encoded in the exact same way, except that the resulting values
> follow the [MessagePack spec][msgp-spec].

## Examples

These examples assume you have a `rexplorer` instance running (and synced!!!),
using the default redis address (`:6379`) and default db slot (`0`).

### Get Coins

There is a Go example that you can checkout at [/examples/getcoins/main.go](/examples/getcoins/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getcoins/main.go 0133021d18cc15467883a34074bb514665380bafd8879d9f1edd171d7f043e800367fd4d1c3ec8
unlocked: 100 TFT
locked:   24691.36 TFT
--------------------
total: 24791.36 TFT
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli HGET a:013302 1d18cc15467883a34074bb514665380bafd8879d9f1edd171d7f043e800367fd4d1c3ec8
"{\"balance\":{\"locked\":\"24691360000000\"}}"
```

As you can see for yourself, the balance of an address is stored as a JSON object,
and the total balance is something which you have to compute yourself.

### Get All Unique Addresses Used

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
(integer) 635
```

### Get Global Statistics

There is a Go example that you can checkout at [/examples/getstats/main.go](/examples/getstats/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getstats/main.go
tfchain network has:
  * a total of 695176892 TFT, of which 690324724.35 TFT is liquid,
    4852167.65 TFT is locked, 77892 TFT is paid out as miner payouts
    and 31.600000001 TFT is paid out as tx fees
  * 99.30202% liquid coins of a total of 695176892 TFT coins
  * 00.69798% locked coins of a total of 695176892 TFT coins
  * a block height of 77892, with the time of the highest block
    being 2018-08-09 08:23:19 +0200 CEST (1533795799)
  * a total of 77893 blocks, 318 value transactions and 357 coin inputs
  * a total of 79368 coin outputs, of which 78626 are liquid, 742 are locked,
    1236 transfer value, 77892 are miner payouts and 240 are tx fees
  * a total of 638 unique addresses that have been used
  * an average of 03.88679% value coin outputs per value transaction
  * an average of 00.00408% value transactions per block
  * 99.06511% liquid outputs of a total of 79368 coin outputs
  * 00.93489% locked outputs of a total of 79368 coin outputs
  * 00.40660% value transactions of a total of 78209 transactions
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli get stats
"{\"timestamp\":1533795799,\"blockHeight\":77892,\"txCount\":78209,\"valueTxCount\":318,\"coinOutputCount\":79368,\"lockedCoinOutputCount\":742,\"coinInputCount\":357,\"minerPayoutCount\":77892,\"txFeeCount\":240,\"minerPayouts\":\"77892000000000\",\"txFees\":\"31600000001\",\"coins\":\"695176892000000000\",\"lockedCoins\":\"4852167650000000\"}"
```

As you can see for yourself, the balance of an address is stored as a JSON object.
In the Golang example we added some extra logic to showcase some examples of
some statistics you can compute based on the tracked global statistical values.

### Get MultiSig Addresses

There is a Go example that you can checkout at [/examples/getmultisigaddresses/main.go](/examples/getmultisigaddresses/main.go),
and you can run it yourself as follows:

```
$ go run ./examples/getmultisigaddresses/main.go --db-slot 101b650391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa
* 0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37
```

You can run the same example directly from the shell —using `redis-cli`— as well:

```
$ redis-cli HGET a:01b650 391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa
"{\"multisignaddresses\":[\"0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37\"]}"
```

This example also works in the opposite direction, where the multisig address will return all owner addresses:

```
$ redis-cli HGET a:0359aa aa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37
"{\"multisign\":{\"owners\":[\"01b650391f06c6292ecf892419dd059c6407bf8bb7220ac2e2a2df92e948fae9980a451ac0a6aa\",\"0114df42a3bb8303a745d23c47062a1333246b3adac446e6d62f4de74f5223faf4c2da465e76af\"],\"signaturesRequired\":1}}"
```

## Testing

### Unit Tests

This project has no unit tests yet, and is mostly tested using manual testing
as well as by using [automated integration tests](#integration-tests).

### Integration Tests

The integration tests have the following conditions:

+ (1) You have the [tfchain][tfchain] network standard synced —using `rexplorer`— to the desired height (be it the network height or not);
+ (2) You have the [tfchain][tfchain] network testnet synced —using `rexplorer`— to the desired height (be it the network height or not);
+ (3) You have no `rexplorer` running when running the integration tests;
+ (4) You still have the Redis server(s) running in the background which contain the aggregated data by the `rexplorer` as mentioned in (1) and (2);

If you meet all conditions listed above you can run the integration tests as follows:

```
$ make integration-tests
go run tests/integration/sumcoins/main.go --db-address ":6379" --db-slot "1"
sumcoins test on block height 2026 passed :)
go run tests/integration/sumcoins/main.go --db-address ":6379" --db-slot "0"
sumcoins test on block height 1710 passed :)
```

[tfchain]: https://github.com/threefoldfoundation/tfchain
[rivine]: https://github.com/rivine/rivine
[redistypes]: https://redis.io/topics/data-types
[msgp-spec]: https://github.com/msgpack/msgpack/blob/master/spec.md#int-format-family
