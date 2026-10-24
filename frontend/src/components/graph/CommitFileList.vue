<script setup lang="ts">
import { useCommitsStore } from '../../stores/commits';

const commits = useCommitsStore();

type FileDiff = (typeof commits.diff)[0];

function fileStatus(file: FileDiff): string {
  if (!file.oldPath) {
    return '+';
  }
  if (!file.newPath || file.newPath === file.oldPath) {
    return file.hunks.length ? '~' : '';
  }
  return '→';
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
      @click="commits.selectFile(fi)"
    >
      <span
        class="file-status"
        :class="{
          'status-add': fileStatus(file) === '+',
          'status-mod': fileStatus(file) === '~',
          'status-ren': fileStatus(file) === '→',
        }"
        >{{ fileStatus(file) }}</span
      >
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
.status-add {
  color: var(--add);
}
.status-mod {
  color: var(--modified);
}
.status-ren {
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
