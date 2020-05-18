package bridge

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	tfeth "github.com/threefoldtech/rivine-extension-erc20/api"
	"github.com/threefoldtech/rivine-extension-erc20/api/bridge/contract"
	erc20types "github.com/threefoldtech/rivine-extension-erc20/types"
)

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

const (
	// retryDelay is the delay to retry calls when there are no peers
	retryDelay = time.Second * 15
)

// BridgeContract exposes a higher lvl api for specific contract bindings. In case of proxy contracts,
// the bridge needs to use the bindings of the implementation contract, but the address of the proxy.
type BridgeContract struct {
	networkConfig tfeth.NetworkConfiguration // Ethereum network

	lc *LightClient

	filter     *contract.TTFT20Filterer
	transactor *contract.TTFT20Transactor
	caller     *contract.TTFT20Caller

	contract *bind.BoundContract
	abi      abi.ABI

	// cache some stats in case they might be usefull
	head    *types.Header // Current head header of the bridge
	balance *big.Int      // The current balance of the bridge (note: ethers only!)
	nonce   uint64        // Current pending nonce of the bridge
	price   *big.Int      // Current gas price to issue funds with

	lock sync.RWMutex // Lock protecting the bridge's internals
}

// GetContractAdress returns the address of this contract
func (bridge *BridgeContract) GetContractAdress() common.Address {
	return bridge.networkConfig.ContractAddress
}

// NewBridgeContract creates a new wrapper for an allready deployed contract
func NewBridgeContract(networkName string, bootnodes []string, contractAddress string, port int, accountJSON, accountPass string, datadir string, cancel <-chan struct{}) (*BridgeContract, error) {
	// load correct network config
	networkConfig, err := tfeth.GetEthNetworkConfiguration(networkName)
	if err != nil {
		return nil, err
	}
	// override contract address if it's provided
	if contractAddress != "" {
		networkConfig.ContractAddress = common.HexToAddress(contractAddress)
		// TODO: validate ABI of contract,
		//       see https://github.com/threefoldtech/rivine-extension-erc20/issues/3
	}

	bootstrapNodes, err := networkConfig.GetBootnodes(bootnodes)
	log.Info("bootnodes", "nodes", bootstrapNodes)
	if err != nil {
		return nil, err
	}
	lc, err := NewLightClient(LightClientConfig{
		Port:           port,
		DataDir:        datadir,
		BootstrapNodes: bootstrapNodes,
		NetworkName:    networkConfig.NetworkName,
		NetworkID:      networkConfig.NetworkID,
		GenesisBlock:   networkConfig.GenesisBlock,
	})
	if err != nil {
		return nil, err
	}
	err = lc.LoadAccount(accountJSON, accountPass)
	if err != nil {
		return nil, err
	}

	filter, err := contract.NewTTFT20Filterer(networkConfig.ContractAddress, lc.Client)
	if err != nil {
		return nil, err
	}

	transactor, err := contract.NewTTFT20Transactor(networkConfig.ContractAddress, lc.Client)
	if err != nil {
		return nil, err
	}

	caller, err := contract.NewTTFT20Caller(networkConfig.ContractAddress, lc.Client)
	if err != nil {
		return nil, err
	}

	contract, abi, err := bindTTFT20(networkConfig.ContractAddress, lc.Client, lc.Client, lc.Client)
	if err != nil {
		return nil, err
	}

	return &BridgeContract{
		networkConfig: networkConfig,
		lc:            lc,
		filter:        filter,
		transactor:    transactor,
		caller:        caller,
		contract:      contract,
		abi:           abi,
	}, nil
}

// Close terminates the Ethereum connection and tears down the stack.
func (bridge *BridgeContract) Close() error {
	return bridge.lc.Close()
}

// AccountAddress returns the account address of the bridge contract
func (bridge *BridgeContract) AccountAddress() (common.Address, error) {
	return bridge.lc.AccountAddress()
}

// LightClient returns the LightClient driving this bridge contract
func (bridge *BridgeContract) LightClient() *LightClient {
	return bridge.lc
}

// ABI returns the parsed and bound ABI driving this bridge contract
func (bridge *BridgeContract) ABI() abi.ABI {
	return bridge.abi
}

// Refresh attempts to retrieve the latest header from the chain and extract the
// associated bridge balance and nonce for connectivity caching.
func (bridge *BridgeContract) Refresh(head *types.Header) error {
	// Ensure a state update does not run for too long
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// If no header was specified, use the current chain head
	var err error
	if head == nil {
		if head, err = bridge.lc.HeaderByNumber(ctx, nil); err != nil {
			return err
		}
	}
	// Retrieve the balance, nonce and gas price from the current head
	var (
		nonce   uint64
		price   *big.Int
		balance *big.Int
	)
	if price, err = bridge.lc.SuggestGasPrice(ctx); err != nil {
		return err
	}
	if balance, err = bridge.lc.AccountBalanceAt(ctx, head.Number); err != nil {
		return err
	}
	// Everything succeeded, update the cached stats
	bridge.lock.Lock()
	bridge.head, bridge.balance = head, balance
	bridge.price, bridge.nonce = price, nonce
	bridge.lock.Unlock()
	return nil
}

// Loop subscribes to new eth heads. If a new head is received, it is passed on the given channel,
// after which the internal stats are updated if no update is already in progress
func (bridge *BridgeContract) Loop(ch chan<- *types.Header) {
	log.Info("Subscribing to eth headers")
	// channel to receive head updates from client on
	heads := make(chan *types.Header, 16)
	// subscribe to head upates
	sub, err := bridge.lc.SubscribeNewHead(context.Background(), heads)
	if err != nil {
		log.Error("Failed to subscribe to head events", "err", err)
	}
	defer sub.Unsubscribe()
	// channel so we can update the internal state from the heads
	update := make(chan *types.Header)
	go func() {
		for head := range update {
			// old heads should be ignored during a chain sync after some downtime
			if err := bridge.Refresh(head); err != nil {
				log.Warn("Failed to update state", "block", head.Number, "err", err)
			}
			log.Debug("Internal stats updated", "block", head.Number, "account balance", bridge.balance, "gas price", bridge.price, "nonce", bridge.nonce)
		}
	}()
	for head := range heads {
		ch <- head
		select {
		// only process new head if another isn't being processed yet
		case update <- head:
			log.Debug("Processing new head")
		default:
			log.Debug("Ignoring current head, update already in progress")
		}
	}
	log.Error("Bridge state update loop ended")
}

// SubscribeTransfers subscribes to new Transfer events on the given contract. This call blocks
// and prints out info about any transfer as it happened
func (bridge *BridgeContract) SubscribeTransfers() error {
	sink := make(chan *contract.TTFT20Transfer)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}
	sub, err := bridge.filter.WatchTransfer(opts, sink, nil, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	for {
		select {
		case err = <-sub.Err():
			return err
		case transfer := <-sink:
			log.Info("Noticed transfer event", "from", transfer.From, "to", transfer.To, "amount", transfer.Tokens)
		}
	}
}

// SubscribeMint subscribes to new Mint events on the given contract. This call blocks
// and prints out info about any mint as it happened
func (bridge *BridgeContract) SubscribeMint() error {
	sink := make(chan *contract.TTFT20Mint)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}
	sub, err := bridge.filter.WatchMint(opts, sink, nil, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	for {
		select {
		case err = <-sub.Err():
			return err
		case mint := <-sink:
			log.Info("Noticed mint event", "receiver", mint.Receiver, "amount", mint.Tokens, "TFT tx id", mint.Txid)
		}
	}
}

// WithdrawEvent holds relevant information about a withdraw event
type WithdrawEvent struct {
	receiver    common.Address
	amount      *big.Int
	txHash      common.Hash
	blockHash   common.Hash
	blockHeight uint64
}

// Receiver of the withdraw
func (w WithdrawEvent) Receiver() common.Address {
	return w.receiver
}

// Amount withdrawn
func (w WithdrawEvent) Amount() *big.Int {
	return w.amount
}

// TxHash hash of the transaction
func (w WithdrawEvent) TxHash() common.Hash {
	return w.txHash
}

// BlockHash of the containing block
func (w WithdrawEvent) BlockHash() common.Hash {
	return w.blockHash
}

// BlockHeight of the containing block
func (w WithdrawEvent) BlockHeight() uint64 {
	return w.blockHeight
}

// GetPastWithdraws gets a list of past withdraw events between two block numbers
func (bridge *BridgeContract) GetPastWithdraws(startHeight uint64, endHeight *uint64) ([]WithdrawEvent, error) {
	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startHeight, End: endHeight}
	iterator, err := bridge.filter.FilterWithdraw(filterOpts, nil)
	for IsNoPeerErr(err) {
		time.Sleep(time.Second * 5)
		log.Debug("Retrying fetching past withdraws")
		iterator, err = bridge.filter.FilterWithdraw(filterOpts, nil)
	}
	if err != nil {
		log.Error("Creating past withdraw event iterator failed", "err", err)
		return nil, err
	}

	var withdraws []WithdrawEvent
	for iterator.Next() {
		withdraw := iterator.Event
		if withdraw.Raw.Removed {
			continue
		}
		withdraws = append(withdraws, WithdrawEvent{receiver: withdraw.Receiver, amount: withdraw.Tokens, txHash: withdraw.Raw.TxHash, blockHash: withdraw.Raw.BlockHash, blockHeight: withdraw.Raw.BlockNumber})
	}
	// Make sure to check the iterator for errors
	return withdraws, iterator.Error()
}

// SubscribeWithdraw subscribes to new Withdraw events on the given contract. This call blocks
// and prints out info about any withdraw as it happened
func (bridge *BridgeContract) SubscribeWithdraw(wc chan<- WithdrawEvent, startHeight uint64) error {
	log.Info("Subscribing to withdraw events", "start height", startHeight)
	sink := make(chan *contract.TTFT20Withdraw)
	watchOpts := &bind.WatchOpts{Context: context.Background(), Start: nil}
	pastWithdraws, err := bridge.GetPastWithdraws(startHeight, nil)
	if err != nil {
		return err
	}
	for _, w := range pastWithdraws {
		// notify about all the past withdraws
		wc <- w
	}
	sub, err := bridge.WatchWithdraw(watchOpts, sink, nil)
	if err != nil {
		log.Error("Subscribing to withdraw events failed", "err", err)
		return err
	}
	defer sub.Unsubscribe()
	for {
		select {
		case err = <-sub.Err():
			return err
		case withdraw := <-sink:
			if withdraw.Raw.Removed {
				// ignore removed events
				continue
			}
			log.Info("Noticed withdraw event", "receiver", withdraw.Receiver, "amount", withdraw.Tokens)
			wc <- WithdrawEvent{
				receiver:    withdraw.Receiver,
				amount:      withdraw.Tokens,
				txHash:      withdraw.Raw.TxHash,
				blockHash:   withdraw.Raw.BlockHash,
				blockHeight: withdraw.Raw.BlockNumber,
			}
		}
	}
}

// WatchWithdraw is a free log subscription operation binding the contract event 0x884edad9ce6fa2440d8a54cc123490eb96d2768479d49ff9c7366125a9424364.
//
// Solidity: e Withdraw(receiver indexed address, tokens uint256)
//
// This method is copied from the generated bindings and slightly modified, so we can add logic to stay backwards compatible with the old withdraw event signature
func (bridge *BridgeContract) WatchWithdraw(opts *bind.WatchOpts, sink chan<- *contract.TTFT20Withdraw, receiver []common.Address) (event.Subscription, error) {

	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := bridge.contract.WatchLogs(opts, "Withdraw", receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(contract.TTFT20Withdraw)
				if err := bridge.contract.UnpackLog(event, "Withdraw", log); err != nil {
					// Could be that this is an old style withdraw event, try unpacking that
					event2 := new(contract.TTFT20WithdrawV0)
					if err2 := bridge.contract.UnpackLog(event2, "withdraw", log); err2 != nil {
						// return the original error though
						return err
					}
					// if we get here we have an old style withdraw, convert this to the new format
					event.Raw = event2.Raw
					event.Receiver = event2.Receiver
					event.Tokens = event2.Tokens
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// SubscribeRegisterWithdrawAddress subscribes to new RegisterWithdrawalAddress events on the given contract. This call blocks
// and prints out info about any RegisterWithdrawalAddress event as it happened
func (bridge *BridgeContract) SubscribeRegisterWithdrawAddress() error {
	sink := make(chan *contract.TTFT20RegisterWithdrawalAddress)
	opts := &bind.WatchOpts{Context: context.Background(), Start: nil}
	sub, err := bridge.filter.WatchRegisterWithdrawalAddress(opts, sink, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	for {
		select {
		case err = <-sub.Err():
			return err
		case withdraw := <-sink:
			log.Info("Noticed withdraw address registration event", "address", withdraw.Addr)
		}
	}
}

// TransferFunds transfers funds from one address to another
func (bridge *BridgeContract) TransferFunds(recipient common.Address, amount *big.Int) error {
	err := bridge.transferFunds(recipient, amount)
	for IsNoPeerErr(err) {
		time.Sleep(retryDelay)
		err = bridge.transferFunds(recipient, amount)
	}
	return err
}

func (bridge *BridgeContract) transferFunds(recipient common.Address, amount *big.Int) error {
	if amount == nil {
		return errors.New("invalid amount")
	}
	accountAddress, err := bridge.lc.AccountAddress()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	opts := &bind.TransactOpts{
		Context: ctx, From: accountAddress,
		Signer: bridge.getSignerFunc(),
		Value:  nil, Nonce: nil, GasLimit: 0, GasPrice: nil,
	}
	_, err = bridge.transactor.Transfer(opts, recipient, amount)
	return err
}

func (bridge *BridgeContract) Mint(receiver erc20types.ERC20Address, amount *big.Int, txID string) error {
	err := bridge.mint(receiver, amount, txID)
	for IsNoPeerErr(err) {
		time.Sleep(retryDelay)
		err = bridge.mint(receiver, amount, txID)
	}
	return err
}

func (bridge *BridgeContract) mint(receiver erc20types.ERC20Address, amount *big.Int, txID string) error {
	log.Info("Calling mint function in contract")
	if amount == nil {
		return errors.New("invalid amount")
	}
	accountAddress, err := bridge.lc.AccountAddress()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	opts := &bind.TransactOpts{
		Context: ctx, From: accountAddress,
		Signer: bridge.getSignerFunc(),
		Value:  nil, Nonce: nil, GasLimit: 0, GasPrice: nil,
	}
	_, err = bridge.transactor.MintTokens(opts, common.Address(receiver), amount, txID)
	return err
}

func (bridge *BridgeContract) IsMintTxID(txID string) (bool, error) {
	res, err := bridge.isMintTxID(txID)
	for IsNoPeerErr(err) {
		time.Sleep(retryDelay)
		res, err = bridge.isMintTxID(txID)
	}
	return res, err
}

func (bridge *BridgeContract) isMintTxID(txID string) (bool, error) {
	log.Info("Calling isMintID")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	opts := &bind.CallOpts{Context: ctx}
	return bridge.caller.IsMintID(opts, txID)
}

func (bridge *BridgeContract) RegisterWithdrawalAddress(address erc20types.ERC20Address) error {
	err := bridge.registerWithdrawalAddress(address)
	for IsNoPeerErr(err) {
		time.Sleep(retryDelay)
		err = bridge.registerWithdrawalAddress(address)
	}
	return err
}

func (bridge *BridgeContract) registerWithdrawalAddress(address erc20types.ERC20Address) error {
	log.Info("Calling register withdrawal address function in contract")
	accountAddress, err := bridge.lc.AccountAddress()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	opts := &bind.TransactOpts{
		Context: ctx, From: accountAddress,
		Signer: bridge.getSignerFunc(),
		Value:  nil, Nonce: nil, GasLimit: 0, GasPrice: nil,
	}
	_, err = bridge.transactor.RegisterWithdrawalAddress(opts, common.Address(address))
	return err
}

func (bridge *BridgeContract) IsWithdrawalAddress(address erc20types.ERC20Address) (bool, error) {
	success, err := bridge.isWithdrawalAddress(address)
	for IsNoPeerErr(err) {
		time.Sleep(retryDelay)
		success, err = bridge.isWithdrawalAddress(address)
	}
	return success, err
}

func (bridge *BridgeContract) isWithdrawalAddress(address erc20types.ERC20Address) (bool, error) {
	log.Info("Calling isWithdrawalAddress function in contract")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	opts := &bind.CallOpts{Context: ctx}
	return bridge.caller.IsWithdrawalAddress(opts, common.Address(address))
}

func (bridge *BridgeContract) getSignerFunc() bind.SignerFn {
	return func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		accountAddress, err := bridge.lc.AccountAddress()
		if err != nil {
			return nil, err
		}
		if address != accountAddress {
			return nil, errors.New("not authorized to sign this account")
		}
		networkID := int64(bridge.networkConfig.NetworkID)
		return bridge.lc.SignTx(tx, big.NewInt(networkID))
	}
}

func (bridge *BridgeContract) TokenBalance(address common.Address) (*big.Int, error) {
	log.Info("Calling TokenBalance function in contract")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	opts := &bind.CallOpts{Context: ctx}
	return bridge.caller.BalanceOf(opts, common.Address(address))
}

func (bridge *BridgeContract) EthBalance() (*big.Int, error) {
	err := bridge.Refresh(nil) // force a refresh
	return bridge.balance, err
}

// bindTTFT20 binds a generic wrapper to an already deployed contract.
//
// This method is copied from the generated bindings as a convenient way to get a *bind.Contract, as this is needed to implement the WatchWithdraw function ourselves
func bindTTFT20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, abi.ABI, error) {
	parsed, err := abi.JSON(strings.NewReader(contract.TTFT20ABI))
	if err != nil {
		return nil, parsed, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), parsed, nil
}
