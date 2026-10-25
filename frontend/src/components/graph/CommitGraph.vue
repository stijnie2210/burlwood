<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { useCommitsStore } from '../../stores/commits';
import type { GraphRow } from '../../stores/commits';
import { useReposStore } from '../../stores/repos';
import { useToastStore } from '../../stores/toast';
import { useBranchesStore } from '../../stores/branches';
import { usePrefsStore } from '../../stores/prefs';
import { formatDate } from '../../utils/date';
import { CELL_W } from './graphRenderer';
import CommitRow from './CommitRow.vue';
import CommitContextMenu from './CommitContextMenu.vue';

const ROW_H = 28;

const repos = useReposStore();
const commits = useCommitsStore();
const toast = useToastStore();
const branches = useBranchesStore();
const prefsStore = usePrefsStore();

const parentRef = ref<HTMLElement | null>(null);
const searching = computed(() => commits.searchQuery.trim() !== '');

const filteredRows = computed(() => {
  const q = commits.searchQuery.trim().toLowerCase();
  if (!q) {
    return commits.rows;
  }
  // Lane edges describe neighbouring rows, which a filtered list no longer has; show dots only.
  return commits.rows
    .filter(
      (r) =>
        r.hash.startsWith(q) ||
        r.subject.toLowerCase().includes(q) ||
        r.author.toLowerCase().includes(q)
    )
    .map((r) => ({ ...r, edges: [], edgesIn: [] }));
});

// A quiet refresh can insert new commits above the viewport; keep the top visible commit in place.
watch(
  () => commits.rows,
  (next, prev) => {
    const el = parentRef.value;
    if (!el || !prev?.length || next === prev || searching.value) {
      return;
    }
    const topIndex = Math.floor(el.scrollTop / ROW_H);
    const topHash = prev[topIndex]?.hash;
    const newIndex = topHash ? next.findIndex((r) => r.hash === topHash) : -1;
    if (newIndex < 0 || newIndex === topIndex) {
      return;
    }
    const offset = el.scrollTop - topIndex * ROW_H;
    nextTick(() => {
      el.scrollTop = newIndex * ROW_H + offset;
    });
  },
  { flush: 'pre' }
);

const selectedIndex = computed(() =>
  commits.selectedHash ? filteredRows.value.findIndex((r) => r.hash === commits.selectedHash) : -1
);

const virtualizer = useVirtualizer(
  computed(() => ({
    count: filteredRows.value.length,
    getScrollElement: () => parentRef.value,
    estimateSize: () => ROW_H,
    overscan: 15,
  }))
);

const items = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());
// Busy repos can have dozens of parallel lanes; the graph column is capped (8 lanes by default)
// and resizable, so subjects don't start halfway across the window.
const GRAPH_WIDTH_KEY = 'burlwood:graph-width';
const DEFAULT_GRAPH_WIDTH = (8 + 2) * CELL_W;
const MIN_GRAPH_WIDTH = 4 * CELL_W;

function loadGraphLimit(): number {
  try {
    const v = parseInt(localStorage.getItem(GRAPH_WIDTH_KEY) ?? '', 10);
    return Number.isFinite(v) ? v : DEFAULT_GRAPH_WIDTH;
  } catch {
    return DEFAULT_GRAPH_WIDTH;
  }
}

const graphLimit = ref(loadGraphLimit());
const naturalGraphWidth = computed(() => (commits.maxColumn + 2) * CELL_W);
const graphWidth = computed(() => Math.min(naturalGraphWidth.value, graphLimit.value));
const graphTruncated = computed(() => naturalGraphWidth.value > graphLimit.value);

function setGraphLimit(px: number) {
  graphLimit.value = Math.max(
    MIN_GRAPH_WIDTH,
    Math.min(px, Math.max(naturalGraphWidth.value, 600))
  );
  try {
    localStorage.setItem(GRAPH_WIDTH_KEY, String(graphLimit.value));
  } catch {
    // width just won't persist
  }
}

function startGraphResize(e: MouseEvent) {
  const startX = e.clientX;
  const startW = graphWidth.value;
  function onMove(ev: MouseEvent) {
    setGraphLimit(startW + ev.clientX - startX);
  }
  function onUp() {
    window.removeEventListener('mousemove', onMove);
    window.removeEventListener('mouseup', onUp);
  }
  window.addEventListener('mousemove', onMove);
  window.addEventListener('mouseup', onUp);
}

// Double-click toggles between showing every lane and the default cap.
function toggleGraphWidth() {
  setGraphLimit(graphTruncated.value ? naturalGraphWidth.value : DEFAULT_GRAPH_WIDTH);
}

// Scroll to top when the search changes so the first match is visible
watch(
  () => commits.searchQuery,
  () => virtualizer.value.scrollToIndex(0)
);

// A commit picked elsewhere (a tag in the sidebar, a parent link) may be off-screen. Only a new
// selection scrolls: rows shifting under a kept selection (auto-fetch, search) must not.
watch(
  () => commits.selectedHash,
  () => {
    const idx = selectedIndex.value;
    if (idx >= 0) {
      virtualizer.value.scrollToIndex(idx, { align: 'auto' });
    }
  }
);

const fetchStatus = computed(() => {
  // The shared clock ticks once a minute, so a fetch can be newer than "now".
  const at = branches.lastFetchedAt;
  const when = at
    ? `fetched ${formatDate(Math.floor(at / 1000), 'relative', Math.max(prefsStore.now, at))}`
    : 'not fetched yet';
  return prefsStore.prefs.autoFetchEnabled ? `Auto-fetch on · ${when}` : `Auto-fetch off · ${when}`;
});

const countStatus = computed(() => {
  const total = commits.rows.length.toLocaleString();
  if (searching.value) {
    return `${filteredRows.value.length.toLocaleString()} of ${total} commits`;
  }
  return `${total} commits${commits.hasMore ? ' loaded' : ''}`;
});

function onGlobalKeyDown(e: KeyboardEvent) {
  if (e.key === 'y' && !e.metaKey && !e.ctrlKey && !e.altKey) {
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
      return;
    }
    if (commits.selectedHash) {
      navigator.clipboard.writeText(commits.selectedHash);
      toast.success('Hash copied');
    }
    return;
  }

  if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') {
    return;
  }
  if (
    e.target instanceof HTMLElement &&
    (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA')
  ) {
    return;
  }

  e.preventDefault();
  const total = filteredRows.value.length;
  if (total === 0 || !repos.activeRepo) {
    return;
  }

  const idx = selectedIndex.value;
  const next =
    e.key === 'ArrowUp'
      ? Math.max(0, idx <= 0 ? 0 : idx - 1)
      : idx < 0
        ? 0
        : Math.min(idx + 1, total - 1);

  if (next === idx && idx >= 0) {
    return;
  }

  commits.selectCommit(repos.activeRepo.path, filteredRows.value[next].hash);
  virtualizer.value.scrollToIndex(next, { align: 'auto' });
}

function onScroll() {
  const el = parentRef.value;
  if (!el || commits.loadingMore || !commits.hasMore) {
    return;
  }
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 300) {
    commits.loadMore();
  }
}

onMounted(() => window.addEventListener('keydown', onGlobalKeyDown));
onUnmounted(() => window.removeEventListener('keydown', onGlobalKeyDown));

function rowLabel(row: GraphRow | undefined): string | undefined {
  if (!row) {
    return undefined;
  }
  const refs = row.labels.map((l) => (l.type === 'remote' ? `${l.remote}/${l.name}` : l.name));
  return [
    row.subject,
    refs.length ? `refs ${refs.join(', ')}` : '',
    row.author,
    row.date,
    row.shortHash,
  ]
    .filter(Boolean)
    .join(', ');
}

function onRowClick(index: number) {
  const row = filteredRows.value[index];
  if (row && repos.activeRepo) {
    commits.selectCommit(repos.activeRepo.path, row.hash);
  }
}

// Context menu
const menu = ref<{ x: number; y: number; index: number } | null>(null);

let menuReturnFocus: HTMLElement | null = null;

function onRowContextMenu(index: number, e: MouseEvent) {
  menuReturnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  menu.value = { x: e.clientX, y: e.clientY, index };
}

function openMenuForSelected() {
  const idx = selectedIndex.value;
  const rowEl = idx >= 0 ? document.getElementById(`commit-row-${idx}`) : null;
  if (!rowEl) {
    return;
  }
  const rect = rowEl.getBoundingClientRect();
  menuReturnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  menu.value = { x: rect.left + 120, y: rect.bottom, index: idx };
}

function closeMenu() {
  menu.value = null;
  const el = menuReturnFocus;
  menuReturnFocus = null;
  if (el?.isConnected) {
    el.focus();
  }
}

async function onMenuRefresh() {
  menu.value = null;
  if (repos.activeRepo) {
    await commits.load(repos.activeRepo.path);
  }
}
</script>

<template>
  <div class="commit-graph-root">
    <div class="graph-columns" aria-hidden="true">
      <span class="col-graph" :style="{ width: `${graphWidth}px` }" />
      <span class="col-hash">Hash</span>
      <span class="col-message">Message</span>
      <span class="col-author">Author</span>
      <span class="col-date">Date</span>
    </div>

    <div class="graph-viewport">
      <div
        ref="parentRef"
        class="graph-scroller"
        role="listbox"
        aria-label="Commits"
        tabindex="0"
        :aria-activedescendant="selectedIndex >= 0 ? `commit-row-${selectedIndex}` : undefined"
        @scroll="onScroll"
        @keydown.shift.f10.prevent="openMenuForSelected"
      >
        <div v-if="commits.loading" class="loading">Loading commits…</div>

        <div v-else-if="commits.error" class="load-error">
          <span>Failed to load commits</span>
          <code>{{ commits.error }}</code>
        </div>

        <div v-else-if="searching && filteredRows.length === 0" class="no-results">
          No commits match "{{ commits.searchQuery.trim() }}"
        </div>

        <div v-else :style="{ height: `${totalSize}px`, position: 'relative' }">
          <div
            v-for="item in items"
            :id="`commit-row-${item.index}`"
            :key="item.index"
            role="option"
            :aria-selected="commits.selectedHash === filteredRows[item.index]?.hash"
            :aria-posinset="item.index + 1"
            :aria-setsize="filteredRows.length"
            :aria-label="rowLabel(filteredRows[item.index])"
            :style="{
              position: 'absolute',
              top: 0,
              transform: `translateY(${item.start}px)`,
              width: '100%',
              height: `${ROW_H}px`,
            }"
            @click="onRowClick(item.index)"
            @contextmenu.prevent="onRowContextMenu(item.index, $event)"
          >
            <CommitRow
              v-if="filteredRows[item.index]"
              :row="filteredRows[item.index]"
              :graph-width="graphWidth"
              :truncated="graphTruncated"
              :row-h="ROW_H"
              :selected="commits.selectedHash === filteredRows[item.index].hash"
            />
          </div>
        </div>

        <CommitContextMenu
          v-if="menu && filteredRows[menu.index]"
          :row="filteredRows[menu.index]"
          :repo-path="repos.activeRepo?.path ?? ''"
          :x="menu.x"
          :y="menu.y"
          @close="closeMenu"
          @refresh="onMenuRefresh"
        />

        <div v-if="commits.loadingMore" class="loading-more">Loading more commits…</div>
      </div>
      <div
        class="graph-resize"
        :style="{ left: `${graphWidth}px` }"
        role="separator"
        aria-orientation="vertical"
        aria-label="Graph column width"
        :aria-valuenow="graphWidth"
        tabindex="0"
        :title="
          graphTruncated
            ? 'Drag to resize · double-click to show all lanes'
            : 'Drag to resize · double-click to reset'
        "
        @mousedown.prevent="startGraphResize"
        @dblclick="toggleGraphWidth"
        @keydown.left.prevent="setGraphLimit(graphWidth - CELL_W)"
        @keydown.right.prevent="setGraphLimit(graphWidth + CELL_W)"
      />
    </div>
    <footer class="graph-status">
      <span
        class="status-dot"
        :class="{ on: prefsStore.prefs.autoFetchEnabled }"
        aria-hidden="true"
      />
      <span class="status-text">{{ fetchStatus }}</span>
      <span class="status-count">{{ countStatus }}</span>
    </footer>
  </div>
</template>

<style scoped>
.commit-graph-root {
  container: graph / inline-size;
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.graph-columns {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: var(--row-h);
  padding-right: var(--space-4);
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-subtle);
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
  white-space: nowrap;
  user-select: none;
  -webkit-user-select: none;
}
.col-graph {
  flex-shrink: 0;
}
.col-hash {
  flex-shrink: 0;
  width: 52px;
}
.col-message {
  flex: 1;
  min-width: 80px;
}
.col-author {
  flex-shrink: 0;
  width: 150px;
}
.col-date {
  flex-shrink: 0;
  width: 80px;
  text-align: right;
}
@container graph (max-width: 720px) {
  .col-author {
    display: none;
  }
}
@container graph (max-width: 520px) {
  .col-date {
    display: none;
  }
}

.graph-status {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: 26px;
  padding: 0 var(--inset-panel);
  flex-shrink: 0;
  background: var(--surface-raised);
  border-top: 1px solid var(--border-subtle);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  white-space: nowrap;
}
.status-dot {
  width: 6px;
  height: 6px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--text-tertiary);
}
.status-dot.on {
  background: var(--add);
}
.status-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.status-count {
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.graph-viewport {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
}

.graph-resize {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 7px;
  margin-left: -3px;
  cursor: col-resize;
  z-index: 1;
}
.graph-resize::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 3px;
  width: 1px;
  background: transparent;
  transition: background 0.15s;
}
.graph-resize:hover::after,
.graph-resize:focus-visible::after {
  background: var(--accent-text);
}

.graph-scroller {
  flex: 1;
  overflow-y: auto;
  overflow-x: auto;
  background: var(--surface-base);
  min-height: 0;
}

.loading {
  padding: 24px;
  color: var(--text-tertiary);
  text-align: center;
  font-size: var(--text-body);
}

.load-error {
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--del);
  font-size: var(--text-body);
}
.load-error code {
  font-size: var(--text-meta);
  color: var(--danger);
  word-break: break-all;
  font-family: var(--font-mono);
}

.no-results {
  padding: 24px;
  color: var(--text-tertiary);
  text-align: center;
  font-size: var(--text-body);
}

.loading-more {
  padding: 12px;
  color: var(--text-tertiary);
  text-align: center;
  font-size: var(--text-row);
}
</style>
