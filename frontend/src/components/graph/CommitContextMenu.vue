<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue';
import {
  clampMenuPosition,
  focusFirstMenuItem,
  handleMenuKeydown,
  keepFocusInMenu,
} from '../../composables/useContextMenu';
import type { GraphRow } from '../../stores/commits';
import { useToastStore } from '../../stores/toast';
import { useCommitsStore } from '../../stores/commits';
import { useBranchesStore } from '../../stores/branches';
import { useStagingStore } from '../../stores/staging';
import { useReposStore } from '../../stores/repos';
import {
  CheckoutRef,
  CreateBranchAt,
  RevertCommit,
  CreateTag,
  MoveTag,
  PushTag,
  ForcePushTag,
  ResetBranch,
  HardResetBranch,
  UndoHardReset,
} from '../../../wailsjs/go/main/App';

const props = defineProps<{
  row: GraphRow;
  repoPath: string;
  x: number;
  y: number;
}>();

const emit = defineEmits<{
  close: [];
  refresh: [];
}>();

const toast = useToastStore();

type Mode = 'menu' | 'branch' | 'tag' | 'move-tag' | 'tag-created' | 'revert-confirm';
const mode = ref<Mode>('menu');
watch(mode, () => nextTick(() => keepFocusInMenu(menuEl.value)));
const inputVal = ref('');
const tagMoved = ref(false);
const inputEl = ref<HTMLInputElement | null>(null);
const menuEl = ref<HTMLElement | null>(null);

const style = ref({ top: '0px', left: '0px' });

onMounted(() => {
  const el = menuEl.value;
  if (!el) {
    return;
  }
  const { x, y } = clampMenuPosition(el, props.x, props.y);
  style.value = { top: `${y}px`, left: `${x}px` };

  document.addEventListener('mousedown', onOutside, true);
  document.addEventListener('keydown', onKey, true);
  focusFirstMenuItem(el);
});

onUnmounted(() => {
  document.removeEventListener('mousedown', onOutside, true);
  document.removeEventListener('keydown', onKey, true);
});

function onOutside(e: MouseEvent) {
  if (menuEl.value && !menuEl.value.contains(e.target as Node)) {
    emit('close');
  }
}

// Capture phase, so arrow keys move through the menu instead of the commit selection behind it.
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close');
    return;
  }
  handleMenuKeydown(e, menuEl.value, () => emit('close'));
}

// ── Actions ──────────────────────────────────────────────────────────────────

async function copyHash(full: boolean) {
  await navigator.clipboard.writeText(full ? props.row.hash : props.row.shortHash);
  toast.success(full ? 'Copied full SHA' : 'Copied short SHA');
  emit('close');
}

async function checkout() {
  try {
    await CheckoutRef(props.repoPath, props.row.hash);
    toast.success(`Checked out ${props.row.shortHash} (detached HEAD)`);
    emit('refresh');
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}

function openBranchInput() {
  mode.value = 'branch';
  inputVal.value = '';
  nextTick(() => inputEl.value?.focus());
}

async function confirmBranch() {
  const name = inputVal.value.trim();
  if (!name) {
    return;
  }
  try {
    await CreateBranchAt(props.repoPath, name, props.row.hash);
    toast.success(`Branch '${name}' created at ${props.row.shortHash}`);
    emit('refresh');
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}

function openRevertConfirm() {
  mode.value = 'revert-confirm';
}

async function revert(commitImmediately: boolean) {
  try {
    await RevertCommit(props.repoPath, props.row.hash, commitImmediately);
    toast.success(
      commitImmediately
        ? `Reverted ${props.row.shortHash} (new commit created)`
        : `Reverted ${props.row.shortHash} — changes staged, ready to commit`
    );
    emit('refresh');
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}

async function reset(resetMode: 'soft' | 'mixed' | 'hard') {
  if (resetMode === 'hard') {
    await hardReset();
    return;
  }
  try {
    await ResetBranch(props.repoPath, props.row.hash, resetMode);
    toast.success(`Branch reset (${resetMode}) to ${props.row.shortHash}`);
    emit('refresh');
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}

async function hardReset() {
  const repo = props.repoPath;
  const short = props.row.shortHash;
  try {
    const undo = await HardResetBranch(repo, props.row.hash);
    await reloadRepo(repo);
    const lost = undo.stash ? ', local changes discarded' : '';
    toast.withUndo(`Hard reset to ${short}${lost}`, async () => {
      try {
        await UndoHardReset(repo, undo);
        toast.success(`Reset undone, back at ${undo.head.slice(0, 7)}`);
      } catch (e) {
        toast.error(e);
      }
      await reloadRepo(repo);
    });
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}

// The menu is unmounted by the time Undo is clicked, so stores are reloaded directly.
async function reloadRepo(repo: string) {
  if (useReposStore().activeRepo?.path !== repo) {
    return;
  }
  await Promise.all([
    useCommitsStore().load(repo),
    useBranchesStore().load(repo),
    useStagingStore().load(repo),
  ]);
}

function openTagInput() {
  mode.value = 'tag';
  tagMoved.value = false;
  inputVal.value = '';
  nextTick(() => inputEl.value?.focus());
}

function openMoveTagInput() {
  mode.value = 'move-tag';
  tagMoved.value = true;
  inputVal.value = '';
  nextTick(() => inputEl.value?.focus());
}

async function confirmMoveTag() {
  const name = inputVal.value.trim();
  if (!name) {
    return;
  }
  try {
    await MoveTag(props.repoPath, name, props.row.hash);
    emit('refresh');
    mode.value = 'tag-created';
  } catch (e) {
    toast.error(e);
    emit('close');
  }
}

async function confirmTag() {
  const name = inputVal.value.trim();
  if (!name) {
    return;
  }
  try {
    await CreateTag(props.repoPath, name, props.row.hash);
    emit('refresh');
    mode.value = 'tag-created';
  } catch (e) {
    toast.error(e);
    emit('close');
  }
}

async function pushCreatedTag() {
  const name = inputVal.value.trim();
  try {
    if (tagMoved.value) {
      await ForcePushTag(props.repoPath, name);
    } else {
      await PushTag(props.repoPath, name);
    }
    toast.success(tagMoved.value ? `Tag '${name}' force-pushed` : `Tag '${name}' pushed`);
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}

async function pushTag(name: string) {
  try {
    await PushTag(props.repoPath, name);
    toast.success(`Tag '${name}' pushed`);
  } catch (e) {
    toast.error(e);
  }
  emit('close');
}
</script>

<template>
  <Teleport to="body">
    <div ref="menuEl" class="ctx-menu" role="menu" :style="style" @click.stop>
      <!-- ── Revert confirmation ──────────────────── -->
      <template v-if="mode === 'revert-confirm'">
        <div class="ctx-input-header">Revert {{ row.shortHash }}</div>
        <div class="ctx-choices">
          <button class="ctx-choice" @click="revert(true)">
            <span class="ctx-choice-title">Revert and commit</span>
            <span class="ctx-choice-desc"
              >Creates a new commit that undoes {{ row.shortHash }}</span
            >
          </button>
          <button class="ctx-choice" @click="revert(false)">
            <span class="ctx-choice-title">Stage the revert</span>
            <span class="ctx-choice-desc"
              >Leaves the changes staged so you can edit them first</span
            >
          </button>
        </div>
      </template>

      <!-- ── Branch input ─────────────────────────── -->
      <template v-else-if="mode === 'branch'">
        <div class="ctx-input-header">Create branch at {{ row.shortHash }}</div>
        <div class="ctx-input-row">
          <input
            ref="inputEl"
            v-model="inputVal"
            class="ctx-input"
            placeholder="branch-name"
            @keydown.enter="confirmBranch"
            @keydown.escape="emit('close')"
          />
          <button class="ctx-confirm" @click="confirmBranch">Create</button>
        </div>
      </template>

      <!-- ── Tag input ────────────────────────────── -->
      <template v-else-if="mode === 'tag'">
        <div class="ctx-input-header">Create tag at {{ row.shortHash }}</div>
        <div class="ctx-input-row">
          <input
            ref="inputEl"
            v-model="inputVal"
            class="ctx-input"
            placeholder="tag-name"
            @keydown.enter="confirmTag"
            @keydown.escape="emit('close')"
          />
          <button class="ctx-confirm" @click="confirmTag">Create</button>
        </div>
      </template>

      <!-- ── Move tag input ────────────────────────────── -->
      <template v-else-if="mode === 'move-tag'">
        <div class="ctx-input-header">Move tag to {{ row.shortHash }}</div>
        <div class="ctx-input-row">
          <input
            ref="inputEl"
            v-model="inputVal"
            class="ctx-input"
            placeholder="existing-tag-name"
            @keydown.enter="confirmMoveTag"
            @keydown.escape="emit('close')"
          />
          <button class="ctx-confirm" @click="confirmMoveTag">Move</button>
        </div>
      </template>

      <!-- ── Tag created ────────────────────────────── -->
      <template v-else-if="mode === 'tag-created'">
        <div class="ctx-input-header">
          Tag '{{ inputVal }}' {{ tagMoved ? 'moved' : 'created' }}
        </div>
        <div v-if="tagMoved" class="ctx-confirm-question">
          The remote still points '{{ inputVal }}' at the old commit. Force-pushing replaces it
          there, and anyone who already fetched it keeps the old one.
        </div>
        <div class="ctx-confirm-row">
          <button
            v-if="tagMoved"
            class="ctx-confirm-btn ctx-confirm-btn--danger"
            @click="pushCreatedTag"
          >
            Force-push tag
          </button>
          <button v-else class="ctx-confirm-btn ctx-confirm-btn--yes" @click="pushCreatedTag">
            Push tag
          </button>
          <button class="ctx-confirm-btn ctx-confirm-btn--cancel" @click="emit('close')">
            Done
          </button>
        </div>
      </template>

      <!-- ── Main menu ────────────────────────────── -->
      <template v-else>
        <div class="ctx-header">{{ row.shortHash }} — {{ row.subject }}</div>

        <div class="ctx-divider" />

        <button role="menuitem" class="ctx-item" @click="copyHash(false)">Copy short SHA</button>
        <button role="menuitem" class="ctx-item" @click="copyHash(true)">Copy full SHA</button>

        <div class="ctx-divider" />

        <button role="menuitem" class="ctx-item" @click="checkout">Checkout commit</button>
        <button role="menuitem" class="ctx-item" @click="openBranchInput">
          Create branch here…
        </button>

        <div class="ctx-divider" />

        <button role="menuitem" class="ctx-item" @click="openRevertConfirm">Revert commit</button>

        <div class="ctx-divider" />

        <div class="ctx-group-label">Reset branch to here</div>
        <button role="menuitem" class="ctx-item ctx-item--indent" @click="reset('soft')">
          Soft — keep staged
        </button>
        <button role="menuitem" class="ctx-item ctx-item--indent" @click="reset('mixed')">
          Mixed — keep working tree
        </button>
        <button
          role="menuitem"
          class="ctx-item ctx-item--indent ctx-item--danger"
          @click="reset('hard')"
        >
          Hard — discard all changes
        </button>

        <div class="ctx-divider" />

        <button role="menuitem" class="ctx-item" @click="openTagInput">Create tag here…</button>
        <button role="menuitem" class="ctx-item" @click="openMoveTagInput">Move tag here…</button>

        <template v-if="row.labels.some((l) => l.type === 'tag')">
          <div class="ctx-divider" />
          <button
            v-for="l in row.labels.filter((l) => l.type === 'tag')"
            :key="l.name"
            class="ctx-item"
            @click="pushTag(l.name)"
          >
            Push tag '{{ l.name }}'
          </button>
        </template>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.ctx-menu {
  position: fixed;
  z-index: 9999;
  min-width: 230px;
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 7px;
  padding: 4px 0;
  box-shadow: var(--shadow-popover);
  user-select: none;
}

.ctx-header {
  padding: 6px 12px 5px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  font-family: var(--font-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 260px;
}

.ctx-divider {
  height: 1px;
  background: var(--border-subtle);
  margin: 3px 0;
}

.ctx-item {
  display: block;
  width: 100%;
  padding: 6px 14px;
  background: none;
  border: none;
  text-align: left;
  color: var(--text-primary);
  font-size: var(--text-row);
  cursor: default;
  white-space: nowrap;
}
.ctx-item:focus-visible,
.ctx-item:hover {
  outline: none;
  background: var(--accent-fill);
  color: var(--text-on-accent);
}

.ctx-item--indent {
  padding-left: 22px;
  color: var(--text-secondary);
  font-size: var(--text-meta);
}
.ctx-item--indent:focus-visible,
.ctx-item--indent:hover {
  color: var(--text-on-accent);
}

.ctx-item--danger {
  color: var(--danger);
}
.ctx-item--danger:focus-visible,
.ctx-item--danger:hover {
  background: var(--accent-fill);
  color: var(--text-on-accent);
}

.ctx-group-label {
  padding: 5px 14px 2px;
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
}

.ctx-choices {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 2px 6px 6px;
}

.ctx-choice {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 7px 10px;
  background: none;
  border: 1px solid transparent;
  border-radius: 5px;
  text-align: left;
  cursor: default;
}
.ctx-choice:hover {
  background: var(--surface-hover);
  border-color: var(--border);
}

.ctx-choice-title {
  font-size: var(--text-row);
  font-weight: 600;
  color: var(--text-primary);
}

.ctx-choice-desc {
  font-size: var(--text-meta);
  color: var(--text-secondary);
  line-height: 1.35;
}

.ctx-confirm-question {
  max-width: 260px;
  padding: 4px 12px 8px;
  font-size: var(--text-row);
  color: var(--text-primary);
  line-height: 1.4;
}

.ctx-confirm-row {
  display: flex;
  gap: 6px;
  padding: 0 10px 10px;
}

.ctx-confirm-btn {
  flex: 1;
  padding: 5px 0;
  border-radius: 4px;
  font-size: var(--text-meta);
  font-weight: 600;
  cursor: default;
  border: 1px solid transparent;
}

.ctx-confirm-btn--yes {
  background: var(--accent-fill);
  color: var(--text-on-accent);
  border-color: var(--accent-fill);
}
.ctx-confirm-btn--yes:hover {
  background: var(--accent-fill-hover);
}
.ctx-confirm-btn--danger {
  background: var(--danger-fill);
  color: var(--text-on-accent);
  border-color: var(--danger-fill);
}
.ctx-confirm-btn--danger:hover {
  background: var(--danger-fill-hover);
}
.ctx-confirm-btn--cancel {
  background: transparent;
  color: var(--text-tertiary);
  border-color: var(--border);
}
.ctx-confirm-btn--cancel:hover {
  color: var(--text-tertiary);
  border-color: var(--border-strong);
}

/* Input mode */
.ctx-input-header {
  padding: 8px 12px 4px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  font-family: var(--font-mono);
}

.ctx-input-row {
  display: flex;
  gap: 4px;
  padding: 4px 10px 8px;
}

.ctx-input {
  flex: 1;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: var(--text-row);
  padding: 4px 8px;
  outline: none;
  font-family: var(--font-mono);
}
.ctx-input:focus {
  border-color: var(--focus-ring);
}
.ctx-input::placeholder {
  color: var(--text-tertiary);
}

.ctx-confirm {
  padding: 4px 10px;
  background: var(--accent-fill);
  border: none;
  border-radius: 4px;
  color: var(--text-on-accent);
  font-size: var(--text-meta);
  cursor: default;
  white-space: nowrap;
}
.ctx-confirm:hover {
  background: var(--accent-fill-hover);
}
</style>
