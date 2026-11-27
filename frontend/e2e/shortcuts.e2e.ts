import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect, type Sandbox } from './harness';

function toast(page: import('@playwright/test').Page) {
  return page.locator('.toast-stack');
}

// Shortcuts do nothing until the session's repo tab is restored.
async function ready(page: import('@playwright/test').Page) {
  await expect(page.locator('.row-inner').first()).toBeVisible();
}

function pushedRepo(sandbox: Sandbox) {
  const repo = sandbox.repo();
  const remote = sandbox.bareRemote(repo);
  sandbox.git(repo, 'push', '-q', '-u', 'origin', 'HEAD');
  return { repo, remote };
}

test.describe('Keyboard shortcuts', () => {
  test('push and pull', async ({ sandbox, launch }) => {
    const { repo, remote } = pushedRepo(sandbox);
    sandbox.commit(repo, 'a.txt', 'a\n', 'local work');
    const page = await launch([repo]);
    await ready(page);

    await page.keyboard.press('ControlOrMeta+KeyP');
    await expect(toast(page)).toContainText('Pushed');
    expect(sandbox.git(remote, 'log', '-1', '--format=%s')).toBe('local work');

    const other = join(sandbox.home, 'other');
    sandbox.git(sandbox.home, 'clone', '-q', remote, other);
    sandbox.git(other, 'config', 'user.email', 'other@example.com');
    sandbox.git(other, 'config', 'user.name', 'Other');
    sandbox.commit(other, 'b.txt', 'b\n', 'their work');
    sandbox.git(other, 'push', '-q');

    await page.keyboard.press('ControlOrMeta+Shift+KeyP');
    await expect.poll(() => sandbox.git(repo, 'log', '-1', '--format=%s')).toBe('their work');
  });

  test('stash and new branch', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'README.md'), '# changed\n');
    const page = await launch([repo]);
    await ready(page);

    await page.keyboard.press('ControlOrMeta+Shift+KeyS');
    await expect(toast(page)).toContainText('Changes stashed');
    expect(sandbox.git(repo, 'stash', 'list')).toContain('stash@{0}');

    await page.keyboard.press('ControlOrMeta+Shift+KeyN');
    const input = page.getByPlaceholder('branch-name');
    await expect(input).toBeFocused();
    await input.fill('from-shortcut');
    await input.press('Enter');
    await expect.poll(() => sandbox.git(repo, 'branch', '--show-current')).toBe('from-shortcut');
  });

  test('Ctrl or ⌘ + Enter commits from the message field', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'README.md'), '# changed\n');
    sandbox.git(repo, 'add', 'README.md');
    const page = await launch([repo]);

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    const summary = page.getByRole('textbox', { name: 'Commit summary' });
    await summary.fill('committed by shortcut');
    await summary.press('Control+Enter');

    await expect
      .poll(() => sandbox.git(repo, 'log', '-1', '--format=%s'))
      .toBe('committed by shortcut');
  });
});
