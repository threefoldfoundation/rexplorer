import argparse, redis, json, msgpack
from enum import Enum

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
    walletBalanceKey = 'b'
    balanceUnlockedKey = 'u'
    balanceLockedKey = 'l'
    balanceTotalKey = 't'
    balanceOutputKey = 'o'
    balanceOutputAmountKey = 'a'
    statsLockedCoinsKey = 'lct'
    statsUnlockedCoinsKey = 'ct'
elif args.encoding == EncodingType.json:
    dec = json.loads
    walletBalanceKey = 'balance'
    balanceUnlockedKey = 'unlocked'
    balanceLockedKey = 'locked'
    balanceTotalKey = 'total'
    balanceOutputKey = 'outputs'
    balanceOutputAmountKey = 'amount'
    statsLockedCoinsKey = 'lockedCoins'
    statsUnlockedCoinsKey = 'coins'
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

    b = r.hget('a:' + addr[:6], addr[6:])
    if b == None:
        continue # skip nil wallets
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
stats = dec(r.get('stats'))
uc = int(stats[statsUnlockedCoinsKey])
if uc != coinsUnlocked:
    raise Exception('unexpected total unlocked coins: ' + str(uc) + ' != ' + str(coinsUnlocked))
lc = int(stats[statsLockedCoinsKey])
if lc != coinsLocked:
    raise Exception('unexpected total locked coins: ' + str(lc) + ' != ' + str(coinsLocked))


print('validated balance of ' + str(ac) + ' wallets successfully, decoding them from ' + str(args.encoding) + ' :)')
