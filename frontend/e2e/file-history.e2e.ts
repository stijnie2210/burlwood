import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect, type Sandbox } from './harness';
import type { Page } from '@playwright/test';

// notes.txt is added, edited, renamed to renamed.txt and edited again; the last commit is unrelated.
function historyRepo(sandbox: Sandbox) {
  const repo = sandbox.repo();
  sandbox.commit(repo, 'notes.txt', 'alpha\nbeta\ngamma\n', 'add notes');
  sandbox.commit(repo, 'notes.txt', 'alpha\nBETA\ngamma\n', 'shout beta');
  sandbox.git(repo, 'mv', 'notes.txt', 'renamed.txt');
  sandbox.git(repo, 'commit', '-q', '-m', 'rename notes');
  sandbox.commit(repo, 'renamed.txt', 'alpha\nBETA\ngamma\ndelta\n', 'add delta');
  sandbox.commit(repo, 'other.txt', 'other\n', 'unrelated');
  return repo;
}

function viewer(page: Page) {
  return page.getByRole('dialog', { name: 'renamed.txt' });
}

async function openFromWorkingTree(page: Page, file: string, item: 'File history' | 'Blame') {
  await page.getByRole('tab', { name: /^Working Tree/ }).click();
  await page.locator('.file-item').filter({ hasText: file }).click({ button: 'right' });
  await page.getByRole('menuitem', { name: item, exact: true }).click();
}

test.describe('File history', () => {
  test('lists the commits that changed a file, across its rename', async ({ sandbox, launch }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'renamed.txt'), 'alpha\nBETA\ngamma\ndelta\nlocal\n');
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'renamed.txt', 'File history');
    const dialog = viewer(page);
    await expect(dialog.getByRole('option')).toHaveText([
      /add delta/,
      /rename notes.*Renamed from notes\.txt/,
      /shout beta/,
      /add notes.*Added/,
    ]);
    await expect(dialog.getByRole('option').first()).toHaveAttribute('aria-selected', 'true');
    await expect(dialog.locator('.diff-line--add .diff-content')).toHaveText('delta');
  });

  test('shows each commit’s change to that file, by mouse and by arrow keys', async ({
    sandbox,
    launch,
  }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'renamed.txt'), 'changed\n');
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'renamed.txt', 'File history');
    const dialog = viewer(page);
    await dialog.getByRole('option', { name: /shout beta/ }).click();
    await expect(dialog.locator('.diff-line--del .diff-content')).toHaveText('beta');
    await expect(dialog.locator('.diff-line--add .diff-content')).toHaveText('BETA');
    await expect(dialog.locator('.fh-diff-path')).toHaveText('notes.txt');

    await dialog.getByRole('option', { name: /shout beta/ }).focus();
    await page.keyboard.press('ArrowUp');
    const rename = dialog.getByRole('option', { name: /rename notes/ });
    await expect(rename).toBeFocused();
    await expect(rename).toHaveAttribute('aria-selected', 'true');
    await expect(dialog).toContainText('the file was only renamed');

    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await expect(dialog.getByRole('option', { name: /add notes/ })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    await expect(dialog.locator('.diff-line--add .diff-content')).toHaveText([
      'alpha',
      'beta',
      'gamma',
    ]);
  });

  test('opens from a file in a commit, on that commit, and can show it in the graph', async ({
    sandbox,
    launch,
  }) => {
    const repo = historyRepo(sandbox);
    const page = await launch([repo]);

    await page.locator('.row-inner').filter({ hasText: 'shout beta' }).click();
    await page.getByText('notes.txt').first().click();
    await page.getByRole('button', { name: 'File history' }).click();

    const dialog = page.getByRole('dialog', { name: 'notes.txt' });
    await expect(dialog.getByRole('option')).toHaveText([
      /rename notes.*Deleted/,
      /shout beta/,
      /add notes.*Added/,
    ]);
    await expect(dialog.getByRole('option', { name: /shout beta/ })).toHaveAttribute(
      'aria-selected',
      'true'
    );

    await dialog.getByRole('option', { name: /add notes/ }).click();
    await dialog.getByRole('button', { name: 'Show in graph' }).click();
    await expect(dialog).toHaveCount(0);
    await expect(page.locator('.row-inner.selected')).toContainText('add notes');
  });

  test('Escape and the close button close it', async ({ sandbox, launch }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'renamed.txt'), 'changed\n');
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'renamed.txt', 'File history');
    await expect(viewer(page)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(viewer(page)).toHaveCount(0);

    await openFromWorkingTree(page, 'renamed.txt', 'File history');
    await viewer(page).getByRole('button', { name: 'Close' }).click();
    await expect(viewer(page)).toHaveCount(0);
  });

  test('is not offered on untracked files', async ({ sandbox, launch }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'brand-new.txt'), 'new\n');
    const page = await launch([repo]);

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await page.locator('.file-item').filter({ hasText: 'brand-new.txt' }).click({
      button: 'right',
    });
    await expect(page.getByRole('menuitem', { name: 'Copy path' })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'File history' })).toHaveCount(0);
    await expect(page.getByRole('menuitem', { name: 'Blame', exact: true })).toHaveCount(0);
  });
});

test.describe('Blame', () => {
  test('names the commit behind each line and marks uncommitted ones', async ({
    sandbox,
    launch,
  }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'renamed.txt'), 'alpha\nBETA\ngamma\ndelta\nlocal\n');
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'renamed.txt', 'Blame');
    const dialog = viewer(page);
    await expect(dialog.getByRole('tab', { name: 'Blame' })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    await expect(dialog).toContainText('Blame at the working tree');

    const chunks = dialog.locator('.blame-chunk');
    await expect(chunks).toHaveCount(5);
    await expect(chunks.nth(0)).toContainText('alpha');
    await expect(chunks.nth(0).getByRole('button')).toHaveAccessibleName(/add notes, Test/);
    await expect(chunks.nth(1)).toContainText('BETA');
    await expect(chunks.nth(1).getByRole('button')).toHaveAccessibleName(/shout beta/);
    await expect(chunks.nth(3).getByRole('button')).toHaveAccessibleName(/add delta/);
    await expect(chunks.nth(4)).toContainText('Not committed yet');
    await expect(chunks.nth(4)).toContainText('local');
    await expect(dialog.locator('.blame-num')).toHaveText(['1', '2', '3', '4', '5']);
  });

  test('clicking a line’s commit opens it in the history', async ({ sandbox, launch }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'renamed.txt'), 'alpha\nBETA\ngamma\ndelta\nlocal\n');
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'renamed.txt', 'Blame');
    const dialog = viewer(page);
    await dialog.locator('.blame-chunk').nth(1).getByRole('button').click();

    await expect(dialog.getByRole('tab', { name: 'History' })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    await expect(dialog.getByRole('option', { name: /shout beta/ })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    await expect(dialog.locator('.diff-line--add .diff-content')).toHaveText('BETA');
  });

  test('blames an older commit under the name the file had then, and back', async ({
    sandbox,
    launch,
  }) => {
    const repo = historyRepo(sandbox);
    writeFileSync(join(repo, 'renamed.txt'), 'alpha\nBETA\ngamma\ndelta\nlocal\n');
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'renamed.txt', 'File history');
    const dialog = viewer(page);
    await dialog.getByRole('option', { name: /add notes/ }).click();
    await dialog.getByRole('button', { name: 'Blame at this commit' }).click();

    await expect(dialog).toContainText(/Blame at \w{7} add notes/);
    await expect(dialog.locator('.blame-chunk')).toHaveCount(1);
    await expect(dialog.locator('.blame-code')).toHaveText(['alpha', 'beta', 'gamma']);

    await dialog.getByRole('button', { name: 'Blame the working tree' }).click();
    await expect(dialog).toContainText('Blame at the working tree');
    await expect(dialog.locator('.blame-chunk')).toHaveCount(5);
  });

  test('opens from a file in a commit, at that commit', async ({ sandbox, launch }) => {
    const repo = historyRepo(sandbox);
    const page = await launch([repo]);

    await page.locator('.row-inner').filter({ hasText: 'shout beta' }).click();
    await page.getByText('notes.txt').first().click();
    await page.getByRole('button', { name: 'Blame', exact: true }).click();

    const dialog = page.getByRole('dialog', { name: 'notes.txt' });
    await expect(dialog).toContainText(/Blame at \w{7} shout beta/);
    await expect(dialog.locator('.blame-code')).toHaveText(['alpha', 'BETA', 'gamma']);
    await expect(dialog.locator('.blame-chunk')).toHaveCount(3);
  });

  test('explains that a binary file has no lines to blame', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'blob.bin'), Buffer.from([1, 0, 2, 0, 3, 10]));
    sandbox.git(repo, 'add', 'blob.bin');
    sandbox.git(repo, 'commit', '-q', '-m', 'add blob');
    writeFileSync(join(repo, 'blob.bin'), Buffer.from([1, 0, 2, 0, 4, 10]));
    const page = await launch([repo]);

    await openFromWorkingTree(page, 'blob.bin', 'Blame');
    await expect(page.getByRole('dialog', { name: 'blob.bin' }).getByRole('alert')).toContainText(
      'binary file'
    );
  });
});
