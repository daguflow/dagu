// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package filenotify

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRemoveStopsWatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "base.yaml")
	require.NoError(t, os.WriteFile(path, []byte("queue: test\n"), 0600))
	item, err := newItemToWatch(path)
	require.NoError(t, err)
	watcher := NewPollingWatcher(time.Hour).(*filePoller)
	stop := make(chan struct{})
	watcher.watches = map[string]chan struct{}{path: stop}
	done := make(chan struct{})
	go func() { defer close(done); watcher.watch(item, stop) }()
	t.Cleanup(func() { require.NoError(t, watcher.Close()); <-done })
	require.NoError(t, watcher.Remove(path))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("removed watch kept running")
	}
	// A replacement directory must be able to establish a fresh watch.
	require.NoError(t, watcher.Add(path))
}
