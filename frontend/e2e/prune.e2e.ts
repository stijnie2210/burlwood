import { test, expect, type Sandbox } from './harness';
import type { Page } from '@playwright/test';

function remoteBranch(page: Page, name: string) {
  return page.locator('[data-list="remote"]').filter({ hasText: new RegExp(`^\\s*${name}\\s*$`) });
}

function toast(page: Page) {
  return page.locator('.toast-stack');
}

function repoWithGoneBranch(sandbox: Sandbox) {
  const repo = sandbox.repo();
  const remote = sandbox.bareRemote(repo);
  sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
  sandbox.git(repo, 'push', '-q', 'origin', 'HEAD:refs/heads/gone');
  sandbox.git(repo, 'fetch', '-q', 'origin');
  sandbox.git(remote, 'branch', '-D', 'gone');
  return { repo, remote };
}

function tracksGone(sandbox: Sandbox, repo: string) {
  return sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/remotes/origin/gone');
}

test.describe('Prune', () => {
  test('the remote menu removes branches the remote no longer has', async ({ sandbox, launch }) => {
    const { repo } = repoWithGoneBranch(sandbox);
    const page = await launch([repo]);
    await expect(remoteBranch(page, 'gone')).toBeVisible();

    await page.getByRole('button', { name: /^origin/ }).click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Remove branches deleted on origin' }).click();

    await expect(toast(page)).toContainText('Removed origin/gone, which origin no longer has');
    await expect(remoteBranch(page, 'gone')).toHaveCount(0);
    await expect(remoteBranch(page, 'main')).toBeVisible();
    expect(tracksGone(sandbox, repo)).toBe(false);
  });

  test('the remote menu fetches that remote', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    const remote = sandbox.bareRemote(repo);
    sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
    const page = await launch([repo]);
    sandbox.git(repo, 'push', '-q', 'origin', 'HEAD:refs/heads/new-on-origin');
    sandbox.git(repo, 'update-ref', '-d', 'refs/remotes/origin/new-on-origin');
    expect(sandbox.gitOk(remote, 'rev-parse', '--verify', '--quiet', 'new-on-origin')).toBe(true);

    await page.getByRole('button', { name: /^origin/ }).click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Fetch origin' }).click();

    await expect(toast(page)).toContainText('Fetched origin');
    await expect(remoteBranch(page, 'new-on-origin')).toBeVisible();
  });

  test('fetching keeps them by default', async ({ sandbox, launch }) => {
    const { repo } = repoWithGoneBranch(sandbox);
    const page = await launch([repo]);
    await page.getByRole('button', { name: 'Fetch', exact: true }).first().click();
    await expect(page.getByText('not fetched yet')).toHaveCount(0);
    expect(tracksGone(sandbox, repo)).toBe(true);
    await expect(remoteBranch(page, 'gone')).toBeVisible();
  });

  test('the setting turns pruning on', async ({ sandbox, launch }) => {
    const { repo } = repoWithGoneBranch(sandbox);
    const page = await launch([repo]);

    await page.getByRole('button', { name: 'Settings' }).click();
    const toggle = page.getByRole('checkbox', { name: 'Remove branches deleted on the remote' });
    await expect(toggle).not.toBeChecked();
    await toggle.focus();
    await page.keyboard.press('Space');
    await expect(toggle).toBeChecked();
    await page.getByRole('button', { name: 'Save' }).click();

    await page.getByRole('button', { name: 'Fetch', exact: true }).first().click();
    await expect(remoteBranch(page, 'gone')).toHaveCount(0);
    expect(tracksGone(sandbox, repo)).toBe(false);
  });
});
