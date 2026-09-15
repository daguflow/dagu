// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package persis

import "context"

const (
	defaultArtifactListLimit = 100
	maxArtifactListLimit     = 500
)

// ArtifactRepository provides application-level access to indexed DAG-run
// artifacts.
type ArtifactRepository struct {
	store ArtifactStore
}

// NewArtifactRepository creates a repository backed by store.
func NewArtifactRepository(store ArtifactStore) *ArtifactRepository {
	return &ArtifactRepository{store: store}
}

// List returns one page of artifact files, newest run first.
func (r *ArtifactRepository) List(ctx context.Context, query ArtifactQuery) (ArtifactPage, error) {
	if r == nil || r.store == nil {
		return ArtifactPage{}, nil
	}
	switch {
	case query.Limit <= 0:
		query.Limit = defaultArtifactListLimit
	case query.Limit > maxArtifactListLimit:
		query.Limit = maxArtifactListLimit
	}
	return r.store.QueryArtifacts(ctx, query)
}
