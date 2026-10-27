<script setup lang="ts">
import { computed } from 'vue';
import type { FileDiff, Hunk } from '../../stores/staging';
import { languageForPath } from '../../utils/highlight';
import { useLineBudget } from '../../composables/useLineBudget';
import { useDiffLayout } from '../../composables/useDiffLayout';
import DiffLines from '../diff/DiffLines.vue';
import DiffLayoutControls from '../diff/DiffLayoutControls.vue';

const props = defineProps<{
  diffs: FileDiff[];
  mode: 'stage' | 'unstage';
  path: string;
  diffLoading: boolean;
  diffError?: string | null;
}>();

const lang = computed(() => languageForPath(props.path));
const { layout, wrap } = useDiffLayout();

const emit = defineEmits<{
  'stage-file': [];
  'unstage-file': [];
  'discard-file': [];
  'stage-hunk': [hunk: Hunk];
  'unstage-hunk': [hunk: Hunk];
}>();

// Flatten hunks across all FileDiffs (usually just one for a single file)
const flatHunks = computed(() => {
  const result: Hunk[] = [];
  for (const fd of props.diffs) {
    for (const h of fd.hunks ?? []) {
      result.push(h);
    }
  }
  return result;
});

const isBinary = computed(() => props.diffs.some((d) => d.isBinary));

const stats = computed(() => {
  let add = 0;
  let del = 0;
  for (const h of flatHunks.value) {
    for (const l of h.lines ?? []) {
      if (l.type === 'add') {
        add++;
      } else if (l.type === 'del') {
        del++;
      }
    }
  }
  return { add, del };
});

const { visible, hiddenLines, revealAll } = useLineBudget(
  () => flatHunks.value,
  () => `${props.path}:${props.mode}`
);
</script>

<template>
  <div class="hunk-selector">
    <div class="file-action-bar">
      <span class="file-action-path" :title="path">{{ path }}</span>
      <span class="file-action-meta">
        {{ mode === 'stage' ? 'Unstaged' : 'Staged' }}
        <template v-if="!diffLoading && !isBinary && flatHunks.length">
          · <span class="stat-add">+{{ stats.add }}</span>
          <span class="stat-del">−{{ stats.del }}</span>
        </template>
      </span>
      <div class="file-action-btns">
        <DiffLayoutControls />
        <button
          v-if="mode === 'stage'"
          class="action-btn discard-btn"
          @click="emit('discard-file')"
        >
          Discard
        </button>
        <button v-if="mode === 'stage'" class="action-btn stage-btn" @click="emit('stage-file')">
          Stage File
        </button>
        <button
          v-if="mode === 'unstage'"
          class="action-btn unstage-btn"
          @click="emit('unstage-file')"
        >
          Unstage File
        </button>
      </div>
    </div>

    <div v-if="diffLoading" class="hunk-loading">Loading diff…</div>

    <div v-else-if="diffError" class="hunk-empty hunk-error" role="alert">
      Couldn't load this diff. {{ diffError }}
    </div>

    <div v-else-if="isBinary" class="hunk-empty">Binary file — no diff available</div>

    <div v-else-if="flatHunks.length === 0" class="hunk-empty">No diff to show</div>

    <div v-else class="hunks-scroll">
      <div v-for="(entry, hi) in visible" :key="entry.hunk.header + hi" class="hunk-block">
        <div class="hunk-header-row">
          <span class="hunk-header-text">{{ entry.hunk.header }}</span>
          <template v-if="!entry.truncated">
            <button
              v-if="mode === 'stage'"
              class="hunk-btn stage-btn"
              @click="emit('stage-hunk', entry.hunk)"
            >
              Stage Hunk
            </button>
            <button v-else class="hunk-btn unstage-btn" @click="emit('unstage-hunk', entry.hunk)">
              Unstage Hunk
            </button>
          </template>
        </div>
        <DiffLines :lines="entry.lines" :lang="lang" :layout="layout" :wrap="wrap" />
      </div>
      <button v-if="hiddenLines" class="diff-show-all" @click="revealAll">
        Show {{ hiddenLines.toLocaleString() }} more lines
      </button>
    </div>
  </div>
</template>

<style scoped>
.hunk-selector {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--surface-sunken);
  font-size: var(--text-row);
  min-width: 0;
}

.file-action-bar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0 var(--inset-panel);
  background: var(--surface-raised);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  height: var(--file-header-h);
  box-sizing: border-box;
}

.file-action-path {
  flex: 0 1 auto;
  min-width: 0;
  font-family: var(--font-mono);
  font-size: var(--text-row);
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-action-meta {
  flex: 1;
  min-width: 0;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  white-space: nowrap;
}
.stat-add {
  color: var(--add);
}
.stat-del {
  color: var(--del);
}

.file-action-btns {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.action-btn {
  padding: 3px 10px;
  border-radius: 4px;
  border: 1px solid;
  font-size: var(--text-meta);
  cursor: default;
  font-weight: 500;
}

.action-btn.stage-btn {
  background: var(--add-soft);
  border-color: var(--add-border);
  color: var(--add);
}
.action-btn.stage-btn:hover {
  background: var(--add-soft-strong);
}
.action-btn.unstage-btn {
  background: var(--accent-soft);
  border-color: var(--accent-border);
  color: var(--accent-text);
}
.action-btn.unstage-btn:hover {
  background: var(--accent-soft-strong);
}
.action-btn.discard-btn {
  background: var(--danger-soft);
  border-color: var(--danger-border);
  color: var(--danger);
}
.action-btn.discard-btn:hover {
  background: var(--danger-soft-hover);
}

.hunk-loading,
.hunk-empty {
  padding: 24px;
  color: var(--text-secondary);
  text-align: center;
  font-size: var(--text-body);
}

.hunk-error {
  color: var(--danger);
}

.hunks-scroll {
  flex: 1;
  overflow-y: auto;
}

.hunk-block {
  border-bottom: 1px solid var(--border-subtle);
}

.hunk-header-row {
  display: flex;
  align-items: center;
  padding: 3px 10px;
  background: var(--surface-raised);
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
  gap: 8px;
}

.hunk-header-text {
  flex: 1;
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.hunk-btn {
  padding: 2px 8px;
  border-radius: 3px;
  border: 1px solid;
  font-size: var(--text-meta);
  cursor: default;
  flex-shrink: 0;
  font-weight: 500;
}
.hunk-btn.stage-btn {
  background: var(--add-soft);
  border-color: var(--add-border);
  color: var(--add);
}
.hunk-btn.stage-btn:hover {
  background: var(--add-soft-strong);
}
.hunk-btn.unstage-btn {
  background: var(--accent-soft);
  border-color: var(--accent-border);
  color: var(--accent-text);
}
.hunk-btn.unstage-btn:hover {
  background: var(--accent-soft-strong);
}

.diff-show-all {
  display: block;
  width: 100%;
  padding: 8px;
  background: var(--surface-raised);
  border: none;
  border-top: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: var(--text-row);
  cursor: default;
}
.diff-show-all:hover {
  background: var(--surface-control);
  color: var(--text-primary);
}
</style>
