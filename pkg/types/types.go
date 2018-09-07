package types

import (
	"fmt"

	rivineencoding "github.com/rivine/rivine/encoding"
	"github.com/rivine/rivine/types"
	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
)

// message pack
//go:generate msgp -marshal=false -io=true

// protobuf (using gogo/protobuf)
//   requires protoc (https://github.com/protocolbuffers/protobuf/releases/tag/v3.6.1) and
// gogofaster (go get -u github.com/gogo/protobuf/proto github.com/gogo/protobuf/gogoproto github.com/gogo/protobuf/protoc-gen-gogofaster)
//go:generate protoc -I=. -I=$GOPATH/src -I=../../vendor/github.com/gogo/protobuf/protobuf --gogofaster_out=. types.proto

// public types
type (
	// NetworkStats collects the global statistics for the blockchain.
	NetworkStats struct {
		Timestamp                             Timestamp   `json:"timestamp" msg:"timestamp"`
		BlockHeight                           BlockHeight `json:"blockHeight" msg:"blockHeight"`
		TransactionCount                      uint64      `json:"txCount" msg:"txCount"`
		CoinCreationTransactionCount          uint64      `json:"coinCreationTxCount" msg:"coinCreationTxCount"`
		CoinCreatorDefinitionTransactionCount uint64      `json:"coinCreatorDefinitionTxCount" msg:"coinCreatorDefinitionTxCount"`
		ValueTransactionCount                 uint64      `json:"valueTxCount" msg:"valueTxCount"`
		CoinOutputCount                       uint64      `json:"coinOutputCount" msg:"coinOutputCount"`
		LockedCoinOutputCount                 uint64      `json:"lockedCoinOutputCount" msg:"lockedCoinOutputCount"`
		CoinInputCount                        uint64      `json:"coinInputCount" msg:"coinInputCount"`
		MinerPayoutCount                      uint64      `json:"minerPayoutCount" msg:"minerPayoutCount"`
		TransactionFeeCount                   uint64      `json:"txFeeCount" msg:"txFeeCount"`
		MinerPayouts                          Currency    `json:"minerPayouts" msg:"minerPayouts"`
		TransactionFees                       Currency    `json:"txFees" msg:"txFees"`
		Coins                                 Currency    `json:"coins" msg:"coins"`
		LockedCoins                           Currency    `json:"lockedCoins" msg:"lockedCoins"`
	}
)

// NewNetworkStats creates a nil (fresh) network state.
func NewNetworkStats() NetworkStats {
	return NetworkStats{}
}

// public wallet types
type (
	// Wallet collects all data for an address in a simple format,
	// focussing on its balance and multisign properties.
	Wallet struct {
		// Balance is optional and defines the balance the wallet currently has.
		Balance WalletBalance `json:"balance,omitempty" msg:"balance,omitempty"`
		// MultiSignAddresses is optional and is only defined if the wallet is part of
		// one or multiple multisign wallets.
		MultiSignAddresses []UnlockHash `json:"multisignAddresses,omitempty" msg:"multisignAddresses,omitempty"`
		// MultiSignData is optional and is only defined if the wallet is a multisign wallet.
		MultiSignData WalletMultiSignData `json:"multisign,omitempty" msg:"multisign,omitempty"`
	}
	// WalletBalance contains the unlocked and/or locked balance of a wallet.
	WalletBalance struct {
		Unlocked WalletUnlockedBalance `json:"unlocked,omitemtpy" msg:"unlocked,omitemtpy"`
		Locked   WalletLockedBalance   `json:"locked,omitemtpy" msg:"locked,omitemtpy"`
	}
	// WalletUnlockedBalance contains the unlocked balance of a wallet,
	// defining the total amount of coins as well as all the outputs that are unlocked.
	WalletUnlockedBalance struct {
		Total   Currency                `json:"total" msg:"total"`
		Outputs WalletUnlockedOutputMap `json:"outputs" msg:"outputs"`
	}
	// WalletUnlockedOutputMap defines the mapping between a coin output ID and its walletUnlockedOutput data
	WalletUnlockedOutputMap map[string]WalletUnlockedOutput
	// WalletUnlockedOutput defines an unlocked output targeted at a wallet.
	WalletUnlockedOutput struct {
		Amount      Currency `json:"amount" msg:"amount"`
		Description string   `json:"description,omitemtpy" msg:"description,omitemtpy"`
	}
	// WalletLockedBalance contains the locked balance of a wallet,
	// defining the total amount of coins as well as all the outputs that are locked.
	WalletLockedBalance struct {
		Total   Currency              `json:"total" msg:"total"`
		Outputs WalletLockedOutputMap `json:"outputs" msg:"outputs"`
	}
	// WalletLockedOutputMap defines the mapping between a coin output ID and its walletLockedOutput data
	WalletLockedOutputMap map[string]WalletLockedOutput
	// WalletLockedOutput defines a locked output targeted at a wallet.
	WalletLockedOutput struct {
		Amount      Currency  `json:"amount" msg:"amount"`
		LockedUntil LockValue `json:"lockedUntil" msg:"lockedUntil"`
		Description string    `json:"description,omitemtpy" msg:"description,omitemtpy"`
	}
	// WalletMultiSignData defines the extra data defined for a MultiSignWallet.
	WalletMultiSignData struct {
		Owners             []UnlockHash `json:"owners" msg:"owners"`
		SignaturesRequired uint64       `json:"signaturesRequired" msg:"signaturesRequired"`
	}
)

var (
	_ encoding.ProtocolBufferMarshaler   = (*NetworkStats)(nil)
	_ encoding.ProtocolBufferUnmarshaler = (*NetworkStats)(nil)

	_ encoding.ProtocolBufferMarshaler   = (*Wallet)(nil)
	_ encoding.ProtocolBufferUnmarshaler = (*Wallet)(nil)
)

// ProtocolBufferMarshal implements encoding.ProtocolBufferMarshaler.ProtocolBufferMarshal
// using the generated code based on the PBNetworkStats Message defined in ./types.proto
func (stats *NetworkStats) ProtocolBufferMarshal(w encoding.ProtocolBufferWriter) error {
	err := w.Marshal(&PBNetworkStats{
		Timestamp:             uint64(stats.Timestamp.Timestamp),
		Blockheight:           uint64(stats.BlockHeight.BlockHeight),
		TxCount:               stats.TransactionCount,
		CoinCreationTxCount:   stats.CoinCreationTransactionCount,
		CoinCreatorDefTxCount: stats.CoinCreatorDefinitionTransactionCount,
		ValueTxCount:          stats.ValueTransactionCount,
		CoinOutputCount:       stats.CoinOutputCount,
		LockedCoinOutputCount: stats.LockedCoinOutputCount,
		CoinInputCount:        stats.CoinInputCount,
		MinerPayoutCount:      stats.MinerPayoutCount,
		TxFeeCount:            stats.TransactionFeeCount,
		MinerPayouts:          rivineencoding.Marshal(stats.MinerPayouts),
		TxFees:                rivineencoding.Marshal(stats.TransactionFees),
		Coins:                 rivineencoding.Marshal(stats.Coins),
		LockedCoins:           rivineencoding.Marshal(stats.LockedCoins),
	})
	if err != nil {
		return fmt.Errorf("NetworkStats: %v", err)
	}
	return nil
}

// ProtocolBufferUnmarshal implements encoding.ProtocolBufferUnmarshaler.ProtocolBufferUnmarshal
// using the generated code based on the PBNetworkStats Message defined in ./types.proto
func (stats *NetworkStats) ProtocolBufferUnmarshal(r encoding.ProtocolBufferReader) error {
	// unmarshal entire protocol buffer message as a whole
	var pb PBNetworkStats
	err := r.Unmarshal(&pb)
	if err != nil {
		return fmt.Errorf("NetworkStats: %v", err)
	}

	// assign all required uint64 values
	stats.Timestamp = AsTimestamp(types.Timestamp(pb.Timestamp))
	stats.BlockHeight = AsBlockHeight(types.BlockHeight(pb.Blockheight))
	stats.TransactionCount = pb.TxCount
	stats.CoinCreationTransactionCount = pb.CoinCreationTxCount
	stats.CoinCreatorDefinitionTransactionCount = pb.CoinCreatorDefTxCount
	stats.ValueTransactionCount = pb.ValueTxCount
	stats.CoinOutputCount = pb.CoinOutputCount
	stats.LockedCoinOutputCount = pb.LockedCoinOutputCount
	stats.CoinInputCount = pb.CoinInputCount
	stats.MinerPayoutCount = pb.MinerPayoutCount
	stats.TransactionFeeCount = pb.TxFeeCount

	// unmarshal all required Currency values
	err = rivineencoding.Unmarshal(pb.MinerPayouts, &stats.MinerPayouts)
	if err != nil {
		return fmt.Errorf("NetworkStats: MinerPayouts: %v", err)
	}
	err = rivineencoding.Unmarshal(pb.TxFees, &stats.TransactionFees)
	if err != nil {
		return fmt.Errorf("NetworkStats: TransactionFees: %v", err)
	}
	err = rivineencoding.Unmarshal(pb.LockedCoins, &stats.LockedCoins)
	if err != nil {
		return fmt.Errorf("NetworkStats: LockedCoins: %v", err)
	}
	err = rivineencoding.Unmarshal(pb.Coins, &stats.Coins)
	if err != nil {
		return fmt.Errorf("NetworkStats: Coins: %v", err)
	}

	// all was unmarshaled, return nil (= no error)
	return nil
}

// ProtocolBufferMarshal implements encoding.ProtocolBufferMarshaler.ProtocolBufferMarshal
// using the generated code based on the PBWallet Message defined in ./types.proto
func (wallet *Wallet) ProtocolBufferMarshal(w encoding.ProtocolBufferWriter) error {
	pb := new(PBWallet)
	// add optional UnlockedBalance only if available
	if !wallet.Balance.Unlocked.Total.IsZero() {
		ub := &PBWalletUnlockedBalance{
			Total:   rivineencoding.Marshal(wallet.Balance.Unlocked.Total),
			Outputs: make(map[string]*PBWalletUnlockedOutput, len(wallet.Balance.Unlocked.Outputs)),
		}
		pb.BalanceUnlocked = ub
		for id, output := range wallet.Balance.Unlocked.Outputs {
			ub.Outputs[id] = &PBWalletUnlockedOutput{
				Amount:      rivineencoding.Marshal(output.Amount),
				Description: output.Description,
			}
		}
	}
	// add optional LockedBalance only if available
	if !wallet.Balance.Locked.Total.IsZero() {
		lb := &PBWalletLockedBalance{
			Total:   rivineencoding.Marshal(wallet.Balance.Locked.Total),
			Outputs: make(map[string]*PBWalletLockedOutput, len(wallet.Balance.Locked.Outputs)),
		}
		pb.BalanceLocked = lb
		for id, output := range wallet.Balance.Locked.Outputs {
			lb.Outputs[id] = &PBWalletLockedOutput{
				Amount:      rivineencoding.Marshal(output.Amount),
				LockedUntil: uint64(output.LockedUntil),
				Description: output.Description,
			}
		}
	}
	// add optional MultiSignAddresses only if available
	if n := len(wallet.MultiSignAddresses); n > 0 {
		pb.MultisignAddresses = make([][]byte, n)
		for idx, uh := range wallet.MultiSignAddresses {
			pb.MultisignAddresses[idx] = rivineencoding.Marshal(uh)
		}
	}
	// add optional MultiSignData only if available
	if wallet.MultiSignData.SignaturesRequired > 0 {
		pb.MultisignData = &PBWalletMultiSignData{
			SignaturesRequired: wallet.MultiSignData.SignaturesRequired,
			Owners:             make([][]byte, len(wallet.MultiSignData.Owners)),
		}
		for idx, uh := range wallet.MultiSignData.Owners {
			pb.MultisignData.Owners[idx] = rivineencoding.Marshal(uh)
		}
	}
	// Marshal the entire wallet into the given ProtocolBufferWriter
	err := w.Marshal(pb)
	if err != nil {
		return fmt.Errorf("Wallet: %v", err)
	}
	return nil
}

// ProtocolBufferUnmarshal implements encoding.ProtocolBufferUnmarshaler.ProtocolBufferUnmarshal
// using the generated code based on the PBWallet Message defined in ./types.proto
func (wallet *Wallet) ProtocolBufferUnmarshal(r encoding.ProtocolBufferReader) error {
	// unmarshal entire protocol buffer message as a whole
	var pb PBWallet
	err := r.Unmarshal(&pb)
	if err != nil {
		return fmt.Errorf("Wallet: %v", err)
	}
	// only unmarshal unlocked balance if it is available, otherwise reset it
	if pb.BalanceUnlocked == nil {
		wallet.Balance.Unlocked = WalletUnlockedBalance{}
	} else {
		err = rivineencoding.Unmarshal(pb.BalanceUnlocked.Total, &wallet.Balance.Unlocked.Total)
		if err != nil {
			return fmt.Errorf("Wallet: Unlocked Balance: Total: %v", err)
		}
		// unmarshal all outputs
		wallet.Balance.Unlocked.Outputs = make(WalletUnlockedOutputMap, len(pb.BalanceUnlocked.Outputs))
		for id, output := range pb.BalanceUnlocked.Outputs {
			var unlockedOutput WalletUnlockedOutput
			// unmarshal amount
			err = rivineencoding.Unmarshal(output.Amount, &unlockedOutput.Amount)
			if err != nil {
				return fmt.Errorf("Wallet: Unlocked Balance: Output %s: Amount: %v", id, err)
			}
			// assign optional description
			unlockedOutput.Description = output.Description
			// assign locked output using its id
			wallet.Balance.Unlocked.Outputs[id] = unlockedOutput
		}
	}
	// only unmarshal locked balance if it is available, otherwise reset it
	if pb.BalanceLocked == nil {
		wallet.Balance.Locked = WalletLockedBalance{}
	} else {
		err = rivineencoding.Unmarshal(pb.BalanceLocked.Total, &wallet.Balance.Locked.Total)
		if err != nil {
			return fmt.Errorf("Wallet: Locked Balance: Total: %v", err)
		}
		// unmarshal all outputs
		wallet.Balance.Locked.Outputs = make(WalletLockedOutputMap, len(pb.BalanceLocked.Outputs))
		for id, output := range pb.BalanceLocked.Outputs {
			var lockedOutput WalletLockedOutput
			// unmarshal amount
			err = rivineencoding.Unmarshal(output.Amount, &lockedOutput.Amount)
			if err != nil {
				return fmt.Errorf("Wallet: Locked Balance: Output %s: Amount: %v", id, err)
			}
			// assign dereferenced lock value
			lockedOutput.LockedUntil = LockValue(output.LockedUntil)
			// assign optional description
			lockedOutput.Description = output.Description
			// assign locked output using its id
			wallet.Balance.Locked.Outputs[id] = lockedOutput
		}
	}
	// unmarshal optional multisign addresses if available
	if n := len(pb.MultisignAddresses); n > 0 {
		wallet.MultiSignAddresses = make([]UnlockHash, n)
		for i, b := range pb.MultisignAddresses {
			err = rivineencoding.Unmarshal(b, &wallet.MultiSignAddresses[i])
			if err != nil {
				return fmt.Errorf("Wallet: MultiSignAddresses: address #%d: %v", i, err)
			}
		}
	} else { // reset it otherwise
		wallet.MultiSignAddresses = nil
	}
	// unmarshal optional multisign data if available, reset otherwise
	if pb.MultisignData == nil {
		wallet.MultiSignData = WalletMultiSignData{}
	} else {
		// assign required signatures required
		wallet.MultiSignData.SignaturesRequired = pb.MultisignData.SignaturesRequired
		// assign all owners
		wallet.MultiSignData.Owners = make([]UnlockHash, len(pb.MultisignData.Owners))
		for i, b := range pb.MultisignData.Owners {
			err = rivineencoding.Unmarshal(b, &wallet.MultiSignData.Owners[i])
			if err != nil {
				return fmt.Errorf("Wallet: MultiSignData: owner #%d: %v", i, err)
			}
		}
	}
	return nil
}

// IsZero returns true if this wallet is Zero
func (wb *WalletBalance) IsZero() bool {
	return wb.Unlocked.Total.IsZero() && wb.Locked.Total.IsZero()
}

// AddUnlockedCoinOutput adds the unique unlocked coin output to the wallet's map of unlocked outputs
// as well as adds the coin output's value to the total amount of unlocked coins registered for this wallet.
func (wub *WalletUnlockedBalance) AddUnlockedCoinOutput(id CoinOutputID, co WalletUnlockedOutput) error {
	idStr := id.String()
	if len(wub.Outputs) == 0 {
		wub.Outputs = make(WalletUnlockedOutputMap)
	} else if _, exists := wub.Outputs[idStr]; exists {
		return fmt.Errorf("trying to add existing unlocked coin output %s", id.String())
	}
	wub.Outputs[idStr] = co
	wub.Total = wub.Total.Add(co.Amount)
	return nil
}

// SubUnlockedCoinOutput tries to remove the unlocked coin output from the wallet's map of unlocked outputs,
// try as it might not exist to never having been added. This method does always
// subtract the coin output's value from the total amount of unlocked coins registered for this wallet.
func (wub *WalletUnlockedBalance) SubUnlockedCoinOutput(id CoinOutputID, amount Currency) error {
	if len(wub.Outputs) != 0 {
		delete(wub.Outputs, id.String())
	}
	wub.Total = wub.Total.Sub(amount)
	return nil
}

// AddLockedCoinOutput adds the unique locked coin output to the wallet's map of locked outputs
// as well as adds the coin output's value to the total amount of locked coins registered for this wallet.
func (wlb *WalletLockedBalance) AddLockedCoinOutput(id CoinOutputID, co WalletLockedOutput) error {
	idStr := id.String()
	if len(wlb.Outputs) == 0 {
		wlb.Outputs = make(WalletLockedOutputMap)
	} else if _, exists := wlb.Outputs[idStr]; exists {
		return fmt.Errorf("trying to add existing locked coin output %s", id.String())
	}
	wlb.Outputs[idStr] = co
	wlb.Total = wlb.Total.Add(co.Amount)
	return nil
}

// SubLockedCoinOutput removes the unique existing locked coin output from the wallet's map of locked outputs,
// as well as subtract the coin output's value from the total amount of locked coins registered for this wallet.
func (wlb *WalletLockedBalance) SubLockedCoinOutput(id CoinOutputID) error {
	if len(wlb.Outputs) == 0 {
		return fmt.Errorf("trying to remove non-existing locked coin output %s", id.String())
	}
	idStr := id.String()
	co, exists := wlb.Outputs[idStr]
	if !exists {
		return fmt.Errorf("trying to remove non-existing locked coin output %s", id.String())
	}
	delete(wlb.Outputs, idStr)
	wlb.Total = wlb.Total.Sub(co.Amount)
	return nil
}

// AddUniqueMultisignAddress adds the given multisign address to the wallet's list of
// multisign addresses which reference this wallet's address.
// It only adds it however if the given multisign address is not known yet.
func (wallet *Wallet) AddUniqueMultisignAddress(address UnlockHash) bool {
	for _, uh := range wallet.MultiSignAddresses {
		if uh.Cmp(address.UnlockHash) == 0 {
			return false // nothing to do
		}
	}
	wallet.MultiSignAddresses = append(wallet.MultiSignAddresses, address)
	return true
}
