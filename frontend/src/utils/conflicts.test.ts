import { describe, it, expect } from 'vitest';
import { parseConflicts, hasConflictMarkers } from './conflicts';
import type { ParsedConflict } from './conflicts';

const merge = [
  'before',
  '<<<<<<< HEAD',
  'ours line',
  '=======',
  'theirs line',
  '>>>>>>> feature',
  'after',
  '',
].join('\n');

const diff3 = [
  'before',
  '<<<<<<< HEAD',
  'ours line',
  '||||||| merged common ancestors',
  'base line',
  '=======',
  'theirs line',
  '>>>>>>> feature',
  'after',
].join('\n');

describe('parseConflicts', () => {
  it('parses the default merge style', () => {
    const chunks = parseConflicts(merge);
    expect(chunks).toHaveLength(3);
    const c = chunks[1] as ParsedConflict;
    expect(c.ours).toEqual(['ours line']);
    expect(c.base).toBeNull();
    expect(c.theirs).toEqual(['theirs line']);
    expect(c.oursLabel).toBe('HEAD');
    expect(c.theirsLabel).toBe('feature');
  });

  it('keeps the diff3 base section out of ours', () => {
    const c = parseConflicts(diff3)[1] as ParsedConflict;
    expect(c.ours).toEqual(['ours line']);
    expect(c.base).toEqual(['base line']);
    expect(c.baseLabel).toBe('merged common ancestors');
    expect(c.theirs).toEqual(['theirs line']);
  });

  it('handles zdiff3 with an empty base and a bare marker', () => {
    const text = ['<<<<<<< HEAD', 'a', '|||||||', '=======', 'b', '>>>>>>> x'].join('\n');
    const c = parseConflicts(text)[0] as ParsedConflict;
    expect(c.ours).toEqual(['a']);
    expect(c.base).toEqual([]);
    expect(c.theirs).toEqual(['b']);
  });

  it('treats a stray ======= outside a conflict as content', () => {
    const chunks = parseConflicts('title\n=======\nbody');
    expect(chunks).toEqual([{ type: 'context', lines: ['title', '=======', 'body'] }]);
  });
});

describe('hasConflictMarkers', () => {
  it('detects any surviving marker', () => {
    expect(hasConflictMarkers('a\n||||||| base\nb')).toBe(true);
    expect(hasConflictMarkers('a\n>>>>>>> x')).toBe(true);
    expect(hasConflictMarkers('Title\n=======\nbody')).toBe(false);
    expect(hasConflictMarkers('a = b\n<<<<<< six')).toBe(false);
  });
});
