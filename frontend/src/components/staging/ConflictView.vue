<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useToastStore } from '../../stores/toast';
import { useStagingStore } from '../../stores/staging';
import { GetConflictContent, ResolveConflict } from '../../../wailsjs/go/main/App';
import { parseConflicts, hasConflictMarkers } from '../../utils/conflicts';
import type { ContextChunk, ParsedConflict } from '../../utils/conflicts';

const props = defineProps<{
  repoPath: string;
  path: string;
}>();

const emit = defineEmits<{
  resolved: [];
}>();

const toast = useToastStore();
// During a rebase git swaps the meaning: "ours" is the branch being rebased onto.
const staging = useStagingStore();
const loading = ref(true);
const saving = ref(false);

type SelectedLine = { side: 'ours' | 'theirs'; index: number };

type ConflictChunk = ParsedConflict & {
  accepted: 'ours' | 'theirs' | 'custom' | null;
  selection: SelectedLine[];
  textOverride: string | null; // set when user manually edits preview; null = derive from selection
};
type Chunk = ContextChunk | ConflictChunk;

const chunks = ref<Chunk[]>([]);

const unresolvedCount = computed(
  () =>
    chunks.value.filter((c) => {
      if (c.type !== 'conflict') {
        return false;
      }
      const cc = c as ConflictChunk;
      return cc.selection.length === 0 && cc.textOverride === null;
    }).length
);

async function loadConflict() {
  loading.value = true;
  chunks.value = [];
  try {
    const content = await GetConflictContent(props.repoPath, props.path);
    chunks.value = parseConflicts(content.working).map((c) =>
      c.type === 'context' ? c : { ...c, accepted: null, selection: [], textOverride: null }
    );
  } catch (e) {
    toast.error(e);
  } finally {
    loading.value = false;
  }
}

onMounted(loadConflict);
watch(() => props.path, loadConflict);

function asConflict(c: Chunk): ConflictChunk {
  return c as ConflictChunk;
}

// ── Selection helpers ────────────────────────────────────────────────────────

function selectionIndex(chunk: ConflictChunk, side: 'ours' | 'theirs', lineIdx: number): number {
  return chunk.selection.findIndex((s) => s.side === side && s.index === lineIdx);
}

function isLineSelected(chunk: ConflictChunk, side: 'ours' | 'theirs', lineIdx: number): boolean {
  return selectionIndex(chunk, side, lineIdx) !== -1;
}

function selectionOrder(chunk: ConflictChunk, side: 'ours' | 'theirs', lineIdx: number): number {
  const i = selectionIndex(chunk, side, lineIdx);
  return i === -1 ? -1 : i + 1;
}

function toggleLine(chunk: ConflictChunk, side: 'ours' | 'theirs', lineIdx: number) {
  const idx = selectionIndex(chunk, side, lineIdx);
  if (idx === -1) {
    chunk.selection.push({ side, index: lineIdx });
  } else {
    chunk.selection.splice(idx, 1);
  }
  chunk.textOverride = null;
  chunk.accepted = deriveAccepted(chunk);
}

function deriveAccepted(chunk: ConflictChunk): 'ours' | 'theirs' | 'custom' | null {
  if (chunk.selection.length === 0) {
    return null;
  }
  const allOurs =
    chunk.selection.length === chunk.ours.length &&
    chunk.selection.every((s, i) => s.side === 'ours' && s.index === i);
  if (allOurs) {
    return 'ours';
  }
  const allTheirs =
    chunk.selection.length === chunk.theirs.length &&
    chunk.selection.every((s, i) => s.side === 'theirs' && s.index === i);
  if (allTheirs) {
    return 'theirs';
  }
  return 'custom';
}

function acceptSide(chunk: ConflictChunk, side: 'ours' | 'theirs') {
  const lines = side === 'ours' ? chunk.ours : chunk.theirs;
  if (chunk.accepted === side && chunk.textOverride === null) {
    chunk.selection = [];
    chunk.accepted = null;
  } else {
    chunk.selection = lines.map((_, i) => ({ side, index: i }));
    chunk.accepted = side;
  }
  chunk.textOverride = null;
}

function acceptAll(side: 'ours' | 'theirs') {
  for (const chunk of chunks.value) {
    if (chunk.type === 'conflict') {
      acceptSide(asConflict(chunk), side);
    }
  }
}

function focusSiblingLine(e: KeyboardEvent, dir: 1 | -1) {
  const el = e.currentTarget as HTMLElement;
  const next = (
    dir === 1 ? el.nextElementSibling : el.previousElementSibling
  ) as HTMLElement | null;
  if (next?.getAttribute('role') === 'checkbox') {
    next.focus();
  }
}

// ── Preview / manual edit ────────────────────────────────────────────────────

function previewText(chunk: ConflictChunk): string {
  return chunk.selection
    .map((s) => (s.side === 'ours' ? chunk.ours : chunk.theirs)[s.index])
    .join('\n');
}

function resolvedText(chunk: ConflictChunk): string {
  return chunk.textOverride ?? previewText(chunk);
}

function onPreviewInput(chunk: ConflictChunk, e: Event) {
  chunk.textOverride = (e.target as HTMLTextAreaElement).value;
}

function resetOverride(chunk: ConflictChunk) {
  chunk.textOverride = null;
}

function clearChunk(chunk: ConflictChunk) {
  chunk.selection = [];
  chunk.accepted = null;
  chunk.textOverride = null;
}

// ── Build resolved content ───────────────────────────────────────────────────

function buildResolved(): string {
  const parts: string[] = [];
  for (const chunk of chunks.value) {
    if (chunk.type === 'context') {
      parts.push(chunk.lines.join('\n'));
    } else {
      const c = chunk as ConflictChunk;
      parts.push(resolvedText(c));
    }
  }
  return parts.join('\n') + '\n';
}

async function saveAndStage() {
  if (unresolvedCount.value > 0 || saving.value) {
    return;
  }
  const resolved = buildResolved();
  if (hasConflictMarkers(resolved)) {
    toast.error('The result still contains conflict markers. Edit the preview before saving.');
    return;
  }
  saving.value = true;
  try {
    await ResolveConflict(props.repoPath, props.path, resolved);
    toast.success(`Resolved ${props.path.split('/').pop()}`);
    emit('resolved');
  } catch (e) {
    toast.error(e);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="conflict-view">
    <!-- Header -->
    <div class="cv-header">
      <span class="cv-filepath">{{ path }}</span>
      <span v-if="!loading" class="cv-status" :class="{ resolved: unresolvedCount === 0 }">
        <template v-if="unresolvedCount > 0"
          >⚠ {{ unresolvedCount }} conflict{{ unresolvedCount > 1 ? 's' : '' }} remaining</template
        >
        <template v-else>✓ All conflicts resolved</template>
      </span>
    </div>

    <div v-if="loading" class="cv-loading">Loading conflict…</div>

    <!-- Conflict blocks -->
    <div v-else class="cv-body">
      <template v-for="(chunk, i) in chunks" :key="i">
        <!-- Context lines -->
        <pre v-if="chunk.type === 'context'" class="cv-context">{{ chunk.lines.join('\n') }}</pre>

        <!-- Conflict block -->
        <div
          v-else
          class="cv-conflict"
          :class="{ 'cv-conflict--resolved': asConflict(chunk).selection.length > 0 }"
        >
          <!-- Ours side -->
          <div class="cv-side-header cv-ours-header">
            <span class="cv-side-label">
              Ours — {{ asConflict(chunk).oursLabel }}
              <span v-if="staging.isInRebase" class="cv-side-hint"
                >the branch you're rebasing onto</span
              >
            </span>
            <button
              class="cv-quick-btn cv-quick-ours"
              :class="{ active: asConflict(chunk).accepted === 'ours' }"
              @click="acceptSide(asConflict(chunk), 'ours')"
            >
              {{ asConflict(chunk).accepted === 'ours' ? 'Using Ours' : 'Use Ours' }}
            </button>
          </div>
          <div class="cv-lines cv-lines--ours" role="group" :aria-label="`Ours lines`">
            <div
              v-for="(line, li) in asConflict(chunk).ours"
              :key="'o' + li"
              class="cv-line"
              :class="{ 'cv-line--selected': isLineSelected(asConflict(chunk), 'ours', li) }"
              role="checkbox"
              :aria-checked="isLineSelected(asConflict(chunk), 'ours', li)"
              :tabindex="li === 0 ? 0 : -1"
              :title="
                isLineSelected(asConflict(chunk), 'ours', li)
                  ? 'Click or Space to deselect'
                  : 'Click or Space to include this line'
              "
              @keydown.space.prevent.stop="toggleLine(asConflict(chunk), 'ours', li)"
              @keydown.enter.prevent.stop="toggleLine(asConflict(chunk), 'ours', li)"
              @keydown.down.prevent.stop="focusSiblingLine($event, 1)"
              @keydown.up.prevent.stop="focusSiblingLine($event, -1)"
              @click="toggleLine(asConflict(chunk), 'ours', li)"
            >
              <span class="cv-line-badge">
                <span
                  v-if="selectionOrder(asConflict(chunk), 'ours', li) > 0"
                  class="cv-badge-num ours"
                  >{{ selectionOrder(asConflict(chunk), 'ours', li) }}</span
                >
                <span v-else class="cv-badge-dot">○</span>
              </span>
              <code class="cv-line-content">{{ line || ' ' }}</code>
            </div>
            <div v-if="asConflict(chunk).ours.length === 0" class="cv-line cv-line--empty">
              (empty)
            </div>
          </div>

          <!-- Base (diff3/zdiff3): read-only reference, never written into the result -->
          <template v-if="asConflict(chunk).base">
            <div class="cv-side-header cv-base-header">
              <span class="cv-side-label">
                Base{{ asConflict(chunk).baseLabel ? ` — ${asConflict(chunk).baseLabel}` : '' }}
              </span>
            </div>
            <div class="cv-lines cv-lines--base">
              <div v-for="(line, li) in asConflict(chunk).base" :key="'b' + li" class="cv-line">
                <span class="cv-line-badge" />
                <code class="cv-line-content">{{ line || ' ' }}</code>
              </div>
              <div v-if="asConflict(chunk).base!.length === 0" class="cv-line cv-line--empty">
                (empty)
              </div>
            </div>
          </template>

          <div class="cv-separator" />

          <!-- Theirs side -->
          <div class="cv-side-header cv-theirs-header">
            <span class="cv-side-label">
              Theirs — {{ asConflict(chunk).theirsLabel }}
              <span v-if="staging.isInRebase" class="cv-side-hint">your commit being replayed</span>
            </span>
            <button
              class="cv-quick-btn cv-quick-theirs"
              :class="{ active: asConflict(chunk).accepted === 'theirs' }"
              @click="acceptSide(asConflict(chunk), 'theirs')"
            >
              {{ asConflict(chunk).accepted === 'theirs' ? 'Using Theirs' : 'Use Theirs' }}
            </button>
          </div>
          <div class="cv-lines cv-lines--theirs" role="group" :aria-label="`Theirs lines`">
            <div
              v-for="(line, li) in asConflict(chunk).theirs"
              :key="'t' + li"
              class="cv-line"
              :class="{ 'cv-line--selected': isLineSelected(asConflict(chunk), 'theirs', li) }"
              role="checkbox"
              :aria-checked="isLineSelected(asConflict(chunk), 'theirs', li)"
              :tabindex="li === 0 ? 0 : -1"
              :title="
                isLineSelected(asConflict(chunk), 'theirs', li)
                  ? 'Click or Space to deselect'
                  : 'Click or Space to include this line'
              "
              @keydown.space.prevent.stop="toggleLine(asConflict(chunk), 'theirs', li)"
              @keydown.enter.prevent.stop="toggleLine(asConflict(chunk), 'theirs', li)"
              @keydown.down.prevent.stop="focusSiblingLine($event, 1)"
              @keydown.up.prevent.stop="focusSiblingLine($event, -1)"
              @click="toggleLine(asConflict(chunk), 'theirs', li)"
            >
              <span class="cv-line-badge">
                <span
                  v-if="selectionOrder(asConflict(chunk), 'theirs', li) > 0"
                  class="cv-badge-num theirs"
                  >{{ selectionOrder(asConflict(chunk), 'theirs', li) }}</span
                >
                <span v-else class="cv-badge-dot">○</span>
              </span>
              <code class="cv-line-content">{{ line || ' ' }}</code>
            </div>
            <div v-if="asConflict(chunk).theirs.length === 0" class="cv-line cv-line--empty">
              (empty)
            </div>
          </div>

          <!-- Preview / editor — shown when any selection exists -->
          <template v-if="asConflict(chunk).selection.length > 0">
            <div class="cv-preview-header">
              <span>
                Preview
                <span v-if="asConflict(chunk).textOverride !== null" class="cv-edited-badge"
                  >edited</span
                >
              </span>
              <div class="cv-preview-actions">
                <button
                  v-if="asConflict(chunk).textOverride !== null"
                  class="cv-clear-btn"
                  title="Reset to selection"
                  @click="resetOverride(asConflict(chunk))"
                >
                  Reset
                </button>
                <button class="cv-clear-btn" @click="clearChunk(asConflict(chunk))">Clear</button>
              </div>
            </div>
            <textarea
              class="cv-preview-editor"
              :value="resolvedText(asConflict(chunk))"
              spellcheck="false"
              @input="onPreviewInput(asConflict(chunk), $event)"
            />
          </template>
        </div>
      </template>
    </div>

    <!-- Footer -->
    <div v-if="!loading" class="cv-footer">
      <button class="cv-footer-btn" @click="acceptAll('ours')">Ours for every conflict</button>
      <button class="cv-footer-btn" @click="acceptAll('theirs')">Theirs for every conflict</button>
      <div class="cv-footer-spacer" />
      <button
        class="cv-save-btn"
        :class="{ 'cv-save-btn--ready': unresolvedCount === 0 }"
        :disabled="unresolvedCount > 0 || saving"
        @click="saveAndStage"
      >
        {{ saving ? 'Saving…' : 'Save & Stage' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.conflict-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
  background: var(--surface-sunken);
}

.cv-side-hint {
  margin-left: var(--space-2);
  font-weight: 400;
  color: var(--text-tertiary);
}

/* Base section (diff3) */
.cv-base-header {
  background: var(--surface-raised);
  border-top: 1px solid var(--border-subtle);
}
.cv-lines--base .cv-line {
  cursor: default;
  color: var(--text-tertiary);
}

/* Header */
.cv-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 14px;
  background: var(--surface-base);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.cv-filepath {
  font-family: var(--font-mono);
  font-size: var(--text-row);
  color: var(--text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cv-status {
  font-size: var(--text-meta);
  color: var(--modified);
  white-space: nowrap;
}
.cv-status.resolved {
  color: var(--add);
}

.cv-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  font-size: var(--text-body);
}

/* Body */
.cv-body {
  flex: 1;
  overflow-y: auto;
  padding: 0;
}

/* Context */
.cv-context {
  margin: 0;
  padding: 3px 14px;
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  line-height: 1.5;
  color: var(--text-tertiary);
  background: var(--surface-sunken);
  white-space: pre-wrap;
  word-break: break-all;
  tab-size: 4;
}

/* Conflict block */
.cv-conflict {
  margin: 8px 10px;
  border: 1px solid var(--danger-border);
  border-radius: 6px;
  overflow: hidden;
}
.cv-conflict--resolved {
  border-color: var(--add-border);
}

/* Side header */
.cv-side-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 5px 8px 5px 10px;
  border-bottom: 1px solid var(--border-subtle);
}
.cv-ours-header {
  background: var(--ours-soft);
  border-color: var(--ours-border);
}
.cv-theirs-header {
  background: var(--theirs-soft);
  border-color: var(--theirs-border);
}

.cv-side-label {
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Quick accept buttons */
.cv-quick-btn {
  flex-shrink: 0;
  padding: 2px 9px;
  border-radius: 3px;
  font-size: var(--text-meta);
  font-weight: 600;
  cursor: default;
  border: 1px solid transparent;
  transition: background 0.1s;
}
.cv-quick-ours {
  background: var(--ours-soft);
  color: var(--ours);
  border-color: var(--ours-border);
}
.cv-quick-ours:hover {
  background: var(--ours-soft-strong);
}
.cv-quick-ours.active {
  background: var(--ours-soft-strong);
  color: var(--ours);
}
.cv-quick-theirs {
  background: var(--theirs-soft);
  color: var(--theirs);
  border-color: var(--theirs-border);
}
.cv-quick-theirs:hover {
  background: var(--theirs-soft);
}
.cv-quick-theirs.active {
  background: var(--theirs-soft-strong);
  color: var(--theirs);
}

/* Line list */
.cv-lines {
  padding: 3px 0;
}
.cv-lines--ours {
  background: var(--ours-line-bg);
}
.cv-lines--theirs {
  background: var(--theirs-line-bg);
}

/* Individual line */
.cv-line {
  display: flex;
  align-items: baseline;
  gap: 0;
  padding: 1px 8px 1px 4px;
  cursor: default;
  user-select: none;
  transition: background 0.08s;
  min-height: 20px;
}
.cv-line:hover {
  background: rgba(255, 255, 255, 0.04);
}
.cv-lines--ours .cv-line--selected {
  background: var(--ours-soft);
}
.cv-lines--theirs .cv-line--selected {
  background: var(--theirs-soft);
}

.cv-line--empty {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  font-style: italic;
  padding: 4px 10px;
  cursor: default;
}

/* Selection badge */
.cv-line-badge {
  width: 22px;
  flex-shrink: 0;
  text-align: center;
  font-size: var(--text-meta);
  line-height: 20px;
}
.cv-badge-dot {
  color: var(--border-strong);
}
.cv-badge-num {
  display: inline-block;
  width: 16px;
  height: 16px;
  line-height: 16px;
  border-radius: 50%;
  font-size: var(--text-meta);
  font-weight: 700;
  text-align: center;
}
.cv-badge-num.ours {
  background: var(--ours-soft-strong);
  color: var(--ours);
}
.cv-badge-num.theirs {
  background: var(--theirs-soft-strong);
  color: var(--theirs);
}

/* Line content */
.cv-line-content {
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-all;
  tab-size: 4;
}
.cv-lines--ours .cv-line-content {
  color: var(--ours-fg);
}
.cv-lines--theirs .cv-line-content {
  color: var(--theirs-fg);
}

/* Separator between ours/theirs */
.cv-separator {
  height: 1px;
  background: var(--border-subtle);
}

/* Preview / editor */
.cv-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 5px 10px 4px;
  background: var(--surface-raised);
  border-top: 1px solid var(--border-subtle);
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
}

.cv-preview-actions {
  display: flex;
  gap: 4px;
}

.cv-edited-badge {
  margin-left: 5px;
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--modified);
  background: var(--warning-soft);
  border: 1px solid var(--warning-border);
  border-radius: 3px;
  padding: 0 4px;
  vertical-align: middle;
}

.cv-clear-btn {
  font-size: var(--text-meta);
  padding: 1px 6px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--text-tertiary);
  cursor: default;
  text-transform: none;
  font-weight: 400;
  letter-spacing: 0;
}
.cv-clear-btn:hover {
  border-color: var(--focus-ring);
  color: var(--accent-text);
}

.cv-preview-editor {
  display: block;
  width: 100%;
  min-height: 60px;
  max-height: 240px;
  resize: vertical;
  background: var(--surface-field);
  border: none;
  border-top: 1px solid var(--border);
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  line-height: 1.55;
  padding: 6px 10px 6px 26px;
  outline: none;
  box-sizing: border-box;
  overflow-y: auto;
}
.cv-preview-editor:focus {
  background: var(--surface-field);
  border-top-color: var(--focus-ring);
}

/* Footer */
.cv-footer {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: var(--surface-base);
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.cv-footer-spacer {
  flex: 1;
}

.cv-footer-btn {
  padding: 4px 10px;
  background: var(--surface-control);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-tertiary);
  font-size: var(--text-meta);
  cursor: default;
}
.cv-footer-btn:hover {
  background: var(--surface-control-hover);
  color: var(--text-tertiary);
  border-color: var(--border-strong);
}

.cv-save-btn {
  padding: 5px 16px;
  border-radius: 4px;
  font-size: var(--text-row);
  font-weight: 600;
  cursor: default;
  border: 1px solid var(--border);
  background: var(--surface-control);
  color: var(--text-tertiary);
  transition: background 0.15s;
}
.cv-save-btn:disabled {
  cursor: default;
}
.cv-save-btn--ready {
  background: var(--accent-fill);
  color: var(--text-on-accent);
  border-color: var(--accent-fill);
}
.cv-save-btn--ready:hover {
  background: var(--accent-fill-hover);
}
</style>
