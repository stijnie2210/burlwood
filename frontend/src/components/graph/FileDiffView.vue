<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useDetailWidth } from '../../composables/useDetailWidth';
import { isImageFile, imageDataUrl } from '../../utils/imageUtils';
import { highlightLine, languageForPath } from '../../utils/highlight';

const fileListWidth = useDetailWidth();

function startFileListResize(e: MouseEvent) {
  const startX = e.clientX;
  const startW = fileListWidth.value;
  function onMove(ev: MouseEvent) {
    fileListWidth.value = Math.max(160, Math.min(600, startW - (ev.clientX - startX)));
  }
  function onUp() {
    window.removeEventListener('mousemove', onMove);
    window.removeEventListener('mouseup', onUp);
  }
  window.addEventListener('mousemove', onMove);
  window.addEventListener('mouseup', onUp);
}
import { useCommitsStore } from '../../stores/commits';
import { useReposStore } from '../../stores/repos';
import { GetFileAtCommitBase64 } from '../../../wailsjs/go/main/App';
import CommitFileList from './CommitFileList.vue';

const commits = useCommitsStore();
const repos = useReposStore();

const file = computed(() =>
  commits.selectedFileIndex !== null ? commits.diff[commits.selectedFileIndex] : null
);

const selectedRow = computed(
  () => commits.rows.find((r) => r.hash === commits.selectedHash) ?? null
);

const lang = computed(() => {
  const f = file.value;
  if (!f) {
    return null;
  }
  return languageForPath(f.newPath || f.oldPath || '');
});

const imageSrc = ref<string | null>(null);
const imageLoading = ref(false);

watch(
  () => commits.selectedFileIndex,
  async () => {
    imageSrc.value = null;
    imageLoading.value = false;
    const f = file.value;
    if (!f?.isBinary) {
      return;
    }
    const path = f.newPath || f.oldPath;
    if (!path || !isImageFile(path)) {
      return;
    }
    const hash = commits.selectedHash;
    const repoPath = repos.activeRepo?.path;
    if (!hash || !repoPath) {
      return;
    }
    imageLoading.value = true;
    try {
      const b64 = await GetFileAtCommitBase64(repoPath, hash, path);
      imageSrc.value = imageDataUrl(path, b64);
    } catch {
      // imageSrc stays null
    } finally {
      imageLoading.value = false;
    }
  },
  { immediate: true }
);

type FileDiff = (typeof commits.diff)[0];

function filePath(f: FileDiff): string {
  return f.newPath || f.oldPath || '';
}

function close() {
  commits.clearFileSelection();
}

function navigateTo(delta: number) {
  if (commits.selectedFileIndex === null) {
    return;
  }
  const next = commits.selectedFileIndex + delta;
  if (next >= 0 && next < commits.diff.length) {
    commits.selectFile(next);
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (
    (e.target as HTMLElement).tagName === 'INPUT' ||
    (e.target as HTMLElement).tagName === 'TEXTAREA'
  ) {
    return;
  }
  if (e.key === 'Escape') {
    close();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    navigateTo(-1);
  } else if (e.key === 'ArrowDown') {
    e.preventDefault();
    navigateTo(1);
  }
}

onMounted(() => document.addEventListener('keydown', onKeyDown));
onUnmounted(() => document.removeEventListener('keydown', onKeyDown));
</script>

<template>
  <div class="fdv">
    <!-- Header bar -->
    <div class="fdv-header">
      <button class="fdv-back" @click="close">
        <span class="fdv-back-arrow">‹</span>
        Commits
      </button>
      <div v-if="file" class="fdv-title">
        <span class="fdv-filepath">{{ filePath(file) }}</span>
      </div>
      <div v-if="selectedRow" class="fdv-commit-chip">
        <span class="fdv-chip-hash">{{ selectedRow.shortHash }}</span>
        <span class="fdv-chip-subject">{{ selectedRow.subject }}</span>
      </div>
    </div>

    <!-- Body -->
    <div class="fdv-body">
      <!-- Diff content -->
      <div v-if="file" class="fdv-diff">
        <template v-if="file.isBinary">
          <template v-if="isImageFile(filePath(file))">
            <div v-if="imageLoading" class="fdv-notice">Loading…</div>
            <div v-else-if="imageSrc" class="fdv-img-wrap">
              <img :src="imageSrc" class="fdv-img" :alt="filePath(file)" />
            </div>
            <div v-else class="fdv-notice">Image not available at this commit</div>
          </template>
          <div v-else class="fdv-notice">Binary file not shown</div>
        </template>
        <template v-else>
          <div v-if="file.hunks.length === 0" class="fdv-notice">No changes</div>
          <div v-for="(hunk, hi) in file.hunks" :key="hunk.header + hi" class="fdv-hunk">
            <div class="fdv-hunk-header">{{ hunk.header }}</div>
            <div
              v-for="(line, li) in hunk.lines"
              :key="li"
              class="diff-line"
              :class="`diff-line--${line.type}`"
            >
              <span class="diff-gutter">{{
                line.type === 'add' ? '+' : line.type === 'del' ? '−' : ' '
              }}</span>
              <span class="diff-lineno old">{{ line.oldLine || '' }}</span>
              <span class="diff-lineno new">{{ line.newLine || '' }}</span>
              <span class="diff-content" v-html="highlightLine(line.content, lang)"></span>
            </div>
          </div>
        </template>
      </div>

      <div class="resize-handle" @mousedown.prevent="startFileListResize" />

      <!-- File list sidebar -->
      <div class="fdv-files" :style="{ width: fileListWidth + 'px' }">
        <div class="fdv-files-title">
          {{ commits.diff.length }} changed file{{ commits.diff.length !== 1 ? 's' : '' }}
        </div>
        <CommitFileList />
      </div>
    </div>
  </div>
</template>

<style scoped>
.fdv {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--surface-base);
}

/* ── Header ── */
.fdv-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 12px;
  height: 38px;
  background: var(--surface-raised);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  min-width: 0;
}

.fdv-back {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--surface-control);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: var(--text-row);
  cursor: pointer;
  flex-shrink: 0;
  transition:
    color 0.15s,
    border-color 0.15s;
}
.fdv-back:hover {
  color: var(--text-primary);
  border-color: var(--focus-ring);
}

.fdv-back-arrow {
  font-size: 16px;
  line-height: 1;
  margin-top: -1px;
}

.fdv-title {
  flex: 1;
  min-width: 0;
}

.fdv-filepath {
  font-family: var(--font-mono);
  font-size: var(--text-row);
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.fdv-commit-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  max-width: 380px;
  min-width: 0;
}

.fdv-chip-hash {
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.fdv-chip-subject {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Body ── */
.fdv-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* ── Diff pane ── */
.fdv-diff {
  flex: 1;
  overflow-y: auto;
  min-width: 0;
}

.fdv-notice {
  padding: 24px;
  color: var(--text-tertiary);
  font-size: var(--text-row);
  text-align: center;
}

.fdv-img-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background: var(--surface-sunken);
}

.fdv-img {
  max-width: 100%;
  max-height: 600px;
  object-fit: contain;
  border-radius: 4px;
  border: 1px solid var(--border-subtle);
}

.fdv-hunk {
  border-bottom: 1px solid var(--border-subtle);
}

.fdv-hunk-header {
  padding: 3px 12px;
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  background: var(--surface-raised);
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
}

/* ── Diff line styles ── */
.diff-line {
  display: flex;
  font-family: var(--font-mono);
  font-size: var(--text-row);
  line-height: 19px;
  white-space: pre;
}
.diff-line--add {
  background: var(--add-line-bg);
}
.diff-line--del {
  background: var(--del-line-bg);
}
.diff-line--context {
  color: var(--text-secondary);
}

.diff-gutter {
  width: 14px;
  text-align: center;
  flex-shrink: 0;
  color: var(--text-tertiary);
  user-select: none;
}
.diff-line--add .diff-gutter {
  color: var(--add);
}
.diff-line--del .diff-gutter {
  color: var(--del);
}

.diff-lineno {
  width: 44px;
  text-align: right;
  padding-right: 8px;
  color: var(--text-tertiary);
  flex-shrink: 0;
  user-select: none;
  font-size: var(--text-meta);
  line-height: 19px;
}
.diff-line--add .diff-lineno.old {
  color: transparent;
}
.diff-line--del .diff-lineno.new {
  color: transparent;
}

.diff-content {
  padding-left: 6px;
  color: var(--text-primary);
  overflow: visible;
}
.diff-line--add .diff-content {
  color: var(--add-fg);
}
.diff-line--del .diff-content {
  color: var(--del-fg);
}

/* ── File list sidebar ── */
.fdv-files {
  flex-shrink: 0;
  border-left: 1px solid var(--border-subtle);
  background: var(--surface-sunken);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.resize-handle {
  width: 4px;
  flex-shrink: 0;
  background: var(--border-subtle);
  cursor: col-resize;
  transition: background 0.15s;
}
.resize-handle:hover {
  background: var(--accent-fill);
}

.fdv-files-title {
  padding: 8px 12px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
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
  cursor: pointer;
}
.diff-show-all:hover {
  background: var(--surface-control);
  color: var(--text-primary);
}
</style>
