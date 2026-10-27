import { acceptHMRUpdate, defineStore } from 'pinia';
import { useToastStore } from './toast';
import { ref, computed } from 'vue';
import {
  OpenRepository,
  CloseRepository,
  ListRecentRepositories,
  PickDirectory,
  LoadSession,
  SaveSession,
  RepositoryGone,
} from '../../bindings/burlwood/app';

export interface RepoTab {
  path: string;
  name: string;
}

// Stores call this after every await: a response for a repo that's no longer active is stale.
export function isActive(repoPath: string): boolean {
  return useReposStore().activeRepo?.path === repoPath;
}

export const useReposStore = defineStore('repos', () => {
  const tabs = ref<RepoTab[]>([]);
  const activeIndex = ref(0);
  const recentRepos = ref<{ path: string; name: string }[]>([]);
  // Starts true so the welcome screen never flashes before the last session is known.
  const restoring = ref(true);
  const restoringNames = ref<string[]>([]);

  const activeRepo = computed(() => tabs.value[activeIndex.value] ?? null);

  async function loadRecents() {
    recentRepos.value = (await ListRecentRepositories()) ?? [];
  }

  function persistSession() {
    SaveSession(
      tabs.value.map((t) => ({ path: t.path, name: t.name })),
      activeIndex.value
    );
  }

  async function restoreSession() {
    try {
      const session = await LoadSession();
      const sessionTabs = session.tabs ?? [];
      restoringNames.value = sessionTabs.map((t) => t.name);
      const results = await Promise.allSettled(sessionTabs.map((t) => OpenRepository(t.path)));
      const gone = new Set<string>();
      for (const [i, result] of results.entries()) {
        const tab = sessionTabs[i];
        if (result.status === 'fulfilled') {
          if (!tabs.value.find((t) => t.path === result.value)) {
            tabs.value.push({ path: result.value, name: tab.name });
          }
        } else if (await RepositoryGone(tab.path)) {
          gone.add(tab.path);
        } else {
          useToastStore().push(
            `Couldn't reopen ${tab.name}`,
            'error',
            undefined,
            undefined,
            String(result.reason)
          );
        }
      }
      const active = results[session.activeIndex];
      const activePath = active?.status === 'fulfilled' ? active.value : null;
      activeIndex.value = Math.max(
        0,
        tabs.value.findIndex((t) => t.path === activePath)
      );
      if (gone.size) {
        // Saving the open tabs would also drop a repo that only failed this time.
        const kept = sessionTabs.filter((t) => !gone.has(t.path));
        const activeTab = sessionTabs[session.activeIndex];
        SaveSession(kept, Math.max(0, kept.indexOf(activeTab)));
        const names = sessionTabs.filter((t) => gone.has(t.path)).map((t) => t.name);
        useToastStore().push(`Closed ${names.join(', ')}: the repository no longer exists`);
      }
    } catch (e) {
      useToastStore().error(e);
    } finally {
      restoring.value = false;
    }
  }

  // Errors are shown here (not thrown), so every "open" entry point reports a bad folder.
  async function openRepo(picked: string): Promise<boolean> {
    let path: string;
    try {
      path = await OpenRepository(picked);
    } catch (e) {
      useToastStore().error(e);
      return false;
    }
    const name = path.split(/[\\/]/).pop() ?? path;
    const existing = tabs.value.findIndex((t) => t.path === path);
    if (existing >= 0) {
      activeIndex.value = existing;
    } else {
      tabs.value.push({ path, name });
      activeIndex.value = tabs.value.length - 1;
    }
    persistSession();
    return true;
  }

  async function pickAndOpen() {
    const path = await PickDirectory();
    if (path) {
      await openRepo(path);
    }
  }

  function closeTab(index: number) {
    const tab = tabs.value[index];
    if (tab) {
      CloseRepository(tab.path);
    }
    tabs.value.splice(index, 1);
    if (activeIndex.value >= tabs.value.length) {
      activeIndex.value = Math.max(0, tabs.value.length - 1);
    }
    persistSession();
  }

  function setActive(index: number) {
    activeIndex.value = index;
    persistSession();
  }

  return {
    tabs,
    activeIndex,
    activeRepo,
    recentRepos,
    restoring,
    restoringNames,
    loadRecents,
    restoreSession,
    openRepo,
    pickAndOpen,
    closeTab,
    setActive,
  };
});

// Without this, editing a store during `wails3 dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useReposStore, import.meta.hot));
}
