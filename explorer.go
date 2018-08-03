package main

import (
	"fmt"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"
)

type (
	ExplorerState struct {
		CurrentChangeID modules.ConsensusChangeID
	}
	NetworkStats struct {
		Timestamp        types.Timestamp   `json:"timestamp"`
		BlockHeight      types.BlockHeight `json:"blockHeight"`
		TransactionCount uint64            `json:"txCount"`
		CointOutputCount uint64            `json:"coinOutputCount"`
		CointInputCount  uint64            `json:"coinInputCount"`
		MinerPayouts     types.Currency    `json:"minerPayouts"`
		Coins            types.Currency    `json:"coins"`
	}
)

type Explorer struct {
	db    Database
	state ExplorerState
	stats NetworkStats

	cs modules.ConsensusSet

	bcInfo   types.BlockchainInfo
	chainCts types.ChainConstants
}

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

func (explorer *Explorer) Close() error {
	explorer.cs.Unsubscribe(explorer)
	return nil
}

func (explorer *Explorer) ProcessConsensusChange(css modules.ConsensusChange) {
	var err error

	// update reverted blocks
	for _, block := range css.RevertedBlocks {
		// revert miner payouts
		for _, mp := range block.MinerPayouts {
			explorer.stats.MinerPayouts = explorer.stats.MinerPayouts.Sub(mp.Value)
			explorer.stats.Coins = explorer.stats.Coins.Sub(mp.Value)
			err = explorer.db.RemoveMinerPayout(mp.UnlockHash, mp.Value)
			if err != nil {
				panic(fmt.Sprintf("failed to remove miner payout of %s to %s: %v",
					mp.UnlockHash.String(), mp.Value.String(), err))
			}
		}
		// revert txs
		for _, tx := range block.Transactions {
			explorer.stats.TransactionCount--
			// revert coin inputs
			for range tx.CoinInputs {
				explorer.stats.CointInputCount--
			}
			// revert coin outputs
			for i := range tx.CoinOutputs {
				explorer.stats.CointOutputCount--
				id := tx.CoinOutputID(uint64(i))
				err = explorer.db.RemoveCoinOutput(id)
				if err != nil {
					panic(fmt.Sprintf("failed to remove coin output %x: %v", id, err))
				}
			}
		}

		explorer.stats.BlockHeight--
		explorer.stats.Timestamp = block.Timestamp
	}

	// update applied blocks
	for _, block := range css.AppliedBlocks {
		// apply miner payouts
		for _, mp := range block.MinerPayouts {
			explorer.stats.MinerPayouts = explorer.stats.MinerPayouts.Add(mp.Value)
			explorer.stats.Coins = explorer.stats.Coins.Add(mp.Value)
			err = explorer.db.AddMinerPayout(mp.UnlockHash, mp.Value)
			if err != nil {
				panic(fmt.Sprintf("failed to add miner payout of %s to %s: %v",
					mp.UnlockHash.String(), mp.Value.String(), err))
			}
		}
		// apply txs
		for _, tx := range block.Transactions {
			explorer.stats.TransactionCount++
			// revert coin inputs
			for _, ci := range tx.CoinInputs {
				explorer.stats.CointInputCount++
				err = explorer.db.RemoveCoinOutput(ci.ParentID)
				if err != nil {
					panic(fmt.Sprintf("failed to remove coin output %x: %v", ci.ParentID, err))
				}
			}
			// apply coin outputs
			for _, co := range tx.CoinOutputs {
				explorer.stats.CointOutputCount++
				if explorer.stats.BlockHeight == 0 {
					// only count coins of outputs for genesis block,
					// as it is currently the only place coins can be created
					explorer.stats.Coins = explorer.stats.Coins.Add(co.Value)
				}
			}
		}

		// apply applied coin output diffs
		for _, diff := range css.CoinOutputDiffs {
			if diff.Direction != modules.DiffApply {
				continue
			}
			err = explorer.db.AddCoinOutput(diff.ID, diff.CoinOutput)
			if err != nil {
				panic(fmt.Sprintf("failed to add coin output %x from %s: %v",
					diff.ID, diff.CoinOutput.Condition.UnlockHash().String(), err))
			}
		}

		explorer.stats.BlockHeight++
		explorer.stats.Timestamp = block.Timestamp
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
