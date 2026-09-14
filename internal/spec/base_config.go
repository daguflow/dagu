// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dagucloud/dagu/v2/internal/cmn/fileutil"
	"golang.org/x/sync/singleflight"
)

const baseConfigCacheSize = 1024

var baseConfigFiles = baseConfigCache{
	files: fileutil.NewCache[*baseConfigSource]("base_config", baseConfigCacheSize, 0),
}

type baseConfigCache struct {
	files *fileutil.Cache[*baseConfigSource]
	loads singleflight.Group
}

type baseDefinition struct {
	definition *dag
	source     *baseConfigSource
}

// baseConfigSource is immutable; each build decodes its own manifest copy.
type baseConfigSource struct {
	raw    []byte
	values map[string]any
	err    error
}

func (c *baseConfigCache) load(path string, read func(string) ([]byte, error)) (*baseConfigSource, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		c.files.Invalidate(path)
		return nil, err
	}
	// Only readers of the same file version may share an in-flight load.
	key := fmt.Sprintf("%s\x00%d\x00%d", path, info.Size(), info.ModTime().UnixNano())
	value, err, _ := c.loads.Do(key, func() (any, error) {
		source, err := c.files.LoadLatest(path, func() (*baseConfigSource, error) {
			data, err := read(path)
			if err != nil {
				return nil, err
			}
			return parseBaseConfig(data), nil
		})
		if err != nil {
			// A missing or unreadable file must not resurrect an earlier entry.
			c.files.Invalidate(path)
		}
		return source, err
	})
	if err != nil {
		return nil, err
	}
	return value.(*baseConfigSource), nil
}

func parseBaseConfig(data []byte) *baseConfigSource {
	values, err := unmarshalData(data)
	return &baseConfigSource{raw: data, values: values, err: err}
}

func (s *baseConfigSource) decode(description string) (*baseDefinition, error) {
	if s == nil || len(s.raw) == 0 {
		return nil, nil
	}
	if s.err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", description, s.err)
	}
	def, err := decode(cloneMap(s.values))
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", description, err)
	}
	return &baseDefinition{definition: def, source: s}, nil
}
