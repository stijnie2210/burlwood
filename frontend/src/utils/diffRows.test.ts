import { describe, it, expect } from 'vitest';
import { changedRanges, pairChanges, splitRows } from './diffRows';
import type { HunkLine } from './diffRows';

function line(type: string, content: string, oldLine = 0, newLine = 0): HunkLine {
  return { type, content, oldLine, newLine };
}

describe('changedRanges', () => {
  it('marks only the edited word', () => {
    expect(changedRanges('<h1>gitgood</h1>', '<h1>Burlwood</h1>')).toEqual([
      { start: 4, end: 11 },
      { start: 4, end: 12 },
    ]);
  });

  it('gives an empty range on the side without the insertion', () => {
    expect(changedRanges('foo()', 'foo(bar)')).toEqual([
      { start: 4, end: 4 },
      { start: 4, end: 7 },
    ]);
  });

  it('returns null for identical lines', () => {
    expect(changedRanges('same', 'same')).toBeNull();
  });

  it('returns null when only whitespace is shared', () => {
    expect(changedRanges('  return a;', '  throw err;')).toBeNull();
    expect(changedRanges('abc', 'xyz')).toBeNull();
  });
});

describe('pairChanges', () => {
  it('pairs the n-th removed line with the n-th added line of a run', () => {
    const lines = [
      line('context', 'a'),
      line('del', 'name: gitgood'),
      line('del', 'title: gitgood'),
      line('add', 'name: burlwood'),
      line('add', 'title: Burlwood'),
      line('add', 'extra'),
    ];
    const ranges = pairChanges(lines);
    expect(ranges[0]).toBeUndefined();
    expect(ranges[1]).toEqual({ start: 6, end: 13 });
    expect(ranges[3]).toEqual({ start: 6, end: 14 });
    expect(ranges[2]).toEqual({ start: 7, end: 14 });
    expect(ranges[4]).toEqual({ start: 7, end: 15 });
    expect(ranges[5]).toBeUndefined();
  });

  it('does not pair across a context line', () => {
    const lines = [line('del', 'x = 1'), line('context', 'y'), line('add', 'x = 2')];
    expect(pairChanges(lines).every((r) => r === undefined)).toBe(true);
  });
});

describe('splitRows', () => {
  it('puts context on both sides and lines up a replaced run', () => {
    const lines = [
      line('context', 'a', 1, 1),
      line('del', 'b', 2, 0),
      line('add', 'B', 0, 2),
      line('add', 'C', 0, 3),
      line('context', 'd', 3, 4),
    ];
    const rows = splitRows(lines);
    expect(rows).toHaveLength(4);
    expect(rows[0].left?.line.content).toBe('a');
    expect(rows[0].right?.line.content).toBe('a');
    expect(rows[1].left?.line.content).toBe('b');
    expect(rows[1].right?.line.content).toBe('B');
    expect(rows[2].left).toBeNull();
    expect(rows[2].right?.line.content).toBe('C');
    expect(rows[3].left?.line.oldLine).toBe(3);
    expect(rows[3].right?.line.newLine).toBe(4);
  });

  it('leaves the right side empty for a pure deletion', () => {
    const rows = splitRows([line('del', 'gone')]);
    expect(rows).toEqual([{ left: { line: line('del', 'gone'), change: undefined }, right: null }]);
  });
});
