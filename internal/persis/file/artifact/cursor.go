// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package artifact

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dagucloud/dagu/v2/internal/persis"
)

const cursorVersion = 1

// cursor marks the last file a page returned. Listing resumes strictly after
// it, so a page boundary neither repeats nor skips an entry.
type cursor struct {
	Version int    `json:"v"`
	Filters string `json:"f"`
	Day     string `json:"d"`
	RunDir  string `json:"r"`
	Path    string `json:"p"`
}

func encodeCursor(query persis.ArtifactQuery, day, runDir, path string) string {
	data, err := json.Marshal(cursor{
		Version: cursorVersion,
		Filters: filterFingerprint(query),
		Day:     day,
		RunDir:  runDir,
		Path:    path,
	})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeCursor rejects a cursor issued for a different set of filters, because
// resuming across a filter change would silently skip or repeat entries.
func decodeCursor(query persis.ArtifactQuery) (*cursor, error) {
	if query.Cursor == "" {
		return nil, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(query.Cursor)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", persis.ErrInvalidArtifactCursor, err)
	}
	var c cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", persis.ErrInvalidArtifactCursor, err)
	}
	if c.Version != cursorVersion || c.Filters != filterFingerprint(query) {
		return nil, persis.ErrInvalidArtifactCursor
	}
	return &c, nil
}

func filterFingerprint(query persis.ArtifactQuery) string {
	var b strings.Builder
	b.WriteString(query.Name)
	b.WriteByte(0)
	b.WriteString(query.From.Format(fingerprintTimeLayout))
	b.WriteByte(0)
	b.WriteString(query.To.Format(fingerprintTimeLayout))
	b.WriteByte(0)
	if f := query.WorkspaceFilter; f != nil && f.Enabled {
		b.WriteString("ws")
		if f.IncludeUnlabelled {
			b.WriteString("+u")
		}
		for _, name := range f.Workspaces {
			b.WriteByte(0)
			b.WriteString(name)
		}
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

const fingerprintTimeLayout = "20060102150405"
