package main

import (
	"fmt"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"
)

type (
	// ExplorerState collects the (internal) state for the explorer.
	ExplorerState struct {
		CurrentChangeID modules.ConsensusChangeID `json:"currentchangeid"`
	}
	// NetworkStats collects the global statistics for the blockchain.
	NetworkStats struct {
		Timestamp              types.Timestamp   `json:"timestamp"`
		BlockHeight            types.BlockHeight `json:"blockHeight"`
		TransactionCount       uint64            `json:"txCount"`
		ValueTransactionCount  uint64            `json:"valueTxCount"`
		CointOutputCount       uint64            `json:"coinOutputCount"`
		LockedCointOutputCount uint64            `json:"lockedCoinOutputCount"`
		CointInputCount        uint64            `json:"coinInputCount"`
		MinerPayoutCount       uint64            `json:"minerPayoutCount"`
		MinerPayouts           types.Currency    `json:"minerPayouts"`
		Coins                  types.Currency    `json:"coins"`
		LockedCoins            types.Currency    `json:"lockedCoins"`
	}
)

// NewExplorerState creates a nil (fresh) explorer state.
func NewExplorerState() ExplorerState {
	return ExplorerState{
		CurrentChangeID: modules.ConsensusChangeBeginning,
	}
}

// NewNetworkStats creates a nil (fresh) network state.
func NewNetworkStats() NetworkStats {
	return NetworkStats{}
}

// Explorer defines the custom (internal) explorer module,
// used to dump the data of a tfchain network in a meaningful way.
type Explorer struct {
	db    Database
	state ExplorerState
	stats NetworkStats

	cs modules.ConsensusSet

	bcInfo   types.BlockchainInfo
	chainCts types.ChainConstants
}

// NewExplorer creates a new custom intenral explorer module.
// See Explorer for more information.
func NewExplorer(db Database, cs modules.ConsensusSet, bcInfo types.BlockchainInfo, chainCts types.ChainConstants) (*Explorer, error) {
	state, err := db.GetExplorerState()
	if err != nil {
		return nil, fmt.Errorf("failed to get explorer state from db: %v", err)
	}
	explorer := &Explorer{
		db:       db,
		state:    state,
		cs:       cs,
		bcInfo:   bcInfo,
		chainCts: chainCts,
	}
	err = cs.ConsensusSetSubscribe(explorer, state.CurrentChangeID)
	if err != nil {
		return nil, fmt.Errorf("explorer: failed to subscribe to consensus set: %v", err)
	}
	return explorer, nil
}

// Close the Explorer module.
func (explorer *Explorer) Close() error {
	explorer.cs.Unsubscribe(explorer)
	return nil
}

// ProcessConsensusChange implements modules.ConsensusSetSubscriber,
// used to apply/revert blocks to/from our Redis-stored data.
func (explorer *Explorer) ProcessConsensusChange(css modules.ConsensusChange) {
	var err error

	// update reverted blocks
	for _, block := range css.RevertedBlocks {
		// revert miner payouts
		for i, mp := range block.MinerPayouts {
			explorer.stats.MinerPayoutCount--
			explorer.stats.CointOutputCount--
			explorer.stats.MinerPayouts = explorer.stats.MinerPayouts.Sub(mp.Value)
			explorer.stats.Coins = explorer.stats.Coins.Sub(mp.Value)
			state, err := explorer.db.RevertCoinOutput(block.MinerPayoutID(uint64(i)))
			if err != nil {
				panic(fmt.Sprintf("failed to revert miner payout of %s to %s: %v",
					mp.UnlockHash.String(), mp.Value.String(), err))
			}
			if state == CoinOutputStateLocked {
				explorer.stats.LockedCointOutputCount--
				explorer.stats.LockedCoins = explorer.stats.LockedCoins.Sub(mp.Value)
			}
		}
		// revert txs
		for _, tx := range block.Transactions {
			explorer.stats.TransactionCount--
			if len(tx.CoinInputs) > 0 || len(tx.BlockStakeOutputs) > 1 {
				explorer.stats.ValueTransactionCount--
			}
			// revert coin inputs
			for _, ci := range tx.CoinInputs {
				explorer.stats.CointInputCount--
				err := explorer.db.RevertCoinInput(ci.ParentID)
				if err != nil {
					panic(fmt.Sprintf("failed to revert coin input %s: %v", ci.ParentID.String(), err))
				}
			}
			// revert coin outputs
			for i, co := range tx.CoinOutputs {
				explorer.stats.CointOutputCount--
				id := tx.CoinOutputID(uint64(i))
				explorer.stats.Coins = explorer.stats.Coins.Sub(co.Value)
				state, err := explorer.db.RevertCoinOutput(id)
				if err != nil {
					panic(fmt.Sprintf("failed to revert coin output %s: %v", id.String(), err))
				}
				if state == CoinOutputStateLocked {
					explorer.stats.LockedCointOutputCount--
					explorer.stats.LockedCoins = explorer.stats.LockedCoins.Sub(co.Value)
				}
			}
		}

		if block.ParentID != (types.BlockID{}) {
			explorer.stats.BlockHeight--
		}
		explorer.stats.Timestamp = block.Timestamp
	}

	// update applied blocks
	for _, block := range css.AppliedBlocks {
		if block.ParentID != (types.BlockID{}) {
			explorer.stats.BlockHeight++
		}
		explorer.stats.Timestamp = block.Timestamp
		// returns the total amount of coins that have been unlocked
		n, coins, err := explorer.db.UpdateLockedCoinOutputs(explorer.stats.BlockHeight, explorer.stats.Timestamp)
		if err != nil {
			panic(fmt.Sprintf("failed to update locked coin outputs at height=%d and time=%d: %v",
				explorer.stats.BlockHeight, explorer.stats.Timestamp, err))
		}
		explorer.stats.LockedCointOutputCount -= n
		explorer.stats.LockedCoins = explorer.stats.LockedCoins.Sub(coins)

		// apply miner payouts
		for i, mp := range block.MinerPayouts {
			explorer.stats.MinerPayoutCount++
			explorer.stats.CointOutputCount++
			explorer.stats.MinerPayouts = explorer.stats.MinerPayouts.Add(mp.Value)
			explorer.stats.Coins = explorer.stats.Coins.Add(mp.Value)
			locked, err := explorer.addCoinOutput(types.CoinOutputID(block.MinerPayoutID(uint64(i))), types.CoinOutput{
				Value: mp.Value,
				Condition: types.NewCondition(
					types.NewTimeLockCondition(
						uint64(explorer.stats.BlockHeight+explorer.chainCts.MaturityDelay),
						types.NewUnlockHashCondition(mp.UnlockHash))),
			})
			if err != nil {
				panic(fmt.Sprintf("failed to add miner payout of %s to %s: %v",
					mp.UnlockHash.String(), mp.Value.String(), err))
			}
			if locked {
				explorer.stats.LockedCointOutputCount++
				explorer.stats.LockedCoins = explorer.stats.LockedCoins.Add(mp.Value)
			}
		}
		// apply txs
		for _, tx := range block.Transactions {
			explorer.stats.TransactionCount++
			if len(tx.CoinInputs) > 0 || len(tx.BlockStakeOutputs) > 1 {
				explorer.stats.ValueTransactionCount++
			}
			// apply coin inputs
			for _, ci := range tx.CoinInputs {
				explorer.stats.CointInputCount++
				err = explorer.db.SpendCoinOutput(ci.ParentID)
				if err != nil {
					panic(fmt.Sprintf("failed to spend coin output %s: %v", ci.ParentID.String(), err))
				}
			}
			// apply coin outputs
			for i, co := range tx.CoinOutputs {
				explorer.stats.CointOutputCount++
				id := tx.CoinOutputID(uint64(i))
				locked, err := explorer.addCoinOutput(id, co)
				if err != nil {
					panic(fmt.Sprintf("failed to add coin output %s from %s: %v",
						id, co.Condition.UnlockHash().String(), err))
				}
				// only count coins of outputs for genesis block,
				// as it is currently the only place coins can be created
				if explorer.stats.BlockHeight == 0 {
					explorer.stats.Coins = explorer.stats.Coins.Add(co.Value)
				}
				// if it is locked, we'll always add it to the locked output
				if locked {
					explorer.stats.LockedCointOutputCount++
					explorer.stats.LockedCoins = explorer.stats.LockedCoins.Add(co.Value)
				}
			}
		}
	}

	// update state
	explorer.state.CurrentChangeID = css.ID

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

// addCoinOutput is an internal function used to be able to store a coin output,
// ensuring we differentiate locked and unlocked coin outputs.
// On top of that it checks for multisig outputs, as to be able to track multisig addresses,
// linking them to the owner addresses as well as storing the owner addresses themself for the multisig wallet.
func (explorer *Explorer) addCoinOutput(id types.CoinOutputID, co types.CoinOutput) (locked bool, err error) {
	// check if it is a multisignature condition, if so, track it
	if ownerAddresses := getMultisigOwnerAddresses(co.Condition); len(ownerAddresses) > 0 {
		multiSigAddress := co.Condition.UnlockHash()
		err := explorer.db.SetMultisigAddresses(multiSigAddress, ownerAddresses)
		if err != nil {
			return false, fmt.Errorf(
				"failed to set multisig addresses for multisig wallet %q: %v",
				multiSigAddress.String(), err)
		}
	}

	// add coin output itself
	isFulfillable := co.Condition.Fulfillable(types.FulfillableContext{
		BlockHeight: explorer.stats.BlockHeight,
		BlockTime:   explorer.stats.Timestamp,
	})
	if isFulfillable {
		return false, explorer.db.AddCoinOutput(id, co)
	}
	// only a TimeLockedCondition can be locked for now
	tlc := co.Condition.Condition.(*types.TimeLockCondition)
	lt := LockTypeTime
	if tlc.LockTime < types.LockTimeMinTimestampValue {
		lt = LockTypeHeight
	}
	return true, explorer.db.AddLockedCoinOutput(id, co, lt, tlc.LockTime)
}

// getMultisigOwnerAddresses gets the owner addresses (= internal addresses of a multisig condition)
// from either a MultiSignatureCondition or a MultiSignatureCondition used as the internal condition of a TimeLockCondition.
func getMultisigOwnerAddresses(condition types.UnlockConditionProxy) []types.UnlockHash {
	ct := condition.ConditionType()
	if ct == types.ConditionTypeTimeLock {
		cg, ok := condition.Condition.(types.MarshalableUnlockConditionGetter)
		if !ok {
			panic(fmt.Sprintf("unexpected Go-type for TimeLockCondition: %T", condition))
		}
		return getMultisigOwnerAddresses(types.NewCondition(cg.GetMarshalableUnlockCondition()))
	}
	switch c := condition.Condition.(type) {
	case types.UnlockHashSliceGetter:
		return dedupOwnerAddresses(c.UnlockHashSlice())
	default:
		return nil
	}
}
func dedupOwnerAddresses(addresses []types.UnlockHash) (deduped []types.UnlockHash) {
	n := len(addresses)
	if n == 0 {
		return
	}
	encountered := make(map[types.UnlockHash]struct{}, n)
	for _, addr := range addresses {
		encountered[addr] = struct{}{}
	}
	for addr := range encountered {
		deduped = append(deduped, addr)
	}
	return
}
