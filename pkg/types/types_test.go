package types

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/glycerine/greenpack/msgp"
	"github.com/rivine/rivine/types"
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
		// decode and encode using a map, to remove the specially added @ field
		var jsonMap map[string]interface{}
		err = json.Unmarshal(jsonBuf.Bytes(), &jsonMap)
		if err != nil {
			t.Errorf("failed to json-decode %s as map[string]interface{}: %v", description, err)
			return
		}
		deepRemoveTaggedField(jsonMap)
		data, err := json.Marshal(jsonMap)
		if err != nil {
			t.Errorf("failed to json-encode %s as map[string]interface{}: %v", description, err)
			return
		}
		// compare encoded result with expected situation
		result := string(data)
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

	testMarshal(Wallet{}, "empty wallet", "{}")
	testMarshal(Wallet{
		MultiSignAddresses: []UnlockHash{unlockHashFromHex(t, "0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37")},
	}, "multisig-referenced-only-wallet",
		`{"ma":["0359aaaa311a10efd7762953418b828bfe2d4e2111dfe6aaf82d4adf6f2fb385688d7f86510d37"]}`)
	testMarshal(Wallet{
		Balance: WalletBalance{
			Unlocked: WalletUnlockedBalance{
				Total: AsCurrency(types.NewCurrency64(1))}}},
		"minimal-unlocked-balance-only-wallet",
		`{"b":{"u":{"t":"1"}}}`)
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
		`{"b":{"u":{"o":{"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef":{"a":"1","d":"for:you"}},"t":"1"}}}`)
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
		`{"b":{"l":{"o":{"1e8cb9bfd8d35a523fefe563b5402e8a49ce86b5555bfde6eafbd226457a23ef":{"a":"1","d":"for:you","lu":2000}},"t":"1"}}}`)
}

func unlockHashFromHex(t *testing.T, str string) (uh UnlockHash) {
	t.Helper()
	err := uh.LoadString(str)
	if err != nil {
		t.Fatalf("failed to decode unlock hash %q: %v", str, err)
	}
	return
}
