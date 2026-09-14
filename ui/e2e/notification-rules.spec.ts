// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import {
  loadStack,
  loginViaAPI,
  loginViaUI,
  uniqueName,
  useDefaultWorkspaceScope,
} from './helpers/e2e';

test('edits notification events and adds a route', async ({
  page,
  request,
}) => {
  const stack = await loadStack();
  const token = await loginViaAPI(
    request,
    stack.auth.adminUsername,
    stack.auth.adminPassword
  );
  const headers = { Authorization: `Bearer ${token}` };
  const routesURL = '/api/v1/notification-routes/global?remoteNode=local';
  const originalResponse = await request.get(routesURL, { headers });
  expect(originalResponse.ok()).toBeTruthy();
  const originalRoutes = await originalResponse.json();
  const channels: Array<{ id: string; name: string }> = [];

  try {
    for (let index = 0; index < 2; index++) {
      const response = await request.post(
        '/api/v1/notification-channels?remoteNode=local',
        {
          headers,
          data: {
            name: uniqueName('e2e-notification'),
            type: 'webhook',
            // Keep test destinations from receiving workflow notifications.
            enabled: false,
            webhook: { url: 'https://example.com/notifications' },
          },
        }
      );
      expect(response.ok()).toBeTruthy();
      channels.push(await response.json());
    }
    const initialResponse = await request.put(routesURL, {
      headers,
      data: {
        enabled: true,
        inheritGlobal: true,
        routes: [
          {
            channelId: channels[0]!.id,
            enabled: true,
            events: ['dag.run.rejected'],
          },
        ],
      },
    });
    expect(initialResponse.ok()).toBeTruthy();

    await loginViaUI(page, stack.auth.adminUsername, stack.auth.adminPassword);
    await useDefaultWorkspaceScope(page);
    await page.goto('/notification-rules');

    const rejected = page.getByRole('checkbox', {
      name: 'Rejected',
      exact: true,
    });
    await rejected.uncheck();
    await expect(rejected).not.toBeChecked();
    await expect(page.getByRole('alert')).toHaveText(
      'Select at least one event before saving.'
    );
    await expect(
      page.getByRole('button', { name: 'Save changes' })
    ).toBeDisabled();

    const failed = page.getByRole('checkbox', { name: 'Failed', exact: true });
    await failed.press('Space');
    await expect(failed).toBeChecked();
    await page.getByRole('button', { name: 'Add rule', exact: true }).click();
    await page
      .getByRole('dialog')
      .getByRole('button', { name: new RegExp(channels[1]!.name) })
      .click();
    await expect(
      page.getByRole('switch', {
        name: `Toggle ${channels[1]!.name}`,
        exact: true,
      })
    ).toBeChecked();
    await page.getByRole('button', { name: 'Save changes' }).click();
    await expect(
      page.getByText('Global rules saved', { exact: true })
    ).toBeVisible();

    await page.reload();
    await expect(
      page.getByRole('checkbox', { name: 'Failed', exact: true }).first()
    ).toBeChecked();
    await expect(
      page.getByRole('checkbox', { name: 'Rejected', exact: true }).first()
    ).not.toBeChecked();
    await expect(
      page.getByRole('switch', {
        name: `Toggle ${channels[1]!.name}`,
        exact: true,
      })
    ).toBeChecked();
    await expect(
      page.getByText('All changes saved', { exact: true })
    ).toBeVisible();
    await page.getByRole('button', { name: 'Add rule', exact: true }).click();
    await expect(
      page
        .getByRole('dialog')
        .getByRole('link', { name: 'Add channel', exact: true })
    ).toBeVisible();
    await page.getByRole('button', { name: 'Close', exact: true }).click();

    await page.setViewportSize({ width: 390, height: 844 });
    await expect(
      page.getByRole('checkbox', { name: 'Failed', exact: true }).first()
    ).toBeVisible();
    await page
      .getByRole('checkbox', { name: 'Failed', exact: true })
      .first()
      .uncheck();
    await page.getByRole('button', { name: 'Cancel', exact: true }).click();
    await expect(
      page.getByRole('checkbox', { name: 'Failed', exact: true }).first()
    ).toBeChecked();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth
      )
    ).toBe(true);

    const savedResponse = await request.get(routesURL, { headers });
    const savedRoutes = await savedResponse.json();
    expect(savedRoutes.routes[0].events).toEqual(['dag.run.failed']);
    expect(savedRoutes.routes[1].channelId).toBe(channels[1]!.id);
  } finally {
    // Restore routes before deleting channels they may still reference.
    const results = await Promise.allSettled([
      request.put(routesURL, { headers, data: originalRoutes }),
    ]);
    results.push(
      ...(await Promise.allSettled(
        channels.map((channel) =>
          request.delete(
            `/api/v1/notification-channels/${channel.id}?remoteNode=local`,
            { headers }
          )
        )
      ))
    );
    for (const result of results) {
      expect.soft(result).toMatchObject({ status: 'fulfilled' });
      if (result.status === 'fulfilled') {
        expect
          .soft(
            result.value.ok(),
            `${result.value.url()}: HTTP ${result.value.status()}`
          )
          .toBeTruthy();
      }
    }
  }
});
