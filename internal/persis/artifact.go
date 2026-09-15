// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package persis

import (
	"context"
	"errors"
	"time"

	"github.com/dagucloud/dagu/v2/internal/workspace"
)

// ErrInvalidArtifactCursor reports a cursor that does not belong to the query
// it was presented with.
var ErrInvalidArtifactCursor = errors.New("invalid artifact cursor")

// ArtifactStore lists files produced by DAG runs, newest run first.
type ArtifactStore interface {
	QueryArtifacts(ctx context.Context, query ArtifactQuery) (ArtifactPage, error)
}

// ArtifactQuery selects a page of artifact files.
type ArtifactQuery struct {
	// Name matches DAG names containing it, case-insensitively.
	Name string

	From TimeInUTC
	To   TimeInUTC

	WorkspaceFilter *workspace.WorkspaceFilter

	Limit  int
	Cursor string
}

// ArtifactPage is one forward-only page of artifact files.
type ArtifactPage struct {
	Items      []ArtifactFile
	NextCursor string
}

// ArtifactFile is a single file produced by a DAG run.
type ArtifactFile struct {
	Name      string
	DAGRunID  string
	StartedAt time.Time

	// Path is relative to the run's artifact directory and uses forward
	// slashes on every platform.
	Path string
	Size int64
}
