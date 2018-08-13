package types

import (
	"fmt"
)

//go:generate msgp -marshal=false -io=true

// public types
type (
	// NetworkStats collects the global statistics for the blockchain.
	NetworkStats struct {
		Timestamp              Timestamp   `json:"timestamp" msg:"timestamp"`
		BlockHeight            BlockHeight `json:"blockHeight" msg:"blockHeight"`
		TransactionCount       uint64      `json:"txCount" msg:"txCount"`
		ValueTransactionCount  uint64      `json:"valueTxCount" msg:"valueTxCount"`
		CointOutputCount       uint64      `json:"coinOutputCount" msg:"coinOutputCount"`
		LockedCointOutputCount uint64      `json:"lockedCoinOutputCount" msg:"lockedCoinOutputCount"`
		CointInputCount        uint64      `json:"coinInputCount" msg:"coinInputCount"`
		MinerPayoutCount       uint64      `json:"minerPayoutCount" msg:"minerPayoutCount"`
		TransactionFeeCount    uint64      `json:"txFeeCount" msg:"txFeeCount"`
		MinerPayouts           Currency    `json:"minerPayouts" msg:"minerPayouts"`
		TransactionFees        Currency    `json:"txFees" msg:"txFees"`
		Coins                  Currency    `json:"coins" msg:"coins"`
		LockedCoins            Currency    `json:"lockedCoins" msg:"lockedCoins"`
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
		Unlocked Currency            `json:"unlocked,omitemtpy" msg:"unlocked,omitemtpy"`
		Locked   WalletLockedBalance `json:"locked,omitemtpy" msg:"locked,omitemtpy"`
	}
	// WalletLockedBalance contains the locked balance of a wallet,
	// defining the total amount of coins as well as all the outputs that are locked.
	WalletLockedBalance struct {
		Total   Currency              `json:"total" msg:"total"`
		Outputs WalletLockedOutputMap `json:"outputs" msg:"outputs"`
	}
	// WalletLockedOutputMap defines the mapping between a coin output ID and its walletLockedOutput data
	WalletLockedOutputMap map[string]WalletLockedOutput
	// WalletLockedOutput defines a locked output targetted at a wallet.
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

// IsZero returns true if this wallet is Zero
func (wb *WalletBalance) IsZero() bool {
	return wb.Unlocked.IsZero() && wb.Locked.Total.IsZero()
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
func (w *Wallet) AddUniqueMultisignAddress(address UnlockHash) bool {
	for _, uh := range w.MultiSignAddresses {
		if uh.Cmp(address.UnlockHash) == 0 {
			return false // nothing to do
		}
	}
	w.MultiSignAddresses = append(w.MultiSignAddresses, address)
	return true
}
