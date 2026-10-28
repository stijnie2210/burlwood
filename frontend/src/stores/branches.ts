import { acceptHMRUpdate, defineStore } from 'pinia';
import { isActive } from './repos';
import { ref } from 'vue';
import {
  ListBranches,
  ListTags,
  ListWorktrees,
  GetAheadBehind,
  SwitchBranch,
  FetchAll,
} from '../../bindings/burlwood/app';

export interface BranchInfo {
  name: string;
  isRemote: boolean;
  remote: string;
  isCurrent: boolean;
  hash: string;
  worktree: string;
}

export interface TagInfo {
  name: string;
  hash: string;
}

export interface WorktreeInfo {
  path: string;
  head: string;
  branch: string;
  isMain: boolean;
  isCurrent: boolean;
  bare: boolean;
  locked: boolean;
  lockReason: string;
  prunable: boolean;
}

export interface AheadBehind {
  ahead: number;
  behind: number;
}

export const useBranchesStore = defineStore('branches', () => {
  const local = ref<BranchInfo[]>([]);
  const remote = ref<BranchInfo[]>([]);
  const tags = ref<TagInfo[]>([]);
  const worktrees = ref<WorktreeInfo[]>([]);
  const loading = ref(false);
  const aheadBehind = ref<AheadBehind>({ ahead: 0, behind: 0 });

  let loadedFor: string | null = null;
  const cache = new Map<
    string,
    {
      local: BranchInfo[];
      remote: BranchInfo[];
      tags: TagInfo[];
      worktrees: WorktreeInfo[];
      aheadBehind: AheadBehind;
    }
  >();
  const fetchedAt = new Map<string, number>();
  const lastFetchedAt = ref<number | null>(null);

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
      tags.value = cached?.tags ?? [];
      worktrees.value = cached?.worktrees ?? [];
      lastFetchedAt.value = fetchedAt.get(repoPath) ?? null;
      aheadBehind.value = cached?.aheadBehind ?? { ahead: 0, behind: 0 };
      loadedFor = repoPath;
    }
    loading.value = true;
    try {
      const [branchList, tagList, worktreeList, ab] = await Promise.all([
        ListBranches(repoPath),
        ListTags(repoPath),
        ListWorktrees(repoPath),
        GetAheadBehind(repoPath),
      ]);
      if (!isActive(repoPath)) {
        return;
      }
      const all = branchList ?? [];
      local.value = all.filter((b) => !b.isRemote);
      remote.value = all.filter((b) => b.isRemote);
      tags.value = tagList ?? [];
      worktrees.value = worktreeList ?? [];
      aheadBehind.value = ab;
      cache.set(repoPath, {
        local: local.value,
        remote: remote.value,
        tags: tags.value,
        worktrees: worktrees.value,
        aheadBehind: ab,
      });
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
      markFetched(repoPath);
    } catch {
      // offline or no remote — ignore, still refresh local state
    }
    await load(repoPath);
  }

  function markFetched(repoPath: string) {
    const now = Date.now();
    fetchedAt.set(repoPath, now);
    if (isActive(repoPath)) {
      lastFetchedAt.value = now;
    }
  }

  function clear() {
    loadedFor = null;
    local.value = [];
    remote.value = [];
    tags.value = [];
    worktrees.value = [];
    lastFetchedAt.value = null;
    aheadBehind.value = { ahead: 0, behind: 0 };
  }

  async function switchBranch(repoPath: string, target: string, trackRemote: boolean) {
    return SwitchBranch(repoPath, target, trackRemote);
  }

  return {
    local,
    remote,
    tags,
    worktrees,
    loading,
    aheadBehind,
    lastFetchedAt,
    load,
    silentFetchAndRefresh,
    markFetched,
    clear,
    switchBranch,
  };
});

// Without this, editing a store during `wails3 dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useBranchesStore, import.meta.hot));
}
