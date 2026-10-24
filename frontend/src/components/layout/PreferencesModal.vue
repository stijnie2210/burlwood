<script setup lang="ts">
import { ref, watch } from 'vue';
import { usePrefsStore } from '../../stores/prefs';
import { useDialog } from '../../composables/useDialog';
import type { AppPrefs } from '../../stores/prefs';

const prefs = usePrefsStore();

const draft = ref<AppPrefs>({ ...prefs.prefs });

watch(
  () => prefs.modalOpen,
  (open) => {
    if (open) {
      draft.value = { ...prefs.prefs };
    }
  }
);

const EDITOR_PRESETS = [
  { label: 'VS Code', cmd: 'code' },
  { label: 'Cursor', cmd: 'cursor' },
  { label: 'Zed', cmd: 'zed' },
  { label: 'Sublime', cmd: 'subl' },
  { label: 'Vim', cmd: 'vim' },
];

const INTERVAL_OPTIONS = [
  { label: '30 seconds', value: 30 },
  { label: '1 minute', value: 60 },
  { label: '5 minutes', value: 300 },
  { label: '15 minutes', value: 900 },
  { label: '30 minutes', value: 1800 },
];

const LIMIT_OPTIONS = [500, 1000, 2000, 5000, 10000];
const CONTEXT_OPTIONS = [1, 2, 3, 5, 10];

async function onSave() {
  await prefs.save(draft.value);
  prefs.closeModal();
}

function onCancel() {
  prefs.closeModal();
}

function onOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) {
    onCancel();
  }
}

const panelRef = ref<HTMLElement | null>(null);
const { onKeydown } = useDialog(() => prefs.modalOpen, panelRef, onCancel);
</script>

<template>
  <Teleport to="body">
    <div v-if="prefs.modalOpen" class="prefs-overlay" @click="onOverlayClick">
      <div
        ref="panelRef"
        class="prefs-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="prefs-title"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <div class="prefs-header">
          <span id="prefs-title" class="prefs-title">Preferences</span>
          <button class="prefs-close" aria-label="Close preferences" @click="onCancel">✕</button>
        </div>

        <div class="prefs-body">
          <!-- General -->
          <section class="prefs-section">
            <h3 class="prefs-section-title">General</h3>

            <div class="prefs-field prefs-field--stacked">
              <label class="prefs-label">External editor</label>
              <input
                v-model="draft.editor"
                class="prefs-input"
                placeholder="e.g. code  (empty = use $VISUAL / $EDITOR)"
                spellcheck="false"
              />
              <div class="editor-presets">
                <button
                  v-for="p in EDITOR_PRESETS"
                  :key="p.cmd"
                  class="preset-btn"
                  :class="{ active: draft.editor === p.cmd }"
                  @click="draft.editor = draft.editor === p.cmd ? '' : p.cmd"
                >
                  {{ p.label }}
                </button>
              </div>
            </div>
          </section>

          <!-- Auto-fetch -->
          <section class="prefs-section">
            <h3 class="prefs-section-title">Auto-fetch</h3>

            <div class="prefs-row">
              <span class="prefs-label">Enable auto-fetch</span>
              <label class="toggle">
                <input v-model="draft.autoFetchEnabled" type="checkbox" />
                <span class="toggle-track"><span class="toggle-thumb" /></span>
              </label>
            </div>

            <div class="prefs-row" :class="{ 'prefs-row--disabled': !draft.autoFetchEnabled }">
              <span class="prefs-label">Interval</span>
              <select
                v-model="draft.autoFetchIntervalSecs"
                class="prefs-select"
                :disabled="!draft.autoFetchEnabled"
              >
                <option v-for="opt in INTERVAL_OPTIONS" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </section>

          <!-- Commit graph -->
          <section class="prefs-section">
            <h3 class="prefs-section-title">Commit graph</h3>

            <div class="prefs-row">
              <span class="prefs-label">Initial commit limit</span>
              <select v-model="draft.commitGraphLimit" class="prefs-select">
                <option v-for="n in LIMIT_OPTIONS" :key="n" :value="n">
                  {{ n.toLocaleString() }}
                </option>
              </select>
            </div>
          </section>

          <!-- Diff -->
          <section class="prefs-section">
            <h3 class="prefs-section-title">Diff</h3>

            <div class="prefs-row">
              <span class="prefs-label">Context lines</span>
              <select v-model="draft.diffContextLines" class="prefs-select">
                <option v-for="n in CONTEXT_OPTIONS" :key="n" :value="n">{{ n }}</option>
              </select>
            </div>
          </section>

          <!-- Dates -->
          <section class="prefs-section">
            <h3 class="prefs-section-title">Dates</h3>

            <div class="prefs-row">
              <span class="prefs-label">Date format</span>
              <div class="radio-group">
                <label class="radio-option">
                  <input v-model="draft.dateFormat" type="radio" value="relative" />
                  <span class="radio-label">Relative <em>3d ago</em></span>
                </label>
                <label class="radio-option">
                  <input v-model="draft.dateFormat" type="radio" value="absolute" />
                  <span class="radio-label">Absolute <em>17 Jun</em></span>
                </label>
              </div>
            </div>
          </section>
        </div>

        <div class="prefs-footer">
          <button class="prefs-btn-cancel" @click="onCancel">Cancel</button>
          <button class="prefs-btn-save" @click="onSave">Save</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.prefs-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.prefs-panel {
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 10px;
  width: 460px;
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-modal);
}

/* Header */
.prefs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.prefs-title {
  font-size: var(--text-title);
  font-weight: 600;
  color: var(--text-primary);
}

.prefs-close {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  font-size: var(--text-row);
  cursor: pointer;
  padding: 3px 7px;
  border-radius: 4px;
  line-height: 1;
}
.prefs-close:hover {
  color: var(--text-secondary);
  background: var(--surface-hover);
}

/* Body */
.prefs-body {
  overflow-y: auto;
  flex: 1;
  padding: 4px 0;
}

.prefs-section {
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-subtle);
}
.prefs-section:last-child {
  border-bottom: none;
}

.prefs-section-title {
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
  margin-bottom: 12px;
}

/* Row layout: label left, control right */
.prefs-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 36px;
  gap: 16px;
}
.prefs-row + .prefs-row {
  margin-top: 4px;
}
.prefs-row--disabled {
  opacity: 0.4;
  pointer-events: none;
}

/* Stacked layout: label above, full-width control below */
.prefs-field--stacked {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.prefs-label {
  font-size: var(--text-body);
  color: var(--text-secondary);
  flex-shrink: 0;
}

/* Text input */
.prefs-input {
  width: 100%;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: var(--text-row);
  font-family: var(--font-mono);
  padding: 7px 10px;
  outline: none;
}
.prefs-input:focus {
  border-color: var(--focus-ring);
}
.prefs-input::placeholder {
  color: var(--text-tertiary);
}

/* Select */
.prefs-select {
  appearance: none;
  -webkit-appearance: none;
  background: var(--surface-field)
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath d='M2 4l4 4 4-4' fill='none' stroke='%23556' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E")
    no-repeat right 10px center;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: var(--text-body);
  padding: 10px 32px 10px 12px;
  outline: none;
  cursor: pointer;
  min-width: 160px;
}
.prefs-select:focus {
  border-color: var(--focus-ring);
}
.prefs-select:disabled {
  cursor: default;
}

/* Editor preset chips */
.editor-presets {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.preset-btn {
  padding: 4px 11px;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 20px;
  color: var(--text-tertiary);
  font-size: var(--text-meta);
  cursor: pointer;
  transition:
    color 0.1s,
    border-color 0.1s;
}
.preset-btn:hover {
  color: var(--text-secondary);
  border-color: var(--border-strong);
}
.preset-btn.active {
  border-color: var(--focus-ring);
  color: var(--accent-text);
  background: var(--accent-soft);
}

/* Toggle switch */
.toggle {
  display: inline-flex;
  cursor: pointer;
  flex-shrink: 0;
}
.toggle input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}
.toggle-track {
  width: 34px;
  height: 18px;
  background: var(--surface-control);
  border: 1px solid var(--border);
  border-radius: 9px;
  position: relative;
  transition:
    background 0.15s,
    border-color 0.15s;
}
.toggle input:checked + .toggle-track {
  background: var(--accent-fill);
  border-color: var(--focus-ring);
}
.toggle-thumb {
  position: absolute;
  width: 12px;
  height: 12px;
  background: var(--text-secondary);
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition:
    left 0.15s,
    background 0.15s;
}
.toggle input:checked + .toggle-track .toggle-thumb {
  left: 18px;
  background: var(--text-on-accent);
}

/* Radio group */
.radio-group {
  display: flex;
  gap: 14px;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}
.radio-option input {
  accent-color: var(--accent-text);
  cursor: pointer;
}
.radio-label {
  font-size: var(--text-row);
  color: var(--text-secondary);
}
.radio-label em {
  color: var(--text-tertiary);
  font-style: normal;
  margin-left: 3px;
}

/* Footer */
.prefs-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.prefs-btn-cancel {
  padding: 6px 16px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-tertiary);
  font-size: var(--text-row);
  cursor: pointer;
}
.prefs-btn-cancel:hover {
  color: var(--text-secondary);
  border-color: var(--border-strong);
}

.prefs-btn-save {
  padding: 6px 20px;
  background: var(--accent-fill);
  border: none;
  border-radius: 6px;
  color: var(--text-on-accent);
  font-size: var(--text-row);
  font-weight: 600;
  cursor: pointer;
}
.prefs-btn-save:hover {
  background: var(--accent-fill-hover);
}
</style>
