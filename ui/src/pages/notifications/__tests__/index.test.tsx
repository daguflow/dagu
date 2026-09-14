// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { AppBarContext } from '@/contexts/AppBarContext';
import { NotificationEventType } from '@/api/v1/schema';

vi.hoisted(() => {
  vi.stubGlobal('getConfig', () => ({
    apiURL: '/api/v1',
    authMode: 'builtin',
  }));
});

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
  client: {
    PUT: vi.fn(),
    POST: vi.fn(),
    DELETE: vi.fn(),
  },
}));

vi.mock('@/hooks/api', () => ({
  useClient: () => mocks.client,
  useQuery: mocks.useQuery,
}));

import NotificationsPage, {
  NotificationChannelsPage,
  NotificationRulesPage,
} from '..';

Object.defineProperty(HTMLElement.prototype, 'hasPointerCapture', {
  configurable: true,
  value: () => false,
});

function renderPage() {
  const setTitle = vi.fn();

  render(
    <MemoryRouter>
      <AppBarContext.Provider value={{ setTitle } as never}>
        <NotificationsPage />
      </AppBarContext.Provider>
    </MemoryRouter>
  );

  return { setTitle };
}

function renderChannelsPage(settings: object) {
  const settingsQuery = {
    data: settings,
    error: undefined,
    isLoading: false,
    mutate: vi.fn(),
  };
  const channelsQuery = {
    data: { channels: [] },
    error: undefined,
    isLoading: false,
    mutate: vi.fn(),
  };
  mocks.useQuery.mockImplementation((path: string) =>
    path === '/notification-settings' ? settingsQuery : channelsQuery
  );

  render(
    <MemoryRouter>
      <AppBarContext.Provider
        value={
          {
            setTitle: vi.fn(),
            selectedRemoteNode: 'local',
          } as never
        }
      >
        <NotificationChannelsPage />
      </AppBarContext.Provider>
    </MemoryRouter>
  );
}

function renderRulesPage(
  extraChannels: object[] = [],
  events = [
    NotificationEventType.dag_run_aborted,
    NotificationEventType.dag_run_rejected,
  ],
  workspaceName?: string
) {
  const queries: Record<string, object> = {
    '/notification-channels': {
      channels: [
        { id: 'slack', name: 'slack-test', type: 'slack', enabled: true },
        ...extraChannels,
      ],
    },
    '/notification-routes/workspaces/{workspaceName}': {
      enabled: true,
      inheritGlobal: true,
      routes: [],
    },
    '/notification-routes/global': {
      enabled: true,
      inheritGlobal: true,
      routes: [
        {
          id: 'route',
          channelId: 'slack',
          enabled: true,
          events,
        },
      ],
    },
  };
  mocks.useQuery.mockImplementation((path: string) => ({
    data: queries[path],
    isLoading: false,
    mutate: vi.fn((data: object) => {
      queries[path] = data;
    }),
  }));

  render(
    <MemoryRouter>
      <AppBarContext.Provider
        value={
          {
            setTitle: vi.fn(),
            selectedRemoteNode: 'local',
            workspaceSelection: workspaceName
              ? { kind: 'workspace', workspace: workspaceName }
              : { kind: 'all' },
          } as never
        }
      >
        <NotificationRulesPage />
      </AppBarContext.Provider>
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.client.PUT.mockImplementation(
    async (_path: string, { body }: { body: object }) => ({ data: body })
  );
});

describe('NotificationsPage', () => {
  it('renders notification links by section', () => {
    const { setTitle } = renderPage();

    expect(
      screen.getByRole('heading', { name: /^notifications$/i })
    ).toBeVisible();
    const rulesLink = screen.getByRole('link', { name: /^rules/i });
    const channelsLink = screen.getByRole('link', { name: /^channels/i });
    expect(rulesLink).toHaveAttribute('href', '/notification-rules');
    expect(channelsLink).toHaveAttribute('href', '/notification-channels');
    expect(
      rulesLink.compareDocumentPosition(channelsLink) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
    expect(
      screen.getByText('Set Global defaults and workspace overrides.')
    ).toBeVisible();
    expect(
      screen.getByText(
        'Manage Slack, email, webhook, and Telegram destinations.'
      )
    ).toBeVisible();
    expect(setTitle).toHaveBeenCalledWith('Notifications');
  });
});

describe('NotificationChannelsPage', () => {
  it('blocks channel controls when delivery is unavailable', () => {
    mocks.useQuery.mockReturnValue({
      data: undefined,
      error: new Error('Notification delivery is unavailable'),
      isLoading: false,
      mutate: vi.fn(),
    });

    render(
      <MemoryRouter>
        <AppBarContext.Provider value={{ setTitle: vi.fn() } as never}>
          <NotificationChannelsPage />
        </AppBarContext.Provider>
      </MemoryRouter>
    );

    expect(
      screen.getByText('Notification delivery is unavailable')
    ).toBeVisible();
    expect(
      screen.queryByRole('button', { name: /^save$/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /add channel/i })
    ).not.toBeInTheDocument();
  });

  it('preserves the configured password indicator when toggling authentication modes', async () => {
    const user = userEvent.setup();
    renderChannelsPage({
      smtp: {
        host: 'smtp.example.com',
        port: '587',
        username: 'sender@example.com',
        passwordConfigured: true,
      },
    });

    expect(screen.getByPlaceholderText('Password configured')).toBeVisible();

    await user.click(screen.getByLabelText('SMTP authentication'));
    await user.click(screen.getByRole('option', { name: 'OAuth 2.0' }));
    await user.click(screen.getByLabelText('SMTP authentication'));
    await user.click(screen.getByRole('option', { name: 'Password' }));

    expect(screen.getByPlaceholderText('Password configured')).toBeVisible();
  });

  it('preserves configured OAuth indicators when toggling authentication modes', async () => {
    const user = userEvent.setup();
    renderChannelsPage({
      smtp: {
        host: 'smtp.office365.com',
        port: '587',
        username: 'sender@example.com',
        oauth: {
          provider: 'microsoft',
          tenantId: 'tenant',
          clientId: 'client',
          clientSecretConfigured: true,
          refreshTokenConfigured: false,
          serviceAccountJsonConfigured: false,
        },
      },
    });

    expect(
      screen.getByPlaceholderText('Client secret configured')
    ).toBeVisible();

    await user.click(screen.getByLabelText('SMTP authentication'));
    await user.click(screen.getByRole('option', { name: 'Password' }));
    await user.click(screen.getByLabelText('SMTP authentication'));
    await user.click(screen.getByRole('option', { name: 'OAuth 2.0' }));

    expect(
      screen.getByPlaceholderText('Client secret configured')
    ).toBeVisible();
  });

  it('keeps typed OAuth secrets when identity fields change', async () => {
    const user = userEvent.setup();
    renderChannelsPage({});

    await user.click(screen.getByLabelText('SMTP authentication'));
    await user.click(screen.getByRole('option', { name: 'OAuth 2.0' }));

    const clientSecret = screen.getByPlaceholderText('Client secret');
    await user.type(clientSecret, 'typed-secret');
    await user.type(
      screen.getByPlaceholderText('Microsoft tenant ID'),
      'tenant'
    );
    await user.type(screen.getByPlaceholderText('Client ID'), 'client');
    await user.type(
      screen.getByPlaceholderText('Sender mailbox'),
      'sender@example.com'
    );

    expect(clientSecret).toHaveValue('typed-secret');
  });
});

describe('NotificationRulesPage', () => {
  it('loads operational defaults for saved routes without events', () => {
    renderRulesPage([], []);

    for (const name of ['Failed', 'Aborted', 'Rejected', 'Waiting']) {
      expect(screen.getByRole('checkbox', { name })).toBeChecked();
    }
    expect(
      screen.getByRole('checkbox', { name: 'Succeeded' })
    ).not.toBeChecked();
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled();
  });

  it('allows clearing events while editing and requires a selection to save', async () => {
    const user = userEvent.setup();
    renderRulesPage();

    await user.click(screen.getByRole('checkbox', { name: 'Aborted' }));
    await user.click(screen.getByRole('checkbox', { name: 'Rejected' }));

    for (const checkbox of screen.getAllByRole('checkbox')) {
      expect(checkbox).not.toBeChecked();
    }
    expect(
      screen.getByText('Select at least one event before saving.')
    ).toBeVisible();
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled();

    await user.click(screen.getByRole('checkbox', { name: 'Failed' }));
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled();
  });

  it('offers channel creation when every destination has a rule', async () => {
    const user = userEvent.setup();
    renderRulesPage();
    await user.click(screen.getByRole('button', { name: 'Add rule' }));
    const dialog = within(screen.getByRole('dialog'));
    expect(
      dialog.getByText(
        'Every channel already has a rule. Edit an existing rule or create another channel.'
      )
    ).toBeVisible();
    expect(dialog.getByRole('link', { name: 'Add channel' })).toHaveAttribute(
      'href',
      '/notification-channels'
    );
  });

  it('adds a route for an unused channel', async () => {
    const user = userEvent.setup();
    renderRulesPage([
      { id: 'email', name: 'email-test', type: 'smtp', enabled: true },
    ]);

    await user.click(screen.getByRole('button', { name: 'Add rule' }));
    await user.click(
      within(screen.getByRole('dialog')).getByRole('button', {
        name: /email-test/,
      })
    );

    expect(
      screen.getByRole('switch', { name: 'Toggle email-test' })
    ).toBeChecked();
    expect(
      screen.getAllByRole('checkbox', { name: 'Failed' })[1]
    ).toBeChecked();
  });

  it('edits events through checkboxes and labels and saves them', async () => {
    const user = userEvent.setup();
    renderRulesPage();

    const failed = screen.getByRole('checkbox', {
      name: 'Failed',
    });
    const rejected = screen.getByRole('checkbox', {
      name: 'Rejected',
    });
    expect(failed).not.toBeChecked();
    expect(rejected).toBeChecked();

    await user.click(failed);
    expect(failed).toBeChecked();
    await user.click(rejected.closest('label')!);
    expect(rejected).not.toBeChecked();

    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    expect(mocks.client.PUT).toHaveBeenCalledWith(
      '/notification-routes/global',
      {
        params: { query: { remoteNode: 'local' } },
        body: {
          enabled: true,
          inheritGlobal: true,
          routes: [
            {
              id: 'route',
              channelId: 'slack',
              enabled: true,
              events: [
                NotificationEventType.dag_run_aborted,
                NotificationEventType.dag_run_failed,
              ],
            },
          ],
        },
      }
    );
  });
  it('cancels edits without saving and clears the dirty state', async () => {
    const user = userEvent.setup();
    renderRulesPage();
    await user.click(screen.getByRole('checkbox', { name: 'Failed' }));
    expect(screen.getByText('Unsaved changes')).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.getByRole('checkbox', { name: 'Failed' })).not.toBeChecked();
    expect(screen.getByRole('checkbox', { name: 'Rejected' })).toBeChecked();
    expect(screen.getByText('All changes saved')).toBeVisible();
    expect(mocks.client.PUT).not.toHaveBeenCalled();
  });

  it('deletes a rule through its menu and can undo the deletion', async () => {
    const user = userEvent.setup();
    renderRulesPage();
    await user.click(
      screen.getByRole('button', { name: 'Rule actions for slack-test' })
    );
    await user.click(screen.getByRole('menuitem', { name: 'Delete rule' }));
    expect(screen.getByText('No notification rules yet')).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(
      screen.getByRole('switch', { name: 'Toggle slack-test' })
    ).toBeChecked();
  });

  it('retains edits when saving fails', async () => {
    const user = userEvent.setup();
    mocks.client.PUT.mockResolvedValue({ error: { message: 'Save failed' } });
    renderRulesPage();
    await user.click(screen.getByRole('checkbox', { name: 'Failed' }));
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    expect(screen.getByText('Save failed')).toBeVisible();
    expect(screen.getByText('Unsaved changes')).toBeVisible();
    expect(screen.getByRole('checkbox', { name: 'Failed' })).toBeChecked();
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled();
  });

  it('tests the destination without saving rule edits', async () => {
    const user = userEvent.setup();
    mocks.client.POST.mockResolvedValue({
      data: { results: [{ delivered: true }] },
    });
    renderRulesPage();
    await user.click(screen.getByRole('checkbox', { name: 'Failed' }));
    await user.click(screen.getByRole('button', { name: 'Test channel' }));
    expect(screen.getByText('Test delivered')).toBeVisible();
    expect(mocks.client.POST).toHaveBeenCalledWith(
      '/notification-channels/{channelId}/test',
      {
        params: {
          path: { channelId: 'slack' },
          query: { remoteNode: 'local' },
        },
      }
    );
    expect(mocks.client.PUT).not.toHaveBeenCalled();
    expect(screen.getByText('Unsaved changes')).toBeVisible();
  });

  it('shows delivery failures', async () => {
    const user = userEvent.setup();
    mocks.client.POST.mockResolvedValue({
      data: {
        results: [{ delivered: false, error: 'Destination unavailable' }],
      },
    });
    renderRulesPage();
    await user.click(screen.getByRole('button', { name: 'Test channel' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Destination unavailable'
    );
  });

  it('configures workspace overrides independently and restores inheritance', async () => {
    const user = userEvent.setup();
    renderRulesPage([], [NotificationEventType.dag_run_failed], 'ops');
    await user.click(screen.getByLabelText('Applies to'));
    await user.click(screen.getByRole('option', { name: 'ops workspace' }));
    expect(screen.getByText('Inheriting Global rules')).toBeVisible();
    expect(screen.getByRole('checkbox', { name: 'Failed' })).toBeDisabled();
    await user.click(
      screen.getByRole('button', { name: 'Configure workspace' })
    );
    await user.click(screen.getByRole('checkbox', { name: 'Succeeded' }));
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    expect(mocks.client.PUT).toHaveBeenLastCalledWith(
      '/notification-routes/workspaces/{workspaceName}',
      expect.objectContaining({
        params: {
          path: { workspaceName: 'ops' },
          query: { remoteNode: 'local' },
        },
        body: expect.objectContaining({
          inheritGlobal: false,
          routes: [
            expect.objectContaining({
              events: ['dag.run.failed', 'dag.run.succeeded'],
            }),
          ],
        }),
      })
    );
    expect(screen.getByText('All changes saved')).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'Use Global rules' }));
    expect(
      screen.getByRole('checkbox', { name: 'Succeeded' })
    ).not.toBeChecked();
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    expect(mocks.client.PUT).toHaveBeenLastCalledWith(
      '/notification-routes/workspaces/{workspaceName}',
      expect.objectContaining({
        body: expect.objectContaining({ inheritGlobal: true }),
      })
    );
  });
});
