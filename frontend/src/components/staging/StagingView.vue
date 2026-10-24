<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted, nextTick } from 'vue';
import type { Ref } from 'vue';
import { isImageFile, imageDataUrl } from '../../utils/imageUtils';
import { useReposStore } from '../../stores/repos';
import { useBranchesStore } from '../../stores/branches';
import { useCommitsStore } from '../../stores/commits';
import { useStagingStore, isConflictedFile } from '../../stores/staging';
import { useToastStore } from '../../stores/toast';
import HunkSelector from './HunkSelector.vue';
import ConflictView from './ConflictView.vue';
import type { Hunk } from '../../stores/staging';
import {
  clampMenuPosition,
  focusFirstMenuItem,
  handleMenuKeydown,
  keepFocusInMenu,
  menuEventFor,
} from '../../composables/useContextMenu';
import {
  AbortMerge,
  ContinueRebase,
  AbortRebase,
  GetFileBase64,
  OpenInDefaultApp,
  ShowInFinder,
  OpenInEditor,
  StashFile,
  AppendToGitignore,
  SavePatchFile,
  DeleteWorkingFile,
} from '../../../wailsjs/go/main/App';

const repos = useReposStore();
const branches = useBranchesStore();
const commits = useCommitsStore();
const staging = useStagingStore();
const toast = useToastStore();

const STAGING_FL_KEY = 'gitgood:stagingFileListWidth';
const fileListWidth = ref(parseInt(localStorage.getItem(STAGING_FL_KEY) ?? '300', 10));
watch(fileListWidth, (v) => localStorage.setItem(STAGING_FL_KEY, String(v)));

function makeResizer(width: Ref<number>, min: number, max: number) {
  let startX = 0;
  let startW = 0;
  function onMove(e: MouseEvent) {
    width.value = Math.max(min, Math.min(max, startW + e.clientX - startX));
  }
  function onUp() {
    window.removeEventListener('mousemove', onMove);
    window.removeEventListener('mouseup', onUp);
  }
  return (e: MouseEvent) => {
    startX = e.clientX;
    startW = width.value;
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  };
}

const startResize = makeResizer(fileListWidth, 180, 520);

// Keep the top-level directory: in a monorepo it tells you which package the file is in.
function shortPath(p: string) {
  const parts = p.split('/');
  if (parts.length > 3) {
    return `${parts[0]}/…/${parts.slice(-2).join('/')}`;
  }
  return p;
}

const repoPath = () => repos.activeRepo?.path ?? '';

function onSelectFile(path: string, mode: 'staged' | 'unstaged') {
  staging.selectFile(repoPath(), path, mode);
}
async function onStageFile(path: string) {
  const prevIdx = staging.unstagedFiles.findIndex((f) => f.path === path);
  const wasSelected = staging.selectedPath === path;
  try {
    await staging.stageFile(repoPath(), path);
  } catch (e) {
    toast.error(e);
    return;
  }
  if (!wasSelected) {
    return;
  }
  const next = staging.unstagedFiles[prevIdx] ?? staging.unstagedFiles[prevIdx - 1];
  if (next) {
    onSelectFile(next.path, 'unstaged');
  } else if (staging.stagedFiles.length) {
    onSelectFile(staging.stagedFiles[0].path, 'staged');
  } else {
    staging.clearSelection();
  }
}

async function onUnstageFile(path: string) {
  const prevIdx = staging.stagedFiles.findIndex((f) => f.path === path);
  const wasSelected = staging.selectedPath === path;
  try {
    await staging.unstageFile(repoPath(), path);
  } catch (e) {
    toast.error(e);
    return;
  }
  if (!wasSelected) {
    return;
  }
  const next = staging.stagedFiles[prevIdx] ?? staging.stagedFiles[prevIdx - 1];
  if (next) {
    onSelectFile(next.path, 'staged');
  } else if (staging.unstagedFiles.length) {
    onSelectFile(staging.unstagedFiles[0].path, 'unstaged');
  } else {
    staging.clearSelection();
  }
}

async function onDiscardFile(path: string) {
  const repo = repoPath();
  const name = path.split('/').pop() ?? path;
  try {
    const snapshot = await staging.discardFile(repo, path);
    if (!snapshot.undoable) {
      toast.success(`Discarded changes to ${name}`);
      return;
    }
    toast.withUndo(`Discarded changes to ${name}`, async () => {
      try {
        await staging.undoDiscard(repo, snapshot);
        if (repoPath() === repo) {
          await staging.load(repo);
        }
        toast.success(`Restored ${name}`);
      } catch (e) {
        toast.error(e);
      }
    });
  } catch (e) {
    toast.error(e);
  }
}

async function onStageHunk(hunk: Hunk) {
  if (!staging.selectedPath) {
    return;
  }
  try {
    await staging.stageHunk(repoPath(), staging.selectedPath, hunk);
  } catch (e) {
    toast.error(e);
  }
}

async function onUnstageHunk(hunk: Hunk) {
  if (!staging.selectedPath) {
    return;
  }
  try {
    await staging.unstageHunk(repoPath(), staging.selectedPath, hunk);
  } catch (e) {
    toast.error(e);
  }
}
async function onStageAll() {
  try {
    await staging.stageFiles(
      repoPath(),
      staging.unstagedFiles.map((f) => f.path)
    );
  } catch (e) {
    toast.error(e);
  }
}
async function onUnstageAll() {
  try {
    await staging.unstageFiles(
      repoPath(),
      staging.stagedFiles.map((f) => f.path)
    );
  } catch (e) {
    toast.error(e);
  }
}

// ── Commit panel ─────────────────────────────────────────────────────────────
const commitSummary = ref('');
const commitDescription = ref('');

watch(
  () => staging.isInMerge,
  (inMerge) => {
    if (inMerge && staging.mergeMessage && !commitSummary.value) {
      const lines = staging.mergeMessage.trim().split('\n');
      commitSummary.value = lines[0] ?? '';
      if (lines.length > 2) {
        commitDescription.value = lines.slice(2).join('\n').trim();
      }
    }
  }
);
const amend = ref(false);
const committing = ref(false);

const currentBranch = computed(() => branches.local.find((b) => b.isCurrent)?.name ?? 'HEAD');

// subject line: conventional max 72 chars
const summaryRemaining = computed(() => 72 - commitSummary.value.length);

const canCommit = computed(() => {
  if (committing.value) {
    return false;
  }
  if (amend.value) {
    return true; // amend: staged files optional
  }
  return staging.stagedFiles.length > 0 && commitSummary.value.trim().length > 0;
});

let amendPrefill: { summary: string; description: string } | null = null;
const amendPushed = ref(false);
const descInputRef = ref<HTMLTextAreaElement | null>(null);

// Prefill the whole message (not just the subject) so amending never silently drops the body.
async function onAmendToggle() {
  if (amend.value) {
    const info = await staging.getAmendInfo(repoPath());
    amendPushed.value = info.pushed;
    if (!commitSummary.value && !commitDescription.value) {
      amendPrefill = { summary: info.summary, description: info.description };
      commitSummary.value = info.summary;
      commitDescription.value = info.description;
    }
    return;
  }
  amendPushed.value = false;
  if (
    amendPrefill &&
    commitSummary.value === amendPrefill.summary &&
    commitDescription.value === amendPrefill.description
  ) {
    commitSummary.value = '';
    commitDescription.value = '';
  }
  amendPrefill = null;
}

async function onCommit() {
  if (!canCommit.value) {
    return;
  }
  committing.value = true;
  try {
    const parts = [commitSummary.value.trim()];
    if (commitDescription.value.trim()) {
      parts.push('', commitDescription.value.trim());
    }
    const branch = currentBranch.value;
    const wasAmend = amend.value;
    await staging.commit(repoPath(), parts.join('\n'), amend.value);
    await Promise.all([commits.load(repoPath()), branches.load(repoPath())]);
    commitSummary.value = '';
    commitDescription.value = '';
    amend.value = false;
    amendPrefill = null;
    amendPushed.value = false;
    toast.success(wasAmend ? `Amended commit on ${branch}` : `Committed to ${branch}`);
  } catch (e) {
    toast.error(e);
  } finally {
    committing.value = false;
  }
}

const STATUS_LABELS: Record<string, string> = {
  M: 'M',
  A: 'A',
  D: 'D',
  R: 'R',
  C: 'C',
  '?': '?',
  U: '!',
};

const STATUS_NAMES: Record<string, string> = {
  M: 'Modified',
  A: 'Added',
  D: 'Deleted',
  R: 'Renamed',
  C: 'Copied',
  '?': 'Untracked',
  U: 'Unmerged',
};

function statusClass(code: string): string {
  const c = code.trim();
  return c === '?' || c === '' ? 'q' : c;
}

const selectedIsConflict = computed(() => {
  if (!staging.selectedPath) {
    return false;
  }
  const f = staging.files.find((x) => x.path === staging.selectedPath);
  return f ? isConflictedFile(f) : false;
});

function onSelectConflict(path: string) {
  staging.setSelectedConflict(path);
}

// ── File context menu ─────────────────────────────────────────────────────────

type FileCtxMode = 'default' | 'ignore' | 'confirm-delete';

interface FileCtxMenu {
  x: number;
  y: number;
  path: string;
  section: 'conflict' | 'unstaged' | 'staged';
  mode: FileCtxMode;
}

const fileCtxMenu = ref<FileCtxMenu | null>(null);
const fileCtxMenuEl = ref<HTMLElement | null>(null);

async function openFileCtxMenu(e: MouseEvent, path: string, section: FileCtxMenu['section']) {
  e.preventDefault();
  e.stopPropagation();
  fileMenuReturnFocus =
    document.activeElement instanceof HTMLElement ? document.activeElement : null;
  fileCtxMenu.value = { x: e.clientX, y: e.clientY, path, section, mode: 'default' };
  await nextTick();
  const menu = fileCtxMenuEl.value;
  if (menu && fileCtxMenu.value) {
    const { x, y } = clampMenuPosition(menu, e.clientX, e.clientY);
    fileCtxMenu.value = { ...fileCtxMenu.value, x, y };
  }
  focusFirstMenuItem(menu);
}

let fileMenuReturnFocus: HTMLElement | null = null;

watch(
  () => fileCtxMenu.value?.mode,
  () => nextTick(() => keepFocusInMenu(fileCtxMenuEl.value))
);

function closeFileCtxMenu() {
  fileCtxMenu.value = null;
  const el = fileMenuReturnFocus;
  fileMenuReturnFocus = null;
  if (el?.isConnected) {
    el.focus();
  }
}

function onFileCtxMouseDown(e: MouseEvent) {
  if (fileCtxMenuEl.value && !fileCtxMenuEl.value.contains(e.target as Node)) {
    closeFileCtxMenu();
  }
}

onMounted(() => window.addEventListener('mousedown', onFileCtxMouseDown));
onUnmounted(() => window.removeEventListener('mousedown', onFileCtxMouseDown));

function fileExt(path: string): string {
  return path.split('.').pop() ?? '';
}

function fileDir(path: string): string {
  const parts = path.split('/');
  return parts.length > 1 ? parts.slice(0, -1).join('/') + '/' : '';
}

async function ctxStageFile() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  await onStageFile(m.path);
}

async function ctxUnstageFile() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  await onUnstageFile(m.path);
}

async function ctxDiscardFile() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  await onDiscardFile(m.path);
}

async function ctxStashFile() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  try {
    await StashFile(repoPath(), m.path);
    await staging.load(repoPath());
    toast.success(`Stashed ${m.path}`);
  } catch (e) {
    toast.error(e);
  }
}

async function ctxIgnore(mode: 'file' | 'ext' | 'dir') {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  let pattern: string;
  if (mode === 'file') {
    pattern = m.path;
  } else if (mode === 'ext') {
    pattern = `*.${fileExt(m.path)}`;
  } else {
    pattern = fileDir(m.path) || m.path;
  }
  closeFileCtxMenu();
  try {
    await AppendToGitignore(repoPath(), pattern);
    await staging.load(repoPath());
    toast.success(`Added "${pattern}" to .gitignore`);
  } catch (e) {
    toast.error(e);
  }
}

async function ctxOpenDefaultApp() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  try {
    await OpenInDefaultApp(repoPath(), m.path);
  } catch (e) {
    toast.error(e);
  }
}

async function ctxShowInFinder() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  try {
    await ShowInFinder(repoPath(), m.path);
  } catch (e) {
    toast.error(e);
  }
}

async function ctxOpenInEditor() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  try {
    await OpenInEditor(repoPath(), m.path);
  } catch (e) {
    toast.error(e);
  }
}

function ctxCopyPath() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  navigator.clipboard.writeText(`${repoPath()}/${m.path}`);
  toast.success('Path copied');
}

async function ctxSavePatch() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  closeFileCtxMenu();
  try {
    await SavePatchFile(repoPath(), m.path);
  } catch (e) {
    toast.error(e);
  }
}

async function ctxDeleteFile() {
  const m = fileCtxMenu.value;
  if (!m) {
    return;
  }
  if (m.mode !== 'confirm-delete') {
    fileCtxMenu.value = { ...m, mode: 'confirm-delete' };
    return;
  }
  const path = m.path;
  closeFileCtxMenu();
  try {
    await DeleteWorkingFile(repoPath(), path);
    await staging.load(repoPath());
    if (staging.selectedPath === path) {
      staging.clearSelection();
    }
    toast.success(`Deleted ${path}`);
  } catch (e) {
    toast.error(e);
  }
}

// ── Arrow key navigation ──────────────────────────────────────────────────────

type NavEntry = { path: string; section: 'conflict' | 'unstaged' | 'staged' };

const navigableFiles = computed((): NavEntry[] => [
  ...staging.conflictedFiles.map((f) => ({ path: f.path, section: 'conflict' as const })),
  ...staging.unstagedFiles.map((f) => ({ path: f.path, section: 'unstaged' as const })),
  ...staging.stagedFiles.map((f) => ({ path: f.path, section: 'staged' as const })),
]);

const currentNavIndex = computed(() => {
  if (!staging.selectedPath) {
    return -1;
  }
  if (selectedIsConflict.value) {
    return navigableFiles.value.findIndex(
      (f) => f.path === staging.selectedPath && f.section === 'conflict'
    );
  }
  const section = staging.selectedMode === 'staged' ? 'staged' : 'unstaged';
  return navigableFiles.value.findIndex(
    (f) => f.path === staging.selectedPath && f.section === section
  );
});

const fileListScrollRef = ref<HTMLElement | null>(null);

function navigateTo(delta: number) {
  const list = navigableFiles.value;
  if (!list.length) {
    return;
  }
  let idx = currentNavIndex.value;
  if (idx === -1) {
    idx = delta > 0 ? -1 : list.length;
  }
  const next = Math.max(0, Math.min(list.length - 1, idx + delta));
  const entry = list[next];
  if (entry.section === 'conflict') {
    onSelectConflict(entry.path);
  } else {
    onSelectFile(entry.path, entry.section);
  }
  const listEl = fileListScrollRef.value;
  const focusFollows =
    !!listEl &&
    (listEl.contains(document.activeElement) || document.activeElement === document.body);
  nextTick(() => {
    const selectedEl = fileListScrollRef.value?.querySelector<HTMLElement>('.file-item.selected');
    selectedEl?.scrollIntoView({ block: 'nearest' });
    if (focusFollows) {
      selectedEl?.focus({ preventScroll: true });
    }
  });
}

function onKeyDown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return;
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault();
    navigateTo(-1);
  } else if (e.key === 'ArrowDown') {
    e.preventDefault();
    navigateTo(1);
  } else if (e.key === ' ' && !e.metaKey && !e.ctrlKey) {
    if (e.target instanceof HTMLButtonElement) {
      return;
    }
    const entry = navigableFiles.value[currentNavIndex.value];
    if (!entry || entry.section === 'conflict') {
      return;
    }
    e.preventDefault();
    if (entry.section === 'unstaged') {
      onStageFile(entry.path);
    } else {
      onUnstageFile(entry.path);
    }
  }
}

onMounted(() => document.addEventListener('keydown', onKeyDown));
onUnmounted(() => document.removeEventListener('keydown', onKeyDown));

// ── Image preview ─────────────────────────────────────────────────────────────

const selectedIsImage = computed(() => !!staging.selectedPath && isImageFile(staging.selectedPath));

const imageLoading = ref(false);
const imageSrc = ref<string | null>(null);

watch(
  () => [staging.selectedPath, staging.selectedMode] as const,
  async ([path]) => {
    imageSrc.value = null;
    if (!path || !isImageFile(path)) {
      return;
    }
    imageLoading.value = true;
    try {
      const b64 = await GetFileBase64(repoPath(), path);
      imageSrc.value = imageDataUrl(path, b64);
    } catch {
      imageSrc.value = null;
    } finally {
      imageLoading.value = false;
    }
  }
);

async function onConflictResolved() {
  await staging.load(repoPath());
  if (!staging.conflictedFiles.find((f) => f.path === staging.selectedPath)) {
    staging.clearSelection();
  }
}

const confirmingAbort = ref(false);

async function doAbortMerge() {
  confirmingAbort.value = false;
  try {
    await AbortMerge(repoPath());
    await Promise.all([
      staging.load(repoPath()),
      commits.load(repoPath()),
      branches.load(repoPath()),
    ]);
    staging.clearSelection();
    toast.success('Merge aborted');
  } catch (e) {
    toast.error(e);
  }
}

const confirmingAbortRebase = ref(false);

async function doContinueRebase() {
  try {
    await ContinueRebase(repoPath());
    await Promise.all([
      staging.load(repoPath()),
      commits.load(repoPath()),
      branches.load(repoPath()),
    ]);
    staging.clearSelection();
    if (staging.isInRebase) {
      toast.success('Conflict resolved — next commit has conflicts, keep resolving');
    } else {
      toast.success('Rebase complete');
    }
  } catch (e) {
    await staging.load(repoPath());
    if (staging.isInRebase) {
      toast.success('Conflict resolved — next commit has conflicts, keep resolving');
    } else {
      toast.error(e);
    }
  }
}

async function doAbortRebase() {
  confirmingAbortRebase.value = false;
  try {
    await AbortRebase(repoPath());
    await Promise.all([
      staging.load(repoPath()),
      commits.load(repoPath()),
      branches.load(repoPath()),
    ]);
    staging.clearSelection();
    toast.success('Rebase aborted');
  } catch (e) {
    toast.error(e);
  }
}
</script>

<template>
  <div class="staging-outer">
    <!-- Merge in progress banner -->
    <div v-if="staging.isInMerge" class="merge-banner">
      <span class="merge-banner-text">Merge in progress — resolve all conflicts, then commit</span>
      <div v-if="confirmingAbort" class="merge-abort-confirm">
        <span class="merge-abort-confirm-text">Discard all resolutions?</span>
        <button class="merge-abort-btn merge-abort-btn--confirm" @click="doAbortMerge">
          Yes, abort
        </button>
        <button class="merge-abort-btn merge-abort-btn--cancel" @click="confirmingAbort = false">
          Cancel
        </button>
      </div>
      <button v-else class="merge-abort-btn" @click="confirmingAbort = true">Abort Merge</button>
    </div>

    <!-- Rebase in progress banner -->
    <div v-if="staging.isInRebase" class="rebase-banner">
      <div class="rebase-banner-left">
        <span class="rebase-banner-title">Rebasing</span>
        <span v-if="staging.rebaseState.total" class="rebase-step">
          {{ staging.rebaseState.step }}/{{ staging.rebaseState.total }}
        </span>
        <span v-if="staging.rebaseState.onto" class="rebase-onto"
          >onto {{ staging.rebaseState.onto }}</span
        >
        <span v-if="staging.rebaseState.message" class="rebase-msg">{{
          staging.rebaseState.message
        }}</span>
      </div>
      <div v-if="confirmingAbortRebase" class="merge-abort-confirm">
        <span class="merge-abort-confirm-text">Discard all rebase progress?</span>
        <button class="merge-abort-btn merge-abort-btn--confirm" @click="doAbortRebase">
          Yes, abort
        </button>
        <button
          class="merge-abort-btn merge-abort-btn--cancel"
          @click="confirmingAbortRebase = false"
        >
          Cancel
        </button>
      </div>
      <button v-else class="merge-abort-btn" @click="confirmingAbortRebase = true">
        Abort Rebase
      </button>
    </div>

    <div class="staging-view">
      <!-- File list panel (fixed width, has its own scroll + commit panel) -->
      <div class="file-panel" :style="{ width: fileListWidth + 'px' }">
        <!-- Scrollable file list area -->
        <div
          ref="fileListScrollRef"
          class="file-list-scroll"
          role="listbox"
          aria-label="Changed files"
        >
          <div v-if="staging.loading" class="list-loading">Loading…</div>
          <template v-else>
            <!-- Conflicts section (only shown during a merge) -->
            <template v-if="staging.conflictedFiles.length">
              <div class="section-header conflict-header-section">
                <span>Conflicts ({{ staging.conflictedFiles.length }})</span>
              </div>
              <div
                v-for="file in staging.conflictedFiles"
                :key="file.path + ':conflict'"
                class="file-item file-item--conflict"
                :class="{ selected: staging.selectedPath === file.path }"
                role="option"
                :aria-selected="staging.selectedPath === file.path"
                :tabindex="staging.selectedPath === file.path ? 0 : -1"
                @click="onSelectConflict(file.path)"
                @contextmenu="openFileCtxMenu($event, file.path, 'conflict')"
                @keydown.shift.f10.prevent="
                  openFileCtxMenu(menuEventFor($event), file.path, 'conflict')
                "
              >
                <span class="status-code sc-conflict">!</span>
                <span class="file-name" :title="file.path">{{ shortPath(file.path) }}</span>
              </div>
            </template>

            <!-- Unstaged section -->
            <div class="section-header">
              <span>Unstaged ({{ staging.unstagedFiles.length }})</span>
              <div class="section-actions">
                <button
                  class="section-icon-btn"
                  title="Refresh status"
                  aria-label="Refresh status"
                  @click="staging.load(repoPath())"
                >
                  <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                    <path
                      d="M10.5 6a4.5 4.5 0 1 1-1.3-3.2M10.5 1.5v2.8H7.7"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="1.3"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                  </svg>
                </button>
                <button v-if="staging.unstagedFiles.length" class="section-btn" @click="onStageAll">
                  Stage All
                </button>
              </div>
            </div>
            <div v-if="!staging.unstagedFiles.length" class="empty-section">Clean</div>
            <div
              v-for="file in staging.unstagedFiles"
              :key="file.path"
              class="file-item"
              :class="{
                selected: staging.selectedPath === file.path && staging.selectedMode === 'unstaged',
              }"
              role="option"
              :aria-selected="
                staging.selectedPath === file.path && staging.selectedMode === 'unstaged'
              "
              :tabindex="
                staging.selectedPath === file.path && staging.selectedMode === 'unstaged' ? 0 : -1
              "
              @click="onSelectFile(file.path, 'unstaged')"
              @contextmenu="openFileCtxMenu($event, file.path, 'unstaged')"
              @keydown.shift.f10.prevent="
                openFileCtxMenu(menuEventFor($event), file.path, 'unstaged')
              "
            >
              <span
                class="status-code"
                :class="`sc-${statusClass(file.unstaged)}`"
                :title="STATUS_NAMES[file.unstaged]"
              >
                {{ STATUS_LABELS[file.unstaged] ?? file.unstaged }}
              </span>
              <span class="file-name" :title="file.path">{{ shortPath(file.path) }}</span>
              <button
                class="inline-btn stage-inline"
                tabindex="-1"
                title="Stage file (Space)"
                :aria-label="`Stage ${file.path}`"
                @click.stop="onStageFile(file.path)"
              >
                +
              </button>
            </div>

            <!-- Staged section -->
            <div class="section-header staged-header">
              <span>Staged ({{ staging.stagedFiles.length }})</span>
              <button v-if="staging.stagedFiles.length" class="section-btn" @click="onUnstageAll">
                Unstage All
              </button>
            </div>
            <div v-if="!staging.stagedFiles.length" class="empty-section">Nothing staged</div>
            <div
              v-for="file in staging.stagedFiles"
              :key="file.path + ':staged'"
              class="file-item"
              :class="{
                selected: staging.selectedPath === file.path && staging.selectedMode === 'staged',
              }"
              role="option"
              :aria-selected="
                staging.selectedPath === file.path && staging.selectedMode === 'staged'
              "
              :tabindex="
                staging.selectedPath === file.path && staging.selectedMode === 'staged' ? 0 : -1
              "
              @click="onSelectFile(file.path, 'staged')"
              @contextmenu="openFileCtxMenu($event, file.path, 'staged')"
              @keydown.shift.f10.prevent="
                openFileCtxMenu(menuEventFor($event), file.path, 'staged')
              "
            >
              <span
                class="status-code"
                :class="`sc-${statusClass(file.staged)}`"
                :title="STATUS_NAMES[file.staged]"
              >
                {{ STATUS_LABELS[file.staged] ?? file.staged }}
              </span>
              <span class="file-name" :title="file.path">{{ shortPath(file.path) }}</span>
              <button
                class="inline-btn unstage-inline"
                tabindex="-1"
                title="Unstage file (Space)"
                :aria-label="`Unstage ${file.path}`"
                @click.stop="onUnstageFile(file.path)"
              >
                −
              </button>
            </div>
          </template>
        </div>

        <!-- Commit panel — pinned at bottom -->
        <div class="commit-panel">
          <!-- During rebase: show Continue button instead of normal commit form -->
          <template v-if="staging.isInRebase">
            <div class="rebase-action-hint">
              Resolve all conflicts above, then continue the rebase.
            </div>
            <button
              class="commit-btn rebase-continue-btn"
              :disabled="staging.conflictedFiles.length > 0"
              @click="doContinueRebase"
            >
              <span v-if="staging.conflictedFiles.length > 0"
                >Resolve {{ staging.conflictedFiles.length }} conflict(s) first</span
              >
              <span v-else>Continue Rebase</span>
            </button>
          </template>

          <template v-else>
            <div class="summary-row">
              <input
                v-model="commitSummary"
                class="summary-input"
                type="text"
                placeholder="Summary"
                aria-label="Commit summary"
                maxlength="200"
                spellcheck="true"
                @keydown.enter.exact.prevent="descInputRef?.focus()"
                @keydown.meta.enter="onCommit"
              />
              <span
                class="char-count"
                aria-hidden="true"
                :class="{
                  over: summaryRemaining < 0,
                  warn: summaryRemaining < 10 && summaryRemaining >= 0,
                }"
              >
                {{ summaryRemaining }}
              </span>
            </div>

            <textarea
              ref="descInputRef"
              v-model="commitDescription"
              class="desc-input"
              placeholder="Description (optional)"
              aria-label="Commit description"
              rows="3"
              @keydown.meta.enter="onCommit"
            />

            <label class="amend-row">
              <input v-model="amend" type="checkbox" @change="onAmendToggle" />
              <span>Amend previous commit</span>
            </label>
            <p v-if="amend && amendPushed" class="amend-warning" role="status">
              This commit is already pushed. Amending rewrites it, so the next push will need a
              force push.
            </p>

            <button
              class="commit-btn"
              :class="{ ready: canCommit && commitSummary.trim() }"
              :disabled="!canCommit"
              @click="onCommit"
            >
              <span v-if="committing">Committing…</span>
              <span v-else-if="!commitSummary.trim() && !amend">Type a message to commit</span>
              <span v-else-if="amend">
                Amend last commit on <strong>{{ currentBranch }}</strong>
              </span>
              <span v-else>
                Commit to <strong>{{ currentBranch }}</strong>
              </span>
            </button>
          </template>
        </div>
      </div>

      <!-- Resize handle -->
      <div class="resize-handle" @mousedown.prevent="startResize" />

      <!-- Diff / hunk panel -->
      <div class="diff-panel">
        <div v-if="!staging.selectedPath" class="diff-empty">Select a file to view changes</div>
        <ConflictView
          v-else-if="selectedIsConflict"
          :repo-path="repoPath()"
          :path="staging.selectedPath!"
          @resolved="onConflictResolved"
        />
        <div v-else-if="selectedIsImage" class="image-preview-panel">
          <div class="image-preview-header">{{ staging.selectedPath }}</div>
          <div v-if="imageLoading" class="image-preview-loading">Loading…</div>
          <div v-else-if="imageSrc" class="image-preview-container">
            <img :src="imageSrc" class="image-preview-img" :alt="staging.selectedPath ?? ''" />
          </div>
          <div v-else class="image-preview-missing">File not found on disk</div>
        </div>
        <HunkSelector
          v-else
          :diffs="staging.diff"
          :mode="staging.selectedMode === 'staged' ? 'unstage' : 'stage'"
          :path="staging.selectedPath"
          :diff-loading="staging.diffLoading"
          :diff-error="staging.diffError"
          @stage-file="onStageFile(staging.selectedPath!)"
          @unstage-file="onUnstageFile(staging.selectedPath!)"
          @discard-file="onDiscardFile(staging.selectedPath!)"
          @stage-hunk="onStageHunk"
          @unstage-hunk="onUnstageHunk"
        />
      </div>
    </div>
  </div>

  <!-- File context menu (teleported to escape overflow clipping) -->
  <Teleport to="body">
    <div
      v-if="fileCtxMenu"
      ref="fileCtxMenuEl"
      class="ctx-menu"
      role="menu"
      :style="{ left: fileCtxMenu.x + 'px', top: fileCtxMenu.y + 'px' }"
      @keydown="handleMenuKeydown($event, fileCtxMenuEl, closeFileCtxMenu)"
    >
      <div class="ctx-menu-label">{{ fileCtxMenu.path.split('/').pop() }}</div>

      <!-- Default mode -->
      <template v-if="fileCtxMenu.mode === 'default'">
        <!-- Unstaged-only actions -->
        <template v-if="fileCtxMenu.section === 'unstaged'">
          <button role="menuitem" class="ctx-menu-item ctx-menu-item-primary" @click="ctxStageFile">
            Stage file
          </button>
          <button
            role="menuitem"
            class="ctx-menu-item ctx-menu-item-danger"
            @click="ctxDiscardFile"
          >
            Discard changes
          </button>
          <div class="ctx-menu-divider" />
          <button role="menuitem" class="ctx-menu-item" @click="fileCtxMenu!.mode = 'ignore'">
            Ignore <span style="float: right; opacity: 0.5">▸</span>
          </button>
          <button role="menuitem" class="ctx-menu-item" @click="ctxStashFile">Stash file</button>
          <div class="ctx-menu-divider" />
        </template>

        <!-- Staged-only actions -->
        <template v-else-if="fileCtxMenu.section === 'staged'">
          <button
            role="menuitem"
            class="ctx-menu-item ctx-menu-item-primary"
            @click="ctxUnstageFile"
          >
            Unstage file
          </button>
          <div class="ctx-menu-divider" />
        </template>

        <!-- Common actions -->
        <button role="menuitem" class="ctx-menu-item" @click="ctxOpenDefaultApp">
          Open in default app
        </button>
        <button role="menuitem" class="ctx-menu-item" @click="ctxShowInFinder">
          Show in Finder
        </button>
        <button role="menuitem" class="ctx-menu-item" @click="ctxOpenInEditor">
          Open in editor
        </button>
        <div class="ctx-menu-divider" />
        <button role="menuitem" class="ctx-menu-item" @click="ctxCopyPath">Copy path</button>
        <button
          v-if="fileCtxMenu.section !== 'conflict'"
          role="menuitem"
          class="ctx-menu-item"
          @click="ctxSavePatch"
        >
          Save patch…
        </button>

        <!-- Unstaged: delete -->
        <template v-if="fileCtxMenu.section === 'unstaged'">
          <div class="ctx-menu-divider" />
          <button role="menuitem" class="ctx-menu-item ctx-menu-item-danger" @click="ctxDeleteFile">
            Delete file
          </button>
        </template>
      </template>

      <!-- Ignore submenu -->
      <template v-else-if="fileCtxMenu.mode === 'ignore'">
        <button role="menuitem" class="ctx-menu-item" @click="fileCtxMenu!.mode = 'default'">
          ← Back
        </button>
        <div class="ctx-menu-divider" />
        <button role="menuitem" class="ctx-menu-item" @click="ctxIgnore('file')">
          Ignore this file
        </button>
        <button
          role="menuitem"
          class="ctx-menu-item"
          :disabled="!fileExt(fileCtxMenu.path)"
          @click="ctxIgnore('ext')"
        >
          Ignore all *.{{ fileExt(fileCtxMenu.path) }} files
        </button>
        <button
          role="menuitem"
          class="ctx-menu-item"
          :disabled="!fileDir(fileCtxMenu.path)"
          @click="ctxIgnore('dir')"
        >
          Ignore containing folder
        </button>
      </template>

      <!-- Delete confirm -->
      <template v-else-if="fileCtxMenu.mode === 'confirm-delete'">
        <div class="ctx-menu-confirm">
          <p class="ctx-menu-confirm-text">
            Delete "{{ fileCtxMenu.path.split('/').pop() }}" from disk? This can't be undone.
          </p>
          <div class="ctx-menu-row">
            <button class="ctx-menu-btn ctx-menu-btn-danger" @click="ctxDeleteFile">Delete</button>
            <button class="ctx-menu-btn" @click="closeFileCtxMenu">Cancel</button>
          </div>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.staging-outer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

/* Merge banner */
.merge-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--inset-panel);
  background: var(--warning-soft);
  border-bottom: 1px solid var(--warning-border);
  flex-shrink: 0;
}

.merge-banner-text {
  font-size: var(--text-row);
  color: var(--warning);
}

.merge-abort-confirm {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.merge-abort-confirm-text {
  font-size: var(--text-meta);
  color: var(--danger);
}

.merge-abort-btn {
  padding: var(--space-1) var(--space-2);
  background: var(--danger-soft);
  border: 1px solid var(--danger-border);
  border-radius: 4px;
  color: var(--danger);
  font-size: var(--text-meta);
  cursor: pointer;
}
.merge-abort-btn:hover {
  background: var(--danger-soft-hover);
  color: var(--danger);
}
.merge-abort-btn--confirm {
  background: var(--danger-soft);
  border-color: var(--danger-border);
  color: var(--danger);
}
.merge-abort-btn--confirm:hover {
  background: var(--danger-soft-hover);
  color: var(--danger);
}
.merge-abort-btn--cancel {
  background: transparent;
  border-color: var(--border);
  color: var(--text-tertiary);
  margin-left: 2px;
}
.merge-abort-btn--cancel:hover {
  border-color: var(--border-strong);
  color: var(--text-secondary);
}

/* Rebase banner */
.rebase-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--inset-panel);
  background: var(--accent-soft);
  border-bottom: 1px solid var(--accent-border);
  flex-shrink: 0;
  gap: 8px;
}

.rebase-banner-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
}

.rebase-banner-title {
  font-size: var(--text-row);
  font-weight: 600;
  color: var(--accent-text);
  flex-shrink: 0;
}

.rebase-step {
  font-size: var(--text-meta);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.rebase-onto {
  font-size: var(--text-meta);
  color: var(--text-secondary);
  flex-shrink: 0;
}

.rebase-msg {
  font-size: var(--text-meta);
  color: var(--text-primary);
  font-style: italic;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rebase-action-hint {
  font-size: var(--text-meta);
  color: var(--text-secondary);
  padding: 2px 2px 4px;
  text-align: center;
}

.rebase-continue-btn {
  background: var(--accent-fill) !important;
  color: var(--text-on-accent) !important;
  border: 1px solid var(--accent-fill) !important;
}
.rebase-continue-btn:hover:not(:disabled) {
  background: var(--accent-fill-hover) !important;
}
.rebase-continue-btn:disabled {
  cursor: default;
  color: var(--text-tertiary) !important;
  border-color: var(--border) !important;
  background: var(--surface-control) !important;
}

.staging-view {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* File list panel — no overflow here; children manage it */
.file-panel {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface-chrome);
  border-right: 1px solid var(--border);
  overflow: hidden;
  min-height: 0;
}

/* Scrollable file list */
.file-list-scroll {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.section-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: none;
  border: none;
  border-radius: 4px;
  color: var(--text-tertiary);
  cursor: pointer;
}
.section-icon-btn:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.list-loading {
  padding: 16px;
  color: var(--text-secondary);
  font-size: var(--text-row);
  text-align: center;
}

.section-header {
  user-select: none;
  -webkit-user-select: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--inset-panel) var(--space-1);
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-secondary);
  flex-shrink: 0;
}
.staged-header {
  margin-top: 4px;
  border-top: 1px solid var(--border-subtle);
  padding-top: 8px;
}

.section-btn {
  font-size: var(--text-meta);
  padding: 1px 6px;
  background: transparent;
  border: 1px solid var(--border-strong);
  border-radius: 3px;
  color: var(--text-tertiary);
  cursor: pointer;
}
.section-btn:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}

.empty-section {
  padding: var(--space-1) var(--inset-panel) var(--space-2);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  font-style: italic;
}

.file-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0 var(--inset-panel);
  font-size: var(--text-row);
  cursor: pointer;
  min-width: 0;
  position: relative;
  min-height: var(--row-h);
}
.file-item:hover {
  background: var(--surface-hover);
}
.file-item.selected {
  background: var(--surface-selected);
}

.status-code {
  font-size: var(--text-meta);
  font-weight: 700;
  font-family: var(--font-mono);
  width: 14px;
  text-align: center;
  flex-shrink: 0;
}
.sc-M {
  color: var(--modified);
}
.sc-A {
  color: var(--add);
}
.sc-D {
  color: var(--del);
}
.sc-R {
  color: var(--accent-text);
}
.sc-q {
  color: var(--untracked);
}
.sc-conflict {
  color: var(--danger);
}

.conflict-header-section {
  color: var(--danger);
}

.file-item--conflict .file-name {
  color: var(--danger);
}
.file-item--conflict.selected .file-name {
  color: var(--danger);
}
.file-item--conflict:hover {
  background: var(--surface-hover);
}

.file-name {
  flex: 1;
  font-size: var(--text-meta);
  font-family: var(--font-mono);
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-item.selected .file-name {
  color: var(--text-primary);
}

.inline-btn {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border-radius: 3px;
  border: none;
  font-size: var(--text-body);
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.1s;
  position: absolute;
  right: var(--space-2);
  top: 50%;
  margin-top: -9px;
}
.file-item:hover .inline-btn,
.file-item:focus-within .inline-btn,
.file-item.selected .inline-btn {
  opacity: 1;
}
.file-item:hover .file-name,
.file-item:focus-within .file-name,
.file-item.selected .file-name {
  padding-right: 22px;
}
.stage-inline {
  background: var(--add-soft);
  color: var(--add);
}
.stage-inline:hover {
  background: var(--add-soft-strong);
}
.unstage-inline {
  background: var(--accent-soft);
  color: var(--accent-text);
}
.unstage-inline:hover {
  background: var(--accent-soft-strong);
}

/* ── Commit panel ─────────────────────────── */
.commit-panel {
  flex-shrink: 0;
  border-top: 1px solid var(--border-subtle);
  background: var(--surface-raised);
  padding: var(--inset-panel);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.amend-warning {
  margin: 0;
  font-size: var(--text-meta);
  line-height: 1.4;
  color: var(--warning);
}

.amend-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--text-meta);
  color: var(--text-secondary);
  cursor: pointer;
  user-select: none;
  padding: 0 2px;
}
.amend-row input[type='checkbox'] {
  accent-color: var(--accent-text);
  cursor: pointer;
}
.amend-row:hover span {
  color: var(--text-primary);
}

.summary-row {
  position: relative;
}

.summary-input {
  width: 100%;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: var(--text-row);
  padding: 6px 30px 6px 8px;
  outline: none;
  font-family: inherit;
  line-height: 1.4;
  box-sizing: border-box;
}
.summary-input:focus {
  border-color: var(--focus-ring);
}
.summary-input::placeholder {
  color: var(--text-tertiary);
}

.char-count {
  position: absolute;
  right: 6px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  pointer-events: none;
  top: 50%;
  transform: translateY(-50%);
}
.char-count.warn {
  color: var(--modified);
}
.char-count.over {
  color: var(--del);
}

.desc-input {
  width: 100%;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: var(--text-row);
  padding: 5px 8px;
  resize: none;
  outline: none;
  font-family: inherit;
  line-height: 1.4;
  box-sizing: border-box;
}
.desc-input:focus {
  border-color: var(--focus-ring);
}
.desc-input::placeholder {
  color: var(--text-tertiary);
}

.commit-btn {
  width: 100%;
  padding: 8px;
  border: none;
  border-radius: 5px;
  font-size: var(--text-row);
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
  background: var(--surface-control);
  color: var(--text-secondary);
  margin-top: var(--space-1);
}
.commit-btn:disabled {
  cursor: default;
}
.commit-btn.ready {
  background: var(--accent-fill);
  color: var(--text-on-accent);
  border: 1px solid var(--accent-fill);
}
.commit-btn.ready:hover {
  background: var(--accent-fill-hover);
}
.commit-btn strong {
  font-weight: 700;
}

/* Resize handle */
.resize-handle {
  width: 4px;
  flex-shrink: 0;
  background: var(--border-subtle);
  cursor: col-resize;
  transition: background 0.15s;
}
.resize-handle:hover {
  background: var(--accent-fill);
}

/* Diff panel */
.diff-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.diff-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: var(--text-secondary);
  font-size: var(--text-body);
}

/* Image preview */
.image-preview-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.image-preview-header {
  flex-shrink: 0;
  padding: 0 var(--inset-panel);
  font-size: var(--text-meta);
  font-family: var(--font-mono);
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-base);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  height: var(--file-header-h);
  box-sizing: border-box;
  display: flex;
  align-items: center;
}

.image-preview-loading,
.image-preview-missing {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  font-size: var(--text-body);
}

.image-preview-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  padding: 24px;
  background: var(--surface-sunken);
}

.image-preview-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 4px;
  border: 1px solid var(--border-subtle);
}
</style>
