// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveKey(t *testing.T) {
	t.Setenv("DAGU_ENCRYPTION_KEY", "")

	t.Run("Existing", func(t *testing.T) {
		dir := t.TempDir()
		key, err := ResolveKey(dir, true)
		require.NoError(t, err)
		require.NotEmpty(t, key)
		got, err := ResolveKey(dir, false)
		require.NoError(t, err)
		require.Equal(t, key, got)
	})

	t.Run("Missing", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ResolveKey(dir, false)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.NoFileExists(t, filepath.Join(dir, keyDirName, keyFileName))
	})

	t.Run("Empty", func(t *testing.T) {
		dir := t.TempDir()
		keyPath := filepath.Join(dir, keyDirName, keyFileName)
		require.NoError(t, os.MkdirAll(filepath.Dir(keyPath), keyDirPerms))
		require.NoError(t, os.WriteFile(keyPath, nil, keyFilePerms))

		_, err := ResolveKey(dir, true)
		require.Error(t, err)
		data, err := os.ReadFile(keyPath)
		require.NoError(t, err)
		require.Empty(t, data)
	})

	t.Run("Unreadable", func(t *testing.T) {
		dir := t.TempDir()
		keyPath := filepath.Join(dir, keyDirName, keyFileName)
		// A directory produces a read error even when tests run as root.
		require.NoError(t, os.MkdirAll(keyPath, keyDirPerms))
		_, err := ResolveKey(dir, true)
		require.Error(t, err)
		require.DirExists(t, keyPath)
	})

	t.Run("Concurrent", func(t *testing.T) {
		dir := t.TempDir()
		type result struct {
			key string
			err error
		}
		const clients = 32
		results := make(chan result, clients)
		start := make(chan struct{})
		for range clients {
			go func() {
				<-start
				key, err := ResolveKey(dir, true)
				results <- result{key: key, err: err}
			}()
		}
		close(start)
		keys := make([]string, 0, clients)
		for range clients {
			got := <-results
			require.NoError(t, got.err)
			keys = append(keys, got.key)
		}
		persisted, err := ResolveKey(dir, true)
		require.NoError(t, err)
		for _, key := range keys {
			require.Equal(t, persisted, key)
		}
	})

	t.Run("Environment", func(t *testing.T) {
		t.Setenv("DAGU_ENCRYPTION_KEY", "environment-key")
		key, err := ResolveKey(t.TempDir(), false)
		require.NoError(t, err)
		require.Equal(t, "environment-key", key)
	})
}
