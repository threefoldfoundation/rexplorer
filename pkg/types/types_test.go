package types

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rivine/rivine/types"
	"github.com/tinylib/msgp/msgp"
)

func TestJSONEncodeMinimalWallets(t *testing.T) {
	testMarshal := func(wallet Wallet, description, expected string) {
		t.Helper()

		// encode
		data, err := json.Marshal(wallet)
		if err != nil {
			t.Errorf("failed to encode %s: %v", description, err)
			return
		}

		// compare encoded result with expected situation
		result := string(data)
		if expected != result {
			t.Errorf("unexpected encoded result for %s: %s != %s", description, expected, result)
			return
		}

		// decode again
		var decodedWallet Wallet
		err = json.Unmarshal(data, &decodedWallet)
		if err != nil {
			t.Errorf("failed to decode encoded %s: %v", description, err)
			return
		}

		// compare decoded wallet with given wallet
		if !reflect.DeepEqual(wallet, decodedWallet) {
			t.Errorf("unexpected decoded result for %s: %v != %v", description, wallet, decodedWallet)
		}
	}

	testMarshal(Wallet{}, "empty wallet", "{}")
	testMarshal(Wallet{
		MultiSignAddresses: []UnlockHash{unlockHashFromHex(t, "0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37")},
	}, "multisig-referenced-only-wallet",
		`{"multisignAddresses":["0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"]}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Unlocked: WalletUnlockedBalance{
				Total: AsCurrency(types.NewCurrency64(1))}}},
		"minimal-unlocked-balance-only-wallet",
		`{"balance":{"unlocked":{"total":"1"}}}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Unlocked: WalletUnlockedBalance{
				Total: AsCurrency(types.NewCurrency64(1)),
				Outputs: WalletUnlockedOutputMap{
					"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef": {
						Amount:      AsCurrency(types.NewCurrency64(1)),
						Description: "for:you",
					},
				},
			}}},
		"minimal-unlocked-balance-with-output-wallet",
		`{"balance":{"unlocked":{"total":"1","outputs":{"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef":{"amount":"1","description":"for:you"}}}}}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Locked: WalletLockedBalance{
				Total: AsCurrency(types.NewCurrency64(1)),
				Outputs: WalletLockedOutputMap{
					"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef": {
						Amount:      AsCurrency(types.NewCurrency64(1)),
						LockedUntil: 2000,
						Description: "for:you",
					},
				},
			}}},
		"minimal-locked-balance-wallet",
		`{"balance":{"locked":{"total":"1","outputs":{"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef":{"amount":"1","lockedUntil":2000,"description":"for:you"}}}}}`)
}

func deepRemoveTaggedField(v interface{}) interface{} {
	m, ok := v.(map[string]interface{})
	if !ok {
		return v
	}
	delete(m, "@")
	for k, v := range m {
		m[k] = deepRemoveTaggedField(v)
	}
	return m
}

func TestMessagePackEncodeMinimalWallets(t *testing.T) {
	testMarshal := func(wallet Wallet, description, expected string) {
		t.Helper()

		buf := bytes.NewBuffer(nil)
		w := msgp.NewWriter(buf)

		// encode
		err := wallet.EncodeMsg(w)
		if err != nil {
			t.Errorf("failed to encode %s: %v", description, err)
			return
		}
		err = w.Flush()
		if err != nil {
			t.Errorf("failed to flush msgp writer for %s: %v", description, err)
			return
		}

		// comopare to expected json
		jsonBuf := bytes.NewBuffer(nil)
		b := make([]byte, len(buf.Bytes()))
		copy(b, buf.Bytes())
		_, err = msgp.UnmarshalAsJSON(jsonBuf, b)
		if err != nil {
			t.Errorf("failed to decode %s as json: %v", description, err)
			return
		}
		// compare encoded result with expected situation
		result := string(jsonBuf.Bytes())
		if expected != result {
			t.Errorf("unexpected JSON-encoded result for %s: %s != %s", description, expected, result)
			return
		}

		r := msgp.NewReader(buf)

		// decode again
		var decodedWallet Wallet
		err = decodedWallet.DecodeMsg(r)
		if err != nil {
			t.Errorf("failed to decode encoded %s: %v", description, err)
			return
		}

		// compare decoded wallet with given wallet
		if !reflect.DeepEqual(wallet, decodedWallet) {
			t.Errorf("unexpected decoded result for %s: %v != %v", description, wallet, decodedWallet)
		}
	}

	testMarshal(Wallet{}, "empty wallet", `{"b":null,"ma":[],"m":null}`)
	testMarshal(Wallet{
		MultiSignAddresses: []UnlockHash{unlockHashFromHex(t, "0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37")},
	}, "multisig-referenced-only-wallet",
		`{"b":null,"ma":["A1mqqjEaEO/XdilTQYuCi/4tTiER3+aq+C1K328vs4Vo"],"m":null}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Unlocked: WalletUnlockedBalance{
				Total:   AsCurrency(types.NewCurrency64(1)),
				Outputs: make(WalletUnlockedOutputMap),
			}}},
		"minimal-unlocked-balance-only-wallet",
		`{"b":{"u":{"t":"AQ==","o":{}},"l":null},"ma":[],"m":null}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Unlocked: WalletUnlockedBalance{
				Total: AsCurrency(types.NewCurrency64(1)),
				Outputs: WalletUnlockedOutputMap{
					"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef": {
						Amount:      AsCurrency(types.NewCurrency64(1)),
						Description: "for:you",
					},
				},
			}}},
		"minimal-unlocked-balance-with-output-wallet",
		`{"b":{"u":{"t":"AQ==","o":{"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef":{"a":"AQ==","d":"for:you"}}},"l":null},"ma":[],"m":null}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Locked: WalletLockedBalance{
				Total: AsCurrency(types.NewCurrency64(1)),
				Outputs: WalletLockedOutputMap{
					"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef": {
						Amount:      AsCurrency(types.NewCurrency64(1)),
						LockedUntil: 2000,
						Description: "for:you",
					},
				},
			}}},
		"minimal-locked-balance-wallet",
		`{"b":{"u":null,"l":{"t":"AQ==","o":{"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef":{"a":"AQ==","lu":2000,"d":"for:you"}}}},"ma":[],"m":null}`)
}

func TestUnmarshalJsonBotRecord(t *testing.T) {
	const input = `{"id":1,"addresses":["example.org","91.198.174.192"],"names":["chatbot.example"],"publickey":"ed25519:00bde9571b30e1742c41fcca8c730183402d967df5b17b5f4ced22c677806614","expiration":1543161720}`
	var record BotRecord
	err := json.Unmarshal([]byte(input), &record)
	if err != nil {
		t.Fatal(err)
	}
	if id := record.ID.UInt32(); id != 1 {
		t.Error("unexpected ID", id, "!=", 1)
	}
	if et := record.Expiration.CompactTimestamp; et != 1543161720 {
		t.Error("unexpected Expiration", et, "!=", 1543161720)
	}
	if pk := record.PublicKey.String(); pk != "ed25519:00bde9571b30e1742c41fcca8c730183402d967df5b17b5f4ced22c677806614" {
		t.Error("unexpected PublicKey", pk, "!= ed25519:00bde9571b30e1742c41fcca8c730183402d967df5b17b5f4ced22c677806614")
	}
	if addresses, err := record.Addresses.MarshalJSON(); err != nil {
		t.Error("error while marshaling addresses into a JSON byte slice:", err)
	} else if string(addresses) != `["example.org","91.198.174.192"]` {
		t.Error("unexpected NetworkAddresses", string(addresses), `!= ["example.org","91.198.174.192"]`)
	}
	if names, err := record.Names.MarshalJSON(); err != nil {
		t.Error("error while marshaling names into a JSON byte slice:", err)
	} else if string(names) != `["chatbot.example"]` {
		t.Error("unexpected BotNames", string(names), `!= ["chatbot.example"]`)
	}
}

func unlockHashFromHex(t *testing.T, str string) (uh UnlockHash) {
	t.Helper()
	err := uh.LoadString(str)
	if err != nil {
		t.Fatalf("failed to decode unlock hash %q: %v", str, err)
	}
	return
}
