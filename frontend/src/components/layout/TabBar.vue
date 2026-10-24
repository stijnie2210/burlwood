<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useReposStore } from '../../stores/repos';

const repos = useReposStore();

const showPicker = ref(false);
const pickerPos = ref({ x: 0, y: 0 });
const addBtnRef = ref<HTMLButtonElement | null>(null);

function shortenPath(path: string): string {
  return path.replace(/^\/Users\/[^/]+/, '~');
}

async function openPicker() {
  if (showPicker.value) {
    showPicker.value = false;
    return;
  }
  await repos.loadRecents();
  if (addBtnRef.value) {
    const rect = addBtnRef.value.getBoundingClientRect();
    pickerPos.value = { x: rect.left, y: rect.bottom };
  }
  showPicker.value = true;
}

function closePicker() {
  showPicker.value = false;
}

async function openRecent(path: string) {
  closePicker();
  await repos.openRepo(path);
}

async function browseFolder() {
  closePicker();
  await repos.pickAndOpen();
}

function onWindowMouseDown(e: MouseEvent) {
  const target = e.target as HTMLElement;
  if (!target.closest('.repo-picker') && !target.closest('.tab-add')) {
    closePicker();
  }
}

onMounted(() => window.addEventListener('mousedown', onWindowMouseDown));
onUnmounted(() => window.removeEventListener('mousedown', onWindowMouseDown));
</script>

<template>
  <div class="tab-bar" role="tablist" aria-label="Open repositories">
    <div
      v-for="(tab, i) in repos.tabs"
      :key="tab.path"
      class="tab"
      :class="{ active: repos.activeIndex === i }"
    >
      <button
        class="tab-select"
        role="tab"
        :aria-selected="repos.activeIndex === i"
        :title="tab.path"
        @click="repos.setActive(i)"
      >
        <span class="tab-name">{{ tab.name }}</span>
      </button>
      <button
        class="tab-close"
        :aria-label="`Close ${tab.name}`"
        title="Close tab (⌘W)"
        @click="repos.closeTab(i)"
      >
        ×
      </button>
    </div>
    <button
      ref="addBtnRef"
      class="tab-add"
      title="Open repository (⌘O)"
      aria-label="Open repository"
      @click="openPicker()"
    >
      +
    </button>
  </div>

  <Teleport to="body">
    <div
      v-if="showPicker"
      class="repo-picker"
      :style="{ left: pickerPos.x + 'px', top: pickerPos.y + 'px' }"
    >
      <div class="picker-section-label">Recent</div>
      <div v-if="repos.recentRepos.length === 0" class="picker-empty">No recent repositories</div>
      <button
        v-for="r in repos.recentRepos"
        :key="r.path"
        class="picker-item"
        @click="openRecent(r.path)"
      >
        <span class="picker-item-name">{{ r.name }}</span>
        <span class="picker-item-path">{{ shortenPath(r.path) }}</span>
      </button>
      <div class="picker-divider" />
      <button class="picker-item picker-browse" @click="browseFolder()">
        Browse for folder...
      </button>
    </div>
  </Teleport>
</template>

<style scoped>
.tab-bar {
  user-select: none;
  -webkit-user-select: none;
  display: flex;
  align-items: center;
  background: var(--surface-chrome);
  border-bottom: 1px solid var(--border);
  height: 36px;
  overflow-x: auto;
  flex-shrink: 0;
}

.tab {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 6px 0 0;
  height: 100%;
  border-right: 1px solid var(--border);
  color: var(--text-secondary);
  font-size: var(--text-body);
  white-space: nowrap;
  transition: background 0.1s;
}

.tab-select {
  height: 100%;
  padding: 0 4px 0 12px;
  background: none;
  border: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
  white-space: nowrap;
}

.tab:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.tab.active {
  background: var(--surface-control);
  color: var(--text-primary);
  border-bottom: 2px solid var(--focus-ring);
}

.tab-close {
  background: none;
  border: none;
  border-radius: 3px;
  color: inherit;
  cursor: pointer;
  opacity: 0.4;
  font-size: 16px;
  line-height: 1;
  padding: 0 2px;
}
.tab-close:focus-visible,
.tab-close:hover {
  opacity: 1;
  color: var(--danger);
}

.tab-add {
  padding: 0 12px;
  height: 100%;
  background: none;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
}
.tab-add:hover {
  color: var(--text-secondary);
}
</style>

<style>
.repo-picker {
  position: fixed;
  z-index: 1000;
  min-width: 280px;
  max-width: 360px;
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: var(--shadow-popover);
  padding: 6px 0;
  max-height: 380px;
  overflow-y: auto;
}

.picker-section-label {
  padding: 4px 12px 6px;
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
}

.picker-empty {
  padding: 6px 12px;
  font-size: var(--text-body);
  color: var(--text-tertiary);
}

.picker-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  width: 100%;
  padding: 6px 12px;
  background: none;
  border: none;
  cursor: pointer;
  text-align: left;
  gap: 1px;
}

.picker-item:hover {
  background: var(--surface-hover);
}

.picker-item-name {
  font-size: var(--text-body);
  color: var(--text-primary);
  font-weight: 500;
}

.picker-item-path {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.picker-divider {
  height: 1px;
  background: var(--border);
  margin: 4px 0;
}

.picker-browse {
  color: var(--accent-text);
  font-size: var(--text-body);
  padding: 8px 12px;
}

.picker-browse:hover {
  background: var(--surface-hover);
}
</style>
