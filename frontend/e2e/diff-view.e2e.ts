import { test, expect } from './harness';

test('a diff shows file contents as text and keeps token colours around a change', async ({
  sandbox,
  launch,
}) => {
  const repo = sandbox.repo();
  sandbox.commit(repo, 'app.ts', 'const greeting = "hello world";\n', 'add app');
  sandbox.commit(
    repo,
    'app.ts',
    'const greeting = "hello there";\nconst x = "<img src=x onerror=alert(1)>";\n',
    'edit app'
  );
  const page = await launch([repo]);

  await page.getByText('edit app').click();
  await page.getByText('app.ts').first().click();

  const added = page.locator('.diff-line--add .diff-content');
  await expect(added.last()).toHaveText('const x = "<img src=x onerror=alert(1)>";');
  await expect(page.locator('.diff-content img')).toHaveCount(0);

  const mark = added.first().locator('mark.diff-change');
  await expect(mark).toHaveText('there');
  await expect(mark.locator('.token.string')).toHaveText('there');
  await expect(added.first().locator('.token.string')).toHaveText(['"hello ', 'there', '"']);
});
