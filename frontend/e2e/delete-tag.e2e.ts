import { test, expect } from './harness';
import type { Page } from '@playwright/test';

function tagItem(page: Page, name: string) {
  return page.locator('[data-list="tag"]').filter({ hasText: new RegExp(`^\\s*${name}\\s*$`) });
}

async function openTagMenu(page: Page, name: string) {
  await tagItem(page, name).click({ button: 'right' });
  const menu = page.locator('.ctx-menu');
  await expect(menu.getByRole('menuitem', { name: 'Delete…' })).toBeVisible();
  return menu;
}

function toast(page: Page) {
  return page.locator('.toast-stack');
}

test.describe('Delete tag', () => {
  test('deletes a tag locally, with Cancel focused first, and Undo restores it', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'tag', '-a', 'v1', '-m', 'first release');
    const target = sandbox.git(repo, 'rev-parse', 'refs/tags/v1');
    const page = await launch([repo]);

    const menu = await openTagMenu(page, 'v1');
    await expect(menu.getByRole('menuitem', { name: 'Push to origin' })).toHaveCount(0);
    await menu.getByRole('menuitem', { name: 'Delete…' }).click();
    await expect(menu.getByRole('button', { name: 'Cancel' })).toBeFocused();
    await expect(menu.getByRole('checkbox')).toHaveCount(0);
    await menu.getByRole('button', { name: 'Delete', exact: true }).click();

    await expect(toast(page)).toContainText('Deleted v1');
    await expect(tagItem(page, 'v1')).toHaveCount(0);
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/tags/v1')).toBe(false);

    await toast(page).getByRole('button', { name: 'Undo' }).click();

    await expect(toast(page)).toContainText('Restored v1');
    await expect(tagItem(page, 'v1')).toBeVisible();
    expect(sandbox.git(repo, 'rev-parse', 'refs/tags/v1')).toBe(target);
    expect(sandbox.git(repo, 'tag', '-l', '--format=%(contents:subject)', 'v1')).toBe(
      'first release'
    );
  });

  test('also deletes it from origin when asked', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    const remote = sandbox.bareRemote(repo);
    sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
    sandbox.git(repo, 'tag', 'v1');
    sandbox.git(repo, 'push', '-q', 'origin', 'refs/tags/v1');
    const page = await launch([repo]);

    const menu = await openTagMenu(page, 'v1');
    await menu.getByRole('menuitem', { name: 'Delete…' }).click();
    await menu.getByRole('checkbox', { name: 'Also delete it from origin' }).check();
    await menu.getByRole('button', { name: 'Delete', exact: true }).click();

    await expect(toast(page)).toContainText('Deleted v1 here and on origin');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/tags/v1')).toBe(false);
    expect(sandbox.gitOk(remote, 'rev-parse', '--verify', '--quiet', 'refs/tags/v1')).toBe(false);
  });

  test('leaves origin alone unless asked', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    const remote = sandbox.bareRemote(repo);
    sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
    sandbox.git(repo, 'tag', 'v1');
    sandbox.git(repo, 'push', '-q', 'origin', 'refs/tags/v1');
    const page = await launch([repo]);

    const menu = await openTagMenu(page, 'v1');
    await menu.getByRole('menuitem', { name: 'Delete…' }).click();
    await expect(
      menu.getByRole('checkbox', { name: 'Also delete it from origin' })
    ).not.toBeChecked();
    await menu.getByRole('button', { name: 'Delete', exact: true }).click();

    await expect(toast(page)).toContainText('Deleted v1');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/tags/v1')).toBe(false);
    expect(sandbox.gitOk(remote, 'rev-parse', '--verify', '--quiet', 'refs/tags/v1')).toBe(true);
  });
});
