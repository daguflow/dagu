// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	openapiv1 "github.com/dagucloud/dagu/v2/api/v1"
	"github.com/dagucloud/dagu/v2/internal/cmn/artifactpath"
	"github.com/dagucloud/dagu/v2/internal/cmn/stringutil"
	"github.com/dagucloud/dagu/v2/internal/persis"
	fileartifact "github.com/dagucloud/dagu/v2/internal/persis/file/artifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var artifactTestStart = time.Date(2026, 9, 15, 14, 32, 7, 0, time.UTC)

// newArtifactListAPI builds an API over a populated artifact tree.
func newArtifactListAPI(t *testing.T, runs ...string) *API {
	t.Helper()

	root := t.TempDir()
	for i, dagRunID := range runs {
		at := artifactTestStart.Add(time.Duration(i) * time.Minute)
		dir, err := artifactpath.NewRunDir(context.Background(), root, "", "reporter", dagRunID, at)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "out.txt"), []byte("hello"), 0o600))

		metaPath, ok := artifactpath.MetaPath(root, dir)
		require.True(t, ok)
		require.NoError(t, fileartifact.WriteRecord(metaPath, fileartifact.Record{
			Version:   fileartifact.RecordVersion,
			Name:      "reporter",
			DAGRunID:  dagRunID,
			StartedAt: stringutil.FormatTime(at),
			Dir:       dir,
		}))
	}

	return &API{artifactRepository: persis.NewArtifactRepository(fileartifact.NewStore(root))}
}

func listArtifacts(t *testing.T, a *API, params openapiv1.ListArtifactsParams) openapiv1.ArtifactListResponse {
	t.Helper()

	resp, err := a.ListArtifacts(context.Background(), openapiv1.ListArtifactsRequestObject{Params: params})
	require.NoError(t, err)
	body, ok := resp.(openapiv1.ListArtifacts200JSONResponse)
	require.True(t, ok, "expected a 200 response, got %T", resp)
	return openapiv1.ArtifactListResponse(body)
}

func TestListArtifacts(t *testing.T) {
	t.Run("NewestRunFirst", func(t *testing.T) {
		a := newArtifactListAPI(t, "run-1", "run-2")

		body := listArtifacts(t, a, openapiv1.ListArtifactsParams{})

		require.Len(t, body.Items, 2)
		assert.Equal(t, "run-2", body.Items[0].DagRunId)
		assert.Equal(t, "reporter", body.Items[0].Name)
		assert.Equal(t, "out.txt", body.Items[0].Path)
		assert.Equal(t, int64(len("hello")), body.Items[0].Size)
		require.NotNil(t, body.Items[0].StartedAt)
		assert.Nil(t, body.NextCursor)
	})

	t.Run("PagesWithCursor", func(t *testing.T) {
		a := newArtifactListAPI(t, "run-1", "run-2")
		limit := 1

		first := listArtifacts(t, a, openapiv1.ListArtifactsParams{Limit: &limit})
		require.Len(t, first.Items, 1)
		require.NotNil(t, first.NextCursor)

		second := listArtifacts(t, a, openapiv1.ListArtifactsParams{Limit: &limit, Cursor: first.NextCursor})
		require.Len(t, second.Items, 1)
		assert.NotEqual(t, first.Items[0].DagRunId, second.Items[0].DagRunId)
	})

	t.Run("RejectsMalformedCursor", func(t *testing.T) {
		a := newArtifactListAPI(t, "run-1")
		cursor := "!!!"

		_, err := a.ListArtifacts(context.Background(),
			openapiv1.ListArtifactsRequestObject{Params: openapiv1.ListArtifactsParams{Cursor: &cursor}})

		var apiErr *Error
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	// A deployment that never enabled artifacts has no repository wired.
	t.Run("EmptyWithoutRepository", func(t *testing.T) {
		body := listArtifacts(t, &API{}, openapiv1.ListArtifactsParams{})

		assert.Empty(t, body.Items)
	})
}
