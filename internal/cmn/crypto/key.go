// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dagucloud/dagu/v2/internal/cmn/fileutil"
)

const (
	keyFileName    = "encryption_key"
	keyDirName     = "auth"
	keyFilePerms   = os.FileMode(0600)
	keyDirPerms    = os.FileMode(0750)
	keyRandomBytes = 32
)

// ResolveKey returns DAGU_ENCRYPTION_KEY or the key in dataDir/auth/encryption_key.
// A missing key is generated only when createIfMissing is true. Existing empty
// or unreadable key files return errors and are never replaced.
func ResolveKey(dataDir string, createIfMissing bool) (string, error) {
	if key := os.Getenv("DAGU_ENCRYPTION_KEY"); key != "" {
		return key, nil
	}

	keyDir := filepath.Join(dataDir, keyDirName)
	keyPath := filepath.Join(keyDir, keyFileName)
	key, err := readKey(keyPath)
	if err == nil {
		return key, nil
	}
	if !createIfMissing || !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	rawKey := make([]byte, keyRandomBytes)
	if _, err := rand.Read(rawKey); err != nil {
		return "", fmt.Errorf("crypto: failed to generate random key: %w", err)
	}
	key = base64.StdEncoding.EncodeToString(rawKey)

	if err := os.MkdirAll(keyDir, keyDirPerms); err != nil {
		return "", fmt.Errorf("crypto: failed to create key directory: %w", err)
	}

	if err := fileutil.WriteFileAtomicExclusive(keyPath, []byte(key), keyFilePerms); err != nil {
		if errors.Is(err, os.ErrExist) {
			return readKey(keyPath)
		}
		return "", fmt.Errorf("crypto: failed to persist encryption key: %w", err)
	}

	return key, nil
}

func readKey(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from trusted dataDir
	if err != nil {
		return "", fmt.Errorf("crypto: failed to read encryption key: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("crypto: encryption key file is empty")
	}
	return string(data), nil
}
