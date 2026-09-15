// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package artifact

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dagucloud/dagu/v2/internal/cmn/artifactpath"
	"github.com/dagucloud/dagu/v2/internal/cmn/fileutil"
	"github.com/dagucloud/dagu/v2/internal/cmn/logger"
	"github.com/dagucloud/dagu/v2/internal/cmn/logger/tag"
	"github.com/dagucloud/dagu/v2/internal/cmn/stringutil"
	"github.com/dagucloud/dagu/v2/internal/ir"
	"github.com/dagucloud/dagu/v2/internal/persis"
)

var _ persis.ArtifactStore = (*Store)(nil)

// Store lists DAG-run artifacts from the date-partitioned artifact tree.
//
// The tree holds every DAG, so a descending walk of its day directories is
// already newest-first and needs no merge across DAGs. Each day costs one
// directory read; a record is opened only for an entry that survives the
// filters that the directory name alone can decide.
type Store struct {
	rootDir string
	cache   *fileutil.Cache[*Record]
}

// StoreOption configures artifact listing.
type StoreOption func(*Store)

// WithRecordCache reuses decoded index records across queries.
func WithRecordCache(cache *fileutil.Cache[*Record]) StoreOption {
	return func(s *Store) {
		s.cache = cache
	}
}

// NewStore creates a listing store over the artifact root directory.
func NewStore(rootDir string, opts ...StoreOption) *Store {
	s := &Store{rootDir: rootDir}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// QueryArtifacts implements persis.ArtifactStore.
func (s *Store) QueryArtifacts(ctx context.Context, query persis.ArtifactQuery) (persis.ArtifactPage, error) {
	resume, err := decodeCursor(query)
	if err != nil {
		return persis.ArtifactPage{}, err
	}

	days, err := s.listDaysDesc(query)
	if err != nil {
		return persis.ArtifactPage{}, err
	}

	page := persis.ArtifactPage{}
	for _, day := range days {
		if err := ctx.Err(); err != nil {
			return persis.ArtifactPage{}, err
		}
		if resume != nil && day > resume.Day {
			continue
		}

		done, err := s.collectDay(ctx, query, resume, day, &page)
		if err != nil {
			return persis.ArtifactPage{}, err
		}
		if done {
			return page, nil
		}
	}

	// Every remaining entry was returned, so there is no next page.
	page.NextCursor = ""
	return page, nil
}

// collectDay appends one day's files to page and reports whether the page is
// full.
func (s *Store) collectDay(
	ctx context.Context,
	query persis.ArtifactQuery,
	resume *cursor,
	day string,
	page *persis.ArtifactPage,
) (bool, error) {
	runDirs, err := s.listRunDirsDesc(day)
	if err != nil {
		return false, err
	}

	for _, runDir := range runDirs {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		from := ""
		if resume != nil && day == resume.Day {
			if runDir.name > resume.RunDir {
				continue
			}
			if runDir.name == resume.RunDir {
				from = resume.Path
			}
		}
		if !matchesName(runDir.dagName, query.Name) {
			continue
		}

		rec := s.readRecord(ctx, day, runDir.name)
		if rec == nil {
			continue
		}
		if !query.WorkspaceFilter.MatchesLabels(ir.NewLabels(rec.Labels)) {
			continue
		}
		if !inRange(rec, query) {
			continue
		}

		files, err := listFiles(rec.Dir)
		if err != nil {
			logger.Warn(ctx, "Failed to list artifact files", tag.Error(err), tag.Dir(rec.Dir))
			continue
		}
		startedAt, _ := stringutil.ParseTime(rec.StartedAt)

		for _, file := range files {
			if from != "" && file.path <= from {
				continue
			}
			// The cursor names the last file returned, so the next page can
			// resume strictly after it.
			if len(page.Items) == query.Limit {
				return true, nil
			}
			page.Items = append(page.Items, persis.ArtifactFile{
				Name:      rec.Name,
				DAGRunID:  rec.DAGRunID,
				StartedAt: startedAt,
				Path:      file.path,
				Size:      file.size,
			})
			page.NextCursor = encodeCursor(query, day, runDir.name, file.path)
		}
	}
	return false, nil
}

func (s *Store) readRecord(ctx context.Context, day, runDir string) *Record {
	path := filepath.Join(s.rootDir, filepath.FromSlash(day), runDir+artifactpath.MetaSuffix)

	load := func() (*Record, error) { return ReadRecord(path) }
	if s.cache != nil {
		load = func() (*Record, error) {
			return s.cache.LoadLatest(path, func() (*Record, error) { return ReadRecord(path) })
		}
	}

	rec, err := load()
	if err != nil {
		// A record removed or rewritten mid-scan must not fail the page.
		if !os.IsNotExist(err) {
			logger.Warn(ctx, "Skipping unreadable artifact record", tag.Error(err), tag.File(path))
		}
		return nil
	}
	return rec
}

type runDirEntry struct {
	name    string
	dagName string
}

// listRunDirsDesc returns a day's index records newest first.
func (s *Store) listRunDirsDesc(day string) ([]runDirEntry, error) {
	dayPath := filepath.Join(s.rootDir, filepath.FromSlash(day))
	entries, err := os.ReadDir(dayPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	runDirs := make([]runDirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !artifactpath.IsMetaName(entry.Name()) {
			continue
		}
		name := artifactpath.TrimMetaSuffix(entry.Name())
		parsed, ok := artifactpath.ParseRunDirName(name)
		if !ok {
			continue
		}
		runDirs = append(runDirs, runDirEntry{name: name, dagName: parsed.DAGName})
	}

	sort.Slice(runDirs, func(i, j int) bool { return runDirs[i].name > runDirs[j].name })
	return runDirs, nil
}

// listDaysDesc returns the "YYYY/MM/DD" days present in the tree within the
// query's range, newest first.
func (s *Store) listDaysDesc(query persis.ArtifactQuery) ([]string, error) {
	from, to := dayBounds(query)

	years, err := listNumericDirsDesc(s.rootDir, 4)
	if err != nil {
		return nil, err
	}

	var days []string
	for _, year := range years {
		months, err := listNumericDirsDesc(filepath.Join(s.rootDir, year), 2)
		if err != nil {
			return nil, err
		}
		for _, month := range months {
			daysOfMonth, err := listNumericDirsDesc(filepath.Join(s.rootDir, year, month), 2)
			if err != nil {
				return nil, err
			}
			for _, day := range daysOfMonth {
				key := year + "/" + month + "/" + day
				if (from != "" && key < from) || (to != "" && key > to) {
					continue
				}
				days = append(days, key)
			}
		}
	}
	return days, nil
}

func dayBounds(query persis.ArtifactQuery) (from, to string) {
	if !query.From.IsZero() {
		from = query.From.UTC().Format("2006/01/02")
	}
	if !query.To.IsZero() {
		to = query.To.UTC().Format("2006/01/02")
	}
	return from, to
}

func listNumericDirsDesc(dir string, width int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || len(name) != width || !isDigits(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	return names, nil
}

type artifactFileEntry struct {
	path string
	size int64
}

// listFiles walks a run's artifact directory, returning regular files in a
// stable order.
func listFiles(dir string) ([]artifactFileEntry, error) {
	var files []artifactFileEntry
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, artifactFileEntry{path: filepath.ToSlash(rel), size: info.Size()})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

func inRange(rec *Record, query persis.ArtifactQuery) bool {
	if query.From.IsZero() && query.To.IsZero() {
		return true
	}
	startedAt, err := stringutil.ParseTime(rec.StartedAt)
	if err != nil || startedAt.IsZero() {
		// The day directory already bounded this entry.
		return true
	}
	if !query.From.IsZero() && startedAt.Before(query.From.Time) {
		return false
	}
	if !query.To.IsZero() && startedAt.After(query.To.Time) {
		return false
	}
	return true
}

func matchesName(dagName, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(dagName), strings.ToLower(filter))
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
