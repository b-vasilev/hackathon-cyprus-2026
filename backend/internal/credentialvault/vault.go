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

	"golang.org/x/crypto/argon2"
)

// Vault encrypts configuration with a server-owned master secret.
type Vault struct{ secret string }

type envelope struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// New requires a master secret independent of provider credentials.
func New(secret string) (*Vault, error) {
	if len(secret) < 32 {
		return nil, errors.New("LLM_MASTER_KEY must contain at least 32 characters")
	}
	return &Vault{secret: secret}, nil
}

func (v *Vault) aead(id string, salt []byte) (cipher.AEAD, error) {
	key := argon2.IDKey([]byte(id+":"+v.secret), salt, 3, 64*1024, 4, 32)
	defer clear(key)
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
	aead, err := v.aead(id, data.Salt)
	if err != nil {
		return nil, err
	}
	data.Ciphertext = aead.Seal(nil, data.Nonce, plaintext, []byte(id))
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode credential: %w", err)
	}
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
	aead, err := v.aead(id, data.Salt)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, data.Nonce, data.Ciphertext, []byte(id))
	if err != nil {
		return nil, errors.New("credential decryption failed; check LLM_MASTER_KEY")
	}
	return plaintext, nil
}
