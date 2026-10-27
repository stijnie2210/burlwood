import { computed, ref, watch } from 'vue';

export const LINE_BUDGET = 2000;

export interface VisibleHunk<H extends { lines: unknown[] | null }> {
  hunk: H;
  lines: NonNullable<H['lines']>;
  truncated: boolean;
}

function linesOf<H extends { lines: unknown[] | null }>(hunk: H): NonNullable<H['lines']> {
  return (hunk.lines ?? []) as NonNullable<H['lines']>;
}

// Rendering every line of a huge diff (lockfiles, generated code) freezes the view, so only a prefix
// renders until the user asks for the rest. `hunk` stays the original object so hunk actions see all lines.
export function useLineBudget<H extends { lines: unknown[] | null }>(
  hunks: () => H[],
  resetKey: () => unknown,
  budget = LINE_BUDGET
) {
  const showAll = ref(false);
  watch(resetKey, () => {
    showAll.value = false;
  });

  const totalLines = computed(() => hunks().reduce((n, h) => n + (h.lines?.length ?? 0), 0));

  const visible = computed<VisibleHunk<H>[]>(() => {
    const all = hunks();
    if (showAll.value || totalLines.value <= budget) {
      return all.map((hunk) => ({ hunk, lines: linesOf(hunk), truncated: false }));
    }
    const out: VisibleHunk<H>[] = [];
    let left = budget;
    for (const hunk of all) {
      if (left <= 0) {
        break;
      }
      const hunkLines = linesOf(hunk);
      const lines = hunkLines.slice(0, left) as NonNullable<H['lines']>;
      out.push({ hunk, lines, truncated: lines.length < hunkLines.length });
      left -= lines.length;
    }
    return out;
  });

  const hiddenLines = computed(() => (showAll.value ? 0 : Math.max(0, totalLines.value - budget)));

  function revealAll() {
    showAll.value = true;
  }

  return { visible, hiddenLines, revealAll };
}
