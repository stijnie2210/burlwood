<script setup lang="ts">
import { useCommitsStore } from '../../stores/commits';

const commits = useCommitsStore();

type FileDiff = (typeof commits.diff)[0];

// Same letters as the working tree (git's own A/M/D/R).
function fileStatus(file: FileDiff): string {
  if (!file.oldPath) {
    return 'A';
  }
  if (!file.newPath) {
    return 'D';
  }
  if (file.newPath !== file.oldPath) {
    return 'R';
  }
  return 'M';
}

const STATUS_NAMES: Record<string, string> = {
  A: 'added',
  D: 'deleted',
  R: 'renamed',
  M: 'modified',
};

function focusSibling(e: KeyboardEvent, dir: 1 | -1) {
  const el = e.currentTarget as HTMLElement;
  const next = (
    dir === 1 ? el.nextElementSibling : el.previousElementSibling
  ) as HTMLElement | null;
  if (next?.classList.contains('file-item')) {
    next.focus();
  }
}

function tabStop(fi: number): number {
  const active = commits.selectedFileIndex ?? 0;
  return fi === active ? 0 : -1;
}
</script>

<template>
  <div class="file-list">
    <div v-if="commits.diff.length === 0" class="no-changes">No changes</div>

    <button
      v-for="(file, fi) in commits.diff"
      :key="file.newPath ?? file.oldPath ?? fi"
      class="file-item"
      :class="{ active: fi === commits.selectedFileIndex }"
      :tabindex="tabStop(fi)"
      :aria-label="`${file.newPath || file.oldPath}, ${STATUS_NAMES[fileStatus(file)]}`"
      @click="commits.selectFile(fi)"
      @keydown.down.prevent.stop="focusSibling($event, 1)"
      @keydown.up.prevent.stop="focusSibling($event, -1)"
    >
      <span class="file-status" :class="`status-${fileStatus(file)}`" aria-hidden="true">{{
        fileStatus(file)
      }}</span>
      <span class="file-path">{{ file.newPath || file.oldPath }}</span>
      <span v-if="file.isBinary" class="file-badge">binary</span>
      <span class="file-stats">
        <span class="stat-add"
          >+{{ file.hunks.flatMap((h) => h.lines).filter((l) => l.type === 'add').length }}</span
        >
        <span class="stat-del"
          >−{{ file.hunks.flatMap((h) => h.lines).filter((l) => l.type === 'del').length }}</span
        >
      </span>
    </button>
  </div>
</template>

<style scoped>
.file-list {
  flex: 1;
  overflow-y: auto;
}

.no-changes {
  padding: 16px;
  color: var(--text-tertiary);
  text-align: center;
  font-size: var(--text-row);
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: var(--space-1) var(--inset-panel);
  background: transparent;
  border: none;
  cursor: pointer;
  text-align: left;
  color: var(--text-secondary);
  width: 100%;
  font-size: var(--text-row);
  min-height: var(--row-h);
}
.file-item:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.file-item.active {
  background: var(--surface-selected);
  color: var(--text-primary);
}

.file-status {
  width: 14px;
  text-align: center;
  flex-shrink: 0;
  font-family: var(--font-mono);
}
.status-A {
  color: var(--add);
}
.status-M {
  color: var(--modified);
}
.status-D {
  color: var(--del);
}
.status-R {
  color: var(--accent-text);
}

.file-path {
  font-family: var(--font-mono);
  font-size: var(--text-body);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-badge {
  font-size: var(--text-meta);
  background: var(--surface-control);
  color: var(--text-secondary);
  padding: 1px 4px;
  border-radius: 3px;
  flex-shrink: 0;
}

.file-stats {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  font-size: var(--text-row);
  font-family: var(--font-mono);
}
.stat-add {
  color: var(--add);
}
.stat-del {
  color: var(--del);
}
</style>
