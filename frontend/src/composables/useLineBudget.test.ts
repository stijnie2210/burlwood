import { describe, it, expect } from 'vitest';
import { nextTick, ref } from 'vue';
import { useLineBudget } from './useLineBudget';

function hunk(n: number) {
  return { lines: Array.from({ length: n }, (_, i) => i) };
}

describe('useLineBudget', () => {
  it('shows everything when under budget', () => {
    const hunks = [hunk(3), hunk(2)];
    const { visible, hiddenLines } = useLineBudget(
      () => hunks,
      () => 'a',
      10
    );
    expect(visible.value.map((v) => v.lines.length)).toEqual([3, 2]);
    expect(visible.value.every((v) => !v.truncated)).toBe(true);
    expect(hiddenLines.value).toBe(0);
  });

  it('truncates across hunks and keeps the original hunk object', () => {
    const hunks = [hunk(6), hunk(6), hunk(6)];
    const { visible, hiddenLines } = useLineBudget(
      () => hunks,
      () => 'a',
      8
    );
    expect(visible.value.map((v) => v.lines.length)).toEqual([6, 2]);
    expect(visible.value[1].truncated).toBe(true);
    expect(visible.value[1].hunk).toBe(hunks[1]);
    expect(hiddenLines.value).toBe(10);
  });

  it('reveals all, and resets when the key changes', async () => {
    const key = ref('a');
    const hunks = [hunk(20)];
    const { visible, hiddenLines, revealAll } = useLineBudget(
      () => hunks,
      () => key.value,
      5
    );
    revealAll();
    expect(visible.value[0].lines.length).toBe(20);
    expect(hiddenLines.value).toBe(0);
    key.value = 'b';
    await nextTick();
    expect(visible.value[0].lines.length).toBe(5);
  });
});
