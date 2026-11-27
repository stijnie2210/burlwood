import { acceptHMRUpdate, defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { GetBlame, GetFileDiffAtCommit, GetFileHistory } from '../../bindings/burlwood/app';
import type * as repo from '../../bindings/burlwood/internal/repo';
import { describeGitError } from '../utils/gitErrors';

export type FileHistoryTab = 'history' | 'blame';
export type FileHistoryEntry = repo.FileHistoryEntry;
export type BlameChunk = Omit<repo.BlameChunk, 'lines'> & { lines: string[] };

export const useFileHistoryStore = defineStore('fileHistory', () => {
  const open = ref(false);
  const repoPath = ref('');
  const path = ref('');
  const tab = ref<FileHistoryTab>('history');

  const entries = ref<FileHistoryEntry[]>([]);
  const historyLoading = ref(false);
  const historyError = ref('');
  const selectedHash = ref('');
  const diff = ref<repo.FileDiff[]>([]);
  const diffLoading = ref(false);
  const diffError = ref('');

  // '' blames the working tree, uncommitted lines included.
  const blameRev = ref('');
  const blamePath = ref('');
  const blame = ref<BlameChunk[]>([]);
  const blameLoading = ref(false);
  const blameError = ref('');
  let blameLoadedFor: string | null = null;

  const selected = computed(() => entries.value.find((e) => e.hash === selectedHash.value) ?? null);
  const blameEntry = computed(() => entries.value.find((e) => e.hash === blameRev.value) ?? null);

  let session = 0;

  async function show(repo: string, filePath: string, startTab: FileHistoryTab, rev = '') {
    const mine = ++session;
    repoPath.value = repo;
    path.value = filePath;
    tab.value = startTab;
    entries.value = [];
    selectedHash.value = '';
    diff.value = [];
    historyError.value = '';
    diffError.value = '';
    blame.value = [];
    blameError.value = '';
    blameLoadedFor = null;
    blameRev.value = rev;
    blamePath.value = filePath;
    open.value = true;

    if (startTab === 'blame') {
      loadBlame();
    }
    historyLoading.value = true;
    try {
      const result = (await GetFileHistory(repo, filePath)) ?? [];
      if (mine !== session) {
        return;
      }
      entries.value = result;
      const first = result.find((e) => e.hash === rev) ?? result[0];
      if (first) {
        await select(first.hash);
      }
    } catch (e) {
      if (mine === session) {
        historyError.value = describeGitError(e).message;
      }
    } finally {
      if (mine === session) {
        historyLoading.value = false;
      }
    }
  }

  async function select(hash: string) {
    const entry = entries.value.find((e) => e.hash === hash);
    if (!entry) {
      return;
    }
    const mine = session;
    selectedHash.value = hash;
    diffLoading.value = true;
    diffError.value = '';
    try {
      const result =
        (await GetFileDiffAtCommit(repoPath.value, hash, entry.path, entry.oldPath)) ?? [];
      if (mine === session && selectedHash.value === hash) {
        diff.value = result;
      }
    } catch (e) {
      if (mine === session && selectedHash.value === hash) {
        diff.value = [];
        diffError.value = describeGitError(e).message;
      }
    } finally {
      if (mine === session && selectedHash.value === hash) {
        diffLoading.value = false;
      }
    }
  }

  async function loadBlame() {
    const key = `${blameRev.value}:${blamePath.value}`;
    if (blameLoadedFor === key) {
      return;
    }
    const mine = session;
    blameLoadedFor = key;
    blameLoading.value = true;
    blameError.value = '';
    blame.value = [];
    try {
      const result = (await GetBlame(repoPath.value, blamePath.value, blameRev.value)) ?? [];
      if (mine === session && blameLoadedFor === key) {
        blame.value = result.map((c) => ({ ...c, lines: c.lines ?? [] }));
      }
    } catch (e) {
      if (mine === session && blameLoadedFor === key) {
        blameError.value = describeGitError(e).message;
      }
    } finally {
      if (mine === session && blameLoadedFor === key) {
        blameLoading.value = false;
      }
    }
  }

  function showTab(next: FileHistoryTab) {
    tab.value = next;
    if (next === 'blame') {
      loadBlame();
    }
  }

  // The file may have had another name at that commit, so blame uses the entry's own path.
  function blameAt(entry: FileHistoryEntry | null) {
    blameRev.value = entry?.hash ?? '';
    blamePath.value = entry?.path ?? path.value;
    showTab('blame');
  }

  async function showInHistory(hash: string) {
    tab.value = 'history';
    await select(hash);
  }

  function close() {
    session++;
    open.value = false;
  }

  return {
    open,
    repoPath,
    path,
    tab,
    entries,
    historyLoading,
    historyError,
    selectedHash,
    selected,
    diff,
    diffLoading,
    diffError,
    blameRev,
    blameEntry,
    blame,
    blameLoading,
    blameError,
    show,
    select,
    showTab,
    blameAt,
    showInHistory,
    close,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useFileHistoryStore, import.meta.hot));
}
