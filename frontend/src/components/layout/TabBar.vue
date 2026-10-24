<script setup lang="ts">
import { useReposStore } from '../../stores/repos';
import { useRepoPickerStore } from '../../stores/repoPicker';

const repos = useReposStore();
const picker = useRepoPickerStore();
</script>

<template>
  <div class="tab-bar">
    <div class="tab-strip" role="tablist" aria-label="Open repositories">
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
          {{ tab.name }}
        </button>
        <button
          class="tab-close"
          :aria-label="`Close ${tab.name}`"
          title="Close tab (⌘W)"
          @click="repos.closeTab(i)"
        >
          <svg width="8" height="8" viewBox="0 0 8 8" aria-hidden="true">
            <path d="M1 1l6 6M7 1L1 7" stroke="currentColor" stroke-width="1.4" />
          </svg>
        </button>
      </div>
    </div>
    <button
      class="tab-add"
      data-repo-picker-anchor
      title="Open repository (⌘O)"
      aria-label="Open repository"
      @click="picker.toggle($event.currentTarget as HTMLElement)"
    >
      <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
        <path d="M5 0.5v9M0.5 5h9" stroke="currentColor" stroke-width="1.4" />
      </svg>
    </button>
  </div>
</template>

<style scoped>
.tab-bar {
  user-select: none;
  -webkit-user-select: none;
  display: flex;
  align-items: stretch;
  height: var(--tabbar-h);
  background: var(--surface-chrome);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.tab-strip {
  display: flex;
  flex: 1;
  min-width: 0;
  overflow-x: auto;
}

/* Safari-style: tabs share the width, but never grow past a readable name. */
.tab {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1 1 0;
  min-width: 112px;
  max-width: 240px;
  border-right: 1px solid var(--border-subtle);
  color: var(--text-tertiary);
  font-size: var(--text-row);
}

.tab:hover {
  background: var(--surface-hover);
  color: var(--text-secondary);
}

.tab.active {
  background: var(--surface-base);
  color: var(--text-primary);
  margin-bottom: -1px;
}

.tab-select {
  flex: 1;
  min-width: 0;
  height: 100%;
  padding: 0 var(--space-5) 0 var(--space-3);
  background: none;
  border: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tab-close {
  position: absolute;
  right: var(--space-1);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  background: none;
  border: none;
  border-radius: 3px;
  color: inherit;
  cursor: pointer;
  opacity: 0;
}

.tab:hover .tab-close,
.tab.active .tab-close,
.tab-close:focus-visible {
  opacity: 0.6;
}

.tab-close:hover {
  opacity: 1;
  background: var(--surface-control);
}

.tab-add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
}

.tab-add:hover {
  color: var(--text-primary);
  background: var(--surface-hover);
}
</style>
