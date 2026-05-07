package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const encPrefix = "enc:v1:"

var (
	secretKeys    [][]byte
	cryptoEnabled bool
)

func initSecrets(keyPath string) error {
	keyPath = strings.TrimSpace(keyPath)
	if keyPath == "" {
		secretKeys = nil
		cryptoEnabled = false
		return nil
	}

	keys, err := loadOrCreateSecretKeys(keyPath)
	if err != nil {
		return err
	}
	secretKeys = keys
	cryptoEnabled = true
	return nil
}

func loadOrCreateSecretKeys(path string) ([][]byte, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("secrets key dir: %w", err)
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		k, err := generateSecretKey()
		if err != nil {
			return nil, fmt.Errorf("generate secrets key: %w", err)
		}
		if err := writeSecretKeys(path, [][]byte{k}, 0600); err != nil {
			return nil, fmt.Errorf("write secrets key: %w", err)
		}
		return [][]byte{k}, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat secrets key: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read secrets key: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	keys := make([][]byte, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			return nil, fmt.Errorf("decode secrets key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("invalid secrets key length: expected 32 bytes")
		}
		keys = append(keys, decoded)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid secrets keys found in %s", path)
	}
	return keys, nil
}

func encryptAtRest(plain string) (string, error) {
	if plain == "" || !cryptoEnabled {
		return plain, nil
	}
	return encryptAtRestWithKey(plain, secretKeys[0])
}

func encryptAtRestWithKey(plain string, key []byte) (string, error) {
	if plain == "" {
		return plain, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm init: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	combined := append(nonce, ciphertext...)
	return encPrefix + base64.StdEncoding.EncodeToString(combined), nil
}

func decryptAtRest(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, encPrefix) {
		return stored, nil
	}
	if !cryptoEnabled {
		return "", errors.New("encrypted value present but secrets key is not configured")
	}

	payload := strings.TrimPrefix(stored, encPrefix)
	combined, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("decode encrypted value: %w", err)
	}
	for _, key := range secretKeys {
		plain, decErr := decryptAtRestWithKey(combined, key)
		if decErr == nil {
			return plain, nil
		}
	}
	return "", errors.New("decrypt value: no configured key could decrypt payload")
}

func decryptAtRestWithKey(combined, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm init: %w", err)
	}

	ns := gcm.NonceSize()
	if len(combined) < ns {
		return "", errors.New("invalid encrypted payload")
	}
	nonce := combined[:ns]
	ciphertext := combined[ns:]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plain), nil
}

func generateSecretKey() ([]byte, error) {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}

func writeSecretKeys(path string, keys [][]byte, mode os.FileMode) error {
	encoded := make([]string, 0, len(keys))
	for _, key := range keys {
		encoded = append(encoded, base64.StdEncoding.EncodeToString(key))
	}
	return os.WriteFile(path, []byte(strings.Join(encoded, "\n")+"\n"), mode)
}

func writeSecretKeysAtomic(path string, keys [][]byte, mode os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := writeSecretKeys(tmpPath, keys, mode); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func SetSecretSetting(key, value string) error {
	enc, err := encryptAtRest(value)
	if err != nil {
		return err
	}
	return SetSetting(key, enc)
}

func GetSecretSetting(key, defaultVal string) string {
	stored := GetSetting(key, "")
	if stored == "" {
		return defaultVal
	}
	plain, err := decryptAtRest(stored)
	if err != nil {
		return defaultVal
	}
	return plain
}
