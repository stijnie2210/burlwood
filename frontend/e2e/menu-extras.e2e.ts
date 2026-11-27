import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect } from './harness';
import type { Page } from '@playwright/test';

function toast(page: Page) {
  return page.locator('.toast-stack');
}

async function openCommitMenu(page: Page, subject: string) {
  await page.locator('.row-inner').filter({ hasText: subject }).click({ button: 'right' });
  const menu = page.locator('.ctx-menu');
  await expect(menu).toBeVisible();
  return menu;
}

test.describe('Commit menu extras', () => {
  test('copies the full commit message', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'commit', '-q', '--allow-empty', '-m', 'the subject', '-m', 'the body');
    const page = await launch([repo]);
    await page.evaluate(() => {
      const w = window as unknown as { copied: string };
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: (text: string) => Promise.resolve(void (w.copied = text)) },
      });
    });

    const menu = await openCommitMenu(page, 'the subject');
    await menu.getByRole('menuitem', { name: 'Copy commit message' }).click();

    await expect(toast(page)).toContainText('Copied commit message');
    expect(await page.evaluate(() => (window as unknown as { copied: string }).copied)).toBe(
      'the subject\n\nthe body'
    );
  });

  test('deletes a tag through the sidebar confirmation', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'tag', 'v1');
    const page = await launch([repo]);

    const menu = await openCommitMenu(page, 'initial commit');
    await menu.getByRole('menuitem', { name: "Delete tag 'v1'…" }).click();

    const confirm = page.locator('.ctx-menu');
    await expect(confirm).toContainText('Delete the tag "v1"?');
    await expect(confirm.getByRole('button', { name: 'Cancel' })).toBeFocused();
    await confirm.getByRole('button', { name: 'Delete', exact: true }).click();

    await expect(toast(page)).toContainText('Deleted v1');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/tags/v1')).toBe(false);
  });
});

test.describe('Discard all', () => {
  test('asks first, discards every unstaged change, and Undo restores them', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'README.md'), '# edited\n');
    writeFileSync(join(repo, 'new.txt'), 'untracked\n');
    sandbox.commit(repo, 'kept.txt', 'kept\n', 'add kept');
    writeFileSync(join(repo, 'kept.txt'), 'staged\n');
    sandbox.git(repo, 'add', 'kept.txt');
    const page = await launch([repo]);

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await page.getByRole('button', { name: 'Discard All' }).click();
    const dialog = page.getByRole('alertdialog');
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();
    await dialog.getByRole('button', { name: 'Discard', exact: true }).click();

    await expect(toast(page)).toContainText('Discarded changes to 2 files');
    expect(readFileSync(join(repo, 'README.md'), 'utf8')).toBe('# test\n');
    expect(existsSync(join(repo, 'new.txt'))).toBe(false);
    expect(sandbox.git(repo, 'diff', '--cached', '--name-only')).toBe('kept.txt');

    await toast(page).getByRole('button', { name: 'Undo' }).click();

    await expect(toast(page)).toContainText('Restored 2 files');
    expect(readFileSync(join(repo, 'README.md'), 'utf8')).toBe('# edited\n');
    expect(readFileSync(join(repo, 'new.txt'), 'utf8')).toBe('untracked\n');
  });
});
