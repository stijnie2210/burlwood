<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useReposStore } from '../../stores/repos';
import { useRepoPickerStore } from '../../stores/repoPicker';
import {
  clampMenuPosition,
  focusFirstMenuItem,
  handleMenuKeydown,
} from '../../composables/useContextMenu';

const repos = useReposStore();
const picker = useRepoPickerStore();
const pickerEl = ref<HTMLElement | null>(null);
const pos = ref({ x: 0, y: 0 });

let returnFocus: HTMLElement | null = null;

watch(
  () => picker.open,
  async (open) => {
    if (!open) {
      const el = returnFocus;
      returnFocus = null;
      if (el?.isConnected) {
        el.focus();
      }
      return;
    }
    returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    pos.value = { x: picker.x, y: picker.y };
    await nextTick();
    if (pickerEl.value) {
      pos.value = clampMenuPosition(pickerEl.value, picker.x, picker.y);
    }
    focusFirstMenuItem(pickerEl.value);
  }
);

function shortenPath(path: string): string {
  return path.replace(/^\/Users\/[^/]+/, '~');
}

async function openRecent(path: string) {
  picker.close();
  await repos.openRepo(path);
}

async function browseFolder() {
  picker.close();
  await repos.pickAndOpen();
}

function onWindowMouseDown(e: MouseEvent) {
  const target = e.target as HTMLElement;
  if (!target.closest('.repo-picker') && !target.closest('[data-repo-picker-anchor]')) {
    picker.close();
  }
}

onMounted(() => window.addEventListener('mousedown', onWindowMouseDown));
onUnmounted(() => window.removeEventListener('mousedown', onWindowMouseDown));
</script>

<template>
  <Teleport to="body">
    <div
      v-if="picker.open"
      ref="pickerEl"
      class="repo-picker"
      :style="{ left: pos.x + 'px', top: pos.y + 'px' }"
      @keydown="handleMenuKeydown($event, pickerEl, picker.close)"
    >
      <div class="picker-section-label">Recent</div>
      <div v-if="repos.recentRepos.length === 0" class="picker-empty">No recent repositories</div>
      <button
        v-for="r in repos.recentRepos"
        :key="r.path"
        class="picker-item"
        :class="{ current: repos.activeRepo?.path === r.path }"
        @click="openRecent(r.path)"
      >
        <span class="picker-item-name">{{ r.name }}</span>
        <span class="picker-item-path">{{ shortenPath(r.path) }}</span>
      </button>
      <div class="picker-divider" />
      <button class="picker-item picker-browse" @click="browseFolder()">
        Open Repository…
        <kbd class="picker-shortcut">⌘O</kbd>
      </button>
    </div>
  </Teleport>
</template>

<style scoped>
.repo-picker {
  position: fixed;
  z-index: 9999;
  min-width: 280px;
  max-width: 360px;
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: var(--shadow-popover);
  padding: var(--space-1) 0;
  max-height: 380px;
  overflow-y: auto;
}

.picker-section-label {
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
}

.picker-empty {
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-row);
  color: var(--text-tertiary);
}

.picker-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  width: 100%;
  padding: 6px var(--space-3);
  background: none;
  border: none;
  cursor: default;
  text-align: left;
  gap: 2px;
  color: var(--text-primary);
}

.picker-item:hover,
.picker-item:focus-visible {
  background: var(--accent-fill);
  color: var(--text-on-accent);
  outline: none;
}

.picker-item-name {
  font-size: var(--text-body);
  font-weight: 500;
}

.picker-item.current .picker-item-name::after {
  content: ' · open';
  font-weight: 400;
  color: var(--text-tertiary);
}

.picker-item-path {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.picker-item:hover .picker-item-path,
.picker-item:focus-visible .picker-item-path,
.picker-item:hover.current .picker-item-name::after,
.picker-item:focus-visible.current .picker-item-name::after {
  color: inherit;
  opacity: 0.8;
}

.picker-divider {
  height: 1px;
  background: var(--border-subtle);
  margin: var(--space-1) 0;
}

.picker-browse {
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  font-size: var(--text-body);
}

.picker-shortcut {
  font-family: inherit;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.picker-browse:hover .picker-shortcut,
.picker-browse:focus-visible .picker-shortcut {
  color: inherit;
  opacity: 0.8;
}
</style>
