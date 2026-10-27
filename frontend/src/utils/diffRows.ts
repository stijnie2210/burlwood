import type * as repo from '../../bindings/burlwood/internal/repo';

export type HunkLine = repo.HunkLine;

export interface ChangeRange {
  start: number;
  end: number;
}

export interface DiffCell {
  line: HunkLine;
  change?: ChangeRange;
}

export interface SplitRow {
  left: DiffCell | null;
  right: DiffCell | null;
}

const WORD = /[A-Za-z0-9_]/;

// Trims the common prefix and suffix so only the edited part of a line stands out, widened to
// whole words. Lines that share no letters or digits are a rewrite, not an edit, and get no range.
export function changedRanges(before: string, after: string): [ChangeRange, ChangeRange] | null {
  if (before === after) {
    return null;
  }
  const max = Math.min(before.length, after.length);
  let start = 0;
  while (start < max && before[start] === after[start]) {
    start++;
  }
  let endBefore = before.length;
  let endAfter = after.length;
  while (endBefore > start && endAfter > start && before[endBefore - 1] === after[endAfter - 1]) {
    endBefore--;
    endAfter--;
  }
  while (start > 0 && WORD.test(before[start - 1])) {
    start--;
  }
  while (endBefore < before.length && WORD.test(before[endBefore])) {
    endBefore++;
    endAfter++;
  }
  const common = before.slice(0, start) + before.slice(endBefore);
  if (!/[A-Za-z0-9]/.test(common)) {
    return null;
  }
  return [
    { start, end: endBefore },
    { start, end: endAfter },
  ];
}

interface Run {
  dels: number[];
  adds: number[];
}

function changeRuns(lines: HunkLine[]): (Run | number)[] {
  const out: (Run | number)[] = [];
  let i = 0;
  while (i < lines.length) {
    if (lines[i].type === 'context') {
      out.push(i);
      i++;
      continue;
    }
    const run: Run = { dels: [], adds: [] };
    while (i < lines.length && lines[i].type === 'del') {
      run.dels.push(i++);
    }
    while (i < lines.length && lines[i].type === 'add') {
      run.adds.push(i++);
    }
    out.push(run);
  }
  return out;
}

// The n-th removed line of a run is compared with the n-th added line that follows it.
export function pairChanges(lines: HunkLine[]): (ChangeRange | undefined)[] {
  const ranges: (ChangeRange | undefined)[] = new Array(lines.length);
  for (const item of changeRuns(lines)) {
    if (typeof item === 'number') {
      continue;
    }
    const pairs = Math.min(item.dels.length, item.adds.length);
    for (let n = 0; n < pairs; n++) {
      const d = item.dels[n];
      const a = item.adds[n];
      const r = changedRanges(lines[d].content, lines[a].content);
      if (r) {
        ranges[d] = r[0];
        ranges[a] = r[1];
      }
    }
  }
  return ranges;
}

export function splitRows(lines: HunkLine[]): SplitRow[] {
  const ranges = pairChanges(lines);
  const cell = (i: number): DiffCell => ({ line: lines[i], change: ranges[i] });
  const rows: SplitRow[] = [];
  for (const item of changeRuns(lines)) {
    if (typeof item === 'number') {
      rows.push({ left: cell(item), right: cell(item) });
      continue;
    }
    const height = Math.max(item.dels.length, item.adds.length);
    for (let n = 0; n < height; n++) {
      rows.push({
        left: n < item.dels.length ? cell(item.dels[n]) : null,
        right: n < item.adds.length ? cell(item.adds[n]) : null,
      });
    }
  }
  return rows;
}
