// Package secret provides at-rest encryption for integration credentials. Secrets are
// sealed with AES-256-GCM; the key is derived from the HS_SECRET_KEY passphrase via scrypt
// with a per-install random salt. Key derivation is lazy: if no passphrase is configured,
// the rest of the app runs fine — only sealing/opening a secret requires it.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/crypto/scrypt"
)

// ErrNoKey is returned when a secret operation is attempted without HS_SECRET_KEY set.
var ErrNoKey = errors.New("HS_SECRET_KEY is not set; it is required to store or read credentials")

// Storage persists opaque encrypted blobs keyed by ref.
type Storage interface {
	PutSecret(ref string, nonce, ciphertext []byte) error
	GetSecret(ref string) (nonce, ciphertext []byte, err error)
	DeleteSecret(ref string) error
}

// SaltStore persists the per-install KDF salt (a simple key/value store).
type SaltStore interface {
	Get(key string) (string, bool, error)
	Set(key, value string) error
}

const saltSettingKey = "secret.salt"

// scrypt cost parameters (interactive-grade); derivation happens once and is cached.
const (
	scryptN      = 1 << 15 // 32768
	scryptR      = 8
	scryptP      = 1
	keyLen       = 32 // AES-256
	saltLen      = 16
	refByteCount = 16
)

// Vault seals and opens secrets.
type Vault struct {
	passphrase string
	storage    Storage
	salts      SaltStore

	mu   sync.Mutex
	gcm  cipher.AEAD // cached after first derivation
	derr error       // cached derivation error
}

// New constructs a Vault. passphrase may be empty (see Enabled).
func New(passphrase string, storage Storage, salts SaltStore) *Vault {
	return &Vault{passphrase: passphrase, storage: storage, salts: salts}
}

// Enabled reports whether a passphrase is configured.
func (v *Vault) Enabled() bool { return v.passphrase != "" }

// aead returns the cached AES-GCM cipher, deriving the key on first use.
func (v *Vault) aead() (cipher.AEAD, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.gcm != nil || v.derr != nil {
		return v.gcm, v.derr
	}
	if v.passphrase == "" {
		v.derr = ErrNoKey
		return nil, v.derr
	}

	salt, err := v.loadOrCreateSalt()
	if err != nil {
		v.derr = err
		return nil, err
	}
	key, err := scrypt.Key([]byte(v.passphrase), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		v.derr = fmt.Errorf("derive key: %w", err)
		return nil, v.derr
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		v.derr = err
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		v.derr = err
		return nil, err
	}
	v.gcm = gcm
	return v.gcm, nil
}

func (v *Vault) loadOrCreateSalt() ([]byte, error) {
	if s, ok, err := v.salts.Get(saltSettingKey); err != nil {
		return nil, err
	} else if ok {
		return base64.StdEncoding.DecodeString(s)
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if err := v.salts.Set(saltSettingKey, base64.StdEncoding.EncodeToString(salt)); err != nil {
		return nil, err
	}
	return salt, nil
}

// Seal encrypts plaintext under a fresh ref and stores it, returning the ref.
func (v *Vault) Seal(plaintext []byte) (string, error) {
	gcm, err := v.aead()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	refBytes := make([]byte, refByteCount)
	if _, err := rand.Read(refBytes); err != nil {
		return "", err
	}
	ref := hex.EncodeToString(refBytes)

	if err := v.storage.PutSecret(ref, nonce, ciphertext); err != nil {
		return "", err
	}
	return ref, nil
}

// Open decrypts and returns the secret for ref.
func (v *Vault) Open(ref string) ([]byte, error) {
	gcm, err := v.aead()
	if err != nil {
		return nil, err
	}
	nonce, ciphertext, err := v.storage.GetSecret(ref)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret (wrong HS_SECRET_KEY?): %w", err)
	}
	return plaintext, nil
}

// Delete removes the secret for ref.
func (v *Vault) Delete(ref string) error {
	return v.storage.DeleteSecret(ref)
}
