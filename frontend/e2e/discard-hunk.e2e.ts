import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect } from './harness';

function lines(change: Record<number, string> = {}) {
  return Array.from({ length: 30 }, (_, i) => change[i + 1] ?? `line ${i + 1}`).join('\n') + '\n';
}

test.describe('Discard hunk', () => {
  test('discards one hunk, keeps the other, and Undo brings it back', async ({
    sandbox,
    launch,
  }) => {
    const repo = sandbox.repo();
    sandbox.commit(repo, 'f.txt', lines(), 'add f');
    const edited = lines({ 2: 'changed 2', 28: 'changed 28' });
    writeFileSync(join(repo, 'f.txt'), edited);
    const page = await launch([repo]);

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await page.locator('.file-item').filter({ hasText: 'f.txt' }).click();
    const discard = page.getByRole('button', { name: 'Discard Hunk' });
    await expect(discard).toHaveCount(2);

    await discard.first().click();

    await expect(page.locator('.toast-stack')).toContainText('Discarded a hunk in f.txt');
    await expect(discard).toHaveCount(1);
    expect(readFileSync(join(repo, 'f.txt'), 'utf8')).toBe(lines({ 28: 'changed 28' }));
    expect(sandbox.git(repo, 'diff', '--cached', '--name-only')).toBe('');

    await page.locator('.toast-stack').getByRole('button', { name: 'Undo' }).click();

    await expect(page.locator('.toast-stack')).toContainText('Restored the hunk in f.txt');
    await expect(discard).toHaveCount(2);
    expect(readFileSync(join(repo, 'f.txt'), 'utf8')).toBe(edited);
  });

  test('is not offered on untracked files or staged changes', async ({ sandbox, launch }) => {
    const repo = sandbox.repo();
    writeFileSync(join(repo, 'new.txt'), 'new\n');
    writeFileSync(join(repo, 'README.md'), '# staged\n');
    sandbox.git(repo, 'add', 'README.md');
    const page = await launch([repo]);

    await page.getByRole('tab', { name: /^Working Tree/ }).click();
    await page.locator('.file-item').filter({ hasText: 'new.txt' }).click();
    await expect(page.getByRole('button', { name: 'Stage Hunk' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Discard Hunk' })).toHaveCount(0);

    await page.locator('.file-item').filter({ hasText: 'README.md' }).click();
    await expect(page.getByRole('button', { name: 'Unstage Hunk' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Discard Hunk' })).toHaveCount(0);
  });
});
