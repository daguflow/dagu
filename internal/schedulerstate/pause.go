// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package schedulerstate

import (
	"context"
	"time"
)

// Pause is the cluster-wide scheduler pause state.
//
// While paused, the scheduler creates no scheduler-managed runs. Manual,
// webhook, and sub-DAG runs are unaffected, matching per-DAG suspension.
type Pause struct {
	Paused   bool
	PausedAt time.Time
	PausedBy string
	Reason   string
}

// PauseStore persists the scheduler pause state. Implementations must be safe
// for concurrent use. Get returns an independent snapshot that callers may
// mutate.
//
// A store that has never been written reports the zero value, which is not
// paused.
type PauseStore interface {
	// IsPaused reports whether scheduler-managed dispatch is currently paused.
	IsPaused(ctx context.Context) (bool, error)
	// Get returns the full pause state, including who paused and why.
	Get(ctx context.Context) (Pause, error)
	// Set records the pause state. Clearing the pause discards the recorded
	// actor and reason.
	Set(ctx context.Context, paused bool, actor, reason string) error
}
