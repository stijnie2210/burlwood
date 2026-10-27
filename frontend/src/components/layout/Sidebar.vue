<script setup lang="ts">
import { shortcut } from '../../utils/platform';
import { watch, ref, computed, nextTick, onMounted, onUnmounted } from 'vue';
import {
  clampMenuPosition,
  focusFirstMenuItem,
  handleMenuKeydown,
  keepFocusInMenu,
  menuEventFor,
} from '../../composables/useContextMenu';
import { useReposStore } from '../../stores/repos';
import { useBranchesStore } from '../../stores/branches';
import { useCommitsStore } from '../../stores/commits';
import { useStagingStore } from '../../stores/staging';
import { useToastStore } from '../../stores/toast';
import { pushWithFeedback } from '../../composables/usePush';
import { useStashStore } from '../../stores/stash';
import type { StashEntry } from '../../stores/stash';
import type { BranchInfo, TagInfo } from '../../stores/branches';
import {
  StartRebase,
  DeleteBranch,
  RenameBranch,
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
  tags: boolean;
  stashes: boolean;
}

const EXPANDED: CollapsedState = { local: false, remote: false, tags: false, stashes: false };

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

interface RemoteGroup {
  remote: string;
  items: { branch: BranchInfo; index: number }[];
}

// Index is the branch's position across all remotes, so one roving Tab stop covers every group.
const remoteGroups = computed(() => {
  const groups: RemoteGroup[] = [];
  filteredRemote.value.forEach((branch, index) => {
    let group = groups.find((g) => g.remote === branch.remote);
    if (!group) {
      group = { remote: branch.remote, items: [] };
      groups.push(group);
    }
    group.items.push({ branch, index });
  });
  return groups;
});

const collapsedRemotes = ref<Record<string, boolean>>({});
function remoteExpanded(remote: string): boolean {
  return branchFilter.value.trim() !== '' || !collapsedRemotes.value[remote];
}

const collapsed = ref<CollapsedState>({ ...EXPANDED });
const remoteHeight = ref(200);

function prefsKey(repoPath: string) {
  return `burlwood:sidebar:prefs:${repoPath}`;
}

function loadPrefs(repoPath: string) {
  try {
    const raw = localStorage.getItem(prefsKey(repoPath));
    if (raw) {
      const p = JSON.parse(raw);
      collapsed.value = {
        local: !!p.local,
        remote: !!p.remote,
        tags: !!p.tags,
        stashes: !!p.stashes,
      };
      remoteHeight.value = typeof p.remoteHeight === 'number' ? p.remoteHeight : 200;
      return;
    }
  } catch {
    // ignore malformed saved prefs
  }
  collapsed.value = { ...EXPANDED };
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
      collapsedRemotes.value = {};
    } else {
      branches.clear();
      stash.clear();
      collapsed.value = { ...EXPANDED };
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
    toast.error(e);
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

// One Tab stop per list; arrow keys move within it.
const roving = ref({ local: 0, remote: 0, tag: 0, stash: 0 });
function rovingTab(list: 'local' | 'remote' | 'tag' | 'stash', i: number, length: number) {
  return i === Math.min(roving.value[list], length - 1) ? 0 : -1;
}

// The remote branches' single Tab stop must be on a branch that is rendered, not inside a
// collapsed group.
const remoteTabStop = computed(() => {
  const visible = remoteGroups.value
    .filter((g) => remoteExpanded(g.remote))
    .flatMap((g) => g.items.map((item) => item.index));
  return visible.includes(roving.value.remote) ? roving.value.remote : (visible[0] ?? -1);
});

watch(
  () => ctxMenu.value?.mode,
  () => nextTick(() => keepFocusInMenu(branchCtxMenuEl.value))
);

function focusSiblingItem(e: KeyboardEvent, dir: 1 | -1) {
  const item = e.currentTarget as HTMLElement;
  const list = item.closest('.sidebar-content');
  const options = Array.from(
    list?.querySelectorAll<HTMLElement>(`[data-list="${item.dataset.list}"]`) ?? []
  );
  options[options.indexOf(item) + dir]?.focus();
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
  } catch (e) {
    await staging.load(repoPath);
    if (staging.isInRebase) {
      await Promise.all([branches.load(repoPath), commits.load(repoPath)]);
      toast.push(
        `Rebase onto ${target} stopped on a conflict. Resolve it in the Working Tree (${shortcut('2')}), then continue.`,
        'info',
        0
      );
    } else {
      toast.error(e);
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
    await pushWithFeedback(repoPath, menu.target);
  } catch (e: unknown) {
    toast.error(e);
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
    toast.error(e);
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

async function selectTag(tag: TagInfo) {
  const repoPath = repos.activeRepo?.path;
  if (!repoPath) {
    return;
  }
  await commits.selectCommit(repoPath, tag.hash);
}

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

watch(
  () => stashCtxMenu.value?.mode,
  () => nextTick(() => keepFocusInMenu(stashCtxMenuEl.value))
);

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
    toast.error(e);
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
    toast.error(e);
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
    toast.error(e);
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
    toast.error(e);
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
            <svg
              class="section-chevron"
              :class="{ collapsed: collapsed.local }"
              width="10"
              height="10"
              viewBox="0 0 10 10"
              aria-hidden="true"
            >
              <path d="M2.5 3.75L5 6.25L7.5 3.75" />
            </svg>
            <span class="section-label">Branches</span>
            <span class="section-count">{{ branches.local.length || '' }}</span>
          </button>
        </h3>
        <ul
          v-if="!collapsed.local"
          class="branch-list local-list"
          role="listbox"
          aria-label="Local branches"
        >
          <li
            v-for="(b, i) in filteredLocal"
            :key="b.name"
            class="branch-item"
            :class="{ current: b.isCurrent }"
            role="option"
            data-list="local"
            :aria-current="b.isCurrent ? 'true' : undefined"
            :title="
              b.isCurrent ? b.name + ' (current)' : 'Double-click or Enter to switch to ' + b.name
            "
            :tabindex="rovingTab('local', i, filteredLocal.length)"
            @focus="roving.local = i"
            @dblclick="switchTo(b.name, false)"
            @keydown.enter="switchTo(b.name, false)"
            @keydown.up.prevent.stop="focusSiblingItem($event, -1)"
            @keydown.down.prevent.stop="focusSiblingItem($event, 1)"
            @keydown.shift.f10.prevent="
              openCtxMenu(menuEventFor($event), b.name, b.name, true, b.isCurrent)
            "
            @contextmenu="openCtxMenu($event, b.name, b.name, true, b.isCurrent)"
          >
            <svg
              v-if="b.isCurrent"
              class="branch-icon"
              width="13"
              height="13"
              viewBox="0 0 24 24"
              stroke-width="2.5"
              aria-hidden="true"
            >
              <path d="M20 6L9 17l-5-5" />
            </svg>
            <svg
              v-else
              class="branch-icon"
              width="13"
              height="13"
              viewBox="0 0 24 24"
              stroke-width="2"
              aria-hidden="true"
            >
              <path d="M6 3v12" />
              <circle cx="18" cy="6" r="3" />
              <circle cx="6" cy="18" r="3" />
              <path d="M18 9a9 9 0 0 1-9 9" />
            </svg>
            <span class="branch-name">{{ b.name }}</span>
            <template v-if="b.isCurrent">
              <span
                v-if="branches.aheadBehind.ahead > 0"
                class="sync-badge ahead"
                title="Commits to push"
                :aria-label="`${branches.aheadBehind.ahead} to push`"
                >↑{{ branches.aheadBehind.ahead }}</span
              >
              <span
                v-if="branches.aheadBehind.behind > 0"
                class="sync-badge behind"
                title="Commits to pull"
                :aria-label="`${branches.aheadBehind.behind} to pull`"
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
            <svg
              class="section-chevron"
              :class="{ collapsed: collapsed.remote }"
              width="10"
              height="10"
              viewBox="0 0 10 10"
              aria-hidden="true"
            >
              <path d="M2.5 3.75L5 6.25L7.5 3.75" />
            </svg>
            <span class="section-label">Remotes</span>
            <span class="section-count">{{ remoteGroups.length || '' }}</span>
          </button>
        </h3>
        <div
          v-if="!collapsed.remote"
          class="remote-tree"
          :style="{ maxHeight: remoteHeight + 'px' }"
        >
          <p v-if="!remoteGroups.length" class="empty-stash">
            {{ branchFilter.trim() ? 'No matching remote branches' : 'No remotes' }}
          </p>
          <div v-for="g in remoteGroups" :key="g.remote" class="remote-group">
            <button
              type="button"
              class="remote-group-toggle"
              :aria-expanded="remoteExpanded(g.remote)"
              @click="collapsedRemotes[g.remote] = !collapsedRemotes[g.remote]"
            >
              <svg
                class="section-chevron"
                :class="{ collapsed: !remoteExpanded(g.remote) }"
                width="10"
                height="10"
                viewBox="0 0 10 10"
                aria-hidden="true"
              >
                <path d="M2.5 3.75L5 6.25L7.5 3.75" />
              </svg>
              <span class="remote-group-name">{{ g.remote }}</span>
              <span class="section-count">{{ g.items.length }}</span>
            </button>
            <ul
              v-if="remoteExpanded(g.remote)"
              class="branch-list"
              role="listbox"
              :aria-label="`Branches on ${g.remote}`"
            >
              <li
                v-for="{ branch: b, index: i } in g.items"
                :key="`${b.remote}/${b.name}`"
                class="branch-item remote"
                role="option"
                data-list="remote"
                :title="'Double-click or Enter to check out ' + b.remote + '/' + b.name"
                :tabindex="i === remoteTabStop ? 0 : -1"
                @focus="roving.remote = i"
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
                  openCtxMenu(
                    $event,
                    b.remote + '/' + b.name,
                    b.remote + '/' + b.name,
                    false,
                    false
                  )
                "
              >
                <span class="remote-dot" aria-hidden="true" />
                <span class="branch-name">{{ b.name }}</span>
              </li>
            </ul>
          </div>
        </div>
      </section>

      <div class="section-divider" @mousedown="onSectionDividerDown" />

      <section class="branch-section tag-section">
        <h3 class="section-title">
          <button
            type="button"
            class="section-title--toggle"
            :aria-expanded="!collapsed.tags"
            @click="toggleSection('tags')"
          >
            <svg
              class="section-chevron"
              :class="{ collapsed: collapsed.tags }"
              width="10"
              height="10"
              viewBox="0 0 10 10"
              aria-hidden="true"
            >
              <path d="M2.5 3.75L5 6.25L7.5 3.75" />
            </svg>
            <span class="section-label">Tags</span>
            <span class="section-count">{{ branches.tags.length || '' }}</span>
          </button>
        </h3>
        <ul
          v-if="!collapsed.tags && branches.tags.length > 0"
          class="branch-list tag-list"
          role="listbox"
          aria-label="Tags"
        >
          <li
            v-for="(t, i) in branches.tags"
            :key="t.name"
            class="branch-item tag-item"
            :class="{ 'tag-selected': commits.selectedHash === t.hash }"
            role="option"
            data-list="tag"
            :aria-selected="commits.selectedHash === t.hash"
            :title="`Show the commit tagged ${t.name}`"
            :tabindex="rovingTab('tag', i, branches.tags.length)"
            @focus="roving.tag = i"
            @click="selectTag(t)"
            @keydown.enter="selectTag(t)"
            @keydown.up.prevent.stop="focusSiblingItem($event, -1)"
            @keydown.down.prevent.stop="focusSiblingItem($event, 1)"
          >
            <svg
              class="branch-icon"
              width="13"
              height="13"
              viewBox="0 0 24 24"
              stroke-width="2"
              aria-hidden="true"
            >
              <path d="M12 2H2v10l9.3 9.3a1 1 0 0 0 1.4 0l8.6-8.6a1 1 0 0 0 0-1.4z" />
              <circle cx="7" cy="7" r="1.5" />
            </svg>
            <span class="branch-name">{{ t.name }}</span>
          </li>
        </ul>
        <p v-else-if="!collapsed.tags" class="empty-stash">No tags yet</p>
      </section>

      <section class="branch-section stash-section">
        <h3 class="section-title">
          <button
            type="button"
            class="section-title--toggle"
            :aria-expanded="!collapsed.stashes"
            @click="toggleSection('stashes')"
          >
            <svg
              class="section-chevron"
              :class="{ collapsed: collapsed.stashes }"
              width="10"
              height="10"
              viewBox="0 0 10 10"
              aria-hidden="true"
            >
              <path d="M2.5 3.75L5 6.25L7.5 3.75" />
            </svg>
            <span class="section-label">Stashes</span>
            <span class="section-count">{{ stash.entries.length || '' }}</span>
          </button>
        </h3>
        <ul
          v-if="!collapsed.stashes && stash.entries.length > 0"
          class="branch-list stash-list"
          role="listbox"
          aria-label="Stashes"
        >
          <li
            v-for="(entry, i) in stash.entries"
            :key="entry.ref"
            class="branch-item stash-item"
            :class="{ 'stash-selected': commits.selectedHash === entry.hash }"
            role="option"
            data-list="stash"
            :aria-selected="commits.selectedHash === entry.hash"
            :title="entry.message"
            :tabindex="rovingTab('stash', i, stash.entries.length)"
            @focus="roving.stash = i"
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
      role="menu"
      :style="{ left: ctxMenu.x + 'px', top: ctxMenu.y + 'px' }"
      @keydown="handleMenuKeydown($event, branchCtxMenuEl, closeCtxMenu)"
    >
      <div class="ctx-menu-label">{{ ctxMenu.label }}</div>

      <!-- Default actions -->
      <template v-if="ctxMenu.mode === 'default'">
        <button
          v-if="!ctxMenu.isCurrent"
          role="menuitem"
          class="ctx-menu-item ctx-menu-item-primary"
          @click="
            switchTo(ctxMenu!.target, !ctxMenu!.isLocal);
            closeCtxMenu();
          "
        >
          Checkout
        </button>
        <button v-if="ctxMenu.isLocal" role="menuitem" class="ctx-menu-item" @click="pushBranch()">
          Push
        </button>
        <button role="menuitem" class="ctx-menu-item" @click="copyBranchName()">
          Copy branch name
        </button>
        <button role="menuitem" class="ctx-menu-item" @click="rebaseOnto(ctxMenu!.target)">
          Rebase current branch onto this
        </button>
        <template v-if="ctxMenu.isLocal">
          <div class="ctx-menu-divider" />
          <button role="menuitem" class="ctx-menu-item" @click="startRename()">Rename</button>
          <button
            role="menuitem"
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
      role="menu"
      :style="{ left: stashCtxMenu.x + 'px', top: stashCtxMenu.y + 'px' }"
      @keydown="handleMenuKeydown($event, stashCtxMenuEl, closeStashCtxMenu)"
    >
      <div class="ctx-menu-label">{{ stashCtxMenu.ref }}</div>

      <template v-if="stashCtxMenu.mode === 'default'">
        <button role="menuitem" class="ctx-menu-item ctx-menu-item-primary" @click="applyStash()">
          Apply
        </button>
        <button role="menuitem" class="ctx-menu-item" @click="popStash()">Pop</button>
        <div class="ctx-menu-divider" />
        <button role="menuitem" class="ctx-menu-item" @click="startStashRename()">Rename</button>
        <button
          role="menuitem"
          class="ctx-menu-item ctx-menu-item-danger"
          @click="startStashDrop()"
        >
          Drop
        </button>
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
          <p class="ctx-menu-confirm-text">
            Drop "{{ stashCtxMenu.ref }}"? Its changes are deleted from the stash list and can't be
            restored from Burlwood.
          </p>
          <div class="ctx-menu-row">
            <button class="ctx-menu-btn ctx-menu-btn-danger" @click="confirmStashDrop()">
              Drop
            </button>
            <button class="ctx-menu-btn" @click="closeStashCtxMenu()">Cancel</button>
          </div>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.sidebar {
  flex-shrink: 0;
  background: var(--surface-sidebar);
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
  cursor: default;
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
  flex-shrink: 0;
  transition: transform 0.12s;
}
.section-chevron path {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.4;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.section-chevron.collapsed {
  transform: rotate(-90deg);
}

.section-label {
  flex: 1;
}

.section-count {
  font-weight: 400;
  font-variant-numeric: tabular-nums;
}

.branch-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.branch-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 var(--space-2);
  padding: 4px var(--space-2);
  border-radius: 6px;
  font-size: var(--text-body);
  color: var(--text-secondary);
  cursor: default;
  white-space: nowrap;
  user-select: none;
  -webkit-user-select: none;
}

.branch-item:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.branch-item.current {
  background: var(--accent-soft);
  color: var(--text-primary);
  font-weight: 600;
}
.branch-item.remote {
  color: var(--text-secondary);
}

.branch-icon {
  flex-shrink: 0;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  color: var(--text-tertiary);
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

.remote-tree {
  overflow-y: auto;
}

.remote-group-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  width: calc(100% - 2 * var(--space-2));
  margin: 0 var(--space-2);
  padding: 4px var(--space-2);
  border: none;
  border-radius: 6px;
  background: none;
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--text-row);
  text-align: left;
  cursor: default;
}
.remote-group-toggle:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.remote-group-toggle .section-count {
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}
.remote-group-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.branch-item.remote {
  padding-left: 26px;
}

.remote-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border: 1.5px solid var(--text-tertiary);
  border-radius: 50%;
}

.tag-item.tag-selected {
  background: var(--tag-soft);
  color: var(--text-primary);
}
.tag-item .branch-icon {
  color: var(--tag);
}

.tag-section {
  flex-shrink: 0;
}

.tag-list {
  max-height: 140px;
  overflow-y: auto;
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
  cursor: default;
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
  cursor: default;
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
  cursor: default;
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
