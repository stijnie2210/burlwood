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

const canvasRef = ref<HTMLCanvasElement | null>(null);

watchEffect(() => {
  const canvas = canvasRef.value;
  if (!canvas || !props.row) {
    return;
  }
  canvas.width = props.graphWidth;
  canvas.height = props.rowH;
  const ctx = canvas.getContext('2d');
  if (ctx) {
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
      <span v-for="l in row.labels" :key="l.name" class="label" :class="`label--${l.type}`">{{
        l.type === 'remote' ? `${l.remote}/${l.name}` : l.name
      }}</span>
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
  gap: 8px;
  padding-right: 16px;
  white-space: nowrap;
  overflow: hidden;
  cursor: pointer;
  height: 100%;
}

.row-inner:hover {
  background: rgba(255, 255, 255, 0.04);
}
.row-inner.selected {
  background: var(--accent-soft);
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
  font-size: var(--text-row);
}
.row-author {
  width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-secondary);
  flex-shrink: 0;
  font-size: var(--text-row);
}
.row-date {
  width: 80px;
  color: var(--text-tertiary);
  flex-shrink: 0;
  text-align: right;
  font-size: var(--text-row);
}
</style>
