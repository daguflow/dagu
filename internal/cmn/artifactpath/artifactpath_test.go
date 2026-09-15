// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package artifactpath_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dagucloud/dagu/v2/internal/cmn/artifactpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTime = time.Date(2026, 9, 15, 14, 32, 7, 0, time.UTC)

func TestNewRunDir(t *testing.T) {
	t.Parallel()

	t.Run("Layout", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		dir, err := artifactpath.NewRunDir(context.Background(), base, "", "daily-report", "run-1", testTime)
		require.NoError(t, err)
		require.DirExists(t, dir)

		rel, err := filepath.Rel(base, dir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("2026", "09", "15"), filepath.Dir(rel))

		parsed, ok := artifactpath.ParseRunDirName(filepath.Base(dir))
		require.True(t, ok)
		assert.Equal(t, "143207", parsed.TimeOfDay)
		assert.Equal(t, "daily-report", parsed.DAGName)
		assert.Len(t, parsed.Suffix, 6)
	})

	t.Run("LocalTimeUsesUTCDay", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		// 09:05 JST on the 16th is 00:05 UTC on the 16th; the UTC day wins.
		jst := time.FixedZone("JST", 9*60*60)
		at := time.Date(2026, 9, 16, 9, 5, 0, 0, jst)

		dir, err := artifactpath.NewRunDir(context.Background(), base, "", "tz", "run-1", at)
		require.NoError(t, err)
		assert.Equal(t, artifactpath.DayDir(base, at), filepath.Dir(dir))
		assert.Equal(t, filepath.Join(base, "2026", "09", "16"), filepath.Dir(dir))
	})

	t.Run("OverrideReplacesBase", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()
		override := t.TempDir()

		dir, err := artifactpath.NewRunDir(context.Background(), base, override, "scoped", "run-1", testTime)
		require.NoError(t, err)
		require.DirExists(t, dir)

		rel, err := filepath.Rel(override, dir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("2026", "09", "15"), filepath.Dir(rel))
	})

	// Re-deriving a run's directory must land on the same path, so that a
	// component which lost the recorded value cannot mint a second directory.
	t.Run("SameRunResolvesToSamePath", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		first, err := artifactpath.NewRunDir(context.Background(), base, "", "same", "run-1", testTime)
		require.NoError(t, err)
		second, err := artifactpath.NewRunDir(context.Background(), base, "", "same", "run-1", testTime)
		require.NoError(t, err)

		assert.Equal(t, first, second)
	})

	t.Run("DistinctRunsInSameSecondDiffer", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		first, err := artifactpath.NewRunDir(context.Background(), base, "", "same", "run-1", testTime)
		require.NoError(t, err)
		second, err := artifactpath.NewRunDir(context.Background(), base, "", "same", "run-2", testTime)
		require.NoError(t, err)

		assert.NotEqual(t, first, second)
	})

	t.Run("RejectsEmptyRunID", func(t *testing.T) {
		t.Parallel()

		_, err := artifactpath.NewRunDir(context.Background(), t.TempDir(), "", "dag", " ", testTime)
		require.Error(t, err)
	})

	// RunDir is the pure form used where a path must be derived without
	// creating anything.
	t.Run("RunDirDoesNotCreate", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		dir, err := artifactpath.RunDir(context.Background(), base, "", "dry", "run-1", testTime)
		require.NoError(t, err)
		assert.NoDirExists(t, dir)
	})

	t.Run("RejectsMissingRoot", func(t *testing.T) {
		t.Parallel()

		_, err := artifactpath.NewRunDir(context.Background(), "", "", "orphan", "run-1", testTime)
		require.EqualError(t, err, "artifact directory is not configured")
	})

	t.Run("RejectsEmptyDAGName", func(t *testing.T) {
		t.Parallel()

		_, err := artifactpath.NewRunDir(context.Background(), t.TempDir(), "", "  ", "run-1", testTime)
		require.Error(t, err)
	})
}

// An override that expands to nothing must not silently fall back to the global
// root, which would scatter a DAG's artifacts across two trees.
func TestNewRunDirRejectsOverrideExpandingToEmpty(t *testing.T) {
	t.Setenv("EMPTY_ARTIFACT_DIR", "")

	_, err := artifactpath.NewRunDir(
		context.Background(), t.TempDir(), "${EMPTY_ARTIFACT_DIR}", "scoped", "run-1", testTime)
	require.EqualError(t, err, "artifact directory is empty after expansion")
}

// MetaPath must stay in the global tree so one date walk sees every run, even
// when the run directory itself was relocated by artifacts.dir.
func TestMetaPath(t *testing.T) {
	t.Parallel()

	t.Run("SiblingOfRunDir", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		dir, err := artifactpath.NewRunDir(context.Background(), base, "", "report", "run-1", testTime)
		require.NoError(t, err)

		meta := artifactpath.MetaPath(base, dir, testTime)
		assert.Equal(t, filepath.Dir(dir), filepath.Dir(meta))
		assert.Equal(t, filepath.Base(dir)+artifactpath.MetaSuffix, filepath.Base(meta))
	})

	t.Run("OverriddenRunDirKeepsGlobalSidecar", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()
		override := t.TempDir()

		dir, err := artifactpath.NewRunDir(context.Background(), base, override, "report", "run-1", testTime)
		require.NoError(t, err)

		meta := artifactpath.MetaPath(base, dir, testTime)
		assert.Equal(t, artifactpath.DayDir(base, testTime), filepath.Dir(meta))
		assert.Equal(t, filepath.Base(dir)+artifactpath.MetaSuffix, filepath.Base(meta))
	})
}

func TestParseRunDirName(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct{ name, dagName string }{
			{"143207_daily-report_a7f3c2", "daily-report"},
			{"000000_report_000000", "report"},
			{"235959_my.dag_ffffff", "my.dag"},
			{"143207_with_underscores_abcdef", "with_underscores"},
			{"143207_con_a7f3c2", "con"},
			{"143207_trailing._0a1b2c", "trailing."},
		}
		for _, tt := range tests {
			parsed, ok := artifactpath.ParseRunDirName(tt.name)
			require.True(t, ok, tt.name)
			assert.Equal(t, tt.dagName, parsed.DAGName, tt.name)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{
			"",
			"143207",
			"14320_report_a7f3c2",          // short time
			"1432o7_report_a7f3c2",         // non-digit time
			"143207_report_a7f3k2",         // non-hex suffix
			"143207__a7f3c2",               // empty DAG name
			"143207_report-a7f3c2",         // missing suffix separator
			"dag-run_20260915_143207Z_run", // legacy layout
		} {
			_, ok := artifactpath.ParseRunDirName(name)
			assert.False(t, ok, name)
		}
	})

	t.Run("RoundTripsUnsafeName", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()

		dir, err := artifactpath.NewRunDir(context.Background(), base, "", "spaced name/slash", "run-1", testTime)
		require.NoError(t, err)
		require.DirExists(t, dir)

		parsed, ok := artifactpath.ParseRunDirName(filepath.Base(dir))
		require.True(t, ok)
		assert.Equal(t, "spaced_name_slash", parsed.DAGName)
	})
}

func TestMetaNameHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, artifactpath.IsMetaName("143207_report_a7f3c2"+artifactpath.MetaSuffix))
	assert.False(t, artifactpath.IsMetaName("143207_report_a7f3c2"))
	assert.Equal(t, "143207_report_a7f3c2",
		artifactpath.TrimMetaSuffix("143207_report_a7f3c2"+artifactpath.MetaSuffix))
}
