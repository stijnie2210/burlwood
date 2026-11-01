import { test, expect, type Sandbox } from './harness';
import type { Page } from '@playwright/test';
import { existsSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

function localBranch(page: Page, name: string) {
  return page.locator('[data-list="local"]').filter({ hasText: new RegExp(`^\\s*${name}(\\s|$)`) });
}

function remoteBranch(page: Page, name: string) {
  return page.locator('[data-list="remote"]').filter({ hasText: new RegExp(`^\\s*${name}\\s*$`) });
}

async function openMenu(page: Page, branch: ReturnType<typeof localBranch>) {
  await branch.click({ button: 'right' });
  await expect(page.getByRole('menu')).toBeVisible();
  return page.locator('.ctx-menu');
}

const toast = (page: Page) => page.locator('.toast-stack');

function item(menu: ReturnType<Page['locator']>, name: string | RegExp) {
  return menu.getByRole('menuitem', typeof name === 'string' ? { exact: true, name } : { name });
}

// main is pushed with two commits that base doesn't have; base has moved on by one.
function pushedMainAndBase(sandbox: Sandbox) {
  const repo = sandbox.repo();
  sandbox.bareRemote(repo);
  sandbox.git(repo, 'push', '-q', '-u', 'origin', 'main');
  sandbox.git(repo, 'branch', 'base');
  sandbox.commit(repo, 'a.txt', 'a\n', 'pushed one');
  sandbox.commit(repo, 'b.txt', 'b\n', 'pushed two');
  sandbox.git(repo, 'push', '-q', 'origin', 'main');
  sandbox.git(repo, 'checkout', '-q', 'base');
  sandbox.commit(repo, 'c.txt', 'c\n', 'base moved on');
  sandbox.git(repo, 'checkout', '-q', 'main');
  return repo;
}

test.describe('Rebase', () => {
  test('names both branches and is not offered on the current branch', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'feature');
    const page = await launch([repo]);

    let menu = await openMenu(page, localBranch(page, 'feature'));
    await expect(item(menu, 'Rebase main onto feature')).toBeVisible();
    await page.keyboard.press('Escape');

    menu = await openMenu(page, localBranch(page, 'main'));
    await expect(item(menu, /^Rebase/)).toHaveCount(0);
  });

  test('asks before rewriting pushed commits, with Cancel focused', async ({ sandbox, launch }) => {
    const repo = pushedMainAndBase(sandbox);
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'base'));
    await item(menu, 'Rebase main onto base').click();
    const dialog = page.getByRole('dialog', { name: 'base' });
    await expect(dialog).toHaveAccessibleDescription(
      '2 commits on main are already on origin/main. Rebasing rewrites them, so the next push has to be a force push.'
    );
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();

    await page.keyboard.press('Enter');
    await expect(dialog).toBeHidden();
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);

    await (
      await openMenu(page, localBranch(page, 'base'))
    )
      .getByRole('menuitem', {
        name: 'Rebase main onto base',
      })
      .click();
    await page.getByRole('dialog').getByRole('button', { name: 'Rebase', exact: true }).click();
    await expect(toast(page)).toContainText('Rebased main onto base');
    expect(sandbox.gitOk(repo, 'merge-base', '--is-ancestor', 'base', 'HEAD')).toBe(true);
  });

  test('runs straight away when nothing pushed is rewritten, and says when nothing changed', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'old');
    sandbox.commit(repo, 'x.txt', 'x\n', 'local work');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'old'));
    await item(menu, 'Rebase main onto old').click();
    await expect(toast(page)).toContainText('main is already based on old');
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('is unavailable, with its reason, while a merge is in progress', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'README.md', 'theirs\n', 'feature edit');
    sandbox.git(repo, 'checkout', '-q', 'main');
    sandbox.commit(repo, 'README.md', 'ours\n', 'main edit');
    sandbox.gitOk(repo, 'merge', 'feature');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    const rebase = item(menu, /^Rebase main onto feature/);
    await expect(rebase).toHaveAttribute('aria-disabled', 'true');
    await expect(rebase).toContainText('Finish the merge in progress first');
    await rebase.focus();
    await page.keyboard.press('Enter');
    await expect(menu).toBeVisible();
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });
});

test.describe('Delete', () => {
  test('a merged branch: Cancel is focused, and Undo brings it back', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'done');
    const hash = sandbox.git(repo, 'rev-parse', 'done');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'done'));
    await item(menu, 'Delete…').click();
    const dialog = page.getByRole('dialog', { name: 'done' });
    await expect(dialog).toHaveAccessibleDescription(
      'Delete "done"? Git only deletes it if its commits are in main.'
    );
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();

    await dialog.getByRole('button', { name: 'Delete', exact: true }).click();
    await expect(toast(page)).toContainText('Deleted done');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/heads/done')).toBe(false);

    await toast(page).getByRole('button', { name: 'Undo' }).click();
    await expect(toast(page)).toContainText('Restored done');
    expect(sandbox.git(repo, 'rev-parse', 'done')).toBe(hash);
    await expect(localBranch(page, 'done')).toBeVisible();
  });

  test('an unmerged branch: the force step focuses Keep branch, so repeated Enter keeps it', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'wip');
    sandbox.commit(repo, 'w.txt', 'w\n', 'unmerged work');
    const hash = sandbox.git(repo, 'rev-parse', 'HEAD');
    sandbox.git(repo, 'checkout', '-q', 'main');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'wip'));
    await item(menu, 'Delete…').click();
    await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click();

    const force = page.getByRole('dialog', { name: 'wip' });
    await expect(force).toHaveAccessibleDescription(
      '"wip" has commits that aren\'t in main. Force deleting it leaves them reachable only through the reflog.'
    );
    await expect(force.getByRole('button', { name: 'Keep branch' })).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(force).toBeHidden();
    expect(sandbox.git(repo, 'rev-parse', 'wip')).toBe(hash);

    await item(await openMenu(page, localBranch(page, 'wip')), 'Delete…').click();
    await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click();
    await page.getByRole('button', { name: 'Force delete' }).click();
    await expect(toast(page)).toContainText('Force deleted wip');
    await toast(page).getByRole('button', { name: 'Undo' }).click();
    await expect(toast(page)).toContainText('Restored wip');
    expect(sandbox.git(repo, 'rev-parse', 'wip')).toBe(hash);
  });

  test('is unavailable on the current branch and says why', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'main'));
    const del = item(menu, /^Delete…/);
    await expect(del).toHaveAttribute('aria-disabled', 'true');
    await expect(del).toContainText("You're on this branch");
    await del.focus();
    await page.keyboard.press('Enter');
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });
});

test.describe('Check out a remote branch', () => {
  function repoWithBehindFeature(sandbox: Sandbox) {
    const repo = sandbox.repo();
    sandbox.bareRemote(repo);
    sandbox.git(repo, 'push', '-q', '-u', 'origin', 'main');
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'f.txt', 'one\n', 'one');
    sandbox.git(repo, 'push', '-q', '-u', 'origin', 'feature');
    sandbox.commit(repo, 'g.txt', 'two\n', 'two');
    sandbox.git(repo, 'push', '-q', 'origin', 'feature');
    sandbox.git(repo, 'reset', '-q', '--hard', 'HEAD~1');
    sandbox.git(repo, 'checkout', '-q', 'main');
    return repo;
  }

  test('switches to the existing local branch and fast-forwards it', async ({
    sandbox,
    launch,
  }) => {
    const repo = repoWithBehindFeature(sandbox);
    const page = await launch([repo]);

    const menu = await openMenu(page, remoteBranch(page, 'feature'));
    await item(menu, 'Check out feature').click();
    await expect(toast(page)).toContainText(
      'Switched to feature and fast-forwarded it to origin/feature'
    );
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(
      sandbox.git(repo, 'rev-parse', 'origin/feature')
    );
  });

  test('offers a new local branch when there is none', async ({ sandbox, launch }) => {
    const repo = repoWithBehindFeature(sandbox);
    sandbox.git(repo, 'branch', '-q', '-D', 'feature');
    const page = await launch([repo]);

    const menu = await openMenu(page, remoteBranch(page, 'feature'));
    await item(menu, 'Check out as new local branch').click();
    await expect(toast(page)).toContainText('Switched to feature');
    expect(sandbox.git(repo, 'rev-parse', '--abbrev-ref', 'feature@{upstream}')).toBe(
      'origin/feature'
    );
  });
});

test.describe('Rename and semantics', () => {
  test('an invalid name keeps the menu open with an explanation', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'feature');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await item(menu, 'Rename…').click();
    const input = page.getByRole('textbox', { name: 'New name for feature' });
    await input.fill('foo bar');
    await page.keyboard.press('Enter');

    await expect(page.getByRole('alert')).toHaveText(
      '"foo bar" isn\'t a valid branch name. Leave out spaces and the characters ~ ^ : ? * [ \\.'
    );
    await expect(input).toHaveValue('foo bar');
    await expect(input).toHaveAttribute('aria-invalid', 'true');

    await input.fill('foo-bar');
    await page.keyboard.press('Enter');
    await expect(toast(page)).toContainText('Renamed to foo-bar');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'refs/heads/foo-bar')).toBe(
      true
    );
  });

  test('the menu is named after its branch and its groups are separated', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'feature');
    const page = await launch([repo]);

    await localBranch(page, 'feature').click({ button: 'right' });
    const menu = page.getByRole('menu', { name: 'feature' });
    await expect(menu).toBeVisible();
    await expect(menu.getByRole('separator')).toHaveCount(1);
  });
});

test.describe('Other destructive confirmations focus the safe button', () => {
  test('dropping a stash focuses Cancel', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'README.md'), 'changed\n');
    sandbox.git(repo, 'stash', 'push', '-q', '-m', 'wip');
    const page = await launch([repo]);

    await page.locator('[data-list="stash"]').first().click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Drop' }).click();
    await expect(page.locator('.ctx-menu').getByRole('button', { name: 'Cancel' })).toBeFocused();
    await page.keyboard.press('Enter');
    expect(sandbox.git(repo, 'stash', 'list')).toContain('wip');
  });

  test('removing a worktree focuses Cancel, and the force step focuses Keep worktree', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    const wt = join(sandbox.home, 'wt-review');
    sandbox.git(repo, 'worktree', 'add', '-q', '-b', 'review', wt);
    writeFileSync(join(wt, 'scratch.txt'), 'unsaved\n');
    const page = await launch([repo]);

    await page
      .locator('[data-list="worktree"]')
      .filter({ hasText: 'wt-review' })
      .click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Remove…' }).click();
    const menu = page.locator('.ctx-menu');
    await expect(menu.getByRole('button', { name: 'Cancel' })).toBeFocused();

    await menu.getByRole('button', { name: 'Remove', exact: true }).click();
    await expect(menu.getByRole('button', { name: 'Keep worktree' })).toBeFocused();
    await page.keyboard.press('Enter');
    expect(existsSync(wt)).toBe(true);
  });
});
