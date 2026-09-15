// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  RunDateMode,
  RunDatePreset,
  RunSpecificPeriod,
  ViewSpecType,
  ViewWorkspaceScope,
} from '@/api/v1/schema';
import type { View } from '@/hooks/useViews';
import { AppBarContext } from '@/contexts/AppBarContext';
import { ConfigContext, type Config } from '@/contexts/ConfigContext';
import { WorkspaceKind } from '@/lib/workspace';
import DAGRuns from '..';

const {
  createRunViewMock,
  deleteRunViewMock,
  readSearchStateMock,
  searchStateMock,
  sharedRunViewState,
  updateRunViewMock,
  writeSearchStateMock,
} = vi.hoisted(() => {
  const readState = vi.fn(() => null);
  const writeState = vi.fn();
  return {
    createRunViewMock: vi.fn(),
    deleteRunViewMock: vi.fn(),
    updateRunViewMock: vi.fn(),
    readSearchStateMock: readState,
    searchStateMock: { readState, writeState },
    sharedRunViewState: { views: [] as View[] },
    writeSearchStateMock: writeState,
  };
});

vi.mock('@/contexts/SearchStateContext', () => ({
  useSearchState: () => searchStateMock,
}));

vi.mock('@/contexts/AuthContext', () => ({
  useCanWriteForWorkspace: () => true,
}));

vi.mock('@/hooks/useViews', () => ({
  useViews: () => ({
    views: sharedRunViewState.views,
    isLoading: false,
    error: undefined,
    createView: createRunViewMock,
    updateView: updateRunViewMock,
    deleteView: deleteRunViewMock,
    refresh: vi.fn(),
  }),
}));

vi.mock('@/contexts/UserPreference', () => ({
  useUserPreferences: () => ({
    preferences: {
      dagRunsViewMode: 'list',
    },
    updatePreference: vi.fn(),
  }),
}));

vi.mock('@/hooks/api', () => ({
  useQuery: () => ({
    data: { labels: [] },
  }),
}));

const usePaginatedDAGRunsMock = vi.hoisted(() => vi.fn());

vi.mock('@/features/dag-runs/hooks/dagRunPagination', () => ({
  usePaginatedDAGRuns: usePaginatedDAGRunsMock,
}));

vi.mock('@/features/dag-runs/hooks/useBulkDAGRunSelection', () => ({
  useBulkDAGRunSelection: () => ({
    clearSelection: vi.fn(),
    replaceSelection: vi.fn(),
    selectAllLoaded: vi.fn(),
    selectedKeys: new Set(),
    selectedRuns: [],
    toggleSelection: vi.fn(),
  }),
}));

vi.mock('@/features/dag-runs/components/common/DAGRunBatchActions', () => ({
  default: () => null,
}));

vi.mock('@/features/dag-runs/components/dag-run-details', () => ({
  DAGRunDetailsModal: ({
    name,
    dagRunId,
    initialTab,
    onClose,
  }: {
    name: string;
    dagRunId: string;
    initialTab: string;
    onClose: () => void;
  }) => (
    <div role="dialog">
      Run modal for {name}/{dagRunId} on {initialTab}
      <button type="button" onClick={onClose}>
        Close run
      </button>
    </div>
  ),
}));

vi.mock(
  '@/features/dag-runs/components/dag-run-list/DAGRunGroupedView',
  () => ({
    default: () => <div>Grouped Runs</div>,
  })
);

const dagRunTableProps = vi.hoisted(() => ({
  current: {} as { isLoading?: boolean },
}));

vi.mock('@/features/dag-runs/components/dag-run-list/DAGRunTable', () => ({
  default: (props: {
    isLoading?: boolean;
    onSelectDAGRun: (run: { name: string; dagRunId: string }) => void;
    onViewArtifacts: (run: { name: string; dagRunId: string }) => void;
  }) => {
    dagRunTableProps.current = props;
    const { onSelectDAGRun, onViewArtifacts } = props;
    return (
      <div>
        <div>Run Table</div>
        <button
          type="button"
          onClick={() => onSelectDAGRun({ name: 'demo', dagRunId: 'run-1' })}
        >
          Open run
        </button>
        <button
          type="button"
          onClick={() => onViewArtifacts({ name: 'demo', dagRunId: 'run-1' })}
        >
          Open artifacts
        </button>
      </div>
    );
  },
}));

const config = {
  tzOffsetInSec: undefined,
} as Config;

beforeEach(() => {
  readSearchStateMock.mockReset();
  readSearchStateMock.mockReturnValue(null);
  writeSearchStateMock.mockReset();
  createRunViewMock.mockReset();
  updateRunViewMock.mockReset();
  deleteRunViewMock.mockReset();
  sharedRunViewState.views = [];
  usePaginatedDAGRunsMock.mockReset();
  usePaginatedDAGRunsMock.mockReturnValue({
    dagRuns: [],
    isInitialLoading: false,
    isLoadingMore: false,
    loadMoreError: null,
    hasMore: false,
    refresh: vi.fn(),
    loadMore: vi.fn(),
  });
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function makeRunView(overrides: Partial<View> = {}): View {
  return {
    id: 'failed-runs',
    name: 'Failed runs',
    type: ViewSpecType.run,
    intervalDays: 1,
    dagName: '',
    labels: [],
    runStatus: '5',
    dateMode: RunDateMode.preset,
    datePreset: RunDatePreset.today,
    specificPeriod: RunSpecificPeriod.date,
    specificValue: '',
    pinned: false,
    workspace: '',
    workspaceScope: ViewWorkspaceScope.all,
    createdAt: '2026-09-15T00:00:00Z',
    updatedAt: '2026-09-15T00:00:00Z',
    ...overrides,
  };
}

function lastRunQuery(): Record<string, unknown> {
  const calls = usePaginatedDAGRunsMock.mock.calls;
  return calls[calls.length - 1]?.[0]?.query ?? {};
}

function LocationProbe(): React.JSX.Element {
  const location = useLocation();
  return <output data-testid="location-search">{location.search}</output>;
}

function locationSearchParams(): URLSearchParams {
  return new URLSearchParams(
    screen.getByTestId('location-search').textContent ?? ''
  );
}

function renderPage(setTitle = vi.fn(), initialEntry = '/dag-runs'): void {
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <LocationProbe />
      <ConfigContext.Provider value={config}>
        <AppBarContext.Provider
          value={
            {
              setTitle,
              selectedRemoteNode: 'local',
              workspaceSelection: { kind: WorkspaceKind.all },
            } as never
          }
        >
          <DAGRuns />
        </AppBarContext.Provider>
      </ConfigContext.Provider>
    </MemoryRouter>
  );
}

describe('DAGRuns page', () => {
  it('uses the Executions page title', () => {
    const setTitle = vi.fn();

    renderPage(setTitle);

    expect(
      screen.getByRole('heading', { name: /^executions$/i })
    ).toBeVisible();
    expect(screen.queryByRole('heading', { name: /dag runs/i })).toBeNull();
    expect(setTitle).toHaveBeenCalledWith('Executions');
  });

  it('passes the initial-load state to the runs table', () => {
    usePaginatedDAGRunsMock.mockReturnValue({
      dagRuns: [],
      isInitialLoading: true,
      isLoadingMore: false,
      loadMoreError: null,
      hasMore: false,
      refresh: vi.fn(),
      loadMore: vi.fn(),
    });

    renderPage();

    expect(dagRunTableProps.current.isLoading).toBe(true);
  });

  it('uses consistent filter control sizing', () => {
    renderPage();

    expect(
      screen.getByPlaceholderText('Filter by DAG name...').className
    ).toContain('h-9');
    expect(
      screen.getByPlaceholderText('Filter by Run ID...').className
    ).toContain('h-9');
    expect(
      screen.getByRole('combobox', { name: 'Status' }).className
    ).toContain('h-9');
    expect(
      screen.getByRole('combobox', { name: 'Date preset' }).className
    ).toContain('h-9');
    expect(screen.getByRole('button', { name: 'Search' }).className).toContain(
      'h-9'
    );

    const labelInput = screen.getByRole('combobox', {
      name: 'Filter by labels...',
    });
    expect(labelInput.parentElement?.className).toContain('min-h-9');
    expect(labelInput.parentElement?.className).toContain('bg-card');

    expect(screen.getByRole('combobox', { name: 'Status' })).toHaveTextContent(
      'All Statuses'
    );
  });

  it('stores the selected run in the URL and clears it on close', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Open run' }));
    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Run modal for demo/run-1 on status'
    );
    expect(screen.getByTestId('location-search')).toHaveTextContent(
      '?selectedRunName=demo&selectedRunId=run-1'
    );

    fireEvent.click(screen.getByRole('button', { name: 'Close run' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.getByTestId('location-search')).toHaveTextContent('');
  });

  it('opens a run on the artifacts tab and stores that selection in the URL', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Open artifacts' }));

    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Run modal for demo/run-1 on artifacts'
    );
    expect(screen.getByTestId('location-search')).toHaveTextContent(
      '?selectedRunName=demo&selectedRunId=run-1&selectedRunTab=artifacts'
    );
  });

  it('preserves execution filters when opening a run', async () => {
    renderPage();

    fireEvent.change(screen.getByPlaceholderText('Filter by DAG name...'), {
      target: { value: 'deploy' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));

    await waitFor(() => {
      expect(locationSearchParams().get('name')).toBe('deploy');
    });

    fireEvent.click(screen.getByRole('button', { name: 'Open run' }));

    expect(locationSearchParams().get('name')).toBe('deploy');
    expect(locationSearchParams().get('selectedRunName')).toBe('demo');
    expect(locationSearchParams().get('selectedRunId')).toBe('run-1');
  });

  it('keeps only active date-mode parameters after Search', async () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Search' }));
    await waitFor(() => {
      const params = locationSearchParams();
      expect(params.get('dateMode')).toBe('preset');
      expect(params.get('preset')).toBe('today');
      expect(params.has('specificValue')).toBe(false);
      expect(params.has('specificPeriod')).toBe(false);
    });

    fireEvent.click(
      screen.getByRole('button', { name: 'Specific date/month/year' })
    );
    await waitFor(() => {
      const params = locationSearchParams();
      expect(params.get('dateMode')).toBe('specific');
    });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));
    await waitFor(() => {
      const params = locationSearchParams();
      expect(params.has('preset')).toBe(false);
      expect(params.get('specificValue')).not.toBeNull();
      expect(params.get('specificPeriod')).toBe('date');
    });

    fireEvent.click(screen.getByRole('button', { name: 'Custom range' }));
    await waitFor(() => {
      const params = locationSearchParams();
      expect(params.get('dateMode')).toBe('custom');
    });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));
    await waitFor(() => {
      const params = locationSearchParams();
      expect(params.has('preset')).toBe(false);
      expect(params.has('specificValue')).toBe(false);
      expect(params.has('specificPeriod')).toBe(false);
    });
  });

  it('restores the run and artifact tab from the URL', () => {
    renderPage(
      vi.fn(),
      '/dag-runs?selectedRunName=demo&selectedRunId=run-1&selectedRunTab=artifacts'
    );

    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Run modal for demo/run-1 on artifacts'
    );
  });

  it('applies the default run view to the first request', async () => {
    sharedRunViewState.views.push(
      makeRunView({
        isDefault: true,
        dagName: 'deploy',
      })
    );

    renderPage();

    await waitFor(() => {
      expect(lastRunQuery()['name']).toBe('deploy');
      expect(lastRunQuery()['status']).toEqual([5]);
      expect(lastRunQuery()['fromDate']).toBeTypeOf('number');
    });
    expect(
      screen.getByRole('button', { name: 'Run view: Failed runs' })
    ).toBeVisible();
  });

  it('uses the bookmarked run view from the URL', async () => {
    sharedRunViewState.views.push(
      makeRunView({
        id: 'url-view',
        name: 'Nightly runs',
        runStatus: '1',
        dagName: 'nightly',
      })
    );

    renderPage(vi.fn(), '/dag-runs?view=url-view');

    await waitFor(() => {
      expect(lastRunQuery()['name']).toBe('nightly');
      expect(lastRunQuery()['status']).toEqual([1]);
    });
    expect(
      screen.getByRole('button', { name: 'Run view: Nightly runs' })
    ).toBeVisible();
  });

  it('gives explicit URL filters precedence over the requested run view', async () => {
    sharedRunViewState.views.push(
      makeRunView({ id: 'view-a', name: 'View A', runStatus: '5' })
    );

    renderPage(vi.fn(), '/dag-runs?view=view-a&name=adhoc&status=1');

    await waitFor(() => {
      expect(lastRunQuery()['name']).toBe('adhoc');
      expect(lastRunQuery()['status']).toEqual([1]);
    });
    expect(
      screen.getByRole('button', { name: 'Run view: View A' })
    ).toBeVisible();
  });

  it('saves the current filters as a run view and applies it', async () => {
    const user = userEvent.setup();
    createRunViewMock.mockResolvedValue(
      makeRunView({ id: 'nightly-view', name: 'Nightly runs' })
    );

    renderPage();

    fireEvent.change(screen.getByPlaceholderText('Filter by DAG name...'), {
      target: { value: 'nightly' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));
    await waitFor(() => {
      expect(locationSearchParams().get('name')).toBe('nightly');
    });

    await user.click(
      screen.getByRole('button', { name: 'Run view: Custom view' })
    );
    await user.click(
      screen.getByRole('menuitem', {
        name: 'Save current filters as view…',
      })
    );
    await user.type(
      screen.getByRole('textbox', { name: 'Name' }),
      'Nightly runs'
    );
    await user.click(screen.getByRole('button', { name: 'Save view' }));

    await waitFor(() => {
      expect(createRunViewMock).toHaveBeenCalledWith(
        expect.objectContaining({
          type: ViewSpecType.run,
          name: 'Nightly runs',
          dagName: 'nightly',
          intervalDays: 1,
        })
      );
    });
    await waitFor(() => {
      expect(locationSearchParams().get('view')).toBe('nightly-view');
    });
    // Preset and specific views keep relative date params and derive the
    // concrete range when applied; only custom ranges persist dates.
    expect(locationSearchParams().has('fromDate')).toBe(false);
  });

  it('marks a run view as edited when its filters change and resets via the menu', async () => {
    const user = userEvent.setup();
    sharedRunViewState.views.push(makeRunView({ dagName: 'deploy' }));

    renderPage(vi.fn(), '/dag-runs?view=failed-runs');

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Run view: Failed runs' })
      ).toBeVisible();
    });
    expect(screen.queryByText('Edited')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('Filter by DAG name...'), {
      target: { value: 'etl' },
    });
    expect(screen.getByText('Edited')).toBeVisible();

    await user.click(
      screen.getByRole('button', { name: 'Run view: Failed runs' })
    );
    await user.click(screen.getByRole('menuitem', { name: 'Reset changes' }));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Filter by DAG name...')).toHaveValue(
        'deploy'
      );
    });
  });

  it('deletes the active run view and falls back to All runs', async () => {
    const user = userEvent.setup();
    sharedRunViewState.views.push(makeRunView({ dagName: 'deploy' }));
    deleteRunViewMock.mockResolvedValue(undefined);

    renderPage(vi.fn(), '/dag-runs?view=failed-runs');

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Run view: Failed runs' })
      ).toBeVisible();
    });

    await user.click(
      screen.getByRole('button', { name: 'Run view: Failed runs' })
    );
    await user.click(screen.getByRole('menuitem', { name: 'Manage views…' }));
    await user.click(
      screen.getByRole('button', { name: 'Delete Failed runs' })
    );
    await user.click(
      await screen.findByRole('button', { name: 'Delete view' })
    );

    await waitFor(() => {
      expect(deleteRunViewMock).toHaveBeenCalledWith('failed-runs');
    });
    await waitFor(() => {
      expect(locationSearchParams().get('view')).toBe('all');
    });
  });
});
