// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { createServer } from 'node:http';
import { loadStack, loginViaAPI, loginViaUI, uniqueName } from './helpers/e2e';

test('manages and tests notification channels on desktop and mobile', async ({
  page,
  request,
}) => {
  let deliveries = 0;
  const destination = createServer((req, res) => {
    req.resume();
    deliveries++;
    res.writeHead(204).end();
  });
  await new Promise<void>((resolve) =>
    destination.listen(0, '127.0.0.1', resolve)
  );
  const address = destination.address();
  if (!address || typeof address === 'string') {
    throw new Error('Test destination did not start');
  }
  const stack = await loadStack();
  const token = await loginViaAPI(
    request,
    stack.auth.adminUsername,
    stack.auth.adminPassword
  );
  const headers = { Authorization: `Bearer ${token}` };
  let channelId: string | undefined;
  const name = uniqueName('e2e-channel');

  try {
    await loginViaUI(page, stack.auth.adminUsername, stack.auth.adminPassword);
    await page.goto('/notification-channels');
    await page
      .getByRole('button', { name: 'Add channel', exact: true })
      .click();
    const editor = page.getByRole('dialog');
    await editor.getByLabel('Provider', { exact: true }).click();
    await page
      .getByRole('option', { name: 'Generic Webhook', exact: true })
      .click();
    await editor.getByLabel('Channel name').fill(name);
    await editor
      .getByLabel('Webhook endpoint URL')
      .fill(`http://127.0.0.1:${address.port}/notifications`);
    await editor
      .getByRole('checkbox', { name: 'Allow HTTP', exact: true })
      .check();
    await editor
      .getByRole('checkbox', { name: 'Allow private network', exact: true })
      .check();
    await editor
      .getByRole('switch', { name: 'Enabled', exact: true })
      .uncheck();
    const created = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        response.url().includes('/notification-channels?')
    );
    await editor
      .getByRole('button', { name: 'Create channel', exact: true })
      .click();
    const createdResponse = await created;
    expect(createdResponse.ok()).toBeTruthy();
    channelId = (await createdResponse.json()).id;
    await expect(editor).not.toBeVisible();

    let row = page.getByRole('listitem', { name, exact: true });
    await row.getByRole('button', { name: 'Test', exact: true }).click();
    await expect(row.getByRole('status')).toHaveText('Test delivered');
    expect(deliveries).toBe(1);
    await row.getByRole('button', { name: 'Edit', exact: true }).click();
    await editor.getByLabel('Channel name').fill(`${name}-updated`);
    await editor
      .getByRole('button', { name: 'Save changes', exact: true })
      .click();
    await expect(editor).not.toBeVisible();
    row = page.getByRole('listitem', { name: `${name}-updated`, exact: true });
    await row.getByRole('button', { name: 'Test', exact: true }).click();
    await expect(row.getByRole('status')).toHaveText('Test delivered');
    expect(deliveries).toBe(2);
    await row.getByRole('switch').click();
    await expect(row.getByRole('switch')).toBeChecked();
    await page.reload();
    await expect(row.getByRole('switch')).toBeChecked();

    await page
      .getByRole('searchbox', { name: 'Search channels' })
      .fill('no-matching-channel');
    await expect(
      page.getByText('No matching channels', { exact: true })
    ).toBeVisible();
    await page.getByRole('searchbox', { name: 'Search channels' }).fill(name);
    await expect(row).toBeVisible();

    await page.setViewportSize({ width: 390, height: 844 });
    await row.getByRole('button', { name: 'Edit', exact: true }).click();
    await expect(
      editor.getByRole('button', { name: 'Save changes', exact: true })
    ).toBeInViewport();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth
      )
    ).toBe(true);
    await editor.getByRole('button', { name: 'Cancel', exact: true }).click();
    await page.getByRole('button', { name: 'Configure', exact: true }).click();
    await expect(editor.getByPlaceholder('SMTP host')).toBeVisible();
    await editor.getByRole('button', { name: 'Cancel', exact: true }).click();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth
      )
    ).toBe(true);

    await row
      .getByRole('button', {
        name: `Channel actions for ${name}-updated`,
        exact: true,
      })
      .click();
    await page
      .getByRole('menuitem', { name: 'Delete channel', exact: true })
      .click();
    await editor.getByRole('button', { name: 'Delete', exact: true }).click();
    await expect(row).not.toBeVisible();
    channelId = undefined;
  } finally {
    if (channelId) {
      const deleted = await request.delete(
        `/api/v1/notification-channels/${channelId}?remoteNode=local`,
        { headers }
      );
      expect(deleted.ok()).toBeTruthy();
    }
    await new Promise<void>((resolve, reject) =>
      destination.close((error) => (error ? reject(error) : resolve()))
    );
  }
});
