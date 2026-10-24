<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { useCommitsStore } from '../../stores/commits';
import { useReposStore } from '../../stores/repos';
import { useToastStore } from '../../stores/toast';
import { CELL_W } from './graphRenderer';
import CommitRow from './CommitRow.vue';
import CommitContextMenu from './CommitContextMenu.vue';

const ROW_H = 28;

const repos = useReposStore();
const commits = useCommitsStore();
const toast = useToastStore();

const parentRef = ref<HTMLElement | null>(null);
const searchQuery = ref('');
const searchVisible = ref(false);
const searchInputRef = ref<HTMLInputElement | null>(null);

const filteredRows = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
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
const graphWidth = computed(() => (commits.maxColumn + 2) * CELL_W);

watch(
  () => repos.activeRepo?.path,
  (path) => {
    if (path) {
      commits.load(path);
    } else {
      commits.clear();
    }
  },
  { immediate: true }
);

// Scroll to top when search results change so first match is visible
watch(filteredRows, () => {
  virtualizer.value.scrollToIndex(0);
});

function openSearch() {
  searchVisible.value = true;
  nextTick(() => searchInputRef.value?.focus());
}

function closeSearch() {
  searchVisible.value = false;
  searchQuery.value = '';
}

function onGlobalKeyDown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
    e.preventDefault();
    openSearch();
    return;
  }

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

function onSearchKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    closeSearch();
  }
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

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeyDown);
  window.addEventListener('gitgood:find', openSearch);
});
onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeyDown);
  window.removeEventListener('gitgood:find', openSearch);
});

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
    <!-- Search bar -->
    <div v-if="searchVisible" class="search-bar">
      <input
        ref="searchInputRef"
        v-model="searchQuery"
        class="search-input"
        placeholder="Search commits…"
        @keydown="onSearchKeyDown"
      />
      <span v-if="searchQuery.trim()" class="search-count">
        {{ filteredRows.length }} result{{ filteredRows.length === 1 ? '' : 's' }}
      </span>
      <button class="search-close" @click="closeSearch">×</button>
    </div>

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

      <div v-else-if="searchQuery.trim() && filteredRows.length === 0" class="no-results">
        No commits match "{{ searchQuery }}"
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
          :aria-label="
            filteredRows[item.index]
              ? `${filteredRows[item.index].shortHash} ${filteredRows[item.index].subject}`
              : undefined
          "
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
      <div
        v-else-if="!commits.hasMore && commits.rows.length > 0 && !searchQuery.trim()"
        class="load-end"
      >
        {{ commits.rows.length }} commits loaded
      </div>
    </div>
  </div>
</template>

<style scoped>
.commit-graph-root {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  background: var(--surface-raised);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: var(--text-body);
  padding: 3px 8px;
  outline: none;
}

.search-input:focus {
  border-color: var(--focus-ring);
}

.search-count {
  font-size: var(--text-row);
  color: var(--text-tertiary);
  white-space: nowrap;
}

.search-close {
  background: none;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 0 2px;
}

.search-close:hover {
  color: var(--text-secondary);
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

.load-end {
  padding: 12px;
  color: var(--text-tertiary);
  text-align: center;
  font-size: var(--text-meta);
}
</style>
