// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { I18nProvider } from '@/i18n/I18nProvider';
import { UserPreferencesProvider } from '@/contexts/UserPreference';
import { useIsAdmin } from '@/contexts/AuthContext';
import { useSchedulerPause } from '@/hooks/useSchedulerPause';
import { SchedulerPauseBanner } from '../SchedulerPauseBanner';

vi.mock('@/hooks/useSchedulerPause', () => ({
  useSchedulerPause: vi.fn(),
}));

vi.mock('@/contexts/AuthContext', () => ({
  useIsAdmin: vi.fn(),
}));

const showError = vi.fn();
vi.mock('@/components/ui/error-modal', () => ({
  useErrorModal: () => ({ showError }),
}));

const useSchedulerPauseMock = vi.mocked(useSchedulerPause);
const useIsAdminMock = vi.mocked(useIsAdmin);

function pauseState(overrides: Record<string, unknown> = {}) {
  return {
    paused: true,
    pausedBy: 'admin',
    pausedAt: '2026-04-27T10:00:00Z',
    reason: 'db migration',
    error: undefined,
    isLoading: false,
    mutate: vi.fn(),
    setPaused: vi.fn(),
    ...overrides,
  } as never;
}

function renderBanner() {
  return render(
    <UserPreferencesProvider>
      <I18nProvider>
        <SchedulerPauseBanner />
      </I18nProvider>
    </UserPreferencesProvider>
  );
}

describe('SchedulerPauseBanner', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useIsAdminMock.mockReturnValue(true);
  });

  it('renders nothing while the scheduler is running', () => {
    useSchedulerPauseMock.mockReturnValue(pauseState({ paused: false }));

    const { container } = renderBanner();

    expect(container).toBeEmptyDOMElement();
  });

  it('explains who paused the scheduler and why', () => {
    useSchedulerPauseMock.mockReturnValue(pauseState());

    renderBanner();

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Scheduler is paused. No scheduled runs will start.'
    );
    expect(screen.getByRole('alert')).toHaveTextContent('paused by admin');
    expect(screen.getByRole('alert')).toHaveTextContent('db migration');
  });

  it('lets an admin resume from the banner', async () => {
    const setPaused = vi.fn().mockResolvedValue(undefined);
    useSchedulerPauseMock.mockReturnValue(pauseState({ setPaused }));

    renderBanner();
    await userEvent.click(screen.getByRole('button', { name: /resume/i }));

    await waitFor(() => expect(setPaused).toHaveBeenCalledWith(false));
  });

  it('hides the resume action from non-admins', () => {
    useIsAdminMock.mockReturnValue(false);
    useSchedulerPauseMock.mockReturnValue(pauseState());

    renderBanner();

    expect(screen.queryByRole('button')).toBeNull();
    expect(screen.getByRole('alert')).toBeInTheDocument();
  });

  it('surfaces a failed resume', async () => {
    const setPaused = vi.fn().mockRejectedValue(new Error('nope'));
    useSchedulerPauseMock.mockReturnValue(pauseState({ setPaused }));

    renderBanner();
    await userEvent.click(screen.getByRole('button', { name: /resume/i }));

    await waitFor(() => expect(showError).toHaveBeenCalled());
  });
});
