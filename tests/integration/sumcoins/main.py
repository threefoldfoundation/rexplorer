import argparse, redis, json, msgpack, rtypes.types_pb2
from enum import Enum

# encoding type used to define a decoder dynamically
class EncodingType(Enum):
    msgp = 'msgp'
    json = 'json'
    protobuf = 'protobuf'

    def __str__(self):
        return self.value

class WalletBalance:
    def __init__(self):
        self.unlocked = 0
        self.unlocked_outputs = []
        self.locked = 0
        self.locked_outputs = []

class ChainStats:
    def __init__(self):
        self.locked_coins = 0
        self.unlocked_coins = []
        self.block_height = 0

# parse CLI arguments
parser = argparse.ArgumentParser(description='Read and validate a MsgPack db.')
parser.add_argument('--redis-db', dest='dbslot',
    default=0, help='slot/index of the redis db')
parser.add_argument('--redis-port', dest='dbport',
    default=6379, help='port of the redis db')
parser.add_argument('--encoding', dest='encoding',
    default=EncodingType.msgp, type=EncodingType, choices=list(EncodingType))
args = parser.parse_args()

# create redis host
r = redis.StrictRedis(host='localhost', port=args.dbport, db=args.dbslot)

# define decode functions
if args.encoding == EncodingType.msgp:
    def decw(b):
        wallet = msgpack.unpackb(b)
        if wallet == None:
            raise Exception('nil wallet for addr: ' + addr)

        balance = WalletBalance()
        if b'b' not in wallet:
            return balance # skip wallets with no balance
        
        if b'u' in wallet[b'b']:
            balance.unlocked = int(wallet[b'b'][b'u'][b't'])
            if b'o' in wallet[b'b'][b'u']:
                for _, output in wallet[b'b'][b'u'][b'o'].items():
                    balance.unlocked_outputs.append(int(output[b'a']))
        if b'l' in wallet[b'b']:
            balance.locked = int(wallet[b'b'][b'l'][b't'])
            for _, output in wallet[b'b'][b'l'][b'o'].items():
                balance.locked_outputs.append(int(output[b'a']))
        return balance
    def decs(b):
        stats = msgpack.unpackb(b)
        if stats == None:
            raise Exception('nil stats')
        chain_stats = ChainStats()
        chain_stats.locked_coins = int(stats[b'lct'])
        chain_stats.unlocked_coins = int(stats[b'ct']) - chain_stats.locked_coins
        chain_stats.block_height = stats[b'cbh']
        return chain_stats
elif args.encoding == EncodingType.json:
    def decw(b):
        wallet = json.loads(b)
        if wallet == None:
            raise Exception('nil wallet for addr: ' + addr)

        balance = WalletBalance()
        if 'balance' not in wallet:
            return balance # skip wallets with no balance
        
        if 'unlocked' in wallet['balance']:
            balance.unlocked = int(wallet['balance']['unlocked']['total'])
            if 'outputs' in wallet['balance']['unlocked']:
                for _, output in wallet['balance']['unlocked']['outputs'].items():
                    balance.unlocked_outputs.append(int(output['amount']))
        if 'locked' in wallet['balance']:
            balance.locked = int(wallet['balance']['locked']['total'])
            for _, output in wallet['balance']['locked']['outputs'].items():
                balance.locked_outputs.append(int(output['amount']))
        return balance
    def decs(b):
        stats = json.loads(b)
        if stats == None:
            raise Exception('nil stats')
        chain_stats = ChainStats()
        chain_stats.locked_coins = int(stats['lockedCoins'])
        chain_stats.unlocked_coins = int(stats['coins']) - chain_stats.locked_coins
        chain_stats.block_height = stats['blockHeight']
        return chain_stats
elif args.encoding == EncodingType.protobuf:
    def decw(b):
        wallet = rtypes.types_pb2.PBWallet()
        wallet.ParseFromString(b)
        if wallet == None:
            raise Exception('nil wallet for addr: ' + addr)
        balance = WalletBalance()

        balance.unlocked = int.from_bytes(wallet.balance_unlocked.total[8:], byteorder='big')
        for _, output in wallet.balance_unlocked.outputs.items():
            balance.unlocked_outputs.append(int.from_bytes(output.amount[8:], byteorder='big'))
        balance.locked = int.from_bytes(wallet.balance_locked.total[8:], byteorder='big')
        for _, output in wallet.balance_locked.outputs.items():
            balance.locked_outputs.append(int.from_bytes(output.amount[8:], byteorder='big'))
        return balance
    def decs(b):
        stats = rtypes.types_pb2.PBNetworkStats()
        stats.ParseFromString(b)
        if stats == None:
            raise Exception('nil stats')
        chain_stats = ChainStats()
        chain_stats.locked_coins = int.from_bytes(stats.locked_coins[8:], byteorder='big')
        chain_stats.unlocked_coins = int.from_bytes(stats.coins[8:], byteorder='big') - chain_stats.locked_coins
        chain_stats.block_height = stats.blockheight
        return chain_stats
else:
    raise Exception('unsupported encoding type: ' + str(args.encoding))

# unique address counter
ac = 0

coinsUnlocked = 0
coinsLocked = 0

# go fetch the wallet for each unique address in chain,
# and ensure the value can be decoded and is valid
for addr in r.sscan_iter(name='addresses'):
    ac += 1

    b = r.hget(b'a:' + addr[:6], addr[6:])
    if b == None:
        continue # skip nil wallets
    # ensure we can encode
    balance = decw(b)
    
    totalUnlocked = sum(balance.unlocked_outputs, 0)
    if totalUnlocked > balance.unlocked:
        raise Exception('invalid total unlocked balance for wallet: ' + addr)
    coinsUnlocked += balance.unlocked
    
    totalLocked = sum(balance.locked_outputs, 0)
    if totalLocked != balance.locked:
        raise Exception('invalid total locked balance for wallet: ' + addr)
    coinsLocked += balance.locked

# get stats and compare the computed total locked and unlocked coin count
stats = decs(r.get('stats'))
if stats.unlocked_coins != coinsUnlocked:
    raise Exception('unexpected total unlocked coins: ' + str(stats.unlocked_coins) + ' != ' + str(coinsUnlocked))
if stats.locked_coins != coinsLocked:
    raise Exception('unexpected total locked coins: ' + str(stats.locked_coins) + ' != ' + str(coinsLocked))


print('sumcoins test --using encoding ' + str(args.encoding) + '-- on block height ' + str(stats.block_height) + ' passed for ' + str(ac) + ' wallets :)')
