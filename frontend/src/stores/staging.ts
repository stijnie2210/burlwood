import { acceptHMRUpdate, defineStore } from 'pinia';
import { ref, computed } from 'vue';
import {
  GetStatus,
  GetWorkingDiff,
  StageFile,
  UnstageFile,
  StageHunk,
  UnstageHunk,
  DiscardFile,
  StageFiles,
  UnstageFiles,
  UndoDiscard,
  FetchAll,
  PullBranch,
  Commit,
  GetAmendInfo,
  IsInMerge,
  GetMergeMessage,
  IsInRebase,
  GetRebaseState,
  SyncAutostash,
} from '../../wailsjs/go/main/App';
import type { gitcli, repo } from '../../wailsjs/go/models';
import { describeGitError } from '../utils/gitErrors';
import { useToastStore } from './toast';
import { isActive } from './repos';

export type FileStatus = repo.FileStatus;
export type FileDiff = repo.FileDiff;
export type Hunk = repo.Hunk;

export function buildHunkPatch(path: string, hunk: Hunk): string {
  const parts = [
    `diff --git a/${path} b/${path}`,
    `--- a/${path}`,
    `+++ b/${path}`,
    hunk.header,
    ...hunk.lines.map((l) => {
      if (l.type === 'add') {
        return `+${l.content}`;
      }
      if (l.type === 'del') {
        return `-${l.content}`;
      }
      return ` ${l.content}`;
    }),
  ];
  return parts.join('\n') + '\n';
}

export function isConflictedFile(f: FileStatus): boolean {
  return (
    f.staged === 'U' ||
    f.unstaged === 'U' ||
    (f.staged === 'A' && f.unstaged === 'A') ||
    (f.staged === 'D' && f.unstaged === 'D')
  );
}

export const useStagingStore = defineStore('staging', () => {
  const files = ref<FileStatus[]>([]);
  const loading = ref(false);
  const selectedPath = ref<string | null>(null);
  const selectedMode = ref<'staged' | 'unstaged'>('unstaged');
  const diff = ref<FileDiff[]>([]);
  const diffLoading = ref(false);
  const diffError = ref<string | null>(null);
  const isInMerge = ref(false);
  const mergeMessage = ref('');
  const isInRebase = ref(false);
  const rebaseState = ref<repo.RebaseState>({ step: 0, total: 0, message: '', onto: '' });

  const conflictedFiles = computed(() => files.value.filter(isConflictedFile));

  const unstagedFiles = computed(() =>
    files.value.filter((f) => !isConflictedFile(f) && f.unstaged !== ' ' && f.unstaged !== '')
  );

  const stagedFiles = computed(() =>
    files.value.filter(
      (f) => !isConflictedFile(f) && f.staged !== ' ' && f.staged !== '?' && f.staged !== ''
    )
  );

  const totalChanges = computed(() => files.value.length);

  let loadedRepoPath: string | null = null;

  // Only the first load of a repo shows the loading state; later loads swap the list in place.
  async function load(repoPath: string) {
    if (!repoPath || !isActive(repoPath)) {
      return;
    }
    if (loadedRepoPath !== repoPath) {
      clear();
      loading.value = true;
    }
    try {
      try {
        await SyncAutostash(repoPath);
      } catch (e) {
        // genuine failure, not the normal "still waiting" case (that resolves with no error)
        console.error('SyncAutostash failed:', e);
      }
      const [statusResult, mergeState, mergeMsg, rebaseActive, rebaseInfo] = await Promise.all([
        GetStatus(repoPath),
        IsInMerge(repoPath),
        GetMergeMessage(repoPath),
        IsInRebase(repoPath),
        GetRebaseState(repoPath),
      ]);
      if (!isActive(repoPath)) {
        return;
      }
      files.value = statusResult;
      isInMerge.value = mergeState;
      mergeMessage.value = mergeMsg;
      isInRebase.value = rebaseActive;
      rebaseState.value = rebaseInfo;
      loadedRepoPath = repoPath;
    } catch (e) {
      if (isActive(repoPath)) {
        useToastStore().error(e);
      }
    } finally {
      if (isActive(repoPath)) {
        loading.value = false;
      }
    }
  }

  // Background refresh (window focus): also re-reads the open diff, since the file may have
  // been edited elsewhere, without flashing the diff's loading state.
  async function refresh(repoPath: string) {
    await load(repoPath);
    const path = selectedPath.value;
    if (!path || !repoPath) {
      return;
    }
    const stillChanged = files.value.some((f) => f.path === path);
    if (!stillChanged) {
      clearSelection();
      return;
    }
    await reloadDiff(repoPath);
  }

  async function selectFile(repoPath: string, path: string, mode: 'staged' | 'unstaged') {
    selectedPath.value = path;
    selectedMode.value = mode;
    diffLoading.value = true;
    diff.value = [];
    diffError.value = null;
    try {
      const result = await GetWorkingDiff(repoPath, path, mode === 'staged');
      if (isActive(repoPath) && selectedPath.value === path) {
        diff.value = result;
      }
    } catch (e) {
      if (isActive(repoPath)) {
        diffError.value = describeGitError(e).message;
      }
    } finally {
      diffLoading.value = false;
    }
  }

  // Swaps the open diff in place (no loading state), so staging a hunk keeps your scroll position.
  async function reloadDiff(repoPath: string) {
    const path = selectedPath.value;
    if (!path) {
      return;
    }
    try {
      const result = await GetWorkingDiff(repoPath, path, selectedMode.value === 'staged');
      if (!isActive(repoPath) || selectedPath.value !== path) {
        return;
      }
      diff.value = result;
      diffError.value = null;
    } catch (e) {
      diffError.value = describeGitError(e).message;
    }
  }

  async function stageFile(repoPath: string, path: string) {
    await StageFile(repoPath, path);
    await load(repoPath);
  }

  async function unstageFile(repoPath: string, path: string) {
    await UnstageFile(repoPath, path);
    await load(repoPath);
  }

  async function stageFiles(repoPath: string, paths: string[]) {
    await StageFiles(repoPath, paths);
    await load(repoPath);
  }

  async function unstageFiles(repoPath: string, paths: string[]) {
    await UnstageFiles(repoPath, paths);
    await load(repoPath);
  }

  async function stageHunk(repoPath: string, path: string, hunk: Hunk) {
    const patch = buildHunkPatch(path, hunk);
    await StageHunk(repoPath, patch);
    await load(repoPath);
    if (selectedPath.value === path) {
      await reloadDiff(repoPath);
    }
  }

  async function unstageHunk(repoPath: string, path: string, hunk: Hunk) {
    const patch = buildHunkPatch(path, hunk);
    await UnstageHunk(repoPath, patch);
    await load(repoPath);
    if (selectedPath.value === path) {
      await reloadDiff(repoPath);
    }
  }

  async function discardFile(repoPath: string, path: string): Promise<gitcli.DiscardSnapshot> {
    const snapshot = await DiscardFile(repoPath, path);
    await load(repoPath);
    if (selectedPath.value === path) {
      diff.value = [];
      selectedPath.value = null;
    }
    return snapshot;
  }

  async function undoDiscard(repoPath: string, snapshot: gitcli.DiscardSnapshot) {
    await UndoDiscard(repoPath, snapshot);
  }

  async function fetchAll(repoPath: string) {
    await FetchAll(repoPath);
  }

  async function pullBranch(repoPath: string, mode: string) {
    try {
      await PullBranch(repoPath, mode);
    } finally {
      // even a failed pull (e.g. merge conflict) needs load() for the conflict UI to show up
      await load(repoPath);
    }
  }

  async function commit(repoPath: string, message: string, amend: boolean) {
    await Commit(repoPath, message, amend);
    selectedPath.value = null;
    diff.value = [];
    await load(repoPath);
  }

  async function getAmendInfo(repoPath: string): Promise<gitcli.AmendInfo> {
    return GetAmendInfo(repoPath);
  }

  function clearSelection() {
    selectedPath.value = null;
    diff.value = [];
  }

  function setSelectedConflict(path: string) {
    selectedPath.value = path;
    selectedMode.value = 'unstaged';
    diff.value = [];
  }

  function clear() {
    loadedRepoPath = null;
    files.value = [];
    selectedPath.value = null;
    diff.value = [];
    isInMerge.value = false;
    mergeMessage.value = '';
    isInRebase.value = false;
    rebaseState.value = { step: 0, total: 0, message: '', onto: '' };
  }

  return {
    files,
    loading,
    selectedPath,
    selectedMode,
    diff,
    diffLoading,
    diffError,
    isInMerge,
    mergeMessage,
    isInRebase,
    rebaseState,
    conflictedFiles,
    unstagedFiles,
    stagedFiles,
    totalChanges,
    load,
    selectFile,
    stageFile,
    unstageFile,
    stageFiles,
    unstageFiles,
    stageHunk,
    unstageHunk,
    discardFile,
    undoDiscard,
    fetchAll,
    pullBranch,
    commit,
    getAmendInfo,
    clearSelection,
    refresh,
    setSelectedConflict,
    clear,
  };
});

// Without this, editing a store during `wails dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useStagingStore, import.meta.hot));
}
