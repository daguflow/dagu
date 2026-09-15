// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package artifact_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dagucloud/dagu/v2/internal/cmn/artifactpath"
	"github.com/dagucloud/dagu/v2/internal/cmn/stringutil"
	"github.com/dagucloud/dagu/v2/internal/ir"
	"github.com/dagucloud/dagu/v2/internal/persis"
	"github.com/dagucloud/dagu/v2/internal/persis/file/artifact"
	"github.com/dagucloud/dagu/v2/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type storeFixture struct {
	root  string
	store *artifact.Store
}

func newStoreFixture(t *testing.T) storeFixture {
	t.Helper()

	root := t.TempDir()
	return storeFixture{root: root, store: artifact.NewStore(root)}
}

// index writes one run's artifacts plus its index record, the way a finished
// run leaves them behind.
func (f storeFixture) index(t *testing.T, dagName, dagRunID string, at time.Time, labels []string, files ...string) string {
	t.Helper()

	dir, err := artifactpath.NewRunDir(context.Background(), f.root, "", dagName, dagRunID, at)
	require.NoError(t, err)
	for _, name := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(name), 0o600))
	}

	metaPath, ok := artifactpath.MetaPath(f.root, dir)
	require.True(t, ok)
	require.NoError(t, artifact.WriteRecord(metaPath, artifact.Record{
		Version:   artifact.RecordVersion,
		Name:      dagName,
		DAGRunID:  dagRunID,
		Status:    ir.Succeeded,
		StartedAt: stringutil.FormatTime(at),
		Labels:    labels,
		Dir:       dir,
	}))
	return dir
}

func (f storeFixture) query(t *testing.T, q persis.ArtifactQuery) persis.ArtifactPage {
	t.Helper()

	if q.Limit == 0 {
		q.Limit = 100
	}
	page, err := f.store.QueryArtifacts(context.Background(), q)
	require.NoError(t, err)
	return page
}

func runIDs(page persis.ArtifactPage) []string {
	ids := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.DAGRunID)
	}
	return ids
}

var (
	day1  = time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	day2  = time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	day2b = time.Date(2026, 9, 15, 18, 0, 0, 0, time.UTC)
)

func TestQueryArtifacts(t *testing.T) {
	t.Run("NewestRunFirstAcrossDays", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "alpha", "run-old", day1, nil, "a.txt")
		f.index(t, "beta", "run-mid", day2, nil, "b.txt")
		f.index(t, "gamma", "run-new", day2b, nil, "c.txt")

		page := f.query(t, persis.ArtifactQuery{})

		assert.Equal(t, []string{"run-new", "run-mid", "run-old"}, runIDs(page))
	})

	t.Run("ReturnsFilesWithRunIdentity", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "alpha", "run-1", day2, nil, "reports/summary.md", "top.txt")

		page := f.query(t, persis.ArtifactQuery{})

		require.Len(t, page.Items, 2)
		assert.Equal(t, "alpha", page.Items[0].Name)
		assert.Equal(t, "run-1", page.Items[0].DAGRunID)
		assert.Equal(t, day2, page.Items[0].StartedAt.UTC())
		assert.Equal(t, "reports/summary.md", page.Items[0].Path)
		assert.Equal(t, int64(len("reports/summary.md")), page.Items[0].Size)
		assert.Equal(t, "top.txt", page.Items[1].Path)
	})

	t.Run("FiltersByDAGName", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "daily-report", "run-1", day2, nil, "a.txt")
		f.index(t, "nightly-sync", "run-2", day2b, nil, "b.txt")

		page := f.query(t, persis.ArtifactQuery{Name: "REPORT"})

		assert.Equal(t, []string{"run-1"}, runIDs(page))
	})

	t.Run("BoundsByDateRange", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "alpha", "run-old", day1, nil, "a.txt")
		f.index(t, "alpha", "run-new", day2, nil, "b.txt")

		page := f.query(t, persis.ArtifactQuery{From: persis.NewUTC(day2.Add(-time.Hour))})

		assert.Equal(t, []string{"run-new"}, runIDs(page))
	})

	// A DAG relocated by artifacts.dir is indexed in the global tree and must
	// still be listed, with its files read from where they actually live.
	t.Run("ListsRelocatedArtifacts", func(t *testing.T) {
		f := newStoreFixture(t)
		elsewhere := t.TempDir()

		dir, err := artifactpath.NewRunDir(context.Background(), elsewhere, "", "scoped", "run-1", day2)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "out.txt"), []byte("x"), 0o600))
		metaPath, ok := artifactpath.MetaPath(f.root, dir)
		require.True(t, ok)
		require.NoError(t, artifact.WriteRecord(metaPath, artifact.Record{
			Version: artifact.RecordVersion, Name: "scoped", DAGRunID: "run-1",
			StartedAt: stringutil.FormatTime(day2), Dir: dir,
		}))

		page := f.query(t, persis.ArtifactQuery{})

		require.Len(t, page.Items, 1)
		assert.Equal(t, "out.txt", page.Items[0].Path)
	})

	t.Run("SkipsUnreadableRecord", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "alpha", "run-1", day2, nil, "a.txt")
		dir := f.index(t, "beta", "run-2", day2b, nil, "b.txt")
		metaPath, ok := artifactpath.MetaPath(f.root, dir)
		require.True(t, ok)
		require.NoError(t, os.WriteFile(metaPath, []byte("{not json"), 0o600))

		page := f.query(t, persis.ArtifactQuery{})

		assert.Equal(t, []string{"run-1"}, runIDs(page))
	})

	t.Run("IgnoresRunDirsWithoutRecord", func(t *testing.T) {
		f := newStoreFixture(t)
		dir, err := artifactpath.NewRunDir(context.Background(), f.root, "", "alpha", "run-1", day2)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o600))

		page := f.query(t, persis.ArtifactQuery{})

		assert.Empty(t, page.Items)
	})
}

func TestQueryArtifactsWorkspaceScoping(t *testing.T) {
	f := newStoreFixture(t)
	f.index(t, "team-a", "run-a", day2, []string{"workspace=alpha"}, "a.txt")
	f.index(t, "team-b", "run-b", day2b, []string{"workspace=beta"}, "b.txt")
	f.index(t, "shared", "run-c", day1, nil, "c.txt")

	t.Run("OnlyMatchingWorkspace", func(t *testing.T) {
		page := f.query(t, persis.ArtifactQuery{
			WorkspaceFilter: &workspace.WorkspaceFilter{Enabled: true, Workspaces: []string{"alpha"}},
		})
		assert.Equal(t, []string{"run-a"}, runIDs(page))
	})

	t.Run("IncludingUnlabelled", func(t *testing.T) {
		page := f.query(t, persis.ArtifactQuery{
			WorkspaceFilter: &workspace.WorkspaceFilter{
				Enabled: true, Workspaces: []string{"alpha"}, IncludeUnlabelled: true,
			},
		})
		assert.Equal(t, []string{"run-a", "run-c"}, runIDs(page))
	})

	t.Run("DisabledFilterSeesAll", func(t *testing.T) {
		page := f.query(t, persis.ArtifactQuery{})
		assert.Len(t, page.Items, 3)
	})
}

func TestQueryArtifactsPagination(t *testing.T) {
	// Paging must not repeat or drop an entry, including where a page boundary
	// falls inside a run and where it falls between days.
	t.Run("WalksEveryFileExactlyOnce", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "alpha", "run-old", day1, nil, "a1.txt", "a2.txt")
		f.index(t, "beta", "run-mid", day2, nil, "b1.txt", "b2.txt", "b3.txt")
		f.index(t, "gamma", "run-new", day2b, nil, "c1.txt")

		for _, limit := range []int{1, 2, 3, 5} {
			var seen []string
			query := persis.ArtifactQuery{Limit: limit}
			for {
				page := f.query(t, query)
				for _, item := range page.Items {
					seen = append(seen, item.DAGRunID+"/"+item.Path)
				}
				if page.NextCursor == "" {
					break
				}
				query.Cursor = page.NextCursor
			}
			assert.Equal(t, []string{
				"run-new/c1.txt",
				"run-mid/b1.txt", "run-mid/b2.txt", "run-mid/b3.txt",
				"run-old/a1.txt", "run-old/a2.txt",
			}, seen, "limit %d", limit)
		}
	})

	t.Run("RejectsCursorFromDifferentFilters", func(t *testing.T) {
		f := newStoreFixture(t)
		f.index(t, "alpha", "run-1", day2, nil, "a1.txt", "a2.txt")

		page := f.query(t, persis.ArtifactQuery{Limit: 1})
		require.NotEmpty(t, page.NextCursor)

		_, err := f.store.QueryArtifacts(context.Background(), persis.ArtifactQuery{
			Limit: 1, Name: "alpha", Cursor: page.NextCursor,
		})
		assert.ErrorIs(t, err, persis.ErrInvalidArtifactCursor)
	})

	t.Run("RejectsMalformedCursor", func(t *testing.T) {
		f := newStoreFixture(t)

		_, err := f.store.QueryArtifacts(context.Background(), persis.ArtifactQuery{Cursor: "!!!"})
		assert.ErrorIs(t, err, persis.ErrInvalidArtifactCursor)
	})
}
