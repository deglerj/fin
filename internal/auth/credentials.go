package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/hkdf"
)

type Credentials struct {
	ServerURL   string `json:"server_url"`
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
}

func deriveKey(provider MachineIDProvider) ([]byte, error) {
	id, err := provider.MachineID()
	if err != nil {
		return nil, err
	}
	r := hkdf.New(sha256.New, []byte(id), []byte("fin-creds-v1"), nil)
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

func Save(creds Credentials, path string, provider MachineIDProvider) error {
	key, err := deriveKey(provider)
	if err != nil {
		return err
	}
	plain, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	enc, err := encrypt(key, plain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0600)
}

func LoadCreds(path string, provider MachineIDProvider) (*Credentials, error) {
	key, err := deriveKey(provider)
	if err != nil {
		return nil, err
	}
	enc, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := decrypt(key, enc)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(plain, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
