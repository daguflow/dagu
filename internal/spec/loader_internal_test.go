// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package spec

import (
	"errors"
	"github.com/dagucloud/dagu/v2/internal/cmn/fileutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExpandHomeDir verifies the loader expands only the current user's home shorthand.
func TestExpandHomeDir(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	expanded, err := expandHomeDir("~")
	require.NoError(t, err)
	assert.Equal(t, homeDir, expanded)

	expanded, err = expandHomeDir("~/dags/test.yaml")
	require.NoError(t, err)
	assert.Equal(
		t,
		filepath.Clean(filepath.Join(homeDir, "dags", "test.yaml")),
		filepath.Clean(expanded),
	)

	expanded, err = expandHomeDir("~alice/dags/test.yaml")
	require.NoError(t, err)
	assert.Equal(t, "~alice/dags/test.yaml", expanded)
}

// TestUnmarshalData verifies manifest decoding handles empty and malformed YAML inputs.
func TestUnmarshalData(t *testing.T) {
	t.Parallel()

	t.Run("EmptyDocument", func(t *testing.T) {
		t.Parallel()

		data, err := unmarshalData(nil)
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("RejectsMalformedYAML", func(t *testing.T) {
		t.Parallel()

		_, err := unmarshalData([]byte("steps: ["))
		require.Error(t, err)
	})
}

// TestDecode verifies manifest decoding preserves raw fields and surfaces validation errors.
func TestDecode(t *testing.T) {
	t.Parallel()

	t.Run("RejectsLabelsAndTagsTogether", func(t *testing.T) {
		t.Parallel()

		_, err := decode(map[string]any{
			"labels": map[string]any{"team": "core"},
			"tags":   []any{"legacy"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "labels and deprecated tags cannot both be set")
	})

	t.Run("CapturesRawHandlerOnAndDefaults", func(t *testing.T) {
		t.Parallel()

		manifest, err := decode(map[string]any{
			"steps": []any{
				map[string]any{
					"name":    "step-1",
					"command": "echo hello",
				},
			},
			"handler_on": map[string]any{
				"failure": map[string]any{
					"command": "echo fail",
				},
			},
			"defaults": map[string]any{
				"shell": "bash",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, manifest)
		require.Contains(t, manifest.handlerOnRaw, "failure")
		require.Equal(t, "echo fail", manifest.handlerOnRaw["failure"]["command"])
		require.Equal(t, "bash", manifest.defaultsRaw["shell"])
	})

	t.Run("ReportsUnknownKeys", func(t *testing.T) {
		t.Parallel()

		_, err := decode(map[string]any{
			"steps": []any{
				map[string]any{
					"name":    "step-1",
					"command": "echo hello",
				},
			},
			"unknown_key": true,
		})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrNameOrPathRequired))
		assert.Contains(t, err.Error(), "unknown_key")
	})

	t.Run("SuggestsSnakeCaseForWebhookForwardHeaders", func(t *testing.T) {
		t.Parallel()

		_, err := decode(map[string]any{
			"steps": []any{
				map[string]any{
					"name":    "step-1",
					"command": "echo hello",
				},
			},
			"webhook": map[string]any{
				"forwardHeaders": []any{"X-GitHub-Event"},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "forwardHeaders -> webhook.forward_headers")
	})
}

// TestNewManifestDecoderSharesInstance verifies callers reuse the shared decoder instance.
func TestNewManifestDecoderSharesInstance(t *testing.T) {
	t.Parallel()

	first := newManifestDecoder()
	second := newManifestDecoder()

	require.Same(t, first, second)
}

func TestBaseCache(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "base.yaml")
	cache := baseConfigCache{files: fileutil.NewCache[*baseConfigSource]("test", 2, 0)}
	var reads atomic.Int32
	read := func(path string) ([]byte, error) {
		reads.Add(1)
		return os.ReadFile(path)
	}
	stamp := time.Unix(1_700_000_000, 0)
	write := func(text string) {
		require.NoError(t, os.WriteFile(path, []byte(text), 0600))
		stamp = stamp.Add(-time.Second)
		require.NoError(t, os.Chtimes(path, stamp, stamp))
	}
	write("queue: old\n")
	ready, start := make(chan struct{}, 2), make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			ready <- struct{}{}
			<-start
			_, err := cache.load(path, read)
			results <- err
		}()
	}
	<-ready
	<-ready
	close(start)
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	for range 2 {
		source, err := cache.load(path, read)
		require.NoError(t, err)
		base, err := source.decode("base config")
		require.NoError(t, err)
		dags, err := loadDAGsFromData(loadBuildContext(t.Context(), OnlyMetadata()), []byte("name: child\n"), "", base)
		require.NoError(t, err)
		require.Equal(t, "old", dags[0].ProcGroup())
		// A caller may mutate its DAG, including the base payload sent to workers.
		dags[0].BaseConfigData[0] = '!'
		*base.definition.Queue = "mutated"
	}
	require.EqualValues(t, 1, reads.Load())
	write("queue: new\n")
	source, err := cache.load(path, read)
	require.NoError(t, err)
	require.Equal(t, "new", source.values["queue"])
	require.EqualValues(t, 2, reads.Load())
	write("queue: [")
	for range 2 {
		source, err = cache.load(path, read)
		require.NoError(t, err)
		_, err = source.decode("base config")
		require.Error(t, err)
	}
	require.EqualValues(t, 3, reads.Load())
	require.NoError(t, os.Remove(path))
	_, err = cache.load(path, read)
	require.ErrorIs(t, err, os.ErrNotExist)
	write("queue: restored\n")
	source, err = cache.load(path, read)
	require.NoError(t, err)
	require.Equal(t, "restored", source.values["queue"])
	require.EqualValues(t, 4, reads.Load())
}

func TestBaseCacheConcurrentUpdate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "base.yaml")
	require.NoError(t, os.WriteFile(path, []byte("queue: old\n"), 0600))
	cache := baseConfigCache{files: fileutil.NewCache[*baseConfigSource]("test", 2, 0)}
	started, release := make(chan struct{}), make(chan struct{})
	var reads atomic.Int32
	read := func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		if reads.Add(1) == 1 {
			close(started)
			<-release
		}
		return data, err
	}
	oldDone := make(chan error, 1)
	t.Cleanup(func() { close(release); require.NoError(t, <-oldDone) })
	go func() {
		_, err := cache.load(path, read)
		oldDone <- err
	}()
	<-started
	// This update must not join the read of the previous file contents.
	require.NoError(t, os.WriteFile(path, []byte("queue: updated\n"), 0600))
	result := make(chan *baseConfigSource, 1)
	go func() {
		source, _ := cache.load(path, read)
		result <- source
	}()
	select {
	case source := <-result:
		require.NotNil(t, source)
		require.Equal(t, "updated", source.values["queue"])
	case <-time.After(time.Second):
		t.Fatal("updated base was blocked by an earlier file revision")
	}
}
