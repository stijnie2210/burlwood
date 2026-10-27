import { acceptHMRUpdate, defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { GetCommitGraph, GetCommitDiff } from '../../bindings/burlwood/app';
import type * as graph from '../../bindings/burlwood/internal/graph';
import type * as repo from '../../bindings/burlwood/internal/repo';
import { usePrefsStore } from './prefs';
import { isActive } from './repos';
import { describeGitError } from '../utils/gitErrors';
import { orderByFolder } from '../utils/folders';

export type GraphRow = graph.GraphRow;
export type FileDiff = repo.FileDiff;

const BATCH_SIZE = 2000;

export const useCommitsStore = defineStore('commits', () => {
  const rows = ref<GraphRow[]>([]);
  const loading = ref(false);
  const loadingMore = ref(false);
  const hasMore = ref(false);
  const loadedLimit = ref(0);
  const error = ref<string | null>(null);
  const selectedHash = ref<string | null>(null);
  const selectedFileIndex = ref<number | null>(null);
  const diff = ref<FileDiff[]>([]);
  const diffLoading = ref(false);
  const diffError = ref<string | null>(null);
  const currentRepoPath = ref<string | null>(null);
  const searchQuery = ref('');

  const maxColumn = computed(() => rows.value.reduce((m, r) => Math.max(m, r.maxColumn), 0));

  // Reloading the repo that is already shown is a quiet refresh: rows swap in place and the
  // selection survives, so routine operations (fetch, push, stash, commit) don't lose your place.
  async function load(repoPath: string) {
    if (!repoPath || !isActive(repoPath)) {
      return;
    }
    const initialLimit = usePrefsStore().prefs.commitGraphLimit;
    const refresh = currentRepoPath.value === repoPath && rows.value.length > 0;
    const limit = refresh ? Math.max(initialLimit, loadedLimit.value) : initialLimit;
    if (!refresh) {
      loading.value = true;
      rows.value = [];
      hasMore.value = false;
      selectedHash.value = null;
      selectedFileIndex.value = null;
      diff.value = [];
      searchQuery.value = '';
    }
    error.value = null;
    currentRepoPath.value = repoPath;
    try {
      const result = (await GetCommitGraph(repoPath, limit)) ?? [];
      if (currentRepoPath.value !== repoPath) {
        return;
      }
      rows.value = result;
      loadedLimit.value = limit;
      hasMore.value = result.length === limit;
      if (selectedHash.value && !result.some((r) => r.hash === selectedHash.value)) {
        selectedHash.value = null;
        selectedFileIndex.value = null;
        diff.value = [];
      }
    } catch (e) {
      error.value = String(e);
      if (!refresh) {
        rows.value = [];
      }
    } finally {
      loading.value = false;
    }
  }

  async function loadMore() {
    if (loadingMore.value || !hasMore.value || !currentRepoPath.value) {
      return;
    }
    loadingMore.value = true;
    const nextLimit = loadedLimit.value + BATCH_SIZE;
    try {
      const result = (await GetCommitGraph(currentRepoPath.value, nextLimit)) ?? [];
      rows.value = result;
      loadedLimit.value = nextLimit;
      hasMore.value = result.length === nextLimit;
    } finally {
      loadingMore.value = false;
    }
  }

  function selectFile(index: number) {
    selectedFileIndex.value = index;
  }

  function clearFileSelection() {
    selectedFileIndex.value = null;
  }

  async function selectCommit(repoPath: string, hash: string) {
    selectedHash.value = hash;
    selectedFileIndex.value = null;
    diffLoading.value = true;
    diff.value = [];
    diffError.value = null;
    try {
      const result = (await GetCommitDiff(repoPath, hash)) ?? [];
      if (currentRepoPath.value === repoPath && selectedHash.value === hash) {
        diff.value = orderByFolder(result, (f) => f.newPath || f.oldPath || '');
      }
    } catch (e) {
      if (currentRepoPath.value === repoPath) {
        diffError.value = describeGitError(e).message;
      }
    } finally {
      diffLoading.value = false;
    }
  }

  function clear() {
    rows.value = [];
    error.value = null;
    selectedHash.value = null;
    selectedFileIndex.value = null;
    diff.value = [];
    hasMore.value = false;
    currentRepoPath.value = null;
    searchQuery.value = '';
  }

  return {
    rows,
    loading,
    loadingMore,
    hasMore,
    error,
    selectedHash,
    selectedFileIndex,
    diff,
    diffLoading,
    diffError,
    searchQuery,
    maxColumn,
    load,
    loadMore,
    selectCommit,
    selectFile,
    clearFileSelection,
    clear,
  };
});

// Without this, editing a store during `wails3 dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useCommitsStore, import.meta.hot));
}
