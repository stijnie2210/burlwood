<script setup lang="ts">
import { useDiffLayout } from '../../composables/useDiffLayout';

const { layout, wrap } = useDiffLayout();
</script>

<template>
  <div class="diff-controls">
    <div class="segmented" role="radiogroup" aria-label="Diff layout">
      <button
        role="radio"
        :aria-checked="layout === 'unified'"
        :class="{ active: layout === 'unified' }"
        @click="layout = 'unified'"
      >
        Unified
      </button>
      <button
        role="radio"
        :aria-checked="layout === 'split'"
        :class="{ active: layout === 'split' }"
        @click="layout = 'split'"
      >
        Split
      </button>
    </div>
    <button
      class="wrap-toggle"
      :class="{ active: wrap || layout === 'split' }"
      :aria-pressed="wrap || layout === 'split'"
      :disabled="layout === 'split'"
      :title="layout === 'split' ? 'Split view always wraps long lines' : 'Wrap long lines'"
      @click="wrap = !wrap"
    >
      Wrap
    </button>
  </div>
</template>

<style scoped>
.diff-controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

.segmented {
  display: flex;
  padding: 2px;
  gap: 2px;
  background: var(--surface-sunken);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
}

.segmented button,
.wrap-toggle {
  height: 20px;
  padding: 0 var(--space-2);
  border: none;
  border-radius: 4px;
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-meta);
  font-weight: 500;
  cursor: default;
}

.segmented button:hover:not(.active),
.wrap-toggle:hover:not(:disabled):not(.active) {
  color: var(--text-primary);
}

.segmented button.active {
  background: var(--surface-control-hover);
  color: var(--text-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.wrap-toggle {
  height: 24px;
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
}
.wrap-toggle.active {
  background: var(--accent-soft);
  border-color: var(--accent-border);
  color: var(--accent-text);
}
.wrap-toggle:disabled {
  opacity: 0.6;
}
</style>
