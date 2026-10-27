<script setup lang="ts">
import { computed } from 'vue';
import { useCommitsStore } from '../../stores/commits';
import { parentDir } from '../../utils/folders';

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

interface FileEntry {
  file: FileDiff;
  index: number;
  path: string;
  name: string;
  add: number;
  del: number;
}

const entries = computed<FileEntry[]>(() =>
  commits.diff.map((file, index) => {
    const path = file.newPath || file.oldPath || '';
    let add = 0;
    let del = 0;
    for (const h of file.hunks ?? []) {
      for (const l of h.lines ?? []) {
        if (l.type === 'add') {
          add++;
        } else if (l.type === 'del') {
          del++;
        }
      }
    }
    return { file, index, path, name: path.split('/').pop() ?? path, add, del };
  })
);

const groups = computed(() => {
  const byDir = new Map<string, FileEntry[]>();
  for (const e of entries.value) {
    const dir = parentDir(e.path) || '/';
    const list = byDir.get(dir);
    if (list) {
      list.push(e);
    } else {
      byDir.set(dir, [e]);
    }
  }
  return [...byDir].map(([dir, files]) => ({ dir, files }));
});

const totals = computed(() =>
  entries.value.reduce((t, e) => ({ add: t.add + e.add, del: t.del + e.del }), { add: 0, del: 0 })
);

function focusSibling(e: KeyboardEvent, dir: 1 | -1) {
  const el = e.currentTarget as HTMLElement;
  const items = Array.from(
    el.closest('.file-list')?.querySelectorAll<HTMLElement>('.file-item') ?? []
  );
  items[items.indexOf(el) + dir]?.focus();
}

function tabStop(fi: number): number {
  const active = commits.selectedFileIndex ?? 0;
  return fi === active ? 0 : -1;
}
</script>

<template>
  <div class="file-list">
    <div v-if="commits.diffError" class="no-changes no-changes--error" role="alert">
      Couldn't load this commit's changes. {{ commits.diffError }}
    </div>
    <div v-else-if="commits.diff.length === 0" class="no-changes">No changes</div>

    <template v-else>
      <div class="file-summary">
        <span>{{ entries.length }} changed file{{ entries.length === 1 ? '' : 's' }}</span>
        <span class="file-stats">
          <span class="stat-add">+{{ totals.add }}</span>
          <span class="stat-del">−{{ totals.del }}</span>
        </span>
      </div>
      <div v-for="g in groups" :key="g.dir" class="file-group">
        <div class="file-dir" :title="g.dir">
          <svg width="12" height="12" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
          </svg>
          <span class="file-dir-path">{{ g.dir }}</span>
        </div>
        <button
          v-for="e in g.files"
          :key="e.path || e.index"
          class="file-item"
          :class="{ active: e.index === commits.selectedFileIndex }"
          :tabindex="tabStop(e.index)"
          :title="e.path"
          :aria-label="`${e.path}, ${STATUS_NAMES[fileStatus(e.file)]}`"
          @click="commits.selectFile(e.index)"
          @keydown.down.prevent.stop="focusSibling($event, 1)"
          @keydown.up.prevent.stop="focusSibling($event, -1)"
        >
          <span class="file-status" :class="`status-${fileStatus(e.file)}`" aria-hidden="true">{{
            fileStatus(e.file)
          }}</span>
          <span class="file-path">{{ e.name }}</span>
          <span v-if="e.file.isBinary" class="file-badge">binary</span>
          <span v-else class="file-stats">
            <span class="stat-add">+{{ e.add }}</span>
            <span class="stat-del">−{{ e.del }}</span>
          </span>
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.file-list {
  flex: 1;
  overflow-y: auto;
  padding-bottom: var(--space-3);
}

.file-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) var(--inset-panel);
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
}
.file-summary .file-stats {
  font-weight: 400;
  font-size: var(--text-meta);
}

.file-group + .file-group {
  margin-top: var(--space-1);
}

.file-dir {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: var(--space-1) var(--inset-panel);
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  min-width: 0;
}
.file-dir svg {
  flex-shrink: 0;
  fill: none;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linejoin: round;
}
.file-dir-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.no-changes--error {
  color: var(--danger);
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
  padding: var(--space-1) var(--inset-panel) var(--space-1) 30px;
  background: transparent;
  border: none;
  cursor: default;
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
  font-size: var(--text-row);
  color: var(--text-primary);
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
