// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package store_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dagucloud/dagu/v2/internal/persis"
	"github.com/dagucloud/dagu/v2/internal/persis/file"
	"github.com/dagucloud/dagu/v2/internal/persis/store"
	"github.com/dagucloud/dagu/v2/internal/persis/testutil"
	"github.com/dagucloud/dagu/v2/internal/schedulerstate"
)

func newSchedulerPauseStore() schedulerstate.PauseStore {
	return store.NewSchedulerPauseStore(testutil.NewMemoryBackend().Collection("scheduler"))
}

func TestSchedulerPauseDefaultsToRunning(t *testing.T) {
	ctx := context.Background()
	s := newSchedulerPauseStore()

	paused, err := s.IsPaused(ctx)
	require.NoError(t, err)
	assert.False(t, paused)

	pause, err := s.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, schedulerstate.Pause{}, pause)
}

func TestSchedulerPauseRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newSchedulerPauseStore()

	require.NoError(t, s.Set(ctx, true, "alice", "db migration"))

	pause, err := s.Get(ctx)
	require.NoError(t, err)
	assert.True(t, pause.Paused)
	assert.Equal(t, "alice", pause.PausedBy)
	assert.Equal(t, "db migration", pause.Reason)
	assert.False(t, pause.PausedAt.IsZero())
}

func TestSchedulerPauseResumeClearsActorAndReason(t *testing.T) {
	ctx := context.Background()
	s := newSchedulerPauseStore()

	require.NoError(t, s.Set(ctx, true, "alice", "db migration"))
	require.NoError(t, s.Set(ctx, false, "bob", "ignored"))

	pause, err := s.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, schedulerstate.Pause{}, pause)
}

// A pause written by another process must be observed without restarting the
// reader, because the scheduler and the API run as separate processes.
func TestSchedulerPauseObservesExternalWrite(t *testing.T) {
	ctx := context.Background()
	col := testutil.NewMemoryBackend().Collection("scheduler")
	reader := store.NewSchedulerPauseStore(col)
	writer := store.NewSchedulerPauseStore(col)

	paused, err := reader.IsPaused(ctx)
	require.NoError(t, err)
	require.False(t, paused)

	require.NoError(t, writer.Set(ctx, true, "alice", "maintenance"))

	paused, err = reader.IsPaused(ctx)
	require.NoError(t, err)
	assert.True(t, paused)

	require.NoError(t, writer.Set(ctx, false, "", ""))

	paused, err = reader.IsPaused(ctx)
	require.NoError(t, err)
	assert.False(t, paused)
}

func TestSchedulerPauseCorruptRecordReportsRunning(t *testing.T) {
	ctx := context.Background()
	col := testutil.NewMemoryBackend().Collection("scheduler")
	s := store.NewSchedulerPauseStore(col)

	require.NoError(t, col.Put(ctx, &persis.Record{ID: "paused", Data: []byte("{not json")}))

	paused, err := s.IsPaused(ctx)
	require.NoError(t, err)
	assert.False(t, paused)
}

func TestSchedulerPauseFileLayout(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	col := file.NewCollection(filepath.Join(root, "scheduler"), file.WithIndentedJSON())
	s := store.NewSchedulerPauseStore(col)

	require.NoError(t, s.Set(ctx, true, "alice", "db migration"))

	raw, err := os.ReadFile(filepath.Join(root, "scheduler", "paused.json"))
	require.NoError(t, err)
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &body))
	assert.Contains(t, body, "paused")
	assert.Contains(t, body, "pausedBy")
	assert.Contains(t, body, "reason")
}

func TestSchedulerPauseConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := newSchedulerPauseStore()

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			if i%2 == 0 {
				require.NoError(t, s.Set(ctx, true, "alice", "maintenance"))
				return
			}
			_, err := s.IsPaused(ctx)
			require.NoError(t, err)
		})
	}
	wg.Wait()
}
