<script setup lang="ts">
import { computed } from 'vue';
import { highlightChange } from '../../utils/highlight';
import { pairChanges, splitRows, type DiffCell, type HunkLine } from '../../utils/diffRows';
import type { DiffLayout } from '../../composables/useDiffLayout';

const props = defineProps<{
  lines: HunkLine[];
  lang: string | null;
  layout: DiffLayout;
  wrap: boolean;
}>();

const ranges = computed(() => (props.layout === 'unified' ? pairChanges(props.lines) : []));
const rows = computed(() => (props.layout === 'split' ? splitRows(props.lines) : []));

const SIGNS: Record<string, string> = { add: '+', del: '−', context: '' };

function oldNo(line: HunkLine): number | string {
  return line.type === 'add' ? '' : line.oldLine || '';
}

function newNo(line: HunkLine): number | string {
  return line.type === 'del' ? '' : line.newLine || '';
}

function cellHtml(cell: DiffCell): string {
  return highlightChange(cell.line.content, props.lang, cell.change);
}
</script>

<template>
  <div class="diff-lines" :class="{ wrap: wrap || layout === 'split' }">
    <template v-if="layout === 'unified'">
      <div
        v-for="(line, li) in lines"
        :key="li"
        class="diff-line"
        :class="`diff-line--${line.type}`"
      >
        <span class="diff-lineno">{{ oldNo(line) }}</span>
        <span class="diff-lineno">{{ newNo(line) }}</span>
        <span class="diff-gutter" aria-hidden="true">{{ SIGNS[line.type] }}</span>
        <span class="diff-content" v-html="highlightChange(line.content, lang, ranges[li])"></span>
      </div>
    </template>
    <template v-else>
      <div v-for="(row, ri) in rows" :key="ri" class="split-row">
        <div
          v-if="row.left"
          class="diff-line split-cell"
          :class="`diff-line--${row.left.line.type}`"
        >
          <span class="diff-lineno">{{ oldNo(row.left.line) }}</span>
          <span class="diff-gutter" aria-hidden="true">{{ SIGNS[row.left.line.type] }}</span>
          <span class="diff-content" v-html="cellHtml(row.left)"></span>
        </div>
        <div v-else class="split-cell split-cell--empty" />
        <div
          v-if="row.right"
          class="diff-line split-cell"
          :class="`diff-line--${row.right.line.type}`"
        >
          <span class="diff-lineno">{{ newNo(row.right.line) }}</span>
          <span class="diff-gutter" aria-hidden="true">{{ SIGNS[row.right.line.type] }}</span>
          <span class="diff-content" v-html="cellHtml(row.right)"></span>
        </div>
        <div v-else class="split-cell split-cell--empty" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.diff-lines {
  font-family: var(--font-mono);
  font-size: var(--text-row);
  line-height: 19px;
  tab-size: 4;
}

/* Unwrapped rows grow to the longest line so their backgrounds reach the end when scrolled. */
.diff-lines:not(.wrap) {
  min-width: max-content;
}

.diff-line {
  display: flex;
  align-items: flex-start;
}
.diff-line--add {
  background: var(--add-line-bg);
}
.diff-line--del {
  background: var(--del-line-bg);
}

.diff-lineno {
  width: 44px;
  padding-right: var(--space-2);
  flex-shrink: 0;
  text-align: right;
  color: var(--text-tertiary);
  font-size: var(--text-meta);
  user-select: none;
  -webkit-user-select: none;
}

.diff-gutter {
  width: 16px;
  flex-shrink: 0;
  text-align: center;
  color: var(--text-tertiary);
  user-select: none;
  -webkit-user-select: none;
}
.diff-line--add .diff-gutter {
  color: var(--add);
}
.diff-line--del .diff-gutter {
  color: var(--del);
}

.diff-content {
  flex: 1;
  min-width: 0;
  padding: 0 var(--space-4) 0 var(--space-1);
  white-space: pre;
  color: var(--text-secondary);
}
.diff-lines.wrap .diff-content {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.diff-line--add .diff-content {
  color: var(--add-fg);
}
.diff-line--del .diff-content {
  color: var(--del-fg);
}

.diff-content :deep(.diff-change) {
  border-radius: 2px;
  color: inherit;
}
.diff-line--add .diff-content :deep(.diff-change) {
  background: var(--add-change-bg);
}
.diff-line--del .diff-content :deep(.diff-change) {
  background: var(--del-change-bg);
}

.split-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}
.split-cell + .split-cell {
  border-left: 1px solid var(--border-subtle);
}
.split-cell--empty {
  background: var(--surface-base);
}
</style>
