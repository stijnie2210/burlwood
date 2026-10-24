<script setup lang="ts">
import { watch, ref, computed, nextTick, onMounted, onUnmounted } from 'vue';
import {
  clampMenuPosition,
  focusFirstMenuItem,
  handleMenuKeydown,
  menuEventFor,
} from '../../composables/useContextMenu';
import { useReposStore } from '../../stores/repos';
import { useBranchesStore } from '../../stores/branches';
import { useCommitsStore } from '../../stores/commits';
import { useStagingStore } from '../../stores/staging';
import { useToastStore } from '../../stores/toast';
import { useStashStore } from '../../stores/stash';
import type { StashEntry } from '../../stores/stash';
import {
  StartRebase,
  DeleteBranch,
  RenameBranch,
  PushNamedBranch,
  ApplyStash,
  DropStash,
  PopStash,
  RenameStash,
} from '../../../wailsjs/go/main/App';

defineOptions({ inheritAttrs: false });

const repos = useReposStore();
const branches = useBranchesStore();
const commits = useCommitsStore();
const staging = useStagingStore();
const toast = useToastStore();
const stash = useStashStore();

interface CollapsedState {
  local: boolean;
  remote: boolean;
  stashes: boolean;
}

const branchFilter = ref('');
const filteredLocal = computed(() => {
  const q = branchFilter.value.trim().toLowerCase();
  return q ? branches.local.filter((b) => b.name.toLowerCase().includes(q)) : branches.local;
});
const filteredRemote = computed(() => {
  const q = branchFilter.value.trim().toLowerCase();
  return q
    ? branches.remote.filter((b) => `${b.remote}/${b.name}`.toLowerCase().includes(q))
    : branches.remote;
});

const collapsed = ref<CollapsedState>({ local: false, remote: false, stashes: false });
const remoteHeight = ref(200);

function prefsKey(repoPath: string) {
  return `gitgood:sidebar:prefs:${repoPath}`;
}

function loadPrefs(repoPath: string) {
  try {
    const raw = localStorage.getItem(prefsKey(repoPath));
    if (raw) {
      const p = JSON.parse(raw);
      collapsed.value = { local: !!p.local, remote: !!p.remote, stashes: !!p.stashes };
      remoteHeight.value = typeof p.remoteHeight === 'number' ? p.remoteHeight : 200;
      return;
    }
  } catch {
    // ignore malformed saved prefs
  }
  collapsed.value = { local: false, remote: false, stashes: false };
  remoteHeight.value = 200;
}

function savePrefs(repoPath: string) {
  localStorage.setItem(
    prefsKey(repoPath),
    JSON.stringify({ ...collapsed.value, remoteHeight: remoteHeight.value })
  );
}

function toggleSection(section: keyof CollapsedState) {
  collapsed.value[section] = !collapsed.value[section];
  const repoPath = repos.activeRepo?.path;
  if (repoPath) {
    savePrefs(repoPath);
  }
}

function onSectionDividerDown(e: MouseEvent) {
  e.preventDefault();
  const startY = e.clientY;
  const startRemote = remoteHeight.value;

  function onMove(ev: MouseEvent) {
    remoteHeight.value = Math.max(60, startRemote + (ev.clientY - startY));
  }

  function onUp() {
    window.removeEventListener('mousemove', onMove);
    window.removeEventListener('mouseup', onUp);
    const repoPath = repos.activeRepo?.path;
    if (repoPath) {
      savePrefs(repoPath);
    }
  }

  window.addEventListener('mousemove', onMove);
  window.addEventListener('mouseup', onUp);
}

watch(
  () => repos.activeRepo?.path,
  (path) => {
    if (path) {
      branches.load(path);
      stash.load(path);
      loadPrefs(path);
    } else {
      branches.clear();
      stash.clear();
      collapsed.value = { local: false, remote: false, stashes: false };
      remoteHeight.value = 200;
    }
  },
  { immediate: true }
);

async function switchTo(target: string, trackRemote: boolean) {
  const repoPath = repos.activeRepo?.path;
  if (!repoPath) {
    return;
  }

  if (!trackRemote) {
    const current = branches.local.find((b) => b.isCurrent);
    if (current?.name === target) {
      return;
    }
  }

  try {
    const result = await branches.switchBranch(repoPath, target, trackRemote);
    await Promise.all([branches.load(repoPath), commits.load(repoPath), staging.load(repoPath)]);
    if (result.conflictFiles && result.conflictFiles.length > 0) {
      toast.error(result.message + ': ' + result.conflictFiles.join(', '));
    } else {
      toast.success(result.message);
    }
  } catch (e: unknown) {
    toast.error(String(e));
  }
}

// ── Context menu ──────────────────────────────────────────────────────────

type CtxMode = 'default' | 'rename' | 'confirm-delete' | 'confirm-force-delete';

interface CtxMenu {
  x: number;
  y: number;
  target: string;
  label: string;
  isLocal: boolean;
  isCurrent: boolean;
  mode: CtxMode;
  renameValue: string;
}

const ctxMenu = ref<CtxMenu | null>(null);
const renameInputRef = ref<HTMLInputElement | null>(null);
const branchCtxMenuEl = ref<HTMLElement | null>(null);

async function openCtxMenu(
  e: MouseEvent,
  target: string,
  label: string,
  isLocal: boolean,
  isCurrent: boolean
) {
  e.preventDefault();
  e.stopPropagation();
  rememberMenuFocus();
  ctxMenu.value = {
    x: e.clientX,
    y: e.clientY,
    target,
    label,
    isLocal,
    isCurrent,
    mode: 'default',
    renameValue: label,
  };
  await nextTick();
  const menu = branchCtxMenuEl.value;
  if (menu && ctxMenu.value) {
    const { x, y } = clampMenuPosition(menu, e.clientX, e.clientY);
    ctxMenu.value = { ...ctxMenu.value, x, y };
  }
  focusFirstMenuItem(menu);
}

function closeCtxMenu() {
  ctxMenu.value = null;
  restoreMenuFocus();
}

let menuReturnFocus: HTMLElement | null = null;

function rememberMenuFocus() {
  menuReturnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
}

function restoreMenuFocus() {
  const el = menuReturnFocus;
  menuReturnFocus = null;
  if (el?.isConnected) {
    el.focus();
  }
}

function focusSiblingItem(e: KeyboardEvent, dir: 1 | -1) {
  const li = e.currentTarget as HTMLElement;
  const next = (
    dir === 1 ? li.nextElementSibling : li.previousElementSibling
  ) as HTMLElement | null;
  next?.focus();
}

function onWindowMouseDown(e: MouseEvent) {
  if (branchCtxMenuEl.value && !branchCtxMenuEl.value.contains(e.target as Node)) {
    closeCtxMenu();
  }
  if (stashCtxMenuEl.value && !stashCtxMenuEl.value.contains(e.target as Node)) {
    closeStashCtxMenu();
  }
}

onMounted(() => window.addEventListener('mousedown', onWindowMouseDown));
onUnmounted(() => window.removeEventListener('mousedown', onWindowMouseDown));

async function rebaseOnto(target: string) {
  closeCtxMenu();
  const repoPath = repos.activeRepo?.path;
  if (!repoPath) {
    return;
  }
  try {
    await StartRebase(repoPath, target);
    await Promise.all([branches.load(repoPath), commits.load(repoPath), staging.load(repoPath)]);
    toast.success('Rebased onto ' + target);
  } catch {
    await staging.load(repoPath);
    if (staging.isInRebase) {
      await Promise.all([branches.load(repoPath), commits.load(repoPath)]);
      toast.success('Rebase started — switch to Working Tree to resolve conflicts');
    } else {
      await staging.load(repoPath);
      toast.error('Rebase failed — check Working Tree view for details');
    }
  }
}

async function pushBranch() {
  const menu = ctxMenu.value;
  const repoPath = repos.activeRepo?.path;
  closeCtxMenu();
  if (!menu || !repoPath) {
    return;
  }
  try {
    await PushNamedBranch(repoPath, menu.target);
    toast.success('Pushed ' + menu.label);
  } catch (e: unknown) {
    toast.error(String(e));
  }
}

async function copyBranchName() {
  const menu = ctxMenu.value;
  closeCtxMenu();
  if (!menu) {
    return;
  }
  await navigator.clipboard.writeText(menu.label);
  toast.success('Copied: ' + menu.label);
}

function startRename() {
  if (!ctxMenu.value) {
    return;
  }
  ctxMenu.value.mode = 'rename';
  ctxMenu.value.renameValue = ctxMenu.value.target;
  nextTick(() => {
    renameInputRef.value?.select();
  });
}

async function confirmRename() {
  const menu = ctxMenu.value;
  const repoPath = repos.activeRepo?.path;
  if (!menu || !repoPath) {
    closeCtxMenu();
    return;
  }
  const newName = menu.renameValue.trim();
  if (!newName || newName === menu.target) {
    closeCtxMenu();
    return;
  }
  closeCtxMenu();
  try {
    await RenameBranch(repoPath, menu.target, newName);
    await branches.load(repoPath);
    toast.success('Renamed to ' + newName);
  } catch (e: unknown) {
    toast.error(String(e));
  }
}

function startDelete() {
  if (!ctxMenu.value) {
    return;
  }
  ctxMenu.value.mode = 'confirm-delete';
}

async function confirmDelete(force: boolean) {
  const menu = ctxMenu.value;
  const repoPath = repos.activeRepo?.path;
  if (!menu || !repoPath) {
    closeCtxMenu();
    return;
  }
  try {
    await DeleteBranch(repoPath, menu.target, force);
    closeCtxMenu();
    await branches.load(repoPath);
    toast.success((force ? 'Force deleted ' : 'Deleted ') + menu.label);
  } catch (e: unknown) {
    const msg = String(e);
    if (
      !force &&
      msg.toLowerCase().includes('not fully merged') &&
      ctxMenu.value?.target === menu.target
    ) {
      ctxMenu.value.mode = 'confirm-force-delete';
      await nextTick();
      const el = branchCtxMenuEl.value;
      if (el && ctxMenu.value) {
        const { x, y } = clampMenuPosition(el, ctxMenu.value.x, ctxMenu.value.y);
        ctxMenu.value = { ...ctxMenu.value, x, y };
      }
      return;
    }
    closeCtxMenu();
    toast.error(msg);
  }
}

// ── Stash ─────────────────────────────────────────────────────────────────

async function selectStash(entry: StashEntry) {
  const repoPath = repos.activeRepo?.path;
  if (!repoPath) {
    return;
  }
  await commits.selectCommit(repoPath, entry.hash);
}

function stashDisplayMsg(entry: StashEntry): string {
  // "WIP on main: abc1234 actual message" → "main: actual message"
  // "On main: abc1234 actual message" → "main: actual message"
  // custom message → returned as-is
  const wip = entry.message.match(/^(?:WIP )?[Oo]n ([^:]+): [0-9a-f]{7,}\s+(.+)$/);
  if (wip) {
    return wip[1] + ': ' + wip[2];
  }
  return entry.message;
}

type StashCtxMode = 'default' | 'rename' | 'confirm-drop';

interface StashCtxMenu {
  x: number;
  y: number;
  ref: string;
  hash: string;
  message: string;
  mode: StashCtxMode;
  renameValue: string;
}

const stashCtxMenu = ref<StashCtxMenu | null>(null);
const stashRenameInputRef = ref<HTMLInputElement | null>(null);
const stashCtxMenuEl = ref<HTMLElement | null>(null);

async function openStashCtxMenu(e: MouseEvent, entry: StashEntry) {
  e.preventDefault();
  e.stopPropagation();
  rememberMenuFocus();
  stashCtxMenu.value = {
    x: e.clientX,
    y: e.clientY,
    ref: entry.ref,
    hash: entry.hash,
    message: entry.message,
    mode: 'default',
    renameValue: entry.message,
  };
  await nextTick();
  const menu = stashCtxMenuEl.value;
  if (menu && stashCtxMenu.value) {
    const { x, y } = clampMenuPosition(menu, e.clientX, e.clientY);
    stashCtxMenu.value = { ...stashCtxMenu.value, x, y };
  }
  focusFirstMenuItem(menu);
}

function closeStashCtxMenu() {
  stashCtxMenu.value = null;
  restoreMenuFocus();
}

async function applyStash() {
  const menu = stashCtxMenu.value;
  const repoPath = repos.activeRepo?.path;
  closeStashCtxMenu();
  if (!menu || !repoPath) {
    return;
  }
  try {
    await ApplyStash(repoPath, menu.ref);
    await Promise.all([stash.load(repoPath), staging.load(repoPath)]);
    toast.success('Applied ' + menu.ref);
  } catch (e: unknown) {
    toast.error(String(e));
  }
}

async function popStash() {
  const menu = stashCtxMenu.value;
  const repoPath = repos.activeRepo?.path;
  closeStashCtxMenu();
  if (!menu || !repoPath) {
    return;
  }
  try {
    await PopStash(repoPath, menu.ref);
    await Promise.all([stash.load(repoPath), staging.load(repoPath), commits.load(repoPath)]);
    toast.success('Popped ' + menu.ref);
  } catch (e: unknown) {
    toast.error(String(e));
  }
}

function startStashDrop() {
  if (!stashCtxMenu.value) {
    return;
  }
  stashCtxMenu.value.mode = 'confirm-drop';
}

async function confirmStashDrop() {
  const menu = stashCtxMenu.value;
  const repoPath = repos.activeRepo?.path;
  closeStashCtxMenu();
  if (!menu || !repoPath) {
    return;
  }
  try {
    await DropStash(repoPath, menu.ref);
    await Promise.all([stash.load(repoPath), commits.load(repoPath)]);
    toast.success('Dropped ' + menu.ref);
  } catch (e: unknown) {
    toast.error(String(e));
  }
}

function startStashRename() {
  if (!stashCtxMenu.value) {
    return;
  }
  stashCtxMenu.value.mode = 'rename';
  nextTick(() => {
    stashRenameInputRef.value?.select();
  });
}

async function confirmStashRename() {
  const menu = stashCtxMenu.value;
  const repoPath = repos.activeRepo?.path;
  if (!menu || !repoPath) {
    closeStashCtxMenu();
    return;
  }
  const newMessage = menu.renameValue.trim();
  if (!newMessage || newMessage === menu.message) {
    closeStashCtxMenu();
    return;
  }
  closeStashCtxMenu();
  try {
    await RenameStash(repoPath, menu.ref, newMessage);
    await Promise.all([stash.load(repoPath), commits.load(repoPath)]);
    toast.success('Renamed stash');
  } catch (e: unknown) {
    toast.error(String(e));
  }
}
</script>

<template>
  <aside class="sidebar" v-bind="$attrs">
    <div v-if="!repos.activeRepo" class="empty-state">
      <p>No repository open</p>
    </div>

    <div v-else class="sidebar-content">
      <div class="branch-filter-row">
        <input
          v-model="branchFilter"
          class="branch-filter"
          type="search"
          placeholder="Filter branches"
          aria-label="Filter branches"
          @keydown.esc="branchFilter = ''"
        />
      </div>
      <section class="branch-section local-section">
        <h3 class="section-title">
          <button
            type="button"
            class="section-title--toggle"
            :aria-expanded="!collapsed.local"
            @click="toggleSection('local')"
          >
            <span class="section-chevron" aria-hidden="true">{{
              collapsed.local ? '▸' : '▾'
            }}</span>
            Local Branches
          </button>
        </h3>
        <ul v-if="!collapsed.local" class="branch-list local-list">
          <li
            v-for="b in filteredLocal"
            :key="b.name"
            class="branch-item"
            :class="{ current: b.isCurrent }"
            :title="b.isCurrent ? b.name + ' (current)' : 'Double-click to switch to ' + b.name"
            tabindex="0"
            @dblclick="switchTo(b.name, false)"
            @keydown.enter="switchTo(b.name, false)"
            @keydown.up.prevent.stop="focusSiblingItem($event, -1)"
            @keydown.down.prevent.stop="focusSiblingItem($event, 1)"
            @keydown.shift.f10.prevent="
              openCtxMenu(menuEventFor($event), b.name, b.name, true, b.isCurrent)
            "
            @contextmenu="openCtxMenu($event, b.name, b.name, true, b.isCurrent)"
          >
            <span class="branch-icon">{{ b.isCurrent ? '●' : '○' }}</span>
            <span class="branch-name">{{ b.name }}</span>
            <template v-if="b.isCurrent">
              <span
                v-if="branches.aheadBehind.ahead > 0"
                class="sync-badge ahead"
                title="Commits to push"
                >↑{{ branches.aheadBehind.ahead }}</span
              >
              <span
                v-if="branches.aheadBehind.behind > 0"
                class="sync-badge behind"
                title="Commits to pull"
                >↓{{ branches.aheadBehind.behind }}</span
              >
            </template>
          </li>
        </ul>
      </section>

      <section class="branch-section">
        <h3 class="section-title">
          <button
            type="button"
            class="section-title--toggle"
            :aria-expanded="!collapsed.remote"
            @click="toggleSection('remote')"
          >
            <span class="section-chevron" aria-hidden="true">{{
              collapsed.remote ? '▸' : '▾'
            }}</span>
            Remote Branches
          </button>
        </h3>
        <ul
          v-if="!collapsed.remote"
          class="branch-list remote-list"
          :style="{ maxHeight: remoteHeight + 'px' }"
        >
          <li
            v-for="b in filteredRemote"
            :key="`${b.remote}/${b.name}`"
            class="branch-item remote"
            :title="'Double-click to check out ' + b.remote + '/' + b.name"
            tabindex="0"
            @dblclick="switchTo(b.remote + '/' + b.name, true)"
            @keydown.enter="switchTo(b.remote + '/' + b.name, true)"
            @keydown.up.prevent.stop="focusSiblingItem($event, -1)"
            @keydown.down.prevent.stop="focusSiblingItem($event, 1)"
            @keydown.shift.f10.prevent="
              openCtxMenu(
                menuEventFor($event),
                b.remote + '/' + b.name,
                b.remote + '/' + b.name,
                false,
                false
              )
            "
            @contextmenu="
              openCtxMenu($event, b.remote + '/' + b.name, b.remote + '/' + b.name, false, false)
            "
          >
            <span class="branch-icon" aria-hidden="true">○</span>
            <span class="branch-name"
              ><span class="remote-label">{{ b.remote }}/</span>{{ b.name }}</span
            >
          </li>
        </ul>
      </section>

      <div class="section-divider" @mousedown="onSectionDividerDown" />

      <section class="branch-section stash-section">
        <h3 class="section-title">
          <button
            type="button"
            class="section-title--toggle"
            :aria-expanded="!collapsed.stashes"
            @click="toggleSection('stashes')"
          >
            <span class="section-chevron" aria-hidden="true">{{
              collapsed.stashes ? '▸' : '▾'
            }}</span>
            Stashes
          </button>
        </h3>
        <ul v-if="!collapsed.stashes && stash.entries.length > 0" class="branch-list stash-list">
          <li
            v-for="entry in stash.entries"
            :key="entry.ref"
            class="branch-item stash-item"
            :class="{ 'stash-selected': commits.selectedHash === entry.hash }"
            :title="entry.message"
            tabindex="0"
            @click="selectStash(entry)"
            @keydown.enter="selectStash(entry)"
            @keydown.up.prevent.stop="focusSiblingItem($event, -1)"
            @keydown.down.prevent.stop="focusSiblingItem($event, 1)"
            @keydown.shift.f10.prevent="openStashCtxMenu(menuEventFor($event), entry)"
            @contextmenu="openStashCtxMenu($event, entry)"
          >
            <span class="stash-index">{{ entry.index }}</span>
            <span class="branch-name stash-msg">{{ stashDisplayMsg(entry) }}</span>
            <span class="stash-date">{{ entry.date }}</span>
          </li>
        </ul>
        <p v-else-if="!collapsed.stashes" class="empty-stash">No stashes</p>
      </section>
    </div>
  </aside>

  <!-- Context menu (teleported so it renders above sidebar overflow) -->
  <Teleport to="body">
    <div
      v-if="ctxMenu"
      ref="branchCtxMenuEl"
      class="ctx-menu"
      :style="{ left: ctxMenu.x + 'px', top: ctxMenu.y + 'px' }"
      @keydown="handleMenuKeydown($event, branchCtxMenuEl, closeCtxMenu)"
    >
      <div class="ctx-menu-label">{{ ctxMenu.label }}</div>

      <!-- Default actions -->
      <template v-if="ctxMenu.mode === 'default'">
        <button
          v-if="!ctxMenu.isCurrent"
          class="ctx-menu-item ctx-menu-item-primary"
          @click="
            switchTo(ctxMenu!.target, !ctxMenu!.isLocal);
            closeCtxMenu();
          "
        >
          Checkout
        </button>
        <button v-if="ctxMenu.isLocal" class="ctx-menu-item" @click="pushBranch()">Push</button>
        <button class="ctx-menu-item" @click="copyBranchName()">Copy branch name</button>
        <button class="ctx-menu-item" @click="rebaseOnto(ctxMenu!.target)">
          Rebase current branch onto this
        </button>
        <template v-if="ctxMenu.isLocal">
          <div class="ctx-menu-divider" />
          <button class="ctx-menu-item" @click="startRename()">Rename</button>
          <button
            class="ctx-menu-item ctx-menu-item-danger"
            :disabled="ctxMenu.isCurrent"
            :title="ctxMenu.isCurrent ? 'Cannot delete the current branch' : ''"
            @click="startDelete()"
          >
            Delete
          </button>
        </template>
      </template>

      <!-- Rename mode -->
      <template v-else-if="ctxMenu.mode === 'rename'">
        <div class="ctx-menu-rename">
          <input
            ref="renameInputRef"
            v-model="ctxMenu.renameValue"
            class="ctx-menu-input"
            placeholder="New branch name"
            @keydown.enter="confirmRename()"
            @keydown.esc="closeCtxMenu()"
          />
          <div class="ctx-menu-row">
            <button class="ctx-menu-btn-primary" @click="confirmRename()">Rename</button>
            <button class="ctx-menu-btn" @click="closeCtxMenu()">Cancel</button>
          </div>
        </div>
      </template>

      <!-- Delete confirmation mode -->
      <template v-else-if="ctxMenu.mode === 'confirm-delete'">
        <div class="ctx-menu-confirm">
          <p class="ctx-menu-confirm-text">
            Delete "{{ ctxMenu.label }}"? Git only deletes it if its commits are merged.
          </p>
          <div class="ctx-menu-row">
            <button class="ctx-menu-btn ctx-menu-btn-danger-soft" @click="confirmDelete(false)">
              Delete
            </button>
            <button class="ctx-menu-btn" @click="closeCtxMenu()">Cancel</button>
          </div>
        </div>
      </template>

      <template v-else-if="ctxMenu.mode === 'confirm-force-delete'">
        <div class="ctx-menu-confirm">
          <p class="ctx-menu-confirm-text ctx-menu-confirm-text--warn">
            "{{ ctxMenu.label }}" has commits that aren't merged into any other branch. Force
            deleting it leaves them reachable only through the reflog.
          </p>
          <div class="ctx-menu-row">
            <button class="ctx-menu-btn ctx-menu-btn-danger" @click="confirmDelete(true)">
              Force delete
            </button>
            <button class="ctx-menu-btn" @click="closeCtxMenu()">Keep branch</button>
          </div>
        </div>
      </template>
    </div>
  </Teleport>

  <!-- Stash context menu -->
  <Teleport to="body">
    <div
      v-if="stashCtxMenu"
      ref="stashCtxMenuEl"
      class="ctx-menu"
      :style="{ left: stashCtxMenu.x + 'px', top: stashCtxMenu.y + 'px' }"
      @keydown="handleMenuKeydown($event, stashCtxMenuEl, closeStashCtxMenu)"
    >
      <div class="ctx-menu-label">{{ stashCtxMenu.ref }}</div>

      <template v-if="stashCtxMenu.mode === 'default'">
        <button class="ctx-menu-item ctx-menu-item-primary" @click="applyStash()">Apply</button>
        <button class="ctx-menu-item ctx-menu-item-primary" @click="popStash()">Pop</button>
        <div class="ctx-menu-divider" />
        <button class="ctx-menu-item" @click="startStashRename()">Rename</button>
        <button class="ctx-menu-item ctx-menu-item-danger" @click="startStashDrop()">Drop</button>
      </template>

      <template v-else-if="stashCtxMenu.mode === 'rename'">
        <div class="ctx-menu-rename">
          <input
            ref="stashRenameInputRef"
            v-model="stashCtxMenu.renameValue"
            class="ctx-menu-input"
            placeholder="Stash message"
            @keydown.enter="confirmStashRename()"
            @keydown.esc="closeStashCtxMenu()"
          />
          <div class="ctx-menu-row">
            <button class="ctx-menu-btn-primary" @click="confirmStashRename()">Rename</button>
            <button class="ctx-menu-btn" @click="closeStashCtxMenu()">Cancel</button>
          </div>
        </div>
      </template>

      <template v-else-if="stashCtxMenu.mode === 'confirm-drop'">
        <div class="ctx-menu-confirm">
          <p class="ctx-menu-confirm-text">Drop "{{ stashCtxMenu.ref }}"?</p>
          <button class="ctx-menu-item ctx-menu-item-danger" @click="confirmStashDrop()">
            Drop
          </button>
          <button class="ctx-menu-item" @click="closeStashCtxMenu()">Cancel</button>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.sidebar {
  flex-shrink: 0;
  background: var(--surface-chrome);
  border-right: 1px solid var(--border);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sidebar-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.empty-state {
  padding: 24px 16px;
  text-align: center;
  color: var(--text-secondary);
  font-size: var(--text-body);
}

.branch-section {
  padding: 8px 0;
}

.section-title {
  padding: 4px 12px;
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
  margin: 0;
}

.section-title--toggle {
  cursor: pointer;
  user-select: none;
  -webkit-user-select: none;
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  padding: 0;
  background: none;
  border: none;
  color: inherit;
  font: inherit;
  text-align: left;
}

.section-title--toggle:hover {
  color: var(--text-secondary);
}

.section-chevron {
  font-size: var(--text-meta);
  flex-shrink: 0;
}

.branch-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.branch-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  font-size: var(--text-body);
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  -webkit-user-select: none;
}

.branch-item:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.branch-item.current {
  color: var(--accent-text);
  font-weight: 500;
  cursor: default;
}
.branch-item.remote {
  color: var(--text-secondary);
}

.branch-icon {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  flex-shrink: 0;
}
.branch-item.current .branch-icon {
  color: var(--accent-text);
}

.branch-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sync-badge {
  flex-shrink: 0;
  font-size: var(--text-meta);
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 8px;
  font-family: var(--font-mono);
}

.sync-badge.ahead {
  background: var(--add-soft);
  color: var(--add);
  border: 1px solid var(--add-soft-strong);
}

.sync-badge.behind {
  background: var(--warning-soft);
  color: var(--modified);
  border: 1px solid var(--warning-border);
}

.remote-label {
  color: var(--text-tertiary);
  font-size: var(--text-meta);
}

.stash-item {
  color: var(--text-secondary);
}
.stash-item:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.stash-item.stash-selected {
  background: var(--stash-soft);
  color: var(--text-primary);
}

.stash-index {
  flex-shrink: 0;
  font-size: var(--text-meta);
  font-family: var(--font-mono);
  color: var(--stash);
  min-width: 16px;
  text-align: right;
}

.stash-msg {
  flex: 1;
  font-size: var(--text-row);
}

.stash-date {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.remote-list {
  overflow-y: auto;
}

.local-section {
  display: flex;
  flex-direction: column;
  flex: 0 1 auto;
  min-height: 0;
  max-height: 45%;
}

.local-list {
  min-height: 0;
  overflow-y: auto;
}

.branch-filter-row {
  padding: var(--space-3) var(--inset-panel) var(--space-1);
  flex-shrink: 0;
}

.branch-filter {
  width: 100%;
  padding: 4px 8px;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: var(--text-row);
  outline: none;
}
.branch-filter::placeholder {
  color: var(--text-tertiary);
}
.branch-filter:focus {
  border-color: var(--focus-ring);
}

.stash-section {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.stash-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.section-divider {
  height: 5px;
  flex-shrink: 0;
  cursor: row-resize;
  background: transparent;
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
}

.section-divider:hover,
.section-divider:active {
  background: var(--accent-soft);
}

.empty-stash {
  padding: 4px 12px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  margin: 0;
}
</style>

<!-- Context menu styles are global (it's teleported to body) -->
<style>
.ctx-menu {
  position: fixed;
  z-index: 9999;
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: var(--shadow-popover);
  min-width: 220px;
  padding: 4px 0;
}

.ctx-menu-label {
  padding: 5px 12px 4px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  font-family: var(--font-mono);
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
  border-bottom: 1px solid var(--border);
  margin-bottom: 3px;
}

.ctx-menu-divider {
  height: 1px;
  background: var(--surface-control);
  margin: 3px 0;
}

.ctx-menu-item {
  display: block;
  width: 100%;
  padding: 6px 12px;
  text-align: left;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: var(--text-row);
  cursor: pointer;
}
.ctx-menu-item:focus-visible:not(:disabled),
.ctx-menu-item:hover:not(:disabled) {
  background: var(--accent-fill);
  color: var(--text-on-accent);
  outline: none;
}
.ctx-menu-item-primary {
  color: var(--accent-text);
  font-weight: 500;
}
.ctx-menu-item:disabled {
  opacity: 0.35;
  cursor: default;
}

.ctx-menu-item-danger {
  color: var(--danger);
}
.ctx-menu-item-danger:focus-visible:not(:disabled),
.ctx-menu-item-danger:hover:not(:disabled) {
  background: var(--accent-fill);
  color: var(--text-on-accent);
}

/* Rename mode */
.ctx-menu-rename {
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ctx-menu-input {
  width: 100%;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: var(--text-row);
  padding: 4px 8px;
  outline: none;
  font-family: var(--font-mono);
}
.ctx-menu-input:focus {
  border-color: var(--focus-ring);
}

.ctx-menu-row {
  display: flex;
  gap: 6px;
}

.ctx-menu-btn-primary {
  flex: 1;
  padding: 4px 0;
  background: var(--accent-fill);
  border: none;
  border-radius: 4px;
  color: var(--text-on-accent);
  font-size: var(--text-row);
  cursor: pointer;
}
.ctx-menu-btn-primary:hover {
  background: var(--accent-fill-hover);
}

.ctx-menu-btn {
  flex: 1;
  padding: 4px 0;
  background: var(--surface-control);
  border: none;
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: var(--text-row);
  cursor: pointer;
}
.ctx-menu-btn:hover {
  background: var(--surface-control-hover);
  color: var(--text-primary);
}

/* Delete confirm mode */
.ctx-menu-confirm {
  max-width: 260px;
  padding: 4px 10px 8px;
}

.ctx-menu-confirm-text {
  margin: 0;
  padding: 4px 2px 8px;
  font-size: var(--text-meta);
  line-height: 1.4;
  color: var(--text-secondary);
  text-align: left;
}
.ctx-menu-confirm-text--warn {
  color: var(--danger);
}

.ctx-menu-btn-danger-soft {
  color: var(--danger);
}
.ctx-menu-btn-danger-soft:hover {
  background: var(--danger-soft);
  color: var(--danger);
}

.ctx-menu-btn-danger {
  background: var(--danger-fill);
  color: var(--text-on-accent);
  font-weight: 600;
}
.ctx-menu-btn-danger:hover {
  background: var(--danger-fill-hover);
  color: var(--text-on-accent);
}
</style>
