// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { UserPreferencesProvider } from '@/contexts/UserPreference';
import { I18nProvider } from '@/i18n/I18nProvider';
import { useIsAdmin } from '@/contexts/AuthContext';
import { useSchedulerPause } from '@/hooks/useSchedulerPause';
import SchedulerPauseControl from '../SchedulerPauseControl';

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

function pauseState(paused: boolean, setPaused = vi.fn()) {
  return {
    paused,
    pausedBy: undefined,
    pausedAt: undefined,
    reason: undefined,
    error: undefined,
    isLoading: false,
    mutate: vi.fn(),
    setPaused,
  } as never;
}

function renderControl() {
  return render(
    <UserPreferencesProvider>
      <I18nProvider>
        <SchedulerPauseControl />
      </I18nProvider>
    </UserPreferencesProvider>
  );
}

describe('SchedulerPauseControl', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useIsAdminMock.mockReturnValue(true);
  });

  it('is hidden from non-admins', () => {
    useIsAdminMock.mockReturnValue(false);
    useSchedulerPauseMock.mockReturnValue(pauseState(false));

    const { container } = renderControl();

    expect(container).toBeEmptyDOMElement();
  });

  // Pausing drops queued scheduled runs and discards catch-up windows, so the
  // confirmation has to say so before the operator commits.
  it('warns about dropped runs before pausing', async () => {
    const setPaused = vi.fn().mockResolvedValue(undefined);
    useSchedulerPauseMock.mockReturnValue(pauseState(false, setPaused));

    renderControl();
    await userEvent.click(screen.getByRole('button', { name: /pause/i }));

    expect(
      screen.getByText(/Queued scheduled runs are dropped/i)
    ).toBeInTheDocument();
    expect(setPaused).not.toHaveBeenCalled();
  });

  it('resumes when already paused', async () => {
    const setPaused = vi.fn().mockResolvedValue(undefined);
    useSchedulerPauseMock.mockReturnValue(pauseState(true, setPaused));

    renderControl();
    await userEvent.click(screen.getByRole('button', { name: /resume/i }));
    const resumeButtons = screen.getAllByRole('button', { name: /^resume$/i });
    await userEvent.click(resumeButtons[resumeButtons.length - 1]!);

    await waitFor(() => expect(setPaused).toHaveBeenCalledWith(false));
  });
});
