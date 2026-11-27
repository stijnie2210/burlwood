export type RebaseAction = 'pick' | 'reword' | 'squash' | 'fixup' | 'drop';

export interface RebaseRow {
  hash: string;
  shortHash: string;
  subject: string;
  message: string;
  author: string;
  pushed: boolean;
  action: RebaseAction;
  newMessage: string;
}

function keeps(row: RebaseRow): boolean {
  return row.action === 'pick' || row.action === 'reword';
}

// The commit a squash or fixup row melts into: the nearest kept row above it.
export function squashTarget(rows: RebaseRow[], index: number): RebaseRow | null {
  for (let i = index - 1; i >= 0; i--) {
    if (keeps(rows[i])) {
      return rows[i];
    }
  }
  return null;
}

// Rows before this index are replayed untouched, so git leaves those commits as they are.
export function firstChangedIndex(rows: RebaseRow[], originalOrder: string[]): number {
  const i = rows.findIndex((row, n) => row.hash !== originalOrder[n] || row.action !== 'pick');
  return i === -1 ? rows.length : i;
}

export function rewrittenPushedCount(rows: RebaseRow[], originalOrder: string[]): number {
  return rows.slice(firstChangedIndex(rows, originalOrder)).filter((row) => row.pushed).length;
}

export function planProblem(rows: RebaseRow[]): string {
  if (!rows.some(keeps)) {
    return 'Keep at least one commit: pick or reword one.';
  }
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    if ((row.action === 'squash' || row.action === 'fixup') && !squashTarget(rows, i)) {
      return `${row.shortHash} has no commit above it to ${row.action} into.`;
    }
    if (row.action === 'reword' && !row.newMessage.trim()) {
      return `${row.shortHash} needs a message.`;
    }
  }
  return '';
}
