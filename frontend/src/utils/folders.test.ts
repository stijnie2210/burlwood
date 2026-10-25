import { describe, it, expect } from 'vitest';
import { orderByFolder, parentDir } from './folders';

describe('parentDir', () => {
  it('returns the folder, or an empty string at the root', () => {
    expect(parentDir('src/a/x.ts')).toBe('src/a');
    expect(parentDir('README.md')).toBe('');
  });
});

describe('orderByFolder', () => {
  it('keeps each folder together in order of first appearance', () => {
    const paths = ['src/a.ts', 'src/a/x.ts', 'src/b.ts', 'README.md'];
    expect(orderByFolder(paths, (p) => p)).toEqual([
      'src/a.ts',
      'src/b.ts',
      'src/a/x.ts',
      'README.md',
    ]);
  });
});
