import { test, expect, type Sandbox } from './harness';
import type { Page } from '@playwright/test';

function remoteBranch(page: Page, name: string) {
  return page.locator('[data-list="remote"]').filter({ hasText: new RegExp(`^\\s*${name}\\s*$`) });
}

function toast(page: Page) {
  return page.locator('.toast-stack');
}

function pushedFeature(sandbox: Sandbox) {
  const repo = sandbox.repo();
  const remote = sandbox.bareRemote(repo);
  sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
  sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
  sandbox.commit(repo, 'feature.txt', 'feature\n', 'feature work');
  sandbox.git(repo, 'push', '-q', '-u', 'origin', 'feature');
  sandbox.git(repo, 'checkout', '-q', '-');
  return { repo, remote };
}

test.describe('Delete remote branch', () => {
  test('deletes it from origin, keeps the local branch, and Undo pushes it back', async ({
    sandbox,
    launch,
  }) => {
    const { repo, remote } = pushedFeature(sandbox);
    const hash = sandbox.git(repo, 'rev-parse', 'feature');
    const page = await launch([repo]);

    await remoteBranch(page, 'feature').click({ button: 'right' });
    const menu = page.locator('.ctx-menu');
    await menu.getByRole('menuitem', { name: 'Delete from origin…' }).click();
    await expect(menu.getByRole('button', { name: 'Cancel' })).toBeFocused();
    await menu.getByRole('button', { name: 'Delete', exact: true }).click();

    await expect(toast(page)).toContainText('Deleted feature from origin');
    await expect(remoteBranch(page, 'feature')).toHaveCount(0);
    expect(sandbox.gitOk(remote, 'rev-parse', '--verify', '--quiet', 'refs/heads/feature')).toBe(
      false
    );
    expect(sandbox.git(repo, 'rev-parse', 'feature')).toBe(hash);

    await toast(page).getByRole('button', { name: 'Undo' }).click();

    await expect(toast(page)).toContainText('Pushed feature back to origin');
    await expect(remoteBranch(page, 'feature')).toBeVisible();
    expect(sandbox.git(remote, 'rev-parse', 'refs/heads/feature')).toBe(hash);
  });

  test('removes a stale ref when origin no longer has the branch', async ({ sandbox, launch }) => {
    const { repo, remote } = pushedFeature(sandbox);
    sandbox.git(remote, 'branch', '-D', 'feature');
    const page = await launch([repo]);

    await remoteBranch(page, 'feature').click({ button: 'right' });
    const menu = page.locator('.ctx-menu');
    await menu.getByRole('menuitem', { name: 'Delete from origin…' }).click();
    await menu.getByRole('button', { name: 'Delete', exact: true }).click();

    await expect(toast(page)).toContainText('origin no longer had feature');
    await expect(remoteBranch(page, 'feature')).toHaveCount(0);
    expect(
      sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/remotes/origin/feature')
    ).toBe(false);
  });

  test('is not offered on local branches', async ({ sandbox, launch }) => {
    const { repo } = pushedFeature(sandbox);
    const page = await launch([repo]);

    await page
      .locator('[data-list="local"]')
      .filter({ hasText: /^\s*feature\s*$/ })
      .click({ button: 'right' });
    const menu = page.locator('.ctx-menu');
    await expect(menu.getByRole('menuitem', { name: 'Delete…' })).toBeVisible();
    await expect(menu.getByRole('menuitem', { name: /^Delete from/ })).toHaveCount(0);
  });
});
