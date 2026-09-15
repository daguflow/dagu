// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package dagrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dagucloud/dagu/v2/internal/cmn/crypto"
)

type snapshotCodec struct {
	dataDir string
}

const snapshotPrefix = "dagu:dag-snapshot:v1:"

func (c *snapshotCodec) encode(data []byte) ([]byte, error) {
	enc, err := c.encryptor(true)
	if err != nil {
		return nil, err
	}
	ciphertext, err := enc.Encrypt(string(data))
	if err != nil {
		return nil, err
	}
	// A JSON string makes legacy readers reject encrypted snapshots.
	return json.Marshal(snapshotPrefix + ciphertext)
}

func (c *snapshotCodec) decode(data []byte) ([]byte, error) {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '{' {
		return data, nil
	}
	var envelope string
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("invalid DAG snapshot envelope: %w", err)
	}
	ciphertext, ok := strings.CutPrefix(envelope, snapshotPrefix)
	if !ok {
		return nil, fmt.Errorf("unsupported DAG snapshot format")
	}
	enc, err := c.encryptor(false)
	if err != nil {
		return nil, err
	}
	plaintext, err := enc.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	return []byte(plaintext), nil
}

func (c *snapshotCodec) encryptor(createIfMissing bool) (*crypto.Encryptor, error) {
	if c == nil || c.dataDir == "" {
		return nil, fmt.Errorf("DAG snapshot encryption is not configured")
	}
	key, err := crypto.ResolveKey(c.dataDir, createIfMissing)
	if err != nil {
		return nil, err
	}
	return crypto.NewEncryptor(key)
}
