import { describe, it, expect } from 'vitest';
import { highlightChange, highlightSegments, type CodeRun } from './highlight';

const text = (runs: CodeRun[]) => runs.flatMap((r) => r.segments.map((s) => s.text)).join('');

describe('highlightSegments', () => {
  it('returns the line as one plain segment without a language', () => {
    expect(highlightSegments('<b>not markup</b>', null)).toEqual([
      { text: '<b>not markup</b>', tokens: [] },
    ]);
  });

  it('returns nothing for an empty line', () => {
    expect(highlightSegments('', 'typescript')).toEqual([]);
  });

  it('labels each segment with its Prism token classes', () => {
    const segments = highlightSegments('const s = "hi";', 'typescript');
    expect(segments.find((s) => s.text === 'const')?.tokens).toEqual([['token', 'keyword']]);
    expect(segments.find((s) => s.text === '"hi"')?.tokens).toEqual([['token', 'string']]);
  });

  it('keeps nested tokens nested, outermost first', () => {
    const segments = highlightSegments('`a${b}`', 'javascript');
    const b = segments.find((s) => s.text === 'b');
    expect(b?.tokens[0]).toEqual(['token', 'template-string']);
    expect(b?.tokens.length).toBeGreaterThan(1);
  });

  it('keeps the original text for every language, including PHP tokenized in hooks', () => {
    const lines: [string, string][] = [
      ['<?php echo $name; ?>', 'php'],
      ['<div class="a">&amp;</div>', 'markup'],
      ['func main() { fmt.Println("x") }', 'go'],
    ];
    for (const [line, lang] of lines) {
      expect(
        highlightSegments(line, lang)
          .map((s) => s.text)
          .join('')
      ).toBe(line);
    }
  });
});

describe('highlightChange', () => {
  it('puts the whole line in one unchanged run without a change', () => {
    const runs = highlightChange('let x = 1;', 'typescript');
    expect(runs).toHaveLength(1);
    expect(runs[0].changed).toBe(false);
    expect(text(runs)).toBe('let x = 1;');
  });

  it('keeps the string colour on both sides of a change inside a string', () => {
    const line = 'const s = "hello world";';
    const start = line.indexOf('world');
    const runs = highlightChange(line, 'typescript', { start, end: start + 5 });

    expect(runs.map((r) => r.changed)).toEqual([false, true, false]);
    expect(runs[1].segments).toEqual([{ text: 'world', tokens: [['token', 'string']] }]);
    expect(runs[0].segments.at(-1)).toEqual({ text: '"hello ', tokens: [['token', 'string']] });
    expect(runs[2].segments[0]).toEqual({ text: '"', tokens: [['token', 'string']] });
    expect(text(runs)).toBe(line);
  });

  it('groups a change spanning several tokens into one run', () => {
    const line = 'a = foo(bar);';
    const start = line.indexOf('foo');
    const runs = highlightChange(line, 'javascript', { start, end: line.indexOf(';') });

    expect(runs.filter((r) => r.changed)).toHaveLength(1);
    expect(
      runs
        .find((r) => r.changed)!
        .segments.map((s) => s.text)
        .join('')
    ).toBe('foo(bar)');
  });

  it('marks a change at the very start or end of the line', () => {
    expect(highlightChange('abc', null, { start: 0, end: 1 }).map((r) => r.changed)).toEqual([
      true,
      false,
    ]);
    expect(highlightChange('abc', null, { start: 2, end: 3 }).map((r) => r.changed)).toEqual([
      false,
      true,
    ]);
  });

  it('ignores an empty change range', () => {
    expect(highlightChange('abc', null, { start: 1, end: 1 })).toEqual([
      { changed: false, segments: [{ text: 'abc', tokens: [] }] },
    ]);
  });
});
