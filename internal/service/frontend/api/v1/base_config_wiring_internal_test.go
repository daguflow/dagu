// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	generated "github.com/dagucloud/dagu/v2/api/v1"
	"github.com/dagucloud/dagu/v2/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dagucloud/dagu/v2/internal/cmn/config"
	"github.com/dagucloud/dagu/v2/internal/dagsettings"
	"github.com/dagucloud/dagu/v2/internal/ir"
	"github.com/dagucloud/dagu/v2/internal/persis"
	"github.com/dagucloud/dagu/v2/internal/runtime"
	"github.com/dagucloud/dagu/v2/internal/testutil"
)

func TestRestoreSnapshotSMTP(t *testing.T) {
	t.Parallel()

	basePath := filepath.Join(t.TempDir(), "base.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte("smtp:\n  host: current.example\n"), 0600))
	a := &API{config: &config.Config{Paths: config.PathsConfig{BaseConfig: basePath}}}
	dag := &ir.DAG{
		Name: "snapshot", YamlData: []byte("steps:\n  - run: echo original\n"),
		BaseConfigData: []byte("smtp:\n  host: old.example\nenv:\n  ORIGINAL: original\n"),
	}
	restored, _, err := a.restoreDAGRunSnapshot(context.Background(), dag, &ir.DAGRunStatus{})
	require.NoError(t, err)
	require.NotNil(t, restored.SMTP)
	assert.Equal(t, "current.example", restored.SMTP.Host)
	assert.Contains(t, restored.Env, "ORIGINAL=original")
	assert.Equal(t, dag.YamlData, restored.YamlData)
}

func TestRestoreLegacyChildSMTP(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := &config.Config{Paths: config.PathsConfig{DAGsDir: t.TempDir()}}
	basePath := workspace.BaseConfigPath(cfg.Paths.DAGsDir, "ops")
	require.NoError(t, os.MkdirAll(filepath.Dir(basePath), 0750))
	require.NoError(t, os.WriteFile(basePath, []byte("smtp:\n  host: workspace.example\n"), 0600))
	repository := testutil.NewFileDAGRunRepository(t.TempDir(), persis.DAGRunRepositoryOptions{})
	root := ir.NewDAGRunRef("root", "root-run")
	parent := ir.NewDAGRunRef("parent", "parent-run")
	for _, status := range []ir.DAGRunStatus{
		{Name: root.Name, DAGRunID: root.ID, Status: ir.Failed},
		{Name: parent.Name, DAGRunID: parent.ID, Root: root, Parent: root, Status: ir.Failed},
	} {
		definition := "steps:\n  - run: echo saved\n"
		if status.DAGRunID == root.ID {
			definition = "labels: [workspace=ops]\n" + definition
		}
		dag := &ir.DAG{Name: status.Name, YamlData: []byte(definition), BaseConfigData: []byte("{}")}
		attempt, err := repository.CreateAttempt(ctx, dag, time.Now(), status.DAGRunID,
			persis.DAGRunCreateAttemptOptions{RootDAGRun: status.Root})
		require.NoError(t, err)
		require.NoError(t, attempt.Open(ctx))
		require.NoError(t, attempt.Write(ctx, status))
		require.NoError(t, attempt.Close(ctx))
	}
	a := &API{config: cfg, dagRunRepository: repository}
	child := &ir.DAG{Name: "child", YamlData: []byte("steps:\n  - run: echo child\n"), BaseConfigData: []byte("{}")}
	restored, _, err := a.restoreDAGRunSnapshot(ctx, child, &ir.DAGRunStatus{Root: root, Parent: parent})
	require.NoError(t, err)
	require.NotNil(t, restored.SMTP)
	assert.Equal(t, "workspace.example", restored.SMTP.Host)
}

type stubBaseConfigStore struct {
	spec string
}

func (s stubBaseConfigStore) GetSpec(context.Context) (string, error) {
	return s.spec, nil
}

func (stubBaseConfigStore) UpdateSpec(context.Context, []byte) error {
	return nil
}

type workspaceStoreStub struct {
	workspace.Store
	item *workspace.Workspace
}

func (s workspaceStoreStub) GetByName(context.Context, string) (*workspace.Workspace, error) {
	return s.item, nil
}

func TestRequireBaseConfigManagementRequiresWorkspaceProvider(t *testing.T) {
	t.Parallel()

	a := &API{baseConfigStore: stubBaseConfigStore{}}

	assert.ErrorIs(t, a.requireBaseConfigManagement(), ErrBaseConfigNotAvailable)

	a.baseConfigProvider = func(string) (dagsettings.BaseConfigStore, error) {
		return stubBaseConfigStore{}, nil
	}
	require.NoError(t, a.requireBaseConfigManagement())
}

func TestNewPanicsWhenBaseConfigStoreHasNoWorkspaceProvider(t *testing.T) {
	t.Parallel()

	require.PanicsWithValue(t,
		"api: workspace base config provider must be configured when base config store is configured",
		func() {
			New(
				nil,
				nil,
				nil,
				nil,
				runtime.Manager{},
				&config.Config{},
				nil,
				nil,
				nil,
				nil,
				WithBaseConfigStore(stubBaseConfigStore{}),
			)
		},
	)
}

func TestGetWorkspaceBaseConfigUsesProvider(t *testing.T) {
	t.Parallel()

	a := &API{
		baseConfigStore: stubBaseConfigStore{},
		baseConfigProvider: func(name string) (dagsettings.BaseConfigStore, error) {
			assert.Equal(t, "operations", name)
			return stubBaseConfigStore{spec: "max_active_runs: 2\n"}, nil
		},
		workspaceStore: workspaceStoreStub{
			item: &workspace.Workspace{Name: "operations"},
		},
	}

	response, err := a.GetWorkspaceBaseConfig(context.Background(), generated.GetWorkspaceBaseConfigRequestObject{
		WorkspaceName: "operations",
	})
	require.NoError(t, err)
	assert.Equal(t, "max_active_runs: 2\n", response.(generated.GetWorkspaceBaseConfig200JSONResponse).Spec)
}
