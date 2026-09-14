// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package persis

import "github.com/dagucloud/dagu/v2/internal/ir"

// DAGChangeType identifies the kind of DAG lifecycle event.
type DAGChangeType int

const (
	DAGChangeAdded DAGChangeType = iota
	DAGChangeUpdated
	DAGChangeDeleted
)

// DAGChangeEvent represents a DAG lifecycle event for a stored DAG definition.
type DAGChangeEvent struct {
	DAGEntry
	Type DAGChangeType
}

// DAGEntry pairs a DAG definition with its stable persistence identity.
type DAGEntry struct {
	DefinitionID string
	DAG          *ir.DAG
}
