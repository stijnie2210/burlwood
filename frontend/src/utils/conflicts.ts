export type ContextChunk = { type: 'context'; lines: string[] };

export type ParsedConflict = {
  type: 'conflict';
  ours: string[];
  base: string[] | null;
  theirs: string[];
  oursLabel: string;
  baseLabel: string;
  theirsLabel: string;
};

export type ParsedChunk = ContextChunk | ParsedConflict;

const OURS = /^<{7}(?: (.*))?$/;
const BASE = /^\|{7}(?: (.*))?$/;
const SEP = /^={7}$/;
const THEIRS = /^>{7}(?: (.*))?$/;
// A bare ======= also underlines Markdown headings, so only the unambiguous markers count.
const ANY_MARKER = /^(<{7}|\|{7}|>{7})( |$)/m;

// Handles the default "merge" style plus diff3/zdiff3, whose |||||||-section holds the common
// ancestor. Treating that section as part of "ours" would write base lines back into the file.
export function parseConflicts(text: string): ParsedChunk[] {
  const result: ParsedChunk[] = [];
  const lines = text.split('\n');

  let state: 'context' | 'ours' | 'base' | 'theirs' = 'context';
  let context: string[] = [];
  let ours: string[] = [];
  let base: string[] | null = null;
  let theirs: string[] = [];
  let oursLabel = '';
  let baseLabel = '';

  for (const line of lines) {
    let m: RegExpMatchArray | null;
    if (state === 'context' && (m = line.match(OURS))) {
      if (context.length) {
        result.push({ type: 'context', lines: context });
      }
      context = [];
      ours = [];
      base = null;
      theirs = [];
      oursLabel = m[1] ?? '';
      baseLabel = '';
      state = 'ours';
    } else if (state === 'ours' && (m = line.match(BASE))) {
      base = [];
      baseLabel = m[1] ?? '';
      state = 'base';
    } else if ((state === 'ours' || state === 'base') && SEP.test(line)) {
      state = 'theirs';
    } else if (state === 'theirs' && (m = line.match(THEIRS))) {
      result.push({
        type: 'conflict',
        ours,
        base,
        theirs,
        oursLabel,
        baseLabel,
        theirsLabel: m[1] ?? '',
      });
      state = 'context';
    } else if (state === 'context') {
      context.push(line);
    } else if (state === 'ours') {
      ours.push(line);
    } else if (state === 'base') {
      base!.push(line);
    } else {
      theirs.push(line);
    }
  }

  while (context.length && context[context.length - 1] === '') {
    context.pop();
  }
  if (context.length) {
    result.push({ type: 'context', lines: context });
  }
  return result;
}

export function hasConflictMarkers(text: string): boolean {
  return ANY_MARKER.test(text);
}
