import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
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

  test('keeps uncommitted and staged changes across the rebase', async ({ sandbox, launch }) => {
    const repo = fourCommits(sandbox);
    writeFileSync(join(repo, 'one.txt'), 'one, edited\n');
    writeFileSync(join(repo, 'README.md'), '# staged\n');
    sandbox.git(repo, 'add', 'README.md');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add three');
    await action(dialog, 'add four').selectOption('fixup');
    await dialog.getByRole('button', { name: 'Start rebase' }).click();

    await expect(toast(page)).toContainText('Rebased main');
    expect(history(sandbox, repo)).toBe('initial commit | add one | add two | add three');
    expect(readFileSync(join(repo, 'one.txt'), 'utf8')).toBe('one, edited\n');
    expect(readFileSync(join(repo, 'README.md'), 'utf8')).toBe('# staged\n');
    expect(sandbox.git(repo, 'status', '--porcelain')).toContain('one.txt');
  });

  test('squash keeps both messages, fixup drops one, reword keeps a body', async ({
    sandbox,
    launch,
  }) => {
    const repo = fourCommits(sandbox);
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add one');
    await action(dialog, 'add one').selectOption('reword');
    await dialog
      .getByRole('textbox', { name: /^New message for/ })
      .fill('one, reworded\n\nWith a body\nof two lines.');
    await action(dialog, 'add two').selectOption('squash');
    await action(dialog, 'add four').selectOption('fixup');
    await expect(dialog).toContainText('dropping this message');
    await dialog.getByRole('button', { name: 'Start rebase' }).click();

    await expect(toast(page)).toContainText('Rebased main');
    expect(history(sandbox, repo)).toBe('initial commit | one, reworded | add three');
    const first = sandbox.git(repo, 'log', '-1', '--format=%B', 'HEAD^');
    expect(first).toContain('With a body\nof two lines.');
    expect(first).toContain('add two');
    expect(sandbox.git(repo, 'log', '-1', '--format=%B', 'HEAD')).toBe('add three');
    expect(sandbox.git(repo, 'ls-tree', '--name-only', 'HEAD')).toContain('four.txt');
  });

  test('starts from the first commit of the repository', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.commit(repo, 'one.txt', 'one\n', 'add one');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'initial commit');
    await expect(dialog.getByRole('listitem')).toHaveCount(2);
    await action(dialog, 'add one').selectOption('fixup');
    await dialog.getByRole('button', { name: 'Start rebase' }).click();

    await expect(toast(page)).toContainText('Rebased main');
    expect(history(sandbox, repo)).toBe('initial commit');
    expect(sandbox.git(repo, 'ls-tree', '--name-only', 'HEAD')).toContain('one.txt');
  });

  test('refuses to drop every commit', async ({ sandbox, launch }) => {
    const repo = fourCommits(sandbox);
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add three');
    await action(dialog, 'add three').selectOption('drop');
    await action(dialog, 'add four').selectOption('drop');

    await expect(dialog.getByRole('status')).toContainText('Keep at least one commit');
    await expect(dialog.getByRole('button', { name: 'Start rebase' })).toBeDisabled();
  });

  test('refuses a reword without a message', async ({ sandbox, launch }) => {
    const repo = fourCommits(sandbox);
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add four');
    await action(dialog, 'add four').selectOption('reword');
    const message = dialog.getByRole('textbox', { name: /^New message for/ });
    await expect(message).toHaveValue('add four');
    await message.fill('   ');

    await expect(dialog.getByRole('status')).toContainText('needs a message');
    await expect(dialog.getByRole('button', { name: 'Start rebase' })).toBeDisabled();
  });

  test('works from the keyboard: focus starts inside, moves follow, Escape returns', async ({
    sandbox,
    launch,
  }) => {
    const repo = fourCommits(sandbox);
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add two');
    await expect(action(dialog, 'add two')).toBeFocused();

    const down = dialog.getByRole('button', { name: /^Move \w+ add two down$/ });
    await down.focus();
    await page.keyboard.press('Enter');
    await expect(down).toBeFocused();
    await page.keyboard.press('Space');
    await expect(dialog.getByRole('listitem')).toHaveText([/add three/, /add four/, /add two/]);
    await expect(down).toBeDisabled();
    await expect(dialog.getByRole('button', { name: /^Move \w+ add two up$/ })).toBeFocused();

    await page.keyboard.press('Escape');
    await expect(dialog).toHaveCount(0);
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
  });

  test('Cancel and a click outside close it without rebasing', async ({ sandbox, launch }) => {
    const repo = fourCommits(sandbox);
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    let dialog = await openEditor(page, 'add two');
    await action(dialog, 'add four').selectOption('drop');
    await dialog.getByRole('button', { name: 'Cancel' }).click();
    await expect(dialog).toHaveCount(0);

    dialog = await openEditor(page, 'add two');
    await expect(action(dialog, 'add four')).toHaveValue('pick');
    await page.mouse.click(5, 300);
    await expect(dialog).toHaveCount(0);
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
  });

  test('resolves conflicts step by step and finishes, with a later reword applied', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.commit(repo, 'README.md', 'first edit\n', 'first edit');
    sandbox.commit(repo, 'README.md', 'second edit\n', 'second edit');
    sandbox.commit(repo, 'other.txt', 'other\n', 'add other');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'first edit');
    await dialog.getByRole('button', { name: /^Move \w+ first edit down$/ }).click();
    await action(dialog, 'add other').selectOption('reword');
    await dialog.getByRole('textbox', { name: /^New message for/ }).fill('other, reworded');
    await dialog.getByRole('button', { name: 'Start rebase' }).click();
    await expect(toast(page)).toContainText('stopped on a conflict');

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await expect(toast(page)).not.toContainText('stopped on a conflict');
    for (const step of ['1/', '2/']) {
      await expect(page.locator('.rebase-step')).toContainText(step);
      await page.locator('.file-item--conflict').filter({ hasText: 'README.md' }).click();
      await page.getByRole('button', { name: 'Theirs for every conflict' }).click();
      await page.getByRole('button', { name: 'Save & Stage' }).click();
      await expect(page.locator('.file-item--conflict')).toHaveCount(0);
      await page.getByRole('button', { name: 'Continue Rebase' }).click();
    }

    await expect(toast(page)).toContainText('Rebase complete');
    await expect(page.locator('.rebase-banner')).toHaveCount(0);
    expect(history(sandbox, repo)).toBe(
      'initial commit | second edit | first edit | other, reworded'
    );
    expect(readFileSync(join(repo, 'README.md'), 'utf8')).toBe('first edit\n');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'REBASE_HEAD')).toBe(false);
  });

  test('a conflicted rebase can be aborted', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.commit(repo, 'README.md', 'first edit\n', 'first edit');
    sandbox.commit(repo, 'README.md', 'second edit\n', 'second edit');
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'first edit');
    await dialog.getByRole('button', { name: /^Move \w+ first edit down$/ }).click();
    await dialog.getByRole('button', { name: 'Start rebase' }).click();
    await expect(toast(page)).toContainText('stopped on a conflict');

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await page.getByRole('button', { name: 'Abort Rebase' }).click();
    await page.getByRole('button', { name: 'Yes, abort' }).click();

    await expect(toast(page)).toContainText('Rebase aborted');
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
    expect(sandbox.git(repo, 'status', '--porcelain')).toBe('');
  });

  test('refuses a range that contains a merge commit', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.commit(repo, 'one.txt', 'one\n', 'add one');
    sandbox.git(repo, 'checkout', '-q', '-b', 'side');
    sandbox.commit(repo, 'side.txt', 'side\n', 'side work');
    sandbox.git(repo, 'checkout', '-q', '-');
    sandbox.commit(repo, 'two.txt', 'two\n', 'add two');
    sandbox.git(repo, 'merge', '-q', '--no-ff', '--no-edit', 'side');
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    await page.locator('.row-inner').filter({ hasText: 'add one' }).click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Interactive rebase from here…' }).click();

    await expect(toast(page)).toContainText('merge commit');
    await expect(page.getByRole('dialog')).toHaveCount(0);
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);

    await page.locator('.row-inner').filter({ hasText: "Merge branch 'side'" }).click({
      button: 'right',
    });
    const item = page.getByRole('menuitem', { name: /^Interactive rebase from here/ });
    await expect(item).toHaveAttribute('aria-disabled', 'true');
    await expect(item).toContainText("A rebase can't start at a merge commit");
  });

  test('refuses a commit that is not on the current branch', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'side');
    sandbox.commit(repo, 'side.txt', 'side\n', 'side work');
    sandbox.git(repo, 'checkout', '-q', '-');
    sandbox.commit(repo, 'main.txt', 'main\n', 'main work');
    const page = await launch([repo]);

    await page.locator('.row-inner').filter({ hasText: 'side work' }).click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Interactive rebase from here…' }).click();

    await expect(toast(page)).toContainText("isn't part of the current branch");
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('is unavailable while a merge is in progress', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'side');
    sandbox.commit(repo, 'README.md', 'theirs\n', 'side edit');
    sandbox.git(repo, 'checkout', '-q', '-');
    sandbox.commit(repo, 'README.md', 'ours\n', 'main edit');
    sandbox.gitOk(repo, 'merge', 'side');
    const page = await launch([repo]);

    await page.locator('.row-inner').filter({ hasText: 'main edit' }).click({ button: 'right' });
    const item = page.getByRole('menuitem', { name: /^Interactive rebase from here/ });
    await expect(item).toHaveAttribute('aria-disabled', 'true');
    await expect(item).toContainText('Finish the merge in progress first');
  });

  test('says so when the branch changed while the editor was open', async ({ sandbox, launch }) => {
    const repo = fourCommits(sandbox);
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add three');
    sandbox.commit(repo, 'five.txt', 'five\n', 'add five');
    const head = sandbox.git(repo, 'rev-parse', 'HEAD');
    await action(dialog, 'add four').selectOption('drop');
    await dialog.getByRole('button', { name: 'Start rebase' }).click();

    await expect(toast(page)).toContainText('The branch has changed since the list was loaded');
    await expect(dialog).toBeVisible();
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(head);
  });

  test('Undo refuses once the branch has moved on', async ({ sandbox, launch }) => {
    const repo = fourCommits(sandbox);
    const page = await launch([repo]);

    const dialog = await openEditor(page, 'add three');
    await action(dialog, 'add four').selectOption('drop');
    await dialog.getByRole('button', { name: 'Start rebase' }).click();
    await expect(toast(page)).toContainText('Rebased main');

    sandbox.commit(repo, 'later.txt', 'later\n', 'later work');
    const head = sandbox.git(repo, 'rev-parse', 'HEAD');
    await toast(page).getByRole('button', { name: 'Undo' }).click();

    await expect(toast(page)).toContainText('The branch has moved since the rebase');
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(head);
  });
});
