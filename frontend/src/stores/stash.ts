import { acceptHMRUpdate, defineStore } from 'pinia';
import { isActive } from './repos';
import { ref } from 'vue';
import { ListStashes } from '../../bindings/burlwood/app';

export interface StashEntry {
  index: number;
  ref: string;
  message: string;
  branch: string;
  hash: string;
  date: string;
}

export const useStashStore = defineStore('stash', () => {
  const entries = ref<StashEntry[]>([]);

  let loadedFor: string | null = null;
  const cache = new Map<string, StashEntry[]>();

  // Same rules as the branch list: never load for an inactive repo, show the cached list on a
  // switch, drop results for a repo that stopped being active.
  async function load(repoPath: string) {
    if (!repoPath || !isActive(repoPath)) {
      return;
    }
    if (loadedFor !== repoPath) {
      entries.value = cache.get(repoPath) ?? [];
      loadedFor = repoPath;
    }
    try {
      const result = await ListStashes(repoPath);
      if (isActive(repoPath)) {
        entries.value = (result as StashEntry[]) ?? [];
        cache.set(repoPath, entries.value);
      }
    } catch {
      if (isActive(repoPath)) {
        entries.value = [];
      }
    }
  }

  function clear() {
    loadedFor = null;
    entries.value = [];
  }

  return { entries, load, clear };
});

// Without this, editing a store during `wails3 dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useStashStore, import.meta.hot));
}
