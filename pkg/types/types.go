package types

import (
	"encoding/json"
	"fmt"

	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	tftypes "github.com/threefoldfoundation/tfchain/pkg/types"
	"github.com/threefoldtech/rivine/pkg/encoding/rivbin"
	"github.com/threefoldtech/rivine/pkg/encoding/siabin"
	"github.com/threefoldtech/rivine/types"
	"github.com/tinylib/msgp/msgp"
)

// message pack (using github.com/tinylib/msgp)
//go:generate msgp -marshal=false -io=true

// protobuf (using gogo/protobuf)
//   requires protoc (https://github.com/protocolbuffers/protobuf/releases/tag/v3.6.1) and
// gogofaster (go get -u github.com/gogo/protobuf/proto github.com/gogo/protobuf/gogoproto github.com/gogo/protobuf/protoc-gen-gogofaster)
//go:generate protoc -I=. -I=$GOPATH/src -I=../../vendor/github.com/gogo/protobuf/protobuf --gogofaster_out=. types.proto

// public types
type (
	// NetworkStats collects the global statistics for the blockchain.
	NetworkStats struct {
		Timestamp                             Timestamp   `json:"timestamp" msg:"cts"`
		BlockHeight                           BlockHeight `json:"blockHeight" msg:"cbh"`
		TransactionCount                      uint64      `json:"txCount" msg:"txc"`
		CoinCreationTransactionCount          uint64      `json:"coinCreationTxCount" msg:"cctxc"`
		CoinCreatorDefinitionTransactionCount uint64      `json:"coinCreatorDefinitionTxCount" msg:"ccdtxc"`
		ThreeBotRegistrationTransactionCount  uint64      `json:"threeBotRegistrationTransactionCount" msg:"tbrtxc"`
		ThreeBotUpdateTransactionCount        uint64      `json:"threeBotUpdateTransactionCount" msg:"tbutxc"`
		ValueTransactionCount                 uint64      `json:"valueTxCount" msg:"vtxc"`
		CoinOutputCount                       uint64      `json:"coinOutputCount" msg:"coc"`
		LockedCoinOutputCount                 uint64      `json:"lockedCoinOutputCount" msg:"lcoc"`
		CoinInputCount                        uint64      `json:"coinInputCount" msg:"cic"`
		MinerPayoutCount                      uint64      `json:"minerPayoutCount" msg:"mpc"`
		TransactionFeeCount                   uint64      `json:"txFeeCount" msg:"txfc"`
		MinerPayouts                          Currency    `json:"minerPayouts" msg:"mpt"`
		TransactionFees                       Currency    `json:"txFees" msg:"txft"`
		Coins                                 Currency    `json:"coins" msg:"ct"`
		LockedCoins                           Currency    `json:"lockedCoins" msg:"lct"`
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
	//
	// Wallet supports MessagePack and JSON encoding,
	// but does so via a custom structure as to ensure
	// it does not encode null values. See `EncodableWallet` to
	// see how this structure looks like.
	//
	//msgp:ignore Wallet
	Wallet struct {
		// Balance is optional and defines the balance the wallet currently has.
		Balance WalletBalance
		// MultiSignAddresses is optional and is only defined if the wallet is part of
		// one or multiple multisign wallets.
		MultiSignAddresses []UnlockHash
		// MultiSignData is optional and is only defined if the wallet is a multisign wallet.
		MultiSignData WalletMultiSignData
	}
	// EncodableWallet is the structure used to encode (and decode) a Wallet.
	// Golang encoding packages cannot omit empty complex types (maps and structures)
	// without them being pointers, hence this structure is used.
	// Pointers on properties for runtime structures makes things however tricky,
	// so the user will work with the actual Wallet instead, not via this structure.
	EncodableWallet struct {
		// Balance is optional and defines the balance the wallet currently has.
		Balance *EncodableWalletBalance `json:"balance,omitempty" msg:"b,omitempty"`
		// MultiSignAddresses is optional and is only defined if the wallet is part of
		// one or multiple multisign wallets.
		MultiSignAddresses []UnlockHash `json:"multisignAddresses,omitempty" msg:"ma,omitempty"`
		// MultiSignData is optional and is only defined if the wallet is a multisign wallet.
		MultiSignData *WalletMultiSignData `json:"multisign,omitempty" msg:"m,omitempty"`
	}

	// WalletBalance contains the unlocked and/or locked balance of a wallet.
	//
	// WalletBalance supports MessagePack and JSON encoding,
	// but does so via a custom structure as to ensure
	// it does not encode null values. See `EncodableWalletBalance` to
	// see how this structure looks like.
	//
	//msgp:ignore WalletBalance
	WalletBalance struct {
		Unlocked WalletUnlockedBalance
		Locked   WalletLockedBalance
	}
	// EncodableWalletBalance is the structure used to encode (and decode) a WalletBalance.
	// Golang encoding packages cannot omit empty complex types (structures)
	// without them being pointers, hence this structure is used.
	// Pointers on properties for runtime structures makes things however tricky,
	// so the user will work with the actual WalletBalance instead, not via this structure.
	EncodableWalletBalance struct {
		Unlocked *WalletUnlockedBalance `json:"unlocked,omitempty" msg:"u,omitempty"`
		Locked   *WalletLockedBalance   `json:"locked,omitempty" msg:"l,omitempty"`
	}

	// WalletUnlockedBalance contains the unlocked balance of a wallet,
	// defining the total amount of coins as well as all the outputs that are unlocked.
	WalletUnlockedBalance struct {
		Total   Currency                `json:"total" msg:"t"`
		Outputs WalletUnlockedOutputMap `json:"outputs,omitempty" msg:"o,omitempty"`
	}
	// WalletUnlockedOutputMap defines the mapping between a coin output ID and its walletUnlockedOutput data
	WalletUnlockedOutputMap map[string]WalletUnlockedOutput
	// WalletUnlockedOutput defines an unlocked output targeted at a wallet.
	WalletUnlockedOutput struct {
		Amount      Currency `json:"amount" msg:"a"`
		Description string   `json:"description,omitempty" msg:"d,omitempty"`
	}

	// WalletLockedBalance contains the locked balance of a wallet,
	// defining the total amount of coins as well as all the outputs that are locked.
	WalletLockedBalance struct {
		Total   Currency              `json:"total" msg:"t"`
		Outputs WalletLockedOutputMap `json:"outputs,omitempty" msg:"o,omitempty"`
	}
	// WalletLockedOutputMap defines the mapping between a coin output ID and its walletLockedOutput data
	WalletLockedOutputMap map[string]WalletLockedOutput
	// WalletLockedOutput defines a locked output targeted at a wallet.
	WalletLockedOutput struct {
		Amount      Currency  `json:"amount" msg:"a"`
		LockedUntil LockValue `json:"lockedUntil" msg:"lu"`
		Description string    `json:"description,omitempty" msg:"d,omitempty"`
	}

	// WalletMultiSignData defines the extra data defined for a MultiSignWallet.
	WalletMultiSignData struct {
		Owners             []UnlockHash `json:"owners" msg:"o"`
		SignaturesRequired uint64       `json:"signaturesRequired" msg:"sr"`
	}

	// BotRecord wraps around the regular (tfchain) bot record, as to be able to define the custom ProtoBuf logic
	BotRecord struct {
		ID         BotID                   `json:"id" msg:"i"`
		Addresses  NetworkAddressSortedSet `json:"addresses,omitempty" msg:"a"`
		Names      BotNameSortedSet        `json:"names,omitempty" msg:"n"`
		PublicKey  PublicKey               `json:"publickey" msg:"k"`
		Expiration CompactTimestamp        `json:"expiration" msg:"e"`
	}
)

var (
	_ encoding.ProtocolBufferMarshaler   = (*NetworkStats)(nil)
	_ encoding.ProtocolBufferUnmarshaler = (*NetworkStats)(nil)

	_ encoding.ProtocolBufferMarshaler   = (*Wallet)(nil)
	_ encoding.ProtocolBufferUnmarshaler = (*Wallet)(nil)

	_ encoding.ProtocolBufferMarshaler   = (*BotRecord)(nil)
	_ encoding.ProtocolBufferUnmarshaler = (*BotRecord)(nil)
)

// ProtocolBufferMarshal implements encoding.ProtocolBufferMarshaler.ProtocolBufferMarshal
// using the generated code based on the PBNetworkStats Message defined in ./types.proto
func (stats *NetworkStats) ProtocolBufferMarshal(w encoding.ProtocolBufferWriter) error {
	err := w.Marshal(&PBNetworkStats{
		Timestamp:                            uint64(stats.Timestamp.Timestamp),
		Blockheight:                          uint64(stats.BlockHeight.BlockHeight),
		TxCount:                              stats.TransactionCount,
		CoinCreationTxCount:                  stats.CoinCreationTransactionCount,
		CoinCreatorDefTxCount:                stats.CoinCreatorDefinitionTransactionCount,
		ThreeBotRegistrationTransactionCount: stats.ThreeBotRegistrationTransactionCount,
		ThreeBotUpdateTransactionCount:       stats.ThreeBotUpdateTransactionCount,
		ValueTxCount:                         stats.ValueTransactionCount,
		CoinOutputCount:                      stats.CoinOutputCount,
		LockedCoinOutputCount:                stats.LockedCoinOutputCount,
		CoinInputCount:                       stats.CoinInputCount,
		MinerPayoutCount:                     stats.MinerPayoutCount,
		TxFeeCount:                           stats.TransactionFeeCount,
		MinerPayouts:                         stats.MinerPayouts.Bytes(),
		TxFees:                               stats.TransactionFees.Bytes(),
		Coins:                                stats.Coins.Bytes(),
		LockedCoins:                          stats.LockedCoins.Bytes(),
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
	stats.ThreeBotRegistrationTransactionCount = pb.ThreeBotRegistrationTransactionCount
	stats.ThreeBotUpdateTransactionCount = pb.ThreeBotUpdateTransactionCount
	stats.ValueTransactionCount = pb.ValueTxCount
	stats.CoinOutputCount = pb.CoinOutputCount
	stats.LockedCoinOutputCount = pb.LockedCoinOutputCount
	stats.CoinInputCount = pb.CoinInputCount
	stats.MinerPayoutCount = pb.MinerPayoutCount
	stats.TransactionFeeCount = pb.TxFeeCount

	// unmarshal all required Currency values
	err = stats.MinerPayouts.LoadBytes(pb.MinerPayouts)
	if err != nil {
		return fmt.Errorf("NetworkStats: MinerPayouts: %v", err)
	}
	err = stats.TransactionFees.LoadBytes(pb.TxFees)
	if err != nil {
		return fmt.Errorf("NetworkStats: TransactionFees: %v", err)
	}
	err = stats.LockedCoins.LoadBytes(pb.LockedCoins)
	if err != nil {
		return fmt.Errorf("NetworkStats: LockedCoins: %v", err)
	}
	err = stats.Coins.LoadBytes(pb.Coins)
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
			Total:   wallet.Balance.Unlocked.Total.Bytes(),
			Outputs: make(map[string]*PBWalletUnlockedOutput, len(wallet.Balance.Unlocked.Outputs)),
		}
		pb.BalanceUnlocked = ub
		for id, output := range wallet.Balance.Unlocked.Outputs {
			ub.Outputs[id] = &PBWalletUnlockedOutput{
				Amount:      output.Amount.Bytes(),
				Description: output.Description,
			}
		}
	}
	// add optional LockedBalance only if available
	if !wallet.Balance.Locked.Total.IsZero() {
		lb := &PBWalletLockedBalance{
			Total:   wallet.Balance.Locked.Total.Bytes(),
			Outputs: make(map[string]*PBWalletLockedOutput, len(wallet.Balance.Locked.Outputs)),
		}
		pb.BalanceLocked = lb
		for id, output := range wallet.Balance.Locked.Outputs {
			lb.Outputs[id] = &PBWalletLockedOutput{
				Amount:      output.Amount.Bytes(),
				LockedUntil: uint64(output.LockedUntil),
				Description: output.Description,
			}
		}
	}
	// add optional MultiSignAddresses only if available
	if n := len(wallet.MultiSignAddresses); n > 0 {
		pb.MultisignAddresses = make([][]byte, n)
		for idx, uh := range wallet.MultiSignAddresses {
			pb.MultisignAddresses[idx] = siabin.Marshal(uh)
		}
	}
	// add optional MultiSignData only if available
	if wallet.MultiSignData.SignaturesRequired > 0 {
		pb.MultisignData = &PBWalletMultiSignData{
			SignaturesRequired: wallet.MultiSignData.SignaturesRequired,
			Owners:             make([][]byte, len(wallet.MultiSignData.Owners)),
		}
		for idx, uh := range wallet.MultiSignData.Owners {
			pb.MultisignData.Owners[idx] = siabin.Marshal(uh)
		}
	}
	// Marshal the entire wallet into the given ProtocolBufferWriter
	err := w.Marshal(pb)
	if err != nil {
		return fmt.Errorf("Wallet: %v", err)
	}
	return nil
}

// ensure custom MessagePack encodable types for our wallet and wallet balance
var (
	_ msgp.Encodable   = (*Wallet)(nil)
	_ msgp.Decodable   = (*Wallet)(nil)
	_ json.Marshaler   = (*Wallet)(nil)
	_ json.Unmarshaler = (*Wallet)(nil)

	_ msgp.Encodable   = (*WalletBalance)(nil)
	_ msgp.Decodable   = (*WalletBalance)(nil)
	_ json.Marshaler   = (*WalletBalance)(nil)
	_ json.Unmarshaler = (*WalletBalance)(nil)
)

// IsNil returns true if this wallet is a nil wallet.
func (wallet *Wallet) IsNil() bool {
	return wallet.Balance.IsZero() &&
		len(wallet.MultiSignAddresses) == 0 &&
		wallet.MultiSignData.SignaturesRequired == 0
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
// Encoding a wallet using the EncodableWallet structure.
func (wallet *Wallet) EncodeMsg(w *msgp.Writer) error {
	ew := &EncodableWallet{
		// already assign multisign addresses,
		// as this value is taken over directly
		MultiSignAddresses: wallet.MultiSignAddresses,
	}
	// only add unlocked balance if it is defined
	if !wallet.Balance.Unlocked.Total.IsZero() {
		ew.Balance = new(EncodableWalletBalance)
		ew.Balance.Unlocked = &wallet.Balance.Unlocked
	}
	// only add locked balance if it is defined
	if !wallet.Balance.Locked.Total.IsZero() {
		if ew.Balance == nil {
			ew.Balance = new(EncodableWalletBalance)
		}
		ew.Balance.Locked = &wallet.Balance.Locked
	}
	// only add multi sign data for multi sig wallets
	if wallet.MultiSignData.SignaturesRequired > 0 {
		ew.MultiSignData = &wallet.MultiSignData
	}
	// encode wallet
	return ew.EncodeMsg(w)
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
// Decoding a wallet using the EncodableWallet structure.
func (wallet *Wallet) DecodeMsg(r *msgp.Reader) error {
	var ew EncodableWallet
	err := ew.DecodeMsg(r)
	if err != nil {
		return err
	}
	// reset
	wallet.Balance = WalletBalance{} // reset
	if ew.Balance != nil {
		// and assign any values that are given
		if ew.Balance.Unlocked != nil {
			wallet.Balance.Unlocked = *ew.Balance.Unlocked
		}
		if ew.Balance.Locked != nil {
			wallet.Balance.Locked = *ew.Balance.Locked
		}
	}
	// always assign multisign addresses, as it is a slice
	wallet.MultiSignAddresses = ew.MultiSignAddresses
	// reset or assign MultiSignData
	if ew.MultiSignData == nil {
		wallet.MultiSignData = WalletMultiSignData{} // reset
	} else {
		wallet.MultiSignData = *ew.MultiSignData // or assign
	}
	// success
	return nil
}

// MarshalJSON implements json.Marshaler.MarshalJSON
// Encoding a Wallet using the EncodableWallet structure.
func (wallet Wallet) MarshalJSON() ([]byte, error) {
	ew := &EncodableWallet{
		// already assign multisign addresses,
		// as this value is taken over directly
		MultiSignAddresses: wallet.MultiSignAddresses,
	}
	// only add unlocked balance if it is defined
	if !wallet.Balance.Unlocked.Total.IsZero() {
		ew.Balance = new(EncodableWalletBalance)
		ew.Balance.Unlocked = &wallet.Balance.Unlocked
	}
	// only add locked balance if it is defined
	if !wallet.Balance.Locked.Total.IsZero() {
		if ew.Balance == nil {
			ew.Balance = new(EncodableWalletBalance)
		}
		ew.Balance.Locked = &wallet.Balance.Locked
	}
	// only add multi sign data for multi sig wallets
	if wallet.MultiSignData.SignaturesRequired > 0 {
		ew.MultiSignData = &wallet.MultiSignData
	}
	// encode wallet
	return json.Marshal(ew)
}

// UnmarshalJSON implements json.Unmarshaler.UnmarshalJSON
// Decoding a Wallet using the EncodableWallet structure.
func (wallet *Wallet) UnmarshalJSON(data []byte) error {
	var ew EncodableWallet
	err := json.Unmarshal(data, &ew)
	if err != nil {
		return err
	}
	// reset
	wallet.Balance = WalletBalance{} // reset
	if ew.Balance != nil {
		// and assign any values that are given
		if ew.Balance.Unlocked != nil {
			wallet.Balance.Unlocked = *ew.Balance.Unlocked
		}
		if ew.Balance.Locked != nil {
			wallet.Balance.Locked = *ew.Balance.Locked
		}
	}
	// always assign multisign addresses, as it is a slice
	wallet.MultiSignAddresses = ew.MultiSignAddresses
	// reset or assign MultiSignData
	if ew.MultiSignData == nil {
		wallet.MultiSignData = WalletMultiSignData{} // reset
	} else {
		wallet.MultiSignData = *ew.MultiSignData // or assign
	}
	// success
	return nil
}

// EncodeMsg implements msgp.Encodable.EncodeMsg
// Encoding a WalletBalance using the EncodableWalletBalance structure.
func (wb *WalletBalance) EncodeMsg(w *msgp.Writer) error {
	if wb.IsZero() {
		return nil // nothing to do
	}
	ewb := new(EncodableWalletBalance)
	if !wb.Unlocked.Total.IsZero() {
		ewb.Unlocked = &wb.Unlocked
	}
	if !wb.Locked.Total.IsZero() {
		ewb.Locked = &wb.Locked
	}
	// encode wallet balance
	return ewb.EncodeMsg(w)
}

// DecodeMsg implements msgp.Decodable.DecodeMsg
// Decoding a WalletBalance using the EncodableWalletBalance structure.
func (wb *WalletBalance) DecodeMsg(r *msgp.Reader) error {
	var ewb EncodableWalletBalance
	err := ewb.DecodeMsg(r)
	if err != nil {
		return err
	}
	// reset or assign unlocked balance
	if ewb.Unlocked == nil {
		wb.Unlocked = WalletUnlockedBalance{} // reset
	} else {
		wb.Unlocked = *ewb.Unlocked // or assign
	}
	// reset or assign locked balance
	if ewb.Locked == nil {
		wb.Locked = WalletLockedBalance{} // reset
	} else {
		wb.Locked = *ewb.Locked // or assign
	}
	// success
	return nil
}

// MarshalJSON implements json.Marshaler.MarshalJSON
// Encoding a WalletBalance using the EncodableWalletBalance structure.
func (wb WalletBalance) MarshalJSON() ([]byte, error) {
	if wb.IsZero() {
		return []byte("null"), nil // nothing to do
	}
	ewb := new(EncodableWalletBalance)
	if !wb.Unlocked.Total.IsZero() {
		ewb.Unlocked = &wb.Unlocked
	}
	if !wb.Locked.Total.IsZero() {
		ewb.Locked = &wb.Locked
	}
	// encode wallet balance
	return json.Marshal(ewb)
}

// UnmarshalJSON implements json.Unmarshaler.UnmarshalJSON
// Decoding a WalletBalance using the EncodableWalletBalance structure.
func (wb *WalletBalance) UnmarshalJSON(data []byte) error {
	var ewb EncodableWalletBalance
	err := json.Unmarshal(data, &ewb)
	if err != nil {
		return err
	}
	// reset or assign unlocked balance
	if ewb.Unlocked == nil {
		wb.Unlocked = WalletUnlockedBalance{} // reset
	} else {
		wb.Unlocked = *ewb.Unlocked // or assign
	}
	// reset or assign locked balance
	if ewb.Locked == nil {
		wb.Locked = WalletLockedBalance{} // reset
	} else {
		wb.Locked = *ewb.Locked // or assign
	}
	// success
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
		err = wallet.Balance.Unlocked.Total.LoadBytes(pb.BalanceUnlocked.Total)
		if err != nil {
			return fmt.Errorf("Wallet: Unlocked Balance: Total: %v", err)
		}
		// unmarshal all outputs
		wallet.Balance.Unlocked.Outputs = make(WalletUnlockedOutputMap, len(pb.BalanceUnlocked.Outputs))
		for id, output := range pb.BalanceUnlocked.Outputs {
			var unlockedOutput WalletUnlockedOutput
			// unmarshal amount
			err = unlockedOutput.Amount.LoadBytes(output.Amount)
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
		err = wallet.Balance.Locked.Total.LoadBytes(pb.BalanceLocked.Total)
		if err != nil {
			return fmt.Errorf("Wallet: Locked Balance: Total: %v", err)
		}
		// unmarshal all outputs
		wallet.Balance.Locked.Outputs = make(WalletLockedOutputMap, len(pb.BalanceLocked.Outputs))
		for id, output := range pb.BalanceLocked.Outputs {
			var lockedOutput WalletLockedOutput
			// unmarshal amount
			err = lockedOutput.Amount.LoadBytes(output.Amount)
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
			err = siabin.Unmarshal(b, &wallet.MultiSignAddresses[i])
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
			err = siabin.Unmarshal(b, &wallet.MultiSignData.Owners[i])
			if err != nil {
				return fmt.Errorf("Wallet: MultiSignData: owner #%d: %v", i, err)
			}
		}
	}
	return nil
}

// IsZero returns true if this wallet's balance is Zero
func (wb *WalletBalance) IsZero() bool {
	return wb.Unlocked.Total.IsZero() && wb.Locked.Total.IsZero()
}

// AddUnlockedCoinOutput adds the unique unlocked coin output to the wallet's map of unlocked outputs
// as well as adds the coin output's value to the total amount of unlocked coins registered for this wallet.
func (wub *WalletUnlockedBalance) AddUnlockedCoinOutput(id CoinOutputID, co WalletUnlockedOutput, addAmount bool) error {
	idStr := id.String()
	if len(wub.Outputs) == 0 {
		wub.Outputs = make(WalletUnlockedOutputMap)
	} else if _, exists := wub.Outputs[idStr]; exists {
		return fmt.Errorf("trying to add existing unlocked coin output %s", idStr)
	}
	wub.Outputs[idStr] = co
	if addAmount {
		wub.Total = wub.Total.Add(co.Amount)
	}
	return nil
}

// SubUnlockedCoinOutput tries to remove the unlocked coin output from the wallet's map of unlocked outputs,
// try as it might not exist to never having been added. This method does always
// subtract the coin output's value from the total amount of unlocked coins registered for this wallet.
func (wub *WalletUnlockedBalance) SubUnlockedCoinOutput(id CoinOutputID, amount Currency, subAmount bool) error {
	if len(wub.Outputs) != 0 {
		delete(wub.Outputs, id.String())
	}
	if subAmount {
		wub.Total = wub.Total.Sub(amount)
	}
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

// BotRecordFromTfchainRecord creates a BotRecord using a tf-typed bot record as source.
func BotRecordFromTfchainRecord(record tftypes.BotRecord) BotRecord {
	return BotRecord{
		ID:         NewBotIDFromTfchainBotID(record.ID),
		Addresses:  NewNetworkAddressSortedSetFromTfchainNetworkAddressSortedSet(record.Addresses),
		Names:      NewBotNameSortedSetFromTfchainBotNameSortedSet(record.Names),
		PublicKey:  NewPublicKeyFromTfchainPublicKey(record.PublicKey),
		Expiration: NewCompactTimeStampFromTfchainCompactTimestamp(record.Expiration),
	}
}

// TfchainRecord creates a tf-typed BotRecord using this bot record as source.
func (record BotRecord) TfchainRecord() tftypes.BotRecord {
	return tftypes.BotRecord{
		ID:         record.ID.TfchainBotID(),
		Addresses:  record.Addresses.TfchainNetworkAddressSortedSet(),
		Names:      record.Names.TfchainBotNameSortedSet(),
		PublicKey:  record.PublicKey.TfchainPublicKey(),
		Expiration: record.Expiration.TfchainCompactTimestamp(),
	}
}

// ProtocolBufferMarshal implements encoding.ProtocolBufferMarshaler.ProtocolBufferMarshal
// using the generated code based on the PBNetworkStats Message defined in ./types.proto
func (record *BotRecord) ProtocolBufferMarshal(w encoding.ProtocolBufferWriter) error {
	err := w.Marshal(&PBThreeBotRecord{
		Id:               uint32(record.ID.TfchainBotID()),
		NetworkAddresses: rivbin.Marshal(record.Addresses),
		Names:            rivbin.Marshal(record.Names),
		ExpirationTime:   rivbin.Marshal(record.Expiration),
		PublicKey:        rivbin.Marshal(record.PublicKey),
	})
	if err != nil {
		return fmt.Errorf("BotRecord: %v", err)
	}
	return nil
}

// ProtocolBufferUnmarshal implements encoding.ProtocolBufferUnmarshaler.ProtocolBufferUnmarshal
// using the generated code based on the PBNetworkStats Message defined in ./types.proto
func (record *BotRecord) ProtocolBufferUnmarshal(r encoding.ProtocolBufferReader) error {
	// unmarshal entire protocol buffer message as a whole
	var pb PBThreeBotRecord
	err := r.Unmarshal(&pb)
	if err != nil {
		return fmt.Errorf("BotRecord: %v", err)
	}

	// assign all values that can be assigned directly
	record.ID = NewBotIDFromTfchainBotID(tftypes.BotID(pb.Id))

	// unmarshal all values that requires decoding
	err = rivbin.Unmarshal(pb.NetworkAddresses, &record.Addresses)
	if err != nil {
		return fmt.Errorf("BotRecord: Addresses: %v", err)
	}
	err = rivbin.Unmarshal(pb.Names, &record.Names)
	if err != nil {
		return fmt.Errorf("BotRecord: Names: %v", err)
	}
	err = rivbin.Unmarshal(pb.ExpirationTime, &record.Expiration)
	if err != nil {
		return fmt.Errorf("BotRecord: Expiration: %v", err)
	}
	err = rivbin.Unmarshal(pb.PublicKey, &record.PublicKey)
	if err != nil {
		return fmt.Errorf("BotRecord: PublicKey: %v", err)
	}

	// all was unmarshaled, return nil (= no error)
	return nil
}
