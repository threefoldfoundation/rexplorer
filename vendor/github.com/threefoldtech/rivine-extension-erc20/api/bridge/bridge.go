package bridge

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	erc20types "github.com/threefoldtech/rivine-extension-erc20/types"
	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/types"
)

const (
	// TFTBlockDelay is the amount of blocks to wait before
	// pushing tft transactions to the ethereum contract
	TFTBlockDelay = 6
	// EthBlockDelay is the amount of blocks to wait before
	// pushing eth transaction to the tfchain network
	EthBlockDelay = 30
)

// Bridge is a high lvl structure which listens on contract events and bridge-related
// tfchain transactions, and handles them
type Bridge struct {
	cs modules.ConsensusSet
	tp modules.TransactionPool

	persistDir string
	persist    persistence
	buffer     *blockBuffer

	bcInfo     types.BlockchainInfo
	chainCts   types.ChainConstants
	txVersions erc20types.TransactionVersions

	bridgeContract *BridgeContract

	mut sync.Mutex
}

// NewBridge creates a new Bridge.
func NewBridge(cs modules.ConsensusSet, tp modules.TransactionPool, ethPort uint16, accountJSON, accountPass string, ethNetworkName string, bootnodes []string, contractAddress string, datadir string, bcInfo types.BlockchainInfo, chainCts types.ChainConstants, txVersions erc20types.TransactionVersions, cancel <-chan struct{}) (*Bridge, error) {
	contract, err := NewBridgeContract(ethNetworkName, bootnodes, contractAddress, int(ethPort), accountJSON, accountPass, filepath.Join(datadir, "eth"), cancel)
	if err != nil {
		return nil, err
	}

	bridge := &Bridge{
		cs:             cs,
		tp:             tp,
		persistDir:     datadir,
		bcInfo:         bcInfo,
		chainCts:       chainCts,
		txVersions:     txVersions,
		bridgeContract: contract,
	}

	err = bridge.initPersist()
	if err != nil {
		return nil, errors.New("bridge persistence startup failed: " + err.Error())
	}

	bridge.buffer = newBlockBuffer(TFTBlockDelay)

	return bridge, nil
}

// commitWithdrawTransaction verifies and (if successfull) commits a withdraw transaction on
// the tfchain (thus creating new tokens)
func (bridge *Bridge) commitWithdrawTransaction(tx erc20types.ERC20CoinCreationTransaction) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	// verify that the transaction is still present in the block (no forks occurred)
	_, _, err := bridge.bridgeContract.lc.FetchTransaction(ctx, common.Hash(tx.BlockID), common.Hash(tx.TransactionID))
	if err != nil {
		return err
	}
	// accept transaction, ignore duplicate errors which might occur if we are syncing a new bridge
	if err := bridge.tp.AcceptTransactionSet([]types.Transaction{tx.Transaction(bridge.txVersions.ERC20CoinCreation)}); err != nil && err != modules.ErrDuplicateTransactionSet {
		return err
	}
	return nil
}

// Close bridge
func (bridge *Bridge) Close() error {
	bridge.mut.Lock()
	defer bridge.mut.Unlock()
	err := bridge.bridgeContract.Close()
	bridge.cs.Unsubscribe(bridge)
	return err
}

func (bridge *Bridge) mint(receiver erc20types.ERC20Address, amount types.Currency, txID types.TransactionID) error {
	// check if we already know this ID
	known, err := bridge.bridgeContract.IsMintTxID(txID.String())
	if err != nil {
		return err
	}
	if known {
		// we already know this withdrawal address, so ignore the transaction
		return nil
	}
	return bridge.bridgeContract.Mint(receiver, amount.Big(), txID.String())
}

func (bridge *Bridge) registerWithdrawalAddress(key types.PublicKey) error {
	// convert public key to unlockhash to eth address
	uh, err := types.NewPubKeyUnlockHash(key)
	if err != nil {
		return err
	}
	erc20addr, err := erc20types.ERC20AddressFromUnlockHash(uh)
	if err != nil {
		return err
	}
	// check if we already know this withdraw address
	known, err := bridge.bridgeContract.IsWithdrawalAddress(erc20addr)
	if err != nil {
		return err
	}
	if known {
		// we already know this withdrawal address, so ignore the transaction
		return nil
	}
	return bridge.bridgeContract.RegisterWithdrawalAddress(erc20addr)
}

// GetClient returns bridgecontract lightclient
func (bridge *Bridge) GetClient() *LightClient {
	return bridge.bridgeContract.LightClient()
}

// GetBridgeContract returns this bridge's contract.
func (bridge *Bridge) GetBridgeContract() *BridgeContract {
	return bridge.bridgeContract
}

// Start the main processing loop of the bridge
func (bridge *Bridge) Start(cs modules.ConsensusSet, erc20Registry erc20types.ERC20Registry, cancel <-chan struct{}) error {
	err := cs.ConsensusSetSubscribe(bridge, bridge.persist.RecentChange, cancel)
	if err != nil {
		return fmt.Errorf("bridged: failed to subscribe to consensus set: %v", err)
	}
	log.Info("Subscribed to tfchain consensus set")

	heads := make(chan *ethtypes.Header)

	go bridge.bridgeContract.Loop(heads)

	// subscribing to these events is not needed for operational purposes, but might be nice to get some info
	go bridge.bridgeContract.SubscribeTransfers()
	go bridge.bridgeContract.SubscribeMint()
	go bridge.bridgeContract.SubscribeRegisterWithdrawAddress()

	withdrawChan := make(chan WithdrawEvent)
	go bridge.bridgeContract.SubscribeWithdraw(withdrawChan, bridge.persist.EthHeight)
	go func() {
		txMap := make(map[erc20types.ERC20Hash]WithdrawEvent)
		for {
			select {
			// Remember new withdraws
			case we := <-withdrawChan:
				// Check if the withdraw is valid
				_, found, err := erc20Registry.GetTFTAddressForERC20Address(erc20types.ERC20Address(we.receiver))
				if err != nil {
					log.Error(fmt.Sprintf("Retrieving TFT address for registered ERC20 address %v errored: %v", we.receiver, err))
					continue
				}
				if !found {
					log.Error(fmt.Sprintf("Failed to retrieve TFT address for registered ERC20 Withdrawal address %v", we.receiver))
					continue
				}
				// remember the withdraw
				txMap[erc20types.ERC20Hash(we.txHash)] = we
				log.Info("Remembering withdraw event", "txHash", we.TxHash(), "height", we.BlockHeight())

			// If we get a new head, check every withdraw we have to see if it has matured
			case head := <-heads:
				bridge.mut.Lock()
				for id := range txMap {
					we := txMap[id]
					if head.Number.Uint64() >= we.blockHeight+EthBlockDelay {
						log.Info("Attempting to create an ERC20 withdraw tx", "ethTx", we.TxHash())
						// we waited long enough, create transaction and push it
						uh, found, err := erc20Registry.GetTFTAddressForERC20Address(erc20types.ERC20Address(we.receiver))
						if err != nil {
							log.Error(fmt.Sprintf("Retrieving TFT address for registered ERC20 address %v errored: %v", we.receiver, err))
							continue
						}
						if !found {
							log.Error(fmt.Sprintf("Failed to retrieve TFT address for registered ERC20 Withdrawal address %v", we.receiver))
							continue
						}

						tx := erc20types.ERC20CoinCreationTransaction{}
						tx.Address = uh

						// define the txFee
						tx.TransactionFee = bridge.chainCts.MinimumTransactionFee

						// define the value, which is the value withdrawn minus the fees
						tx.Value = types.NewCurrency(we.amount).Sub(tx.TransactionFee)

						// fill in the other info
						tx.TransactionID = erc20types.ERC20Hash(we.txHash)
						tx.BlockID = erc20types.ERC20Hash(we.blockHash)

						if err := bridge.commitWithdrawTransaction(tx); err != nil {
							log.Error("Failed to create ERC20 Withdraw transaction", "err", err)
							continue
						}

						log.Info("Created ERC20 -> TFT transaction", "txid", tx.Transaction(bridge.txVersions.ERC20CoinCreation).ID())

						// forget about our tx
						delete(txMap, id)
					}
				}

				bridge.persist.EthHeight = head.Number.Uint64() - EthBlockDelay
				// Check for underflow
				if bridge.persist.EthHeight > head.Number.Uint64() {
					bridge.persist.EthHeight = 0
				}
				if err := bridge.save(); err != nil {
					log.Error("Failed to save bridge persistency", "err", err)
				}

				bridge.mut.Unlock()
			}
		}
	}()
	return nil
}
