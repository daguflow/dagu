// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

/**
 * App-wide notice shown while scheduler-driven run creation is paused.
 *
 * @module components
 */
import { useIsAdmin } from '@/contexts/AuthContext';
import { useErrorModal } from '@/components/ui/error-modal';
import { useSchedulerPause } from '@/hooks/useSchedulerPause';
import { I18nProps } from '@/i18n/I18nProps';
import { I18nText } from '@/i18n/I18nText';
import { useI18n } from '@/i18n/I18nProvider';
import { PauseCircle } from 'lucide-react';
import * as React from 'react';

export function SchedulerPauseBanner() {
  const { ts } = useI18n();
  const isAdmin = useIsAdmin();
  const { showError } = useErrorModal();
  const { paused, pausedBy, reason, setPaused } = useSchedulerPause();
  const [resuming, setResuming] = React.useState(false);

  const handleResume = React.useCallback(async () => {
    setResuming(true);
    try {
      await setPaused(false);
    } catch (err) {
      showError(
        err instanceof Error
          ? err.message
          : ts('Failed to resume the scheduler'),
        ts('Please try again or check the server connection.')
      );
    } finally {
      setResuming(false);
    }
  }, [setPaused, showError, ts]);

  if (!paused) {
    return null;
  }

  const detail = [
    pausedBy ? ts('paused by {user}', { user: pausedBy }) : '',
    reason,
  ]
    .filter(Boolean)
    .join(' — ');

  return (
    <div
      role="alert"
      className="bg-amber-50 dark:bg-amber-950 border-b border-amber-200 dark:border-amber-800 px-4 py-1.5 flex items-center justify-between gap-3 text-sm"
    >
      <span className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
        <PauseCircle className="h-4 w-4 shrink-0" />
        <span>
          <I18nText
            text={'Scheduler is paused. No scheduled runs will start.'}
          />
          {detail ? ` ${detail}` : ''}
        </span>
      </span>
      {isAdmin ? (
        <I18nProps>
          <button
            onClick={handleResume}
            disabled={resuming}
            className="shrink-0 underline hover:no-underline text-amber-900 dark:text-amber-100 disabled:opacity-50"
            aria-label="Resume the scheduler"
          >
            <I18nText text={'Resume'} />
          </button>
        </I18nProps>
      ) : null}
    </div>
  );
}
