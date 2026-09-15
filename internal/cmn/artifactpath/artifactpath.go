// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

// Package artifactpath builds and parses the on-disk layout of DAG-run
// artifact directories.
//
// Artifacts are partitioned by run date so that a date-range listing costs one
// directory read per day. A run's files live in a directory whose name carries
// the time of day and the DAG name; identity beyond that is recorded in a
// sidecar file next to it, keeping the run ID out of the path so that deeply
// nested artifact paths stay within platform path length limits.
package artifactpath

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmnvalue "github.com/dagucloud/dagu/v2/internal/cmn/value"
)

// MetaSuffix is appended to a run directory name to address its sidecar.
const MetaSuffix = ".meta"

const (
	dayLayout       = "2006/01/02"
	timeOfDayLayout = "150405"

	// suffixLen is the length of the hex suffix that disambiguates two runs of
	// the same DAG started within the same second.
	suffixLen = 6

	// maxDAGNameLen bounds the DAG name segment. It matches ir.DAGNameMaxLen,
	// duplicated here to keep this package free of domain imports.
	maxDAGNameLen = 40
)

// RunDirName is a parsed per-run artifact directory name.
type RunDirName struct {
	TimeOfDay string
	DAGName   string
	Suffix    string
}

// NewRunDir creates the artifact directory for a run started at the given time
// and returns its absolute path. overrideDir, when set, replaces baseDir as the
// root; both are expanded for environment references before use.
func NewRunDir(ctx context.Context, baseDir, overrideDir, dagName string, at time.Time) (string, error) {
	root, err := resolveRoot(ctx, baseDir, overrideDir)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dagName) == "" {
		return "", fmt.Errorf("DAG name must not be empty")
	}

	suffix, err := randomSuffix()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(DayDir(root, at), runDirName(at, dagName, suffix))
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("failed to initialize directory %s: %w", dir, err)
	}
	return dir, nil
}

// ResolveRoot expands baseDir, or overrideDir when it is set, and returns the
// artifact root without creating it.
func ResolveRoot(ctx context.Context, baseDir, overrideDir string) (string, error) {
	return resolveRoot(ctx, baseDir, overrideDir)
}

// DayDir returns the directory holding every run started on the given day.
func DayDir(root string, at time.Time) string {
	return filepath.Join(root, filepath.FromSlash(at.UTC().Format(dayLayout)))
}

// MetaPath returns the sidecar path for runDir. The sidecar always lives in the
// global tree under root, even when runDir itself sits under a DAG-level
// override, so that one date walk sees every run.
func MetaPath(root, runDir string, at time.Time) string {
	return filepath.Join(DayDir(root, at), filepath.Base(runDir)+MetaSuffix)
}

// IsMetaName reports whether name addresses a sidecar file.
func IsMetaName(name string) bool {
	return strings.HasSuffix(name, MetaSuffix)
}

// TrimMetaSuffix returns the run directory name a sidecar name refers to.
func TrimMetaSuffix(name string) string {
	return strings.TrimSuffix(name, MetaSuffix)
}

// ParseRunDirName splits a per-run artifact directory name into its parts. The
// DAG name it reports is the sanitized path segment, which is a filter hint
// only; the sidecar holds the authoritative name.
func ParseRunDirName(name string) (RunDirName, bool) {
	const minLen = len(timeOfDayLayout) + 1 + 1 + 1 + suffixLen
	if len(name) < minLen {
		return RunDirName{}, false
	}

	timeOfDay := name[:len(timeOfDayLayout)]
	if !isDigits(timeOfDay) || name[len(timeOfDayLayout)] != '_' {
		return RunDirName{}, false
	}

	suffix := name[len(name)-suffixLen:]
	if !isHex(suffix) || name[len(name)-suffixLen-1] != '_' {
		return RunDirName{}, false
	}

	dagName := name[len(timeOfDayLayout)+1 : len(name)-suffixLen-1]
	if dagName == "" {
		return RunDirName{}, false
	}
	return RunDirName{TimeOfDay: timeOfDay, DAGName: dagName, Suffix: suffix}, true
}

func resolveRoot(ctx context.Context, baseDir, overrideDir string) (string, error) {
	resolver := cmnvalue.NewResolver(cmnvalue.StaticScope{}, cmnvalue.RuntimeScope{})

	baseDir, err := resolver.String(ctx, baseDir, cmnvalue.CoordinatorArtifactBaseDirField("artifacts.base_dir"))
	if err != nil {
		return "", fmt.Errorf("failed to expand base directory: %w", err)
	}
	overrideDir, err = resolver.String(ctx, overrideDir, cmnvalue.CoordinatorArtifactBaseDirField("artifacts.dir"))
	if err != nil {
		return "", fmt.Errorf("failed to expand DAG artifact directory: %w", err)
	}

	if strings.TrimSpace(overrideDir) != "" {
		return overrideDir, nil
	}
	if strings.TrimSpace(baseDir) == "" {
		return "", fmt.Errorf("artifact directory is not set")
	}
	return baseDir, nil
}

func runDirName(at time.Time, dagName, suffix string) string {
	return at.UTC().Format(timeOfDayLayout) + "_" + safeDAGName(dagName) + "_" + suffix
}

// safeDAGName reduces a DAG name to characters that are safe in a path segment
// on every supported platform. Names accepted by DAG validation pass through
// unchanged.
func safeDAGName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-':
			b.WriteRune(r)
		default:
			// '_' separates the name from the surrounding segments, so it is
			// not reproduced here; collapsing to it keeps parsing unambiguous
			// because the boundaries are fixed-width.
			b.WriteRune('_')
		}
	}

	safe := b.String()
	if len(safe) > maxDAGNameLen {
		safe = safe[:maxDAGNameLen]
	}
	if safe == "" {
		safe = "dag"
	}
	return safe
}

func randomSuffix() (string, error) {
	b := make([]byte, suffixLen/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
