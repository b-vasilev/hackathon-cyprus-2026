// Package credentialvault stores connection configuration using authenticated encryption.
// Adapted from antares-app/backend/internal/shared/credentialvault.
package credentialvault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	maxCachedKeys = 64
	keyCacheTTL   = 5 * time.Minute
)

// Vault encrypts configuration with a server-owned master secret.
// Derived keys stay local to this instance; decrypted configurations are never cached.
type Vault struct {
	secret string
	mu     sync.Mutex
	keys   map[derivationID]cachedKey
	used   uint64
	timer  *time.Timer
}

type derivationID struct {
	id   string
	salt [32]byte
}

type cachedKey struct {
	key  []byte
	used uint64
}

type envelope struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// New requires a master secret independent of provider credentials.
func New(secret string) (*Vault, error) {
	if len(secret) < 32 {
		return nil, errors.New("CONNECTION_MASTER_KEY must contain at least 32 characters")
	}
	return &Vault{secret: secret, keys: make(map[derivationID]cachedKey)}, nil
}

// Called under mu so concurrent reads derive a missing key only once and cannot
// race with key erasure. Callers erase their temporary copy after use.
func (v *Vault) key(id string, salt []byte) (derivationID, []byte) {
	identity := derivationID{id: id, salt: [32]byte(salt)}
	if entry, ok := v.keys[identity]; ok {
		return identity, slices.Clone(entry.key)
	}
	return identity, argon2.IDKey([]byte(id+":"+v.secret), salt, 3, 64*1024, 4, 32)
}

// Remember only keys used by a successfully sealed or authenticated envelope.
// The timer expires the entire cache even if the vault receives no more calls.
func (v *Vault) remember(identity derivationID, key []byte) {
	v.used++
	if entry, ok := v.keys[identity]; ok {
		entry.used = v.used
		v.keys[identity] = entry
		return
	}
	if len(v.keys) == maxCachedKeys {
		var oldest derivationID
		oldestUse := v.used
		for id, entry := range v.keys {
			if entry.used < oldestUse {
				oldest, oldestUse = id, entry.used
			}
		}
		clear(v.keys[oldest].key)
		delete(v.keys, oldest)
	}
	v.keys[identity] = cachedKey{key: slices.Clone(key), used: v.used}
	if v.timer == nil {
		v.timer = time.AfterFunc(keyCacheTTL, v.clearCachedKeys)
	}
}

func (v *Vault) clearCachedKeys() {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, entry := range v.keys {
		clear(entry.key)
	}
	clear(v.keys)
	v.used = 0
	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
}

func credentialAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create credential cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create credential authentication: %w", err)
	}
	return aead, nil
}

// Seal binds encrypted data to a connection revision.
func (v *Vault) Seal(id string, plaintext []byte) ([]byte, error) {
	data := envelope{Salt: make([]byte, 32), Nonce: make([]byte, 12)}
	if _, err := rand.Read(data.Salt); err != nil {
		return nil, fmt.Errorf("generate credential salt: %w", err)
	}
	if _, err := rand.Read(data.Nonce); err != nil {
		return nil, fmt.Errorf("generate credential nonce: %w", err)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	identity, key := v.key(id, data.Salt)
	defer clear(key)
	aead, err := credentialAEAD(key)
	if err != nil {
		return nil, err
	}
	data.Ciphertext = aead.Seal(nil, data.Nonce, plaintext, []byte(id))
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode credential: %w", err)
	}
	v.remember(identity, key)
	return encoded, nil
}

// Open authenticates and decrypts a stored configuration.
func (v *Vault) Open(id string, encoded []byte) ([]byte, error) {
	var data envelope
	if err := json.Unmarshal(encoded, &data); err != nil {
		return nil, errors.New("invalid encrypted credential")
	}
	if len(data.Salt) != 32 || len(data.Nonce) != 12 {
		return nil, errors.New("invalid encrypted credential")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	identity, key := v.key(id, data.Salt)
	defer clear(key)
	aead, err := credentialAEAD(key)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, data.Nonce, data.Ciphertext, []byte(id))
	if err != nil {
		return nil, errors.New("credential decryption failed; check CONNECTION_MASTER_KEY")
	}
	v.remember(identity, key)
	return plaintext, nil
}
