<script setup lang="ts">
import { ref, watchEffect, computed } from 'vue';
import type { GraphRow } from '../../stores/commits';
import { drawGraphCell } from './graphRenderer';
import { usePrefsStore } from '../../stores/prefs';
import { formatDate } from '../../utils/date';

const props = defineProps<{
  row: GraphRow;
  graphWidth: number;
  rowH: number;
  selected: boolean;
}>();

const prefsStore = usePrefsStore();

const displayDate = computed(() => {
  if (!props.row.timestamp) {
    return props.row.date;
  }
  return formatDate(props.row.timestamp, prefsStore.prefs.dateFormat, prefsStore.now);
});

const MAX_LABELS = 2;
const visibleLabels = computed(() => props.row.labels.slice(0, MAX_LABELS));
const hiddenLabels = computed(() => props.row.labels.slice(MAX_LABELS));

function labelText(l: GraphRow['labels'][number]): string {
  return l.type === 'remote' ? `${l.remote}/${l.name}` : l.name;
}

const canvasRef = ref<HTMLCanvasElement | null>(null);

watchEffect(() => {
  const canvas = canvasRef.value;
  if (!canvas || !props.row) {
    return;
  }
  // Draw at device resolution so lanes stay crisp on Retina; CSS keeps the logical size.
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.round(props.graphWidth * dpr);
  canvas.height = Math.round(props.rowH * dpr);
  const ctx = canvas.getContext('2d');
  if (ctx) {
    ctx.scale(dpr, dpr);
    drawGraphCell(ctx, props.row, props.rowH);
  }
});
</script>

<template>
  <div class="row-inner" :class="{ selected }">
    <canvas
      ref="canvasRef"
      class="row-canvas"
      :style="{ width: `${graphWidth}px`, height: `${rowH}px` }"
    />
    <span class="row-hash">{{ row.shortHash }}</span>
    <span class="row-labels">
      <span
        v-for="l in visibleLabels"
        :key="`${l.type}:${l.remote ?? ''}:${l.name}`"
        class="label"
        :class="`label--${l.type}`"
        :title="labelText(l)"
        >{{ labelText(l) }}</span
      >
      <span
        v-if="hiddenLabels.length"
        class="label label--more"
        :title="hiddenLabels.map(labelText).join('\n')"
        >+{{ hiddenLabels.length }}</span
      >
    </span>
    <span class="row-subject">{{ row.subject }}</span>
    <span class="row-author">{{ row.author }}</span>
    <span class="row-date">{{ displayDate }}</span>
  </div>
</template>

<style src="../../assets/labels.css" />

<style scoped>
.row-inner {
  display: flex;
  align-items: center;
  width: 100%;
  gap: var(--space-2);
  padding-right: var(--space-4);
  white-space: nowrap;
  overflow: hidden;
  cursor: default;
  height: 100%;
}

.row-inner:hover {
  background: var(--surface-hover);
}
.row-inner.selected {
  background: var(--surface-selected);
}

.row-canvas {
  flex-shrink: 0;
  display: block;
}

.row-hash {
  font-family: var(--font-mono);
  color: var(--text-tertiary);
  font-size: var(--text-meta);
  flex-shrink: 0;
  width: 52px;
}

.row-labels {
  display: flex;
  gap: 4px;
  flex-shrink: 1;
  min-width: 0;
  max-width: 40%;
  overflow: hidden;
}
.row-labels .label {
  min-width: 0;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.row-labels .label--more {
  flex-shrink: 0;
  background: var(--surface-control);
  color: var(--text-secondary);
}

.row-subject {
  flex: 1;
  min-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary);
  font-size: var(--text-body);
}
.row-author {
  width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-tertiary);
  flex-shrink: 0;
  font-size: var(--text-row);
}
.row-date {
  width: 80px;
  color: var(--text-tertiary);
  flex-shrink: 0;
  text-align: right;
  font-size: var(--text-row);
  font-variant-numeric: tabular-nums;
}

/* The subject is what people scan; metadata columns give way first in narrow panes. */
@container graph (max-width: 720px) {
  .row-author {
    display: none;
  }
}
@container graph (max-width: 520px) {
  .row-date {
    display: none;
  }
}
</style>
