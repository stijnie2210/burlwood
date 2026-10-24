import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { GetCommitGraph, GetCommitDiff } from '../../wailsjs/go/main/App';
import type { graph, repo } from '../../wailsjs/go/models';
import { usePrefsStore } from './prefs';

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
  const currentRepoPath = ref<string | null>(null);

  const maxColumn = computed(() => rows.value.reduce((m, r) => Math.max(m, r.maxColumn), 0));

  // Reloading the repo that is already shown is a quiet refresh: rows swap in place and the
  // selection survives, so routine operations (fetch, push, stash, commit) don't lose your place.
  async function load(repoPath: string) {
    if (!repoPath) {
      return;
    }
    const initialLimit = usePrefsStore().prefs.commitGraphLimit;
    const refresh = currentRepoPath.value === repoPath && rows.value.length > 0;
    const limit = refresh ? Math.max(initialLimit, loadedLimit.value) : initialLimit;
    if (!refresh) {
      loading.value = true;
      hasMore.value = false;
      selectedHash.value = null;
      selectedFileIndex.value = null;
      diff.value = [];
    }
    error.value = null;
    currentRepoPath.value = repoPath;
    try {
      const result = await GetCommitGraph(repoPath, limit);
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
      const result = await GetCommitGraph(currentRepoPath.value, nextLimit);
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
    try {
      diff.value = await GetCommitDiff(repoPath, hash);
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
    maxColumn,
    load,
    loadMore,
    selectCommit,
    selectFile,
    clearFileSelection,
    clear,
  };
});
