import { acceptHMRUpdate, defineStore } from 'pinia';
import { isActive } from './repos';
import { ref } from 'vue';
import { ListBranches, GetAheadBehind, SwitchBranch, FetchAll } from '../../wailsjs/go/main/App';

export interface BranchInfo {
  name: string;
  isRemote: boolean;
  remote: string;
  isCurrent: boolean;
  hash: string;
}

export interface AheadBehind {
  ahead: number;
  behind: number;
}

export const useBranchesStore = defineStore('branches', () => {
  const local = ref<BranchInfo[]>([]);
  const remote = ref<BranchInfo[]>([]);
  const loading = ref(false);
  const aheadBehind = ref<AheadBehind>({ ahead: 0, behind: 0 });

  let loadedFor: string | null = null;
  const cache = new Map<
    string,
    { local: BranchInfo[]; remote: BranchInfo[]; aheadBehind: AheadBehind }
  >();

  // Loads only ever touch the list for the active repo. Auto-fetch for a repo you already left
  // calls load() seconds later, so a load for an inactive repo is ignored before it starts, and
  // a result is dropped if the repo stopped being active while git ran. Switching back shows
  // the last known list at once and refreshes it in the background.
  async function load(repoPath: string) {
    if (!repoPath || !isActive(repoPath)) {
      return;
    }
    if (loadedFor !== repoPath) {
      const cached = cache.get(repoPath);
      local.value = cached?.local ?? [];
      remote.value = cached?.remote ?? [];
      aheadBehind.value = cached?.aheadBehind ?? { ahead: 0, behind: 0 };
      loadedFor = repoPath;
    }
    loading.value = true;
    try {
      const [all, ab] = await Promise.all([ListBranches(repoPath), GetAheadBehind(repoPath)]);
      if (!isActive(repoPath)) {
        return;
      }
      local.value = all.filter((b) => !b.isRemote);
      remote.value = all.filter((b) => b.isRemote);
      aheadBehind.value = ab;
      cache.set(repoPath, { local: local.value, remote: remote.value, aheadBehind: ab });
    } finally {
      if (isActive(repoPath)) {
        loading.value = false;
      }
    }
  }

  async function silentFetchAndRefresh(repoPath: string) {
    if (!repoPath) {
      return;
    }
    try {
      await FetchAll(repoPath);
    } catch {
      // offline or no remote — ignore, still refresh local state
    }
    await load(repoPath);
  }

  function clear() {
    loadedFor = null;
    local.value = [];
    remote.value = [];
    aheadBehind.value = { ahead: 0, behind: 0 };
  }

  async function switchBranch(repoPath: string, target: string, trackRemote: boolean) {
    return SwitchBranch(repoPath, target, trackRemote);
  }

  return { local, remote, loading, aheadBehind, load, silentFetchAndRefresh, clear, switchBranch };
});

// Without this, editing a store during `wails dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useBranchesStore, import.meta.hot));
}
