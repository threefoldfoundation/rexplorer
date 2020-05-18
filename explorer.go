package main

import (
	"fmt"
	"sync"

	dtypes "github.com/threefoldfoundation/rexplorer/pkg/database/types"
	"github.com/threefoldfoundation/rexplorer/pkg/types"

	threebottypes "github.com/threefoldfoundation/tfchain/extensions/threebot/types"
	tfchaintypes "github.com/threefoldfoundation/tfchain/pkg/types"
	erc20types "github.com/threefoldtech/rivine-extension-erc20/types"
	minting "github.com/threefoldtech/rivine/extensions/minting"
	"github.com/threefoldtech/rivine/modules"
	rivinetypes "github.com/threefoldtech/rivine/types"
)

// Explorer defines the custom (internal) explorer module,
// used to dump the data of a tfchain network in a meaningful way.
type Explorer struct {
	db    Database
	state dtypes.ExplorerState
	stats types.NetworkStats

	cs modules.ConsensusSet

	bcInfo        rivinetypes.BlockchainInfo
	chainCts      rivinetypes.ChainConstants
	mintingplugin *minting.Plugin

	mut sync.Mutex
}

// NewExplorer creates a new custom intenral explorer module.
// See Explorer for more information.
func NewExplorer(db Database, cs modules.ConsensusSet, bcInfo rivinetypes.BlockchainInfo, chainCts rivinetypes.ChainConstants, mintingplugin *minting.Plugin, cancel <-chan struct{}) (*Explorer, error) {
	state, err := db.GetExplorerState()
	if err != nil {
		return nil, fmt.Errorf("failed to get explorer state from db: %v", err)
	}
	stats, err := db.GetNetworkStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get network stats from db: %v", err)
	}
	explorer := &Explorer{
		db:            db,
		state:         state,
		stats:         stats,
		cs:            cs,
		bcInfo:        bcInfo,
		chainCts:      chainCts,
		mintingplugin: mintingplugin,
	}
	err = cs.ConsensusSetSubscribe(explorer, state.CurrentChangeID.ConsensusChangeID, cancel)
	if err != nil {
		return nil, fmt.Errorf("explorer: failed to subscribe to consensus set: %v", err)
	}
	return explorer, nil
}

// Close the Explorer module.
func (explorer *Explorer) Close() error {
	explorer.mut.Lock()
	defer explorer.mut.Unlock()
	explorer.cs.Unsubscribe(explorer)
	return explorer.db.Close()
}

// ProcessConsensusChange implements modules.ConsensusSetSubscriber,
// used to apply/revert blocks to/from our Redis-stored data.
func (explorer *Explorer) ProcessConsensusChange(css modules.ConsensusChange) {
	explorer.mut.Lock()
	defer explorer.mut.Unlock()

	var err error

	// update reverted blocks
	for _, block := range css.RevertedBlocks {
		// revert miner payouts
		for i, mp := range block.MinerPayouts {
			explorer.stats.CoinOutputCount--
			if i == 0 {
				explorer.stats.MinerPayoutCount--
				explorer.stats.MinerPayouts = explorer.stats.MinerPayouts.Sub(types.AsCurrency(mp.Value))
				// block reward is always created money, no matter what txs the block contains
				explorer.stats.Coins = explorer.stats.Coins.Sub(types.AsCurrency(mp.Value))
			} else if i == 1 {
				explorer.stats.TransactionFeeCount--
				explorer.stats.TransactionFees = explorer.stats.TransactionFees.Sub(types.AsCurrency(mp.Value))
			} else {
				explorer.stats.FoundationFeeCount--
				explorer.stats.FoundationFees = explorer.stats.FoundationFees.Sub(types.AsCurrency(mp.Value))
			}
			state, err := explorer.db.RevertCoinOutput(types.AsCoinOutputID(block.MinerPayoutID(uint64(i))))
			if err != nil {
				panic(fmt.Sprintf("failed to revert miner payout of %s to %s: %v",
					mp.UnlockHash.String(), mp.Value.String(), err))
			}
			if state == dtypes.CoinOutputStateLocked {
				explorer.stats.LockedCoinOutputCount--
				explorer.stats.LockedCoins = explorer.stats.LockedCoins.Sub(types.AsCurrency(mp.Value))
			}
		}
		// revert txs
		for _, tx := range block.Transactions {
			var isCoinCreationTransaction bool

			switch tx.Version {
			case threebottypes.TransactionVersionBotRegistration:
				// decrease the 3bot reg tx count
				explorer.stats.ThreeBotRegistrationTransactionCount--

				// delete the 3Bot record
				err = explorer.db.DeleteBotRecord(explorer.state.NextThreeBotID)
				if err != nil {
					panic(fmt.Sprintf("failed to revert BotRegistrationTransaction: %v", err))
				}
				// decrement the sequential counter used for 3Bots in this explorer
				explorer.state.NextThreeBotID.Decrement()

			case threebottypes.TransactionVersionBotRecordUpdate:
				// decrease the 3bot update tx count
				explorer.stats.ThreeBotUpdateTransactionCount--

				// expose the 3bot record update data from the tx
				butx, err := threebottypes.BotRecordUpdateTransactionFromTransaction(tx)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret v145 tx as a BotRecordUpdateTransaction: %v", err))
				}
				// update the 3Bot record in a revert way
				err = explorer.db.UpdateBotRecord(
					types.NewBotIDFromTfchainBotID(butx.Identifier),
					wrapTfchainBotUpdateRevertFunction(butx.RevertBotRecordUpdate))
				if err != nil {
					panic(fmt.Sprintf("failed to revert BotRecordUpdateTransaction: %v", err))
				}

			case threebottypes.TransactionVersionBotNameTransfer:
				// decrease the 3bot update tx count (also used for name transfers)
				explorer.stats.ThreeBotUpdateTransactionCount--

				// expose the 3bot record name transfer data from the tx
				bnttx, err := threebottypes.BotNameTransferTransactionFromTransaction(tx)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret v146 tx as a BotNameTransferTransaction: %v", err))
				}
				// update both 3Bot records in a revert way
				err = explorer.db.UpdateBotRecord(
					types.NewBotIDFromTfchainBotID(bnttx.Sender.Identifier),
					wrapTfchainBotUpdateRevertFunction(bnttx.RevertSenderBotRecordUpdate))
				if err != nil {
					panic(fmt.Sprintf("failed to revert BotRecordUpdateTransaction for sender: %v", err))
				}
				err = explorer.db.UpdateBotRecord(
					types.NewBotIDFromTfchainBotID(bnttx.Receiver.Identifier),
					wrapTfchainBotUpdateRevertFunction(bnttx.RevertReceiverBotRecordUpdate))
				if err != nil {
					panic(fmt.Sprintf("failed to revert BotRecordUpdateTransaction for receiver: %v", err))
				}

			case tfchaintypes.TransactionVersionERC20Conversion:
				explorer.stats.CoinBurnTransactionCount--

				// expose the ERC20 Conversion Tx data from the Tx
				erc20CTx, err := erc20types.ERC20ConvertTransactionFromTransaction(tx, tx.Version)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret tx as a ERC20 Conversion Tx: %v", err))
				}
				// add value again, as it was burned
				explorer.stats.Coins = explorer.stats.Coins.Add(types.AsCurrency(erc20CTx.Value))

			case tfchaintypes.TransactionVersionERC20CoinCreation:
				explorer.stats.CoinCreationTransactionCount--
				isCoinCreationTransaction = true
				// sub miner fees if it was a coin created tx,
				// as fees are created from new coins in a coin creation tx as well
				for _, fee := range tx.MinerFees {
					explorer.stats.Coins = explorer.stats.Coins.Sub(types.AsCurrency(fee))
				}

			case tfchaintypes.TransactionVersionERC20AddressRegistration:
				// expose the ERC20 Conversion Tx data from the Tx
				erc20ARTx, err := erc20types.ERC20AddressRegistrationTransactionFromTransaction(tx, tx.Version)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret tx as a ERC20 Address Registration Tx: %v", err))
				}
				tftAddress, err := rivinetypes.NewPubKeyUnlockHash(erc20ARTx.PublicKey)
				if err != nil {
					panic(fmt.Sprintf("failed to create a rivine address from a ERC20 Address Registration Tx: %v", err))
				}
				erc20Address, err := erc20types.ERC20AddressFromUnlockHash(tftAddress)
				if err != nil {
					panic(fmt.Sprintf("failed to create an erc20 address from a ERC20 Address Registration Tx: %v", err))
				}
				err = explorer.db.DeleteERC20AddressRegistration(types.AsERC20Address(erc20Address))
				if err != nil {
					panic(fmt.Sprintf("failed to unregister ERC20 Address %s: %v", erc20Address.String(), err))
				}

			case tfchaintypes.TransactionVersionCoinCreation:
				explorer.stats.CoinCreationTransactionCount--
				isCoinCreationTransaction = true
				// sub miner fees if it was a coin created tx,
				// as fees are created from new coins in a coin creation tx as well
				for _, fee := range tx.MinerFees {
					explorer.stats.Coins = explorer.stats.Coins.Sub(types.AsCurrency(fee))
				}

			case tfchaintypes.TransactionVersionMinterDefinition:
				// decrease coin creator tx count
				explorer.stats.CoinCreatorDefinitionTransactionCount--

				// set the previous mint condition as the current coin creators
				previousMintConditionHeight := explorer.stats.BlockHeight.BlockHeight - 1
				previousMintCondition, err := explorer.mintingplugin.GetMintConditionAt(previousMintConditionHeight)
				if err != nil {
					panic(fmt.Sprintf("failed to get mint condition for height %d: %v", previousMintConditionHeight, err))
				}
				// set previous coin creators
				err = explorer.setMintCondition(previousMintCondition)
				if err != nil {
					panic(fmt.Sprintf("failed to set mint condition in the explorer db: %v", err))
				}

				// sub miner fees if it was a minter definition tx,
				// as fees are created from new coins in a minter definition tx as well
				for _, fee := range tx.MinerFees {
					explorer.stats.Coins = explorer.stats.Coins.Sub(types.AsCurrency(fee))
				}
			}

			explorer.stats.TransactionCount--

			if len(tx.CoinInputs) > 0 || len(tx.BlockStakeOutputs) > 1 {
				explorer.stats.ValueTransactionCount--
			}
			// revert coin inputs
			for _, ci := range tx.CoinInputs {
				explorer.stats.CoinInputCount--
				err := explorer.db.RevertCoinInput(types.AsCoinOutputID(ci.ParentID))
				if err != nil {
					panic(fmt.Sprintf("failed to revert coin input %s: %v", ci.ParentID.String(), err))
				}
			}
			// revert coin outputs
			for i, co := range tx.CoinOutputs {
				explorer.stats.CoinOutputCount--
				id := tx.CoinOutputID(uint64(i))
				state, err := explorer.db.RevertCoinOutput(types.AsCoinOutputID(id))
				if err != nil {
					panic(fmt.Sprintf("failed to revert coin output %s: %v", id.String(), err))
				}
				// only revert total coin count if output was part of a coin creation txs,
				// we assume that a genesis block can never revert, as that would change the entire identity of a blockchain
				if isCoinCreationTransaction {
					explorer.stats.Coins = explorer.stats.Coins.Sub(types.AsCurrency(co.Value))
				}
				// always count locked coins
				if state == dtypes.CoinOutputStateLocked {
					explorer.stats.LockedCoinOutputCount--
					explorer.stats.LockedCoins = explorer.stats.LockedCoins.Sub(types.AsCurrency(co.Value))
				}
			}
		}

		if block.ParentID != (rivinetypes.BlockID{}) {
			explorer.stats.BlockHeight.Decrease()
		}
		explorer.stats.Timestamp = types.AsTimestamp(block.Timestamp)

		// returns the total amount of coins that have been locked
		n, coins, err := explorer.db.RevertCoinOutputLocks(explorer.stats.BlockHeight, explorer.stats.Timestamp)
		if err != nil {
			panic(fmt.Sprintf("failed to lock coin outputs at height=%d and time=%d: %v",
				explorer.stats.BlockHeight, explorer.stats.Timestamp, err))
		}
		if n > 0 {
			explorer.stats.LockedCoinOutputCount += n
			explorer.stats.LockedCoins = explorer.stats.LockedCoins.Add(coins)
		}
	}

	// update applied blocks
	for _, block := range css.AppliedBlocks {
		isGenesisBlock := block.ParentID == (rivinetypes.BlockID{})
		if !isGenesisBlock {
			explorer.stats.BlockHeight.Increase()
		} else {
			// no need to increase coin creator tx count, as this is not a coin creator tx

			// get genesis mint condition
			genesisMintCondition, err := explorer.mintingplugin.GetMintConditionAt(0)
			if err != nil {
				panic(fmt.Sprintf("failed to get genesis mint condition: %v", err))
			}
			// set initial coin creators
			err = explorer.setMintCondition(genesisMintCondition)
			if err != nil {
				panic(fmt.Sprintf("failed to set genesis mint condition in the explorer db: %v", err))
			}
		}
		explorer.stats.Timestamp = types.AsTimestamp(block.Timestamp)
		// returns the total amount of coins that have been unlocked
		n, coins, err := explorer.db.ApplyCoinOutputLocks(explorer.stats.BlockHeight, explorer.stats.Timestamp)
		if err != nil {
			panic(fmt.Sprintf("failed to unlock coin outputs at height=%d and time=%d: %v",
				explorer.stats.BlockHeight.LockValue(), explorer.stats.Timestamp.LockValue(), err))
		}
		if n > 0 {
			explorer.stats.LockedCoinOutputCount -= n
			explorer.stats.LockedCoins = explorer.stats.LockedCoins.Sub(coins)
		}

		// apply miner payouts
		for i, mp := range block.MinerPayouts {
			explorer.stats.CoinOutputCount++
			var description string
			if i == 0 {
				explorer.stats.MinerPayoutCount++
				explorer.stats.MinerPayouts = explorer.stats.MinerPayouts.Add(types.AsCurrency(mp.Value))
				description = "reward:block"
				// block rewards are always freshly created money
				explorer.stats.Coins = explorer.stats.Coins.Add(types.AsCurrency(mp.Value))
			} else if i == 1 {
				explorer.stats.TransactionFeeCount++
				explorer.stats.TransactionFees = explorer.stats.TransactionFees.Add(types.AsCurrency(mp.Value))
				description = "reward:tx"
			} else {
				explorer.stats.FoundationFeeCount++
				explorer.stats.FoundationFees = explorer.stats.FoundationFees.Add(types.AsCurrency(mp.Value))
				description = "reward:foundation"
			}
			locked, err := explorer.addCoinOutput(types.AsCoinOutputID(block.MinerPayoutID(uint64(i))), rivinetypes.CoinOutput{
				Value: mp.Value,
				Condition: rivinetypes.NewCondition(
					rivinetypes.NewTimeLockCondition(
						uint64(explorer.stats.BlockHeight.BlockHeight+explorer.chainCts.MaturityDelay),
						rivinetypes.NewUnlockHashCondition(mp.UnlockHash))),
			}, description)
			if err != nil {
				panic(fmt.Sprintf("failed to add miner payout of %s to %s: %v",
					mp.UnlockHash.String(), mp.Value.String(), err))
			}
			if locked {
				explorer.stats.LockedCoinOutputCount++
				explorer.stats.LockedCoins = explorer.stats.LockedCoins.Add(types.AsCurrency(mp.Value))
			}
		}
		// apply txs
		for _, tx := range block.Transactions {
			isCoinCreationTransaction := isGenesisBlock

			switch tx.Version {
			case threebottypes.TransactionVersionBotRegistration:
				// increase the 3bot reg tx count
				explorer.stats.ThreeBotRegistrationTransactionCount++

				// expose the 3bot record registration data from the tx
				brtx, err := threebottypes.BotRegistrationTransactionFromTransaction(tx)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret v144 tx as a BotRegistrationTransaction: %v", err))
				}
				// increment the sequential counter used for 3Bots in this explorer,
				// this is done prior to storage as to ensure we start from 1,
				// just as tfchain does
				explorer.state.NextThreeBotID.Increment()
				// assemble our new tfchain 3Bot record
				record := threebottypes.BotRecord{
					ID:        explorer.state.NextThreeBotID.TfchainBotID(),
					PublicKey: brtx.Identification.PublicKey,
					Expiration: threebottypes.SiaTimestampAsCompactTimestamp(block.Timestamp) +
						threebottypes.CompactTimestamp(brtx.NrOfMonths)*threebottypes.BotMonth,
				}
				err = record.AddNetworkAddresses(brtx.Addresses...)
				if err != nil {
					panic(fmt.Sprintf("failed to apply BotRegistrationTransaction: adding addresses resulted in an error: %v", err))
				}
				err = record.AddNames(brtx.Names...)
				if err != nil {
					panic(fmt.Sprintf("failed to apply BotRegistrationTransaction: adding names resulted in an error: %v", err))
				}
				// create the 3Bot record
				err = explorer.db.CreateBotRecord(types.BotRecordFromTfchainRecord(record))
				if err != nil {
					panic(fmt.Sprintf("failed to apply BotRegistrationTransaction: %v", err))
				}

			case threebottypes.TransactionVersionBotRecordUpdate:
				// increase the 3bot update tx count
				explorer.stats.ThreeBotUpdateTransactionCount++

				// expose the 3bot record update data from the tx
				butx, err := threebottypes.BotRecordUpdateTransactionFromTransaction(tx)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret v145 tx as a BotRecordUpdateTransaction: %v", err))
				}
				// update the 3Bot record
				err = explorer.db.UpdateBotRecord(
					types.NewBotIDFromTfchainBotID(butx.Identifier),
					wrapTfchainBotUpdateFunction(block.Timestamp, butx.UpdateBotRecord))
				if err != nil {
					panic(fmt.Sprintf("failed to apply BotRecordUpdateTransaction: %v", err))
				}

			case threebottypes.TransactionVersionBotNameTransfer:
				// increase the 3bot update tx count (also used for name transfers)
				explorer.stats.ThreeBotUpdateTransactionCount++

				// expose the 3bot record name transfer data from the tx
				bnttx, err := threebottypes.BotNameTransferTransactionFromTransaction(tx)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret v146 tx as a BotNameTransferTransaction: %v", err))
				}
				// update both 3Bot records in a revert way
				err = explorer.db.UpdateBotRecord(
					types.NewBotIDFromTfchainBotID(bnttx.Sender.Identifier),
					wrapTfchainBotUpdateFunction(block.Timestamp, bnttx.UpdateSenderBotRecord))
				if err != nil {
					panic(fmt.Sprintf("failed to apply BotRecordUpdateTransaction for sender: %v", err))
				}
				err = explorer.db.UpdateBotRecord(
					types.NewBotIDFromTfchainBotID(bnttx.Receiver.Identifier),
					wrapTfchainBotUpdateFunction(block.Timestamp, bnttx.UpdateReceiverBotRecord))
				if err != nil {
					panic(fmt.Sprintf("failed to apply BotRecordUpdateTransaction for receiver: %v", err))
				}

			case tfchaintypes.TransactionVersionERC20Conversion:
				explorer.stats.CoinBurnTransactionCount++

				// expose the ERC20 Conversion Tx data from the Tx
				erc20CTx, err := erc20types.ERC20ConvertTransactionFromTransaction(tx, tx.Version)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret tx as a ERC20 Conversion Tx: %v", err))
				}
				// remove value, as it is burned
				explorer.stats.Coins = explorer.stats.Coins.Sub(types.AsCurrency(erc20CTx.Value))

			case tfchaintypes.TransactionVersionERC20CoinCreation:
				explorer.stats.CoinCreationTransactionCount++
				isCoinCreationTransaction = true
				// add miner fees if it was a coin created tx,
				// as fees are created from new coins in a coin creation tx as well
				for _, fee := range tx.MinerFees {
					explorer.stats.Coins = explorer.stats.Coins.Add(types.AsCurrency(fee))
				}

			case tfchaintypes.TransactionVersionERC20AddressRegistration:
				// expose the ERC20 Conversion Tx data from the Tx
				erc20ARTx, err := erc20types.ERC20AddressRegistrationTransactionFromTransaction(tx, tx.Version)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret tx as a ERC20 Address Registration Tx: %v", err))
				}
				tftAddress, err := rivinetypes.NewPubKeyUnlockHash(erc20ARTx.PublicKey)
				if err != nil {
					panic(fmt.Sprintf("failed to create a rivine addressfrom a ERC20 Address Registration Tx: %v", err))
				}
				erc20Address, err := erc20types.ERC20AddressFromUnlockHash(tftAddress)
				if err != nil {
					panic(fmt.Sprintf("failed to create an erc20 addressfrom a ERC20 Address Registration Tx: %v", err))
				}
				err = explorer.db.AddERC20AddressRegistration(types.AsERC20Address(erc20Address), types.AsUnlockHash(tftAddress))
				if err != nil {
					panic(fmt.Sprintf("failed to register ERC20 Address %s: %v", erc20Address.String(), err))
				}

			case tfchaintypes.TransactionVersionCoinCreation:
				explorer.stats.CoinCreationTransactionCount++
				isCoinCreationTransaction = true
				// add miner fees if it was a coin created tx,
				// as fees are created from new coins in a coin creation tx as well
				for _, fee := range tx.MinerFees {
					explorer.stats.Coins = explorer.stats.Coins.Add(types.AsCurrency(fee))
				}

			case tfchaintypes.TransactionVersionMinterDefinition:
				// decrease coin creator tx count
				explorer.stats.CoinCreatorDefinitionTransactionCount++

				// get the mint condition from the tx's extension data
				mdtx, err := minting.MinterDefinitionTransactionFromTransaction(tx, tx.Version, true)
				if err != nil {
					panic(fmt.Sprintf("failed to interpret v128 tx as a MinterDefinitionTransaction: %v", err))
				}

				// set current coin creators
				err = explorer.setMintCondition(mdtx.MintCondition)
				if err != nil {
					panic(fmt.Sprintf("failed to set mint condition in the explorer db: %v", err))
				}

				// add miner fees if it was a minter definition tx,
				// as fees are created from new coins in a minter definition tx as well
				for _, fee := range tx.MinerFees {
					explorer.stats.Coins = explorer.stats.Coins.Add(types.AsCurrency(fee))
				}
			}

			explorer.stats.TransactionCount++

			if len(tx.CoinInputs) > 0 || len(tx.BlockStakeOutputs) > 1 {
				explorer.stats.ValueTransactionCount++
			}
			// apply coin inputs
			for _, ci := range tx.CoinInputs {
				explorer.stats.CoinInputCount++
				err = explorer.db.SpendCoinOutput(types.AsCoinOutputID(ci.ParentID))
				if err != nil {
					panic(fmt.Sprintf("failed to spend coin output %s: %v", ci.ParentID.String(), err))
				}
			}
			// apply coin outputs
			for i, co := range tx.CoinOutputs {
				explorer.stats.CoinOutputCount++
				id := tx.CoinOutputID(uint64(i))
				description := string(tx.ArbitraryData)
				locked, err := explorer.addCoinOutput(types.AsCoinOutputID(id), co, description)
				if err != nil {
					panic(fmt.Sprintf("failed to add coin output %s from %s: %v",
						id, co.Condition.UnlockHash().String(), err))
				}
				// only count coins of outputs for genesis block txs or coin creation txs
				if isCoinCreationTransaction {
					explorer.stats.Coins = explorer.stats.Coins.Add(types.AsCurrency(co.Value))
				}
				// if it is locked, we'll always add it to the locked output
				if locked {
					explorer.stats.LockedCoinOutputCount++
					explorer.stats.LockedCoins = explorer.stats.LockedCoins.Add(types.AsCurrency(co.Value))
				}
			}
		}
	}

	// update state
	explorer.state.CurrentChangeID = types.AsConsensusChangeID(css.ID)

	// store latest state and stats
	err = explorer.db.SetExplorerState(explorer.state)
	if err != nil {
		panic("failed to store explorer state in db: " + err.Error())
	}
	err = explorer.db.SetNetworkStats(explorer.stats)
	if err != nil {
		panic("failed to store network stats in db: " + err.Error())
	}
}

func wrapTfchainBotUpdateFunction(bts rivinetypes.Timestamp, fn func(rivinetypes.Timestamp, *threebottypes.BotRecord) error) func(*types.BotRecord) error {
	return func(br *types.BotRecord) error {
		tfbr := br.TfchainRecord()
		err := fn(bts, &tfbr)
		if err != nil {
			return err
		}
		*br = types.BotRecordFromTfchainRecord(tfbr)
		return nil
	}
}

func wrapTfchainBotUpdateRevertFunction(fn func(*threebottypes.BotRecord) error) func(*types.BotRecord) error {
	return func(br *types.BotRecord) error {
		tfbr := br.TfchainRecord()
		err := fn(&tfbr)
		if err != nil {
			return err
		}
		*br = types.BotRecordFromTfchainRecord(tfbr)
		return nil
	}
}

func (explorer *Explorer) setMintCondition(condition rivinetypes.UnlockConditionProxy) error {
conditionSwitch:
	switch ct := condition.ConditionType(); ct {
	case rivinetypes.ConditionTypeUnlockHash:
		return explorer.db.SetCoinCreators([]types.UnlockHash{types.AsUnlockHash(condition.UnlockHash())})

	case rivinetypes.ConditionTypeMultiSignature:
		uhsg, ok := condition.Condition.(rivinetypes.UnlockHashSliceGetter)
		if !ok {
			return fmt.Errorf("unexpected Go-type for MultiSignatureCondition: %T", condition.Condition)
		}
		ruhs := uhsg.UnlockHashSlice()
		uhs := make([]types.UnlockHash, 0, len(ruhs))
		for _, ruh := range ruhs {
			uhs = append(uhs, types.AsUnlockHash(ruh))
		}
		return explorer.db.SetCoinCreators(uhs)

	case rivinetypes.ConditionTypeTimeLock:
		// time lock conditions are allowed as long as the internal condition is allowed
		cg, ok := condition.Condition.(rivinetypes.MarshalableUnlockConditionGetter)
		if !ok {
			return fmt.Errorf("unexpected Go-type for TimeLockCondition: %T", condition.Condition)
		}
		condition = rivinetypes.NewCondition(cg.GetMarshalableUnlockCondition())
		goto conditionSwitch

	default:
		return fmt.Errorf("unexpected condition type %d as mint condition", ct)
	}
}

// addCoinOutput is an internal function used to be able to store a coin output,
// ensuring we differentiate locked and unlocked coin outputs.
// On top of that it checks for multisig outputs, as to be able to track multisig addresses,
// linking them to the owner addresses as well as storing the owner addresses themself for the multisig wallet.
func (explorer *Explorer) addCoinOutput(id types.CoinOutputID, co rivinetypes.CoinOutput, description string) (locked bool, err error) {
	// check if it is a multisignature condition, if so, track it
	if ownerAddresses, signaturesRequired := getMultisigProperties(co.Condition); len(ownerAddresses) > 0 {
		multiSigAddress := co.Condition.UnlockHash()
		err := explorer.db.SetMultisigAddresses(types.AsUnlockHash(multiSigAddress), ownerAddresses, signaturesRequired)
		if err != nil {
			return false, fmt.Errorf(
				"failed to set multisig addresses for multisig wallet %q: %v",
				multiSigAddress.String(), err)
		}
	}

	// add coin output itself
	isFulfillable := co.Condition.Fulfillable(rivinetypes.FulfillableContext{
		BlockHeight: explorer.stats.BlockHeight.BlockHeight,
		BlockTime:   explorer.stats.Timestamp.Timestamp,
	})
	if isFulfillable {
		return false, explorer.db.AddCoinOutput(id, CoinOutput{
			Value:       types.AsCurrency(co.Value),
			Condition:   co.Condition,
			Description: description,
		})
	}
	// only a TimeLockedCondition can be locked for now
	tlc := co.Condition.Condition.(*rivinetypes.TimeLockCondition)
	lt := dtypes.LockTypeTime
	if tlc.LockTime < rivinetypes.LockTimeMinTimestampValue {
		lt = dtypes.LockTypeHeight
	}
	return true, explorer.db.AddLockedCoinOutput(id, CoinOutput{
		Value:       types.AsCurrency(co.Value),
		Condition:   co.Condition,
		Description: description,
	}, lt, types.LockValue(tlc.LockTime))
}

// getMultisigOwnerAddresses gets the owner addresses (= internal addresses of a multisig condition)
// from either a MultiSignatureCondition or a MultiSignatureCondition used as the internal condition of a TimeLockCondition.
func getMultisigProperties(condition rivinetypes.UnlockConditionProxy) (owners []types.UnlockHash, signaturesRequired uint64) {
	ct := condition.ConditionType()
	if ct == rivinetypes.ConditionTypeTimeLock {
		cg, ok := condition.Condition.(rivinetypes.MarshalableUnlockConditionGetter)
		if !ok {
			panic(fmt.Sprintf("unexpected Go-type for TimeLockCondition: %T", condition))
		}
		return getMultisigProperties(rivinetypes.NewCondition(cg.GetMarshalableUnlockCondition()))
	}

	type multisigCondition interface {
		rivinetypes.UnlockHashSliceGetter
		GetMinimumSignatureCount() uint64
	}
	switch c := condition.Condition.(type) {
	case multisigCondition:
		return dedupOwnerAddresses(c.UnlockHashSlice()), c.GetMinimumSignatureCount()
	default:
		return nil, 0
	}
}
func dedupOwnerAddresses(addresses []rivinetypes.UnlockHash) (deduped []types.UnlockHash) {
	n := len(addresses)
	if n == 0 {
		return
	}
	encountered := make(map[rivinetypes.UnlockHash]struct{}, n)
	for _, addr := range addresses {
		encountered[addr] = struct{}{}
	}
	for addr := range encountered {
		deduped = append(deduped, types.AsUnlockHash(addr))
	}
	return
}
