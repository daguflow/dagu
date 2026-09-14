// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package scheduler

import "context"

// EntryReader is responsible for managing DAG definitions and watching for changes.
type EntryReader interface {
	// Init initializes the DAG registry by loading the current DAG definitions.
	// This must be called before Start.
	Init(ctx context.Context) error
	// Start starts observing DAG definition changes.
	// This method blocks until Stop is called or context is canceled.
	Start(ctx context.Context)
	// Stop stops observing changes.
	Stop()
	// Entries returns a snapshot of all currently loaded DAG definitions.
	Entries() []DAGEntry
	// Events returns lifecycle changes after initialization.
	Events() <-chan DAGChangeEvent
}
