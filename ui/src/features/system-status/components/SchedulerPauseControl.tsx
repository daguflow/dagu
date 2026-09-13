// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Admin control for pausing and resuming scheduler-driven run creation.
 *
 * @module features/system-status/components
 */
import ConfirmModal from '@/components/ui/confirm-dialog';
import { Button } from '@/components/ui/button';
import { useErrorModal } from '@/components/ui/error-modal';
import { useIsAdmin } from '@/contexts/AuthContext';
import { useSchedulerPause } from '@/hooks/useSchedulerPause';
import { I18nProps } from '@/i18n/I18nProps';
import { I18nText } from '@/i18n/I18nText';
import { useI18n } from '@/i18n/I18nProvider';
import { cn } from '@/lib/utils';
import { PauseCircle, PlayCircle } from 'lucide-react';
import * as React from 'react';

function SchedulerPauseControl() {
  const { ts } = useI18n();
  const isAdmin = useIsAdmin();
  const { showError } = useErrorModal();
  const { paused, setPaused } = useSchedulerPause();
  const [showConfirm, setShowConfirm] = React.useState(false);
  const [submitting, setSubmitting] = React.useState(false);

  const handleConfirm = React.useCallback(async () => {
    setShowConfirm(false);
    setSubmitting(true);
    try {
      await setPaused(!paused);
    } catch (err) {
      showError(
        err instanceof Error
          ? err.message
          : ts('Failed to update the scheduler pause state'),
        ts('Please try again or check the server connection.')
      );
    } finally {
      setSubmitting(false);
    }
  }, [paused, setPaused, showError, ts]);

  if (!isAdmin) {
    return null;
  }

  return (
    <>
      <Button
        onClick={() => setShowConfirm(true)}
        disabled={submitting}
        aria-label={ts(paused ? 'Resume the scheduler' : 'Pause the scheduler')}
        title={ts(
          paused
            ? 'Resume scheduler-driven run creation'
            : 'Pause scheduler-driven run creation'
        )}
      >
        {paused ? (
          <PlayCircle className={cn('h-4 w-4', 'text-success')} />
        ) : (
          <PauseCircle className="h-4 w-4" />
        )}
        {paused ? <I18nText text={'Resume'} /> : <I18nText text={'Pause'} />}
      </Button>
      <I18nProps>
        <ConfirmModal
          title={paused ? 'Resume Scheduler' : 'Pause Scheduler'}
          buttonText={paused ? 'Resume' : 'Pause'}
          visible={showConfirm}
          dismissModal={() => setShowConfirm(false)}
          onSubmit={handleConfirm}
        >
          {paused ? (
            <p>
              <I18nText
                text={
                  'Resume scheduled runs for every DAG? Schedules pick up from the next tick.'
                }
              />
            </p>
          ) : (
            <>
              <p>
                <I18nText
                  text={
                    'Pause scheduled runs for every DAG? Manual, webhook, and sub-DAG runs keep working.'
                  }
                />
              </p>
              <p className="mt-2 text-warning">
                <I18nText
                  text={
                    'Queued scheduled runs are dropped and catch-up windows are discarded. Nothing replays on resume.'
                  }
                />
              </p>
            </>
          )}
        </ConfirmModal>
      </I18nProps>
    </>
  );
}

export default SchedulerPauseControl;
