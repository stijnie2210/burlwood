import { test, expect, type Sandbox } from './harness';
import type { Page } from '@playwright/test';

function localBranch(page: Page, name: string) {
  return page.locator('[data-list="local"]').filter({ hasText: new RegExp(`^\\s*${name}\\s*$`) });
}

function remoteBranch(page: Page, name: string) {
  return page.locator('[data-list="remote"]').filter({ hasText: new RegExp(`^\\s*${name}\\s*$`) });
}

async function openMenu(page: Page, branch: ReturnType<typeof localBranch>) {
  await branch.click({ button: 'right' });
  const menu = page.getByRole('menu');
  await expect(menu).toBeVisible();
  return menu;
}

function toast(page: Page) {
  return page.locator('.toast-stack');
}

// main and feature each get a commit after branching, so merging feature needs a merge commit.
function divergedRepo(sandbox: Sandbox) {
  const repo = sandbox.repo();
  sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
  sandbox.commit(repo, 'feature.txt', 'feature\n', 'feature work');
  sandbox.git(repo, 'checkout', '-q', 'main');
  sandbox.commit(repo, 'main.txt', 'main\n', 'main work');
  return repo;
}

test.describe('Merge into current', () => {
  test('creates a merge commit when the branches have diverged', async ({ sandbox, launch }) => {
    const repo = divergedRepo(sandbox);
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Merge into main' }).click();

    await expect(toast(page)).toContainText('Merged feature into main');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'HEAD^2')).toBe(true);
    expect(sandbox.git(repo, 'rev-parse', 'HEAD^2')).toBe(
      sandbox.git(repo, 'rev-parse', 'feature')
    );
    await expect(page.getByText("Merge branch 'feature'")).toBeVisible();
  });

  test('fast-forwards when the current branch has no commits of its own', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'feature.txt', 'feature\n', 'feature work');
    sandbox.git(repo, 'checkout', '-q', 'main');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Merge into main' }).click();

    await expect(toast(page)).toContainText('Fast-forwarded main to feature');
    expect(sandbox.git(repo, 'rev-parse', 'main')).toBe(sandbox.git(repo, 'rev-parse', 'feature'));
  });

  test('says so when the current branch already contains it', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'old');
    const page = await launch([repo]);
    const before = sandbox.git(repo, 'rev-parse', 'HEAD');

    const menu = await openMenu(page, localBranch(page, 'old'));
    await menu.getByRole('menuitem', { exact: true, name: 'Merge into main' }).click();

    await expect(toast(page)).toContainText('main already contains old');
    expect(sandbox.git(repo, 'rev-parse', 'HEAD')).toBe(before);
  });

  test('hands a conflict to the Working Tree and blocks a second merge', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'README.md', 'theirs\n', 'feature edit');
    sandbox.git(repo, 'checkout', '-q', 'main');
    sandbox.commit(repo, 'README.md', 'ours\n', 'main edit');
    sandbox.git(repo, 'branch', 'other');
    const page = await launch([repo]);

    let menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Merge into main' }).click();

    await expect(toast(page)).toContainText('Merging feature into main stopped on a conflict');
    expect(sandbox.gitOk(repo, 'rev-parse', '--verify', '--quiet', 'MERGE_HEAD')).toBe(true);

    menu = await openMenu(page, localBranch(page, 'other'));
    const merge = menu.getByRole('menuitem', { exact: true, name: 'Merge into main' });
    await expect(merge).toBeDisabled();
    await expect(merge).toHaveAttribute('title', /A merge is in progress/);
    await page.keyboard.press('Escape');

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await expect(page.getByText('Merge in progress')).toBeVisible();
    await expect(page.getByText('README.md')).toBeVisible();
  });

  test('merges a remote branch', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.bareRemote(repo);
    sandbox.git(repo, 'push', '-q', 'origin', 'main');
    sandbox.git(repo, 'checkout', '-q', '-b', 'feature');
    sandbox.commit(repo, 'feature.txt', 'feature\n', 'feature work');
    sandbox.git(repo, 'push', '-q', 'origin', 'feature');
    sandbox.git(repo, 'checkout', '-q', 'main');
    sandbox.git(repo, 'branch', '-q', '-D', 'feature');
    const page = await launch([repo]);

    const menu = await openMenu(page, remoteBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Merge into main' }).click();

    await expect(toast(page)).toContainText('Fast-forwarded main to origin/feature');
    expect(sandbox.git(repo, 'rev-parse', 'main')).toBe(
      sandbox.git(repo, 'rev-parse', 'origin/feature')
    );
  });

  test('is not offered on the current branch', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'main'));
    await expect(menu.getByRole('menuitem', { name: /^Merge into/ })).toHaveCount(0);
  });
});

test.describe('Set upstream', () => {
  // feature exists on two remotes and has never been pushed with -u, so it tracks nothing.
  function repoWithRemotes(sandbox: Sandbox) {
    const repo = sandbox.repo();
    sandbox.bareRemote(repo, 'origin');
    sandbox.bareRemote(repo, 'fork');
    sandbox.git(repo, 'branch', 'feature');
    sandbox.git(repo, 'branch', 'other');
    for (const remote of ['origin', 'fork']) {
      sandbox.git(repo, 'push', '-q', remote, 'main', 'feature', 'other');
    }
    return repo;
  }

  function upstreamOf(sandbox: Sandbox, repo: string, branch: string) {
    return sandbox.gitOk(repo, 'rev-parse', '--abbrev-ref', `${branch}@{upstream}`)
      ? sandbox.git(repo, 'rev-parse', '--abbrev-ref', `${branch}@{upstream}`)
      : '';
  }

  test('lists the same-named remote branches first and tracks the picked one', async ({
    sandbox,
    launch,
  }) => {
    const repo = repoWithRemotes(sandbox);
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Set upstream…' }).click();

    await expect(menu.getByText('Pick the remote branch to pull from and push to:')).toBeVisible();
    await expect(menu.getByRole('textbox', { name: 'Filter remote branches' })).toBeFocused();
    const options = menu.locator('.ctx-menu-scroll [role="menuitem"]');
    await expect(options).toHaveText([
      'fork/feature',
      'origin/feature',
      'fork/main',
      'fork/other',
      'origin/main',
      'origin/other',
    ]);

    await menu.getByRole('menuitem', { exact: true, name: 'origin/feature' }).click();
    await expect(toast(page)).toContainText('feature now tracks origin/feature');
    expect(upstreamOf(sandbox, repo, 'feature')).toBe('origin/feature');
  });

  test('filters the list and picks the first match with Enter', async ({ sandbox, launch }) => {
    const repo = repoWithRemotes(sandbox);
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Set upstream…' }).click();
    await page.keyboard.type('origin/oth');
    await expect(menu.locator('.ctx-menu-scroll [role="menuitem"]')).toHaveText(['origin/other']);
    await page.keyboard.press('Enter');

    await expect(toast(page)).toContainText('feature now tracks origin/other');
    expect(upstreamOf(sandbox, repo, 'feature')).toBe('origin/other');
  });

  test('moves from the filter into the list with the arrow keys', async ({ sandbox, launch }) => {
    const repo = repoWithRemotes(sandbox);
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Set upstream…' }).click();
    await page.keyboard.press('ArrowDown');
    await expect(menu.getByRole('menuitem', { exact: true, name: 'fork/feature' })).toBeFocused();
    await page.keyboard.press('ArrowDown');
    await expect(menu.getByRole('menuitem', { exact: true, name: 'origin/feature' })).toBeFocused();
    await page.keyboard.press('Enter');

    await expect(toast(page)).toContainText('feature now tracks origin/feature');
  });

  test('changes and stops tracking an existing upstream', async ({ sandbox, launch }) => {
    const repo = repoWithRemotes(sandbox);
    sandbox.git(repo, 'branch', '-q', '--set-upstream-to=origin/feature', 'feature');
    const page = await launch([repo]);

    let menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Change upstream…' }).click();
    await expect(menu.getByText('Tracks origin/feature.')).toBeVisible();
    await expect(menu.getByRole('menuitem', { exact: true, name: 'origin/feature' })).toHaveCount(
      0
    );
    await menu.getByRole('menuitem', { exact: true, name: 'fork/feature' }).click();
    await expect(toast(page)).toContainText('feature now tracks fork/feature');
    expect(upstreamOf(sandbox, repo, 'feature')).toBe('fork/feature');

    menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Change upstream…' }).click();
    await menu.getByRole('menuitem', { exact: true, name: 'Stop tracking fork/feature' }).click();
    await expect(toast(page)).toContainText('feature no longer tracks fork/feature');
    expect(upstreamOf(sandbox, repo, 'feature')).toBe('');

    menu = await openMenu(page, localBranch(page, 'feature'));
    await expect(menu.getByRole('menuitem', { exact: true, name: 'Set upstream…' })).toBeVisible();
  });

  test('points to pushing when there are no remote branches', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    sandbox.git(repo, 'branch', 'feature');
    const page = await launch([repo]);

    const menu = await openMenu(page, localBranch(page, 'feature'));
    await menu.getByRole('menuitem', { exact: true, name: 'Set upstream…' }).click();
    await expect(
      menu.getByText('There are no remote branches. Push this branch to create one.')
    ).toBeVisible();
  });

  test('is not offered on remote branches', async ({ sandbox, launch }) => {
    const repo = repoWithRemotes(sandbox);
    const page = await launch([repo]);

    const menu = await openMenu(page, remoteBranch(page, 'feature').first());
    await expect(menu.getByRole('menuitem', { name: /upstream/ })).toHaveCount(0);
  });
});
