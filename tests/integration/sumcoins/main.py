import argparse, redis, json, msgpack
from enum import Enum
from build.types_pb2 import PBWallet, PBNetworkStats

class ProtoMsg(object):

    def __init__(self, data, msgcls=None):
        if msgcls is not None:
            self._msg = msgcls()
            self._msg.ParseFromString(data)
        else:
            self._msg = data

    def is_valid(self):
        return self._msg is not None

    def __dir__(self):
        return dir(self._msg)

    def __contains__(self, k):
        if hasattr(self._msg, 'DESCRIPTOR'):
            return k in self._msg.DESCRIPTOR.fields_by_name.keys()
        return k in self._msg if self._msg else False

    def __getitem__(self, k):
        try:
            item = getattr(self._msg, k)
            if hasattr(item, 'DESCRIPTOR'):
                return ProtoMsg(item)
            else:
                return item
        except:
            if self._msg:
                return self._msg.__getitem__(k)

# encoding type used to define a decoder dynamically
class EncodingType(Enum):
    msgp = 'msgp'
    json = 'json'
    protobuf = 'protobuf'

    def __str__(self):
        return self.value

# parse CLI arguments
parser = argparse.ArgumentParser(description='Read and validate a MsgPack db.')
parser.add_argument('--db-slot', dest='dbslot',
    default=0, help='slot/index of the redis db')
parser.add_argument('--db-port', dest='dbport',
    default=6379, help='port of the redis db')
parser.add_argument('--encoding', dest='encoding',
    default=EncodingType.msgp, type=EncodingType, choices=list(EncodingType))
args = parser.parse_args()

# create redis host
r = redis.StrictRedis(host='localhost', port=args.dbport, db=args.dbslot)

walletBalanceKey = None
balanceUnlockedKey = None
balanceLockedKey = None
balanceTotalKey = None
balanceOutputKey = None
balanceOutputAmountKey = None
statsLockedCoinsKey = None
statsUnlockedCoinsKey = None

# define decode function
dec = None
if args.encoding == EncodingType.msgp:
    dec = msgpack.unpackb
    decode_stats = dec
    walletBalanceKey = 'b'
    balanceUnlockedKey = 'u'
    balanceLockedKey = 'l'
    balanceTotalKey = 't'
    balanceOutputKey = 'o'
    balanceOutputAmountKey = 'a'
    statsLockedCoinsKey = 'lct'
    statsCoinsKey = 'ct'
elif args.encoding == EncodingType.json:
    dec = json.loads
    decode_stats = dec
    walletBalanceKey = 'balance'
    balanceUnlockedKey = 'unlocked'
    balanceLockedKey = 'locked'
    balanceTotalKey = 'total'
    balanceOutputKey = 'outputs'
    balanceOutputAmountKey = 'amount'
    statsLockedCoinsKey = 'lockedCoins'
    statsCoinsKey = 'coins'

elif args.encoding == EncodingType.protobuf:
    decode_wallet = lambda data: ProtoMsg(data, PBWallet)
    dec = decode_wallet
    decode_stats = lambda data: ProtoMsg(data, PBNetworkStats)
    walletBalanceKey = 'balance'
    balanceUnlockedKey = 'balance_unlocked'
    balanceLockedKey = 'balance_locked'
    balanceTotalKey = 'total'
    balanceOutputKey = 'outputs'
    balanceOutputAmountKey = 'amount'
    statsLockedCoinsKey = 'lockedCoins'
    statsCoinsKey = 'coins'

    print('using protobuf')
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

    if args.encoding == EncodingType.protobuf:

        wallet = dec(b)
        # add check for valid wallet
        
        # *1) balance key isn't defined in the proto schema
        # go directly to balance unlocked

        if balanceOutputKey in wallet[balanceUnlockedKey]:
            totalUnlocked = 0
            import ipdb; ipdb.set_trace()
            for id, output in wallet[balanceUnlockedKey][balanceOutputKey].items():
                totalUnlocked += int('>I', output[balanceOutputAmountKey])
            if totalUnlocked > int(wallet[balanceUnlockedKey][balanceTotalKey]):
                raise Exception('invalid total unlock balance for wallet: ' + addr)
        coinsUnlocked += int(wallet[balanceUnlockedKey][balanceTotalKey])

        totalLocked = 0
        for id, output in wallet[balanceLockedKey][balanceOutputKey].items():
            print(id, output)
            totalLocked += int(output[balanceOutputAmountKey])
        if totalLocked != int(wallet[balanceLockedKey][balanceTotalKey]):
            raise Exception('invalid total locked balance for wallet: ' + addr)
        coinsLocked += int(wallet[balanceLockedKey][balanceTotalKey])
    else:

        # ensure we can encode
        wallet = dec(b)
        if wallet == None:
            raise Exception('nil wallet for addr: ' + addr)
        if walletBalanceKey not in wallet:
            continue # skip wallets with no balance

        if balanceUnlockedKey in wallet[walletBalanceKey]:
            if balanceOutputKey in wallet[walletBalanceKey][balanceUnlockedKey]:
                totalUnlocked = 0
                for id, output in wallet[walletBalanceKey][balanceUnlockedKey][balanceOutputKey].items():
                    totalUnlocked += int('>I', output[balanceOutputAmountKey])
                if totalUnlocked > int(wallet[walletBalanceKey][balanceUnlockedKey][balanceTotalKey]):
                    raise Exception('invalid total unlock balance for wallet: ' + addr)
            coinsUnlocked += int(wallet[walletBalanceKey][balanceUnlockedKey][balanceTotalKey])
        
        if balanceLockedKey in wallet[walletBalanceKey]:
            totalLocked = 0
            for id, output in wallet[walletBalanceKey][balanceLockedKey][balanceOutputKey].items():
                totalLocked += int(output[balanceOutputAmountKey])
            if totalLocked != int(wallet[walletBalanceKey][balanceLockedKey][balanceTotalKey]):
                raise Exception('invalid total locked balance for wallet: ' + addr)
            coinsLocked += int(wallet[walletBalanceKey][balanceLockedKey][balanceTotalKey])

# get stats and compare the computed total locked and unlocked coin count
stats = decode_stats(r.get('stats'))
uc = int(stats[statsCoinsKey]) - int(stats[statsLockedCoinsKey])
if uc != coinsUnlocked:
    raise Exception('unexpected total unlocked coins: ' + str(uc) + ' != ' + str(coinsUnlocked))
lc = int(stats[statsLockedCoinsKey])
if lc != coinsLocked:
    raise Exception('unexpected total locked coins: ' + str(lc) + ' != ' + str(coinsLocked))

print('validated balance of ' + str(ac) + ' wallets successfully, decoding them from ' + str(args.encoding) + ' :)')
