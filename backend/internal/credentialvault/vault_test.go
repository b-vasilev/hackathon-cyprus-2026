package credentialvault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/argon2"
)

func testVault(t *testing.T, secret string) *Vault {
	t.Helper()
	v, err := New(secret)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(v.clearCachedKeys)
	return v
}

func decodeEnvelope(t *testing.T, encoded []byte) envelope {
	t.Helper()
	var data envelope
	if err := json.Unmarshal(encoded, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCachedKeyStillAuthenticatesEveryEnvelope(t *testing.T) {
	master := strings.Repeat("m", 32)
	writer := testVault(t, master)
	sealed, err := writer.Seal("connection:1", []byte("provider secret"))
	if err != nil {
		t.Fatal(err)
	}
	reader := testVault(t, master)
	plain, err := reader.Open("connection:1", sealed)
	if err != nil {
		t.Fatal(err)
	}
	plain[0] = '!'
	data := decodeEnvelope(t, sealed)
	identity := derivationID{id: "connection:1", salt: [32]byte(data.Salt)}
	cached := reader.keys[identity].key
	for range 3 {
		plain, err = reader.Open("connection:1", sealed)
		if err != nil || string(plain) != "provider secret" {
			t.Fatal("repeated read changed plaintext or reused a caller's buffer")
		}
		if &reader.keys[identity].key[0] != &cached[0] {
			t.Fatal("repeated read replaced the cached derived key")
		}
	}
	for _, field := range []string{"ciphertext", "nonce", "salt"} {
		t.Run(field, func(t *testing.T) {
			tampered := decodeEnvelope(t, sealed)
			switch field {
			case "ciphertext":
				tampered.Ciphertext[0] ^= 1
			case "nonce":
				tampered.Nonce[0] ^= 1
			case "salt":
				tampered.Salt[0] ^= 1
			}
			encoded, encodeErr := json.Marshal(tampered)
			if encodeErr != nil {
				t.Fatal(encodeErr)
			}
			if value, openErr := reader.Open("connection:1", encoded); openErr == nil || value != nil || len(reader.keys) != 1 {
				t.Fatal("cache hit bypassed authenticated decryption")
			}
		})
	}
	if _, err := reader.Open("connection:2", sealed); err == nil || len(reader.keys) != 1 {
		t.Fatal("cached key crossed revision identity or a failed read populated the cache")
	}
	wrongMaster := testVault(t, strings.Repeat("x", 32))
	if _, err := wrongMaster.Open("connection:1", sealed); err == nil || len(wrongMaster.keys) != 0 {
		t.Fatal("cache crossed vaults or retained an unauthenticated key")
	}
}

func TestVaultReadsLegacyEnvelopeAndSeparatesSalts(t *testing.T) {
	master := strings.Repeat("m", 32)
	id := "connection:1"
	// Construct the original envelope independently of Vault.Seal.
	legacy := envelope{Salt: bytes.Repeat([]byte{1}, 32), Nonce: bytes.Repeat([]byte{2}, 12)}
	key := argon2.IDKey([]byte(id+":"+master), legacy.Salt, 3, 64*1024, 4, 32)
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	legacy.Ciphertext = aead.Seal(nil, legacy.Nonce, []byte("legacy secret"), []byte(id))
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	v := testVault(t, master)
	if plain, openErr := v.Open(id, encoded); openErr != nil || string(plain) != "legacy secret" {
		t.Fatal("original envelope is no longer readable")
	}
	fresh, err := v.Seal(id, []byte("new secret"))
	if err != nil {
		t.Fatal(err)
	}
	if plain, openErr := v.Open(id, fresh); openErr != nil || string(plain) != "new secret" {
		t.Fatal("different salts for the same ID shared the wrong key")
	}
	if len(v.keys) != 2 {
		t.Fatal("salt was omitted from the cache identity")
	}
	if plain, openErr := v.Open(id, encoded); openErr != nil || string(plain) != "legacy secret" {
		t.Fatal("sealing a new envelope displaced the original salt's key")
	}
}

func TestVaultConcurrentColdReads(t *testing.T) {
	master := strings.Repeat("m", 32)
	writer := testVault(t, master)
	sealed, err := writer.Seal("connection:1", []byte("provider secret"))
	if err != nil {
		t.Fatal(err)
	}
	reader := testVault(t, master)
	var workers sync.WaitGroup
	for range 12 {
		workers.Go(func() {
			plain, openErr := reader.Open("connection:1", sealed)
			if openErr != nil || string(plain) != "provider secret" {
				t.Error("concurrent read failed")
			}
		})
	}
	workers.Wait()
	if len(reader.keys) != 1 || reader.timer == nil {
		t.Fatal("cold reads did not share an expiring key cache")
	}
}

func TestCacheEvictionAndExpiryEraseDerivedKeys(t *testing.T) {
	v := testVault(t, strings.Repeat("m", 32))
	key := bytes.Repeat([]byte{7}, 32)
	for i := range maxCachedKeys {
		v.remember(derivationID{id: strconv.Itoa(i)}, key)
	}
	first := derivationID{id: "0"}
	second := derivationID{id: "1"}
	evicted := v.keys[second].key
	v.remember(first, key) // Keep the recently used key when capacity is reached.
	v.remember(derivationID{id: "new"}, key)
	if len(v.keys) != maxCachedKeys || v.keys[first].key == nil || v.keys[second].key != nil {
		t.Fatal("cache exceeded its bound or evicted the recently used key")
	}
	if !bytes.Equal(evicted, make([]byte, 32)) {
		t.Fatal("eviction did not erase the retained key buffer")
	}
	retained := v.keys[first].key
	v.clearCachedKeys() // The expiry timer invokes this same cleanup.
	if len(v.keys) != 0 || v.timer != nil || !bytes.Equal(retained, make([]byte, 32)) {
		t.Fatal("expiry did not erase retained keys and reset the timer")
	}
	if !bytes.Equal(key, bytes.Repeat([]byte{7}, 32)) {
		t.Fatal("cache retained and erased its caller's key buffer")
	}
}
