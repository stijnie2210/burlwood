import { describe, it, expect } from 'vitest';
import { buildHunkPatch, isConflictedFile } from './staging';
import type { FileStatus } from './staging';
import type * as repo from '../../bindings/burlwood/internal/repo';

function makeHunk(
  header: string,
  lines: { type: string; content: string; oldLine: number; newLine: number }[]
): repo.Hunk {
  return { header, lines };
}

describe('buildHunkPatch', () => {
  it('produces the correct patch format for add lines', () => {
    const hunk = makeHunk('@@ -0,0 +1,2 @@', [
      { type: 'add', content: 'first line', oldLine: 0, newLine: 1 },
      { type: 'add', content: 'second line', oldLine: 0, newLine: 2 },
    ]);
    const patch = buildHunkPatch('foo.go', hunk);

    expect(patch).toContain('diff --git a/foo.go b/foo.go');
    expect(patch).toContain('--- a/foo.go');
    expect(patch).toContain('+++ b/foo.go');
    expect(patch).toContain('@@ -0,0 +1,2 @@');
    expect(patch).toContain('+first line');
    expect(patch).toContain('+second line');
  });

  it('prefixes del lines with -', () => {
    const hunk = makeHunk('@@ -1,1 +0,0 @@', [
      { type: 'del', content: 'remove me', oldLine: 1, newLine: 0 },
    ]);
    const patch = buildHunkPatch('bar.go', hunk);

    expect(patch).toContain('-remove me');
    expect(patch).not.toContain('+remove me');
  });

  it('prefixes context lines with a space', () => {
    const hunk = makeHunk('@@ -1,3 +1,3 @@', [
      { type: 'context', content: 'before', oldLine: 1, newLine: 1 },
      { type: 'del', content: 'old', oldLine: 2, newLine: 0 },
      { type: 'add', content: 'new', oldLine: 0, newLine: 2 },
      { type: 'context', content: 'after', oldLine: 3, newLine: 3 },
    ]);
    const patch = buildHunkPatch('baz.go', hunk);

    expect(patch).toContain(' before');
    expect(patch).toContain(' after');
  });

  it('ends with a newline (required by git apply)', () => {
    const hunk = makeHunk('@@ -1,1 +1,1 @@', [
      { type: 'add', content: 'x', oldLine: 0, newLine: 1 },
    ]);
    const patch = buildHunkPatch('f.go', hunk);
    expect(patch.endsWith('\n')).toBe(true);
  });

  it('matches the Go BuildHunkPatch output for a mixed hunk', () => {
    const hunk = makeHunk('@@ -1,2 +1,2 @@', [
      { type: 'context', content: 'ctx', oldLine: 1, newLine: 1 },
      { type: 'del', content: 'old', oldLine: 2, newLine: 0 },
      { type: 'add', content: 'new', oldLine: 0, newLine: 2 },
    ]);
    const patch = buildHunkPatch('file.go', hunk);
    const expected = [
      'diff --git a/file.go b/file.go',
      '--- a/file.go',
      '+++ b/file.go',
      '@@ -1,2 +1,2 @@',
      ' ctx',
      '-old',
      '+new',
      '',
    ].join('\n');

    expect(patch).toBe(expected);
  });
});

describe('isConflictedFile', () => {
  const file = (staged: string, unstaged: string): FileStatus =>
    ({ staged, unstaged, path: 'f', oldPath: '' }) as FileStatus;

  it('returns true when staged is U (both modified conflict)', () => {
    expect(isConflictedFile(file('U', 'U'))).toBe(true);
  });

  it('returns true when staged is U and unstaged is something else', () => {
    expect(isConflictedFile(file('U', ' '))).toBe(true);
  });

  it('returns true when unstaged is U', () => {
    expect(isConflictedFile(file(' ', 'U'))).toBe(true);
  });

  it('returns true for AA (both added) conflict', () => {
    expect(isConflictedFile(file('A', 'A'))).toBe(true);
  });

  it('returns true for DD (both deleted) conflict', () => {
    expect(isConflictedFile(file('D', 'D'))).toBe(true);
  });

  it('returns false for normal staged-only modification', () => {
    expect(isConflictedFile(file('M', ' '))).toBe(false);
  });

  it('returns false for normal unstaged modification', () => {
    expect(isConflictedFile(file(' ', 'M'))).toBe(false);
  });

  it('returns false for untracked files', () => {
    expect(isConflictedFile(file('?', '?'))).toBe(false);
  });

  it('returns false for staged new file (A + space)', () => {
    expect(isConflictedFile(file('A', ' '))).toBe(false);
  });
});
