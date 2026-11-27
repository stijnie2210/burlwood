<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { useDialog } from '../../composables/useDialog';
import { useDiffLayout } from '../../composables/useDiffLayout';
import { useFileHistoryStore, type FileHistoryEntry } from '../../stores/fileHistory';
import { usePrefsStore } from '../../stores/prefs';
import { formatDate } from '../../utils/date';
import { languageForPath } from '../../utils/highlight';
import CodeLine from '../diff/CodeLine';
import DiffLines from '../diff/DiffLines.vue';
import DiffLayoutControls from '../diff/DiffLayoutControls.vue';

const history = useFileHistoryStore();
const prefs = usePrefsStore();
const { layout, wrap } = useDiffLayout();

const BLAME_LINE_BUDGET = 3000;

const panelRef = ref<HTMLElement | null>(null);
const { onKeydown } = useDialog(
  () => history.open,
  panelRef,
  () => history.close()
);

const lang = computed(() => languageForPath(history.path));
const hunks = computed(() => history.diff.flatMap((d) => d.hunks ?? []));
const isBinary = computed(() => history.diff.some((d) => d.isBinary));

const STATUS_NOTES: Record<string, string> = { A: 'Added', D: 'Deleted', C: 'Copied' };

function statusNote(entry: FileHistoryEntry): string {
  if (entry.status === 'R') {
    return `Renamed from ${entry.oldPath}`;
  }
  return STATUS_NOTES[entry.status] ?? '';
}

function date(ts: number): string {
  return formatDate(ts, prefs.prefs.dateFormat, prefs.now);
}

function onOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) {
    history.close();
  }
}

async function moveSelection(by: 1 | -1) {
  const i = history.entries.findIndex((e) => e.hash === history.selectedHash);
  const next = history.entries[i + by];
  if (!next) {
    return;
  }
  history.select(next.hash);
  await nextTick();
  panelRef.value?.querySelector<HTMLElement>(`[data-hash="${next.hash}"]`)?.focus();
}

function showInGraph(hash: string) {
  history.close();
  window.dispatchEvent(new CustomEvent('burlwood:show-commit', { detail: hash }));
}

const showAllBlame = ref(false);
watch(
  () => history.blame,
  () => {
    showAllBlame.value = false;
  }
);

const blameLineCount = computed(() => history.blame.reduce((n, c) => n + c.lines.length, 0));

const visibleBlame = computed(() => {
  if (showAllBlame.value || blameLineCount.value <= BLAME_LINE_BUDGET) {
    return history.blame;
  }
  const out = [];
  let left = BLAME_LINE_BUDGET;
  for (const chunk of history.blame) {
    if (left <= 0) {
      break;
    }
    out.push(chunk.lines.length > left ? { ...chunk, lines: chunk.lines.slice(0, left) } : chunk);
    left -= chunk.lines.length;
  }
  return out;
});

const blameTarget = computed(() => {
  if (!history.blameRev) {
    return 'the working tree';
  }
  const entry = history.blameEntry;
  return entry ? `${entry.shortHash} ${entry.subject}` : history.blameRev.slice(0, 7);
});

function onBlameCommit(hash: string) {
  if (history.entries.some((e) => e.hash === hash)) {
    history.showInHistory(hash);
  } else {
    showInGraph(hash);
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="history.open" class="fh-overlay" @click="onOverlayClick">
      <div
        ref="panelRef"
        class="fh-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="fh-title"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <div class="fh-header">
          <h2 id="fh-title" class="fh-title" :title="history.path">{{ history.path }}</h2>
          <div class="fh-tabs" role="tablist" aria-label="View">
            <button
              role="tab"
              class="fh-tab"
              :class="{ active: history.tab === 'history' }"
              :aria-selected="history.tab === 'history'"
              @click="history.showTab('history')"
            >
              History
            </button>
            <button
              role="tab"
              class="fh-tab"
              :class="{ active: history.tab === 'blame' }"
              :aria-selected="history.tab === 'blame'"
              @click="history.showTab('blame')"
            >
              Blame
            </button>
          </div>
          <DiffLayoutControls v-if="history.tab === 'history'" />
          <button class="fh-close" aria-label="Close" @click="history.close()">✕</button>
        </div>

        <div v-if="history.tab === 'history'" class="fh-body">
          <div class="fh-commits">
            <p v-if="history.historyLoading" class="fh-notice">Loading history…</p>
            <p v-else-if="history.historyError" class="fh-notice fh-error" role="alert">
              {{ history.historyError }}
            </p>
            <p v-else-if="!history.entries.length" class="fh-notice">
              No commits touch this file yet.
            </p>
            <ul
              v-else
              class="fh-list"
              role="listbox"
              :aria-label="`Commits that changed this file`"
            >
              <li
                v-for="entry in history.entries"
                :key="entry.hash"
                class="fh-commit"
                :class="{ selected: entry.hash === history.selectedHash }"
                role="option"
                :aria-selected="entry.hash === history.selectedHash"
                :tabindex="entry.hash === history.selectedHash ? 0 : -1"
                :data-hash="entry.hash"
                @click="history.select(entry.hash)"
                @keydown.up.prevent.stop="moveSelection(-1)"
                @keydown.down.prevent.stop="moveSelection(1)"
              >
                <span class="fh-commit-subject">{{ entry.subject }}</span>
                <span class="fh-commit-meta">
                  <span class="fh-hash">{{ entry.shortHash }}</span>
                  {{ entry.author }} · {{ date(entry.timestamp) }}
                </span>
                <span v-if="statusNote(entry)" class="fh-commit-note">{{ statusNote(entry) }}</span>
              </li>
            </ul>
          </div>

          <div class="fh-diff">
            <div v-if="history.selected" class="fh-diff-bar">
              <span class="fh-diff-path">{{ history.selected.path }}</span>
              <button class="fh-btn" @click="history.blameAt(history.selected)">
                Blame at this commit
              </button>
              <button class="fh-btn" @click="showInGraph(history.selected.hash)">
                Show in graph
              </button>
            </div>
            <div class="fh-diff-scroll">
              <p v-if="history.diffLoading" class="fh-notice">Loading diff…</p>
              <p v-else-if="history.diffError" class="fh-notice fh-error" role="alert">
                {{ history.diffError }}
              </p>
              <p v-else-if="isBinary" class="fh-notice">Binary file not shown</p>
              <p v-else-if="history.selected && !hunks.length" class="fh-notice">
                No line changes in this commit{{
                  history.selected.status === 'R' ? ': the file was only renamed' : ''
                }}.
              </p>
              <template v-else>
                <div v-for="(hunk, hi) in hunks" :key="hunk.header + hi">
                  <div class="fh-hunk-header">{{ hunk.header }}</div>
                  <DiffLines :lines="hunk.lines ?? []" :lang="lang" :layout="layout" :wrap="wrap" />
                </div>
              </template>
            </div>
          </div>
        </div>

        <div v-else class="fh-blame">
          <div class="fh-diff-bar">
            <span class="fh-diff-path">Blame at {{ blameTarget }}</span>
            <button v-if="history.blameRev" class="fh-btn" @click="history.blameAt(null)">
              Blame the working tree
            </button>
          </div>
          <div class="fh-diff-scroll">
            <p v-if="history.blameLoading" class="fh-notice">Loading blame…</p>
            <p v-else-if="history.blameError" class="fh-notice fh-error" role="alert">
              {{ history.blameError }}
            </p>
            <p v-else-if="!history.blame.length" class="fh-notice">This file has no lines.</p>
            <template v-else>
              <div
                v-for="chunk in visibleBlame"
                :key="chunk.startLine"
                class="blame-chunk"
                :class="{ uncommitted: chunk.uncommitted }"
              >
                <div class="blame-who">
                  <span v-if="chunk.uncommitted" class="blame-uncommitted">Not committed yet</span>
                  <button
                    v-else
                    class="blame-commit"
                    :title="chunk.summary"
                    :aria-label="`${chunk.shortHash} ${chunk.summary}, ${chunk.author}, ${date(chunk.timestamp)}`"
                    @click="onBlameCommit(chunk.hash)"
                  >
                    <span class="fh-hash">{{ chunk.shortHash }}</span>
                    <span class="blame-author">{{ chunk.author }}</span>
                    <span class="blame-date">{{ date(chunk.timestamp) }}</span>
                  </button>
                </div>
                <div class="blame-lines">
                  <div v-for="(line, li) in chunk.lines" :key="li" class="blame-line">
                    <span class="blame-num">{{ chunk.startLine + li }}</span>
                    <span class="blame-code"><CodeLine :content="line" :lang="lang" /></span>
                  </div>
                </div>
              </div>
              <button
                v-if="visibleBlame !== history.blame"
                class="fh-show-all"
                @click="showAllBlame = true"
              >
                Show {{ (blameLineCount - BLAME_LINE_BUDGET).toLocaleString() }} more lines
              </button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.fh-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.fh-panel {
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 10px;
  width: min(1180px, 94vw);
  height: 86vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: var(--shadow-modal);
}

.fh-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px 10px 18px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.fh-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-family: var(--font-mono);
  font-size: var(--text-body);
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fh-tabs {
  display: flex;
  padding: 2px;
  background: var(--surface-sunken);
  border: 1px solid var(--border-subtle);
  border-radius: 7px;
  flex-shrink: 0;
}
.fh-tab {
  padding: 3px 12px;
  background: transparent;
  border: none;
  border-radius: 5px;
  color: var(--text-secondary);
  font-size: var(--text-row);
  cursor: default;
}
.fh-tab.active {
  background: var(--surface-control);
  color: var(--text-primary);
  font-weight: 600;
}

.fh-close {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  font-size: var(--text-row);
  cursor: default;
  padding: 3px 7px;
  border-radius: 4px;
  line-height: 1;
}
.fh-close:hover {
  color: var(--text-secondary);
  background: var(--surface-hover);
}

.fh-tab:focus-visible,
.fh-close:focus-visible,
.fh-btn:focus-visible,
.fh-commit:focus-visible,
.blame-commit:focus-visible,
.fh-show-all:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: -2px;
}

.fh-body {
  flex: 1;
  min-height: 0;
  display: flex;
}

.fh-commits {
  width: 320px;
  flex-shrink: 0;
  overflow-y: auto;
  border-right: 1px solid var(--border-subtle);
  background: var(--surface-raised);
}
.fh-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.fh-commit {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 7px 14px;
  border-bottom: 1px solid var(--border-subtle);
  cursor: default;
}
.fh-commit:hover {
  background: var(--surface-hover);
}
.fh-commit.selected {
  background: var(--surface-selected);
}
.fh-commit-subject {
  font-size: var(--text-row);
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fh-commit-meta,
.fh-commit-note {
  font-size: var(--text-meta);
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fh-commit-note {
  color: var(--modified);
}
.fh-hash {
  font-family: var(--font-mono);
  color: var(--text-tertiary);
}

.fh-diff,
.fh-blame {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface-sunken);
}
.fh-diff-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--surface-raised);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.fh-diff-path {
  flex: 1;
  min-width: 0;
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fh-btn {
  flex-shrink: 0;
  padding: 3px 10px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: var(--text-meta);
  cursor: default;
}
.fh-btn:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}
.fh-diff-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  font-size: var(--text-row);
}
.fh-hunk-header {
  padding: 3px 12px;
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  background: var(--surface-raised);
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
}

.fh-notice {
  margin: 0;
  padding: 24px;
  text-align: center;
  font-size: var(--text-row);
  color: var(--text-secondary);
}
.fh-error {
  color: var(--danger);
}

.blame-chunk {
  display: flex;
  border-bottom: 1px solid var(--border-subtle);
}
.blame-chunk.uncommitted {
  background: var(--warning-soft);
}
.blame-who {
  width: 250px;
  flex-shrink: 0;
  border-right: 1px solid var(--border-subtle);
  background: var(--surface-base);
}
.blame-commit {
  display: flex;
  align-items: baseline;
  gap: 8px;
  width: 100%;
  padding: 2px 10px;
  background: transparent;
  border: none;
  text-align: left;
  font-size: var(--text-meta);
  color: var(--text-secondary);
  cursor: default;
}
.blame-commit:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.blame-author {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.blame-date {
  flex-shrink: 0;
  color: var(--text-tertiary);
}
.blame-uncommitted {
  display: block;
  padding: 2px 10px;
  font-size: var(--text-meta);
  color: var(--warning);
}
.blame-lines {
  flex: 1;
  min-width: 0;
}
.blame-line {
  display: flex;
  font-family: var(--font-mono);
  font-size: var(--text-row);
  line-height: 20px;
  white-space: pre;
}
.blame-num {
  width: 48px;
  flex-shrink: 0;
  padding-right: 10px;
  text-align: right;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  user-select: none;
}
.blame-code {
  color: var(--text-primary);
}

.fh-show-all {
  display: block;
  width: 100%;
  padding: 8px;
  background: var(--surface-raised);
  border: none;
  border-top: 1px solid var(--border-subtle);
  color: var(--accent-text);
  font-size: var(--text-row);
  cursor: default;
}
</style>
