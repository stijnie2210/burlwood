import { test, expect, type Sandbox } from './harness';
import type { Page } from '@playwright/test';

function toast(page: Page) {
  return page.locator('.toast-stack');
}

function fourCommits(sandbox: Sandbox) {
  const repo = sandbox.repo();
  for (const n of ['one', 'two', 'three', 'four']) {
    sandbox.commit(repo, `${n}.txt`, `${n}\n`, `add ${n}`);
  }
  return repo;
}

async function openEditor(page: Page, subject: string) {
  await page.locator('.row-inner').filter({ hasText: subject }).click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Interactive rebase from here…' }).click();
  const dialog = page.getByRole('dialog', { name: /^Interactive rebase of/ });
  await expect(dialog).toBeVisible();
  return dialog;
}

function action(dialog: ReturnType<Page['getByRole']>, subject: string) {
  return dialog.getByRole('combobox', { name: new RegExp(`^Action for \\w+ ${subject}$`) });
}

function history(sandbox: Sandbox, repo: string) {
  return sandbox.git(repo, 'log', '--reverse', '--format=%s').split('\n').join(' | ');
}

test.describe('Interactive rebase', () => {
  test('rewords, squashes, drops and reorders, and Undo puts the branch back', async ({
    sandbox,
    launch,
  }) => {
    const repo = fourCommits(sandbox);
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add one');
    await expect(dialog).toContainText('4 commits, oldest first');
    const start = dialog.getByRole('button', { name: 'Start rebase' });
    await expect(start).toBeDisabled();
    await expect(dialog.getByRole('status')).toHaveText('Nothing changed yet.');

    await action(dialog, 'add one').selectOption('reword');
    await dialog.getByRole('textbox', { name: /^New message for/ }).fill('add one and two');
    await action(dialog, 'add two').selectOption('squash');
    await expect(dialog).toContainText('keeping both messages');
    await action(dialog, 'add three').selectOption('drop');
    await dialog.getByRole('button', { name: /^Move \w+ add four up$/ }).click();
    await expect(dialog.getByRole('button', { name: /^Move \w+ add four up$/ })).toBeFocused();

    await expect(dialog.getByRole('listitem')).toHaveText([
      /add one/,
      /add two/,
      /add four/,
      /add three/,
    ]);
    await start.click();

    await expect(toast(page)).toContainText('Rebased main');
    expect(history(sandbox, repo)).toBe('initial commit | add one and two | add four');
    expect(sandbox.git(repo, 'log', '-1', '--format=%B', 'HEAD^')).toContain('add two');
    expect(sandbox.git(repo, 'ls-tree', '--name-only', 'HEAD').split('\n')).toEqual([
      'README.md',
      'four.txt',
      'one.txt',
      'two.txt',
    ]);

    await toast(page).getByRole('button', { name: 'Undo' }).click();
    await expect(toast(page)).toContainText('Rebase undone');
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
  });

  test('refuses a squash with nothing above it, and Cancel changes nothing', async ({
    sandbox,
    launch,
  }) => {
    const repo = fourCommits(sandbox);
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add three');
    await expect(dialog.getByRole('listitem')).toHaveCount(2);
    await action(dialog, 'add three').selectOption('fixup');

    await expect(dialog.getByRole('status')).toContainText('has no commit above it to fixup into');
    await expect(dialog.getByRole('button', { name: 'Start rebase' })).toBeDisabled();

    await page.keyboard.press('Escape');
    await expect(dialog).toHaveCount(0);
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
  });

  test('hands a conflict to the Working Tree', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.commit(repo, 'README.md', 'first edit\n', 'first edit');
    sandbox.commit(repo, 'README.md', 'second edit\n', 'second edit');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'first edit');
    await dialog.getByRole('button', { name: /^Move \w+ first edit down$/ }).click();
    await dialog.getByRole('button', { name: 'Start rebase' }).click();

    await expect(toast(page)).toContainText('The rebase of main stopped on a conflict');
    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await expect(page.locator('.rebase-banner')).toBeVisible();
    await expect(page.getByText('README.md').first()).toBeVisible();
  });

  test('warns only when pushed commits get rewritten', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.bareRemote(repo);
    sandbox.commit(repo, 'one.txt', 'one\n', 'add one');
    sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
    sandbox.commit(repo, 'two.txt', 'two\n', 'add two');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add one');
    await action(dialog, 'add two').selectOption('reword');
    await expect(dialog.getByText('force push')).toHaveCount(0);

    await action(dialog, 'add one').selectOption('reword');
    await expect(dialog).toContainText(
      'This rewrites 1 commit that is already on origin/main, so the next push has to be a force push.'
    );
  });
});
