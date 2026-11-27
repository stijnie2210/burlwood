import { existsSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect } from './harness';

test.describe('Stash untracked files', () => {
  test('are left alone by default, and stashed once the setting is on', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'README.md'), '# changed\n');
    writeFileSync(join(repo, 'new.txt'), 'untracked\n');
    const page = await launch([repo]);
    const stashButton = page.getByRole('button', { name: 'Stash changes' });

    await stashButton.click();
    await expect(page.locator('.toast-stack')).toContainText('Changes stashed');
    expect(existsSync(join(repo, 'new.txt'))).toBe(true);
    sandbox.git(repo, 'stash', 'pop', '-q');

    await page.getByRole('button', { name: 'Settings' }).click();
    const toggle = page.getByRole('checkbox', { name: 'Include untracked files when stashing' });
    await expect(toggle).not.toBeChecked();
    await toggle.focus();
    await page.keyboard.press('Space');
    await expect(toggle).toBeChecked();
    await page.getByRole('button', { name: 'Save' }).click();

    await stashButton.click();
    await expect.poll(() => existsSync(join(repo, 'new.txt'))).toBe(false);
    expect(sandbox.git(repo, 'status', '--porcelain')).toBe('');
    sandbox.git(repo, 'stash', 'pop', '-q');
    expect(existsSync(join(repo, 'new.txt'))).toBe(true);
  });
});
