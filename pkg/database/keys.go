package database

import (
	"strings"

	"github.com/threefoldfoundation/rexplorer/pkg/types"
)

// Root Keys used for public Database values
const (
	StatsKey = "stats"

	AddressesKey = "addresses"

	CoinCreatorsKey = "coincreators"

	AddressKeyPrefix      = "a:"
	CoinOutputKeyPrefix   = "c:"
	ThreeBotKeyPrefix     = "b:"
	ERC20AddressKeyPrefix = "e:"
)

// GetAddressKeyAndField gets the key and field hash for an unlock hash.
func GetAddressKeyAndField(uh types.UnlockHash) (key, field string) {
	str := uh.String()
	key, field = AddressKeyPrefix+str[:6], str[6:]
	return
}

// GetCoinOutputKeyAndField gets the key and field hash for a coin output ID.
func GetCoinOutputKeyAndField(id types.CoinOutputID) (key, field string) {
	str := id.String()
	key, field = CoinOutputKeyPrefix+str[:4], str[4:]
	return
}

// GetThreeBotKeyAndField gets the key and field hash for a 3Bot (record) ID.
func GetThreeBotKeyAndField(id types.BotID) (string, string) {
	if id.UInt32() < 100 {
		return ThreeBotKeyPrefix + "0", id.String()
	}
	str := id.String()
	cop := len(str) - 2
	field := strings.TrimPrefix(str[cop:], "0")
	if field == "" {
		field = "0"
	}
	key := ThreeBotKeyPrefix + str[:cop]
	return key, field
}

// GetERC20AddressKeyAndField gets the key and field hash for an ERC20 Address.
func GetERC20AddressKeyAndField(addr types.ERC20Address) (string, string) {
	str := addr.String()
	return ERC20AddressKeyPrefix + str[:6], str[6:]
}
