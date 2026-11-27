import { test, expect } from './harness';
import type { Page } from '@playwright/test';

async function openCommitMenu(page: Page, subject: string) {
  await page.locator('.row-inner').filter({ hasText: subject }).click({ button: 'right' });
  const menu = page.locator('.ctx-menu');
  await expect(menu).toBeVisible();
  return menu;
}

function toast(page: Page) {
  return page.locator('.toast-stack');
}

test.describe('Cherry-pick', () => {
  test('copies a commit from another branch onto the current one', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'fix.txt', 'fix\n', 'the fix');
    sandbox.git(repo, 'checkout', '-q', 'main');
    const page = await launch([repo]);

    const menu = await openCommitMenu(page, 'the fix');
    await menu.getByRole('menuitem', { name: 'Cherry-pick commit…' }).click();
    await menu.getByRole('button', { name: /^Cherry-pick and commit/ }).click();

    await expect(toast(page)).toContainText('onto main');
    expect(sandbox.git(repo, 'log', '-1', '--format=%s', 'main')).toBe('the fix');
    expect(sandbox.git(repo, 'rev-parse', 'main')).not.toBe(
      sandbox.git(repo, 'rev-parse', 'feature')
    );
  });

  test('says so when the branch already has the changes', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'fix.txt', 'fix\n', 'the fix');
    sandbox.git(repo, 'checkout', '-q', 'main');
    // Without a commit of main's own, the pick can come out byte-identical to the original.
    sandbox.commit(repo, 'main.txt', 'main\n', 'main work');
    sandbox.git(repo, 'cherry-pick', 'feature');
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    await page
      .locator('.row-inner')
      .filter({ hasText: 'the fix' })
      .filter({ hasText: 'feature' })
      .click({ button: 'right' });
    const menu = page.locator('.ctx-menu');
    await menu.getByRole('menuitem', { name: 'Cherry-pick commit…' }).click();
    await menu.getByRole('button', { name: /^Cherry-pick and commit/ }).click();

    await expect(toast(page)).toContainText('main already has the changes from');
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'CHERRY_PICK_HEAD')).toBe(false);
  });

  test('hands a conflict to the Working Tree, where it can be aborted', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'README.md', 'theirs\n', 'feature edit');
    sandbox.git(repo, 'checkout', '-q', 'main');
    sandbox.commit(repo, 'README.md', 'ours\n', 'main edit');
    const head = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    let menu = await openCommitMenu(page, 'feature edit');
    await menu.getByRole('menuitem', { name: 'Cherry-pick commit…' }).click();
    await menu.getByRole('button', { name: /^Cherry-pick and commit/ }).click();

    await expect(toast(page)).toContainText('Cherry-picking');
    await expect(toast(page)).toContainText('stopped on a conflict');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'CHERRY_PICK_HEAD')).toBe(true);

    menu = await openCommitMenu(page, 'feature edit');
    const pick = menu.getByRole('menuitem', { name: /^Cherry-pick commit/ });
    await expect(pick).toHaveAttribute('aria-disabled', 'true');
    await expect(pick).toContainText('Finish the merge in progress first');
    await page.keyboard.press('Escape');

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await expect(page.getByText('Cherry-pick in progress')).toBeVisible();
    await page.getByRole('button', { name: 'Abort Cherry-pick' }).click();
    await page.getByRole('button', { name: 'Yes, abort' }).click();

    await expect(toast(page)).toContainText('Cherry-pick aborted');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'CHERRY_PICK_HEAD')).toBe(false);
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(head);
  });

  test('is unavailable on the current commit', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    const page = await launch([repo]);

    const menu = await openCommitMenu(page, 'initial commit');
    const pick = menu.getByRole('menuitem', { name: /^Cherry-pick commit/ });
    await expect(pick).toHaveAttribute('aria-disabled', 'true');
    await expect(pick).toContainText('This is the current commit');
  });
});
