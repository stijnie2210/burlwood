<script setup lang="ts">
import { shortcut } from './utils/platform';
import { onMounted, onUnmounted, ref, watch, type Ref, computed, nextTick } from 'vue';
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';
import { useDetailWidth } from './composables/useDetailWidth';
import TabBar from './components/layout/TabBar.vue';
import ToolBar from './components/layout/ToolBar.vue';
import Sidebar from './components/layout/Sidebar.vue';
import CommitGraph from './components/graph/CommitGraph.vue';
import CommitDetail from './components/graph/CommitDetail.vue';
import FileDiffView from './components/graph/FileDiffView.vue';
import StagingView from './components/staging/StagingView.vue';
import ToastStack from './components/layout/ToastStack.vue';
import PreferencesModal from './components/layout/PreferencesModal.vue';
import LogViewerModal from './components/layout/LogViewerModal.vue';
import RepoPicker from './components/layout/RepoPicker.vue';
import { useReposStore } from './stores/repos';
import { useCommitsStore } from './stores/commits';
import { useStagingStore } from './stores/staging';
import { useBranchesStore } from './stores/branches';
import { usePrefsStore } from './stores/prefs';
import { useToastStore } from './stores/toast';
import logoUrl from './assets/images/logo-icon.svg';

const repos = useReposStore();
const commits = useCommitsStore();
const staging = useStagingStore();
const branches = useBranchesStore();
const prefsStore = usePrefsStore();
const toast = useToastStore();
onMounted(async () => {
  await prefsStore.load().catch((e) => toast.error(e));
  await repos.restoreSession();
  repos.loadRecents();
});

// Keep staging badge up-to-date regardless of which tab is active
watch(
  () => repos.activeRepo?.path,
  (path) => {
    if (path) {
      staging.load(path);
    } else {
      staging.clear();
    }
  },
  { immediate: true }
);

let fetchPollTimer: ReturnType<typeof setInterval> | null = null;

function restartFetchTimer(path: string | undefined) {
  if (fetchPollTimer !== null) {
    clearInterval(fetchPollTimer);
    fetchPollTimer = null;
  }
  if (!path || !prefsStore.prefs.autoFetchEnabled) {
    return;
  }
  autoFetch(path);
  fetchPollTimer = setInterval(
    () => autoFetch(path),
    prefsStore.prefs.autoFetchIntervalSecs * 1000
  );
}

async function autoFetch(path: string) {
  await branches.silentFetchAndRefresh(path);
  if (repos.activeRepo?.path === path) {
    await commits.load(path);
  }
}

function onWindowFocusChange(e: FocusEvent) {
  document.documentElement.classList.toggle('window-inactive', e.type === 'blur');
}

// Coming back from the editor is when the working tree most likely changed.
let lastFocusRefresh = 0;
function refreshOnFocus() {
  const path = repoPath.value;
  if (!path || document.visibilityState === 'hidden' || Date.now() - lastFocusRefresh < 1500) {
    return;
  }
  lastFocusRefresh = Date.now();
  staging.refresh(path);
  branches.load(path);
  commits.load(path);
}

watch(
  [
    () => repos.activeRepo?.path,
    () => prefsStore.prefs.autoFetchEnabled,
    () => prefsStore.prefs.autoFetchIntervalSecs,
  ],
  ([path]) => restartFetchTimer(path as string | undefined),
  { immediate: true }
);

onUnmounted(() => {
  if (fetchPollTimer !== null) {
    clearInterval(fetchPollTimer);
  }
});

const viewMode = ref<'commits' | 'staging'>('commits');

// Switching repos always lands on the graph: an open diff or working-tree file belongs to the
// repo you left. App owns this load because the graph isn't mounted while a file diff is open.
watch(
  () => repos.activeRepo?.path,
  (path, previous) => {
    if (previous !== undefined && path !== previous) {
      viewMode.value = 'commits';
    }
    if (path) {
      commits.load(path);
    } else {
      commits.clear();
    }
  },
  { immediate: true }
);

// Generic resize helper
function makeResizer(
  width: Ref<number>,
  min: number,
  max: number,
  direction: 'right' | 'left' = 'right'
) {
  let startX = 0;
  let startW = 0;

  function onMove(e: MouseEvent) {
    const delta = e.clientX - startX;
    width.value = Math.max(min, Math.min(max, startW + (direction === 'right' ? delta : -delta)));
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

const SIDEBAR_KEY = 'burlwood:sidebar-width';
const sidebarWidth = ref(parseInt(localStorage.getItem(SIDEBAR_KEY) ?? '240', 10));
watch(sidebarWidth, (v) => localStorage.setItem(SIDEBAR_KEY, String(v)));
const detailWidth = useDetailWidth();

const startSidebarResize = makeResizer(sidebarWidth, 140, 500, 'right');
const startDetailResize = makeResizer(detailWidth, 200, 800, 'left');

const repoPath = computed(() => repos.activeRepo?.path ?? '');

function runAction(action: string) {
  if (action === 'view-commits' && repos.activeRepo) {
    viewMode.value = 'commits';
  } else if (action === 'view-staging' && repos.activeRepo) {
    viewMode.value = 'staging';
  } else if (action === 'prev-tab' && repos.tabs.length > 1) {
    repos.setActive((repos.activeIndex - 1 + repos.tabs.length) % repos.tabs.length);
  } else if (action === 'next-tab' && repos.tabs.length > 1) {
    repos.setActive((repos.activeIndex + 1) % repos.tabs.length);
  } else if (action === 'close-tab' && repos.tabs.length > 0) {
    repos.closeTab(repos.activeIndex);
  } else if (action === 'fetch') {
    const path = repoPath.value;
    if (path) {
      staging.fetchAll(path).then(() => {
        branches.markFetched(path);
        commits.load(path);
        staging.load(path);
        branches.load(path);
      });
    }
  } else if (action === 'open-repo') {
    repos.pickAndOpen();
  } else if (action === 'preferences') {
    prefsStore.openModal();
  } else if (action === 'find' && repos.activeRepo) {
    viewMode.value = 'commits';
    nextTick(() => window.dispatchEvent(new CustomEvent('burlwood:find')));
  }
}

const SHORTCUTS: Record<string, string> = {
  '1': 'view-commits',
  '2': 'view-staging',
  '[': 'prev-tab',
  ']': 'next-tab',
  w: 'close-tab',
  r: 'fetch',
  o: 'open-repo',
  ',': 'preferences',
};

function onGlobalKeyDown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return;
  }
  if (!e.metaKey && !e.ctrlKey) {
    return;
  }
  // Outside text fields, ⌘Z undoes the last discard or hard reset while its Undo toast is up.
  if (e.key === 'z' && !e.shiftKey && toast.undoLatest()) {
    e.preventDefault();
    return;
  }
  const action = SHORTCUTS[e.key];
  if (action) {
    e.preventDefault();
    runAction(action);
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeyDown);
  window.addEventListener('focus', refreshOnFocus);
  window.addEventListener('focus', onWindowFocusChange);
  window.addEventListener('blur', onWindowFocusChange);
  document.addEventListener('visibilitychange', refreshOnFocus);
  EventsOn('open-preferences', () => prefsStore.openModal());
  EventsOn('menu', runAction);
});
onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeyDown);
  window.removeEventListener('focus', refreshOnFocus);
  window.removeEventListener('focus', onWindowFocusChange);
  window.removeEventListener('blur', onWindowFocusChange);
  document.removeEventListener('visibilitychange', refreshOnFocus);
  EventsOff('open-preferences');
  EventsOff('menu');
});
</script>

<template>
  <div class="app">
    <ToolBar v-model:view-mode="viewMode" />
    <TabBar v-if="repos.tabs.length > 1" />
    <RepoPicker />
    <ToastStack />
    <PreferencesModal />
    <LogViewerModal />
    <div class="workspace">
      <template v-if="repos.activeRepo">
        <Sidebar :style="{ width: sidebarWidth + 'px' }" />
        <div class="resize-handle" @mousedown.prevent="startSidebarResize" />
      </template>
      <main class="main-panel">
        <div v-if="!repos.activeRepo && repos.restoring" class="welcome" aria-busy="true">
          <div v-if="repos.restoringNames.length" class="welcome-inner restoring">
            <img :src="logoUrl" class="welcome-logo" alt="" />
            <p class="welcome-lede" role="status">
              <span class="restoring-spinner" aria-hidden="true" />
              Opening {{ repos.restoringNames.join(', ') }}…
            </p>
          </div>
        </div>

        <div v-else-if="!repos.activeRepo" class="welcome">
          <div class="welcome-inner">
            <img :src="logoUrl" class="welcome-logo" alt="" />
            <h1 class="welcome-wordmark">burlwood</h1>
            <p class="welcome-lede">Open a repository to get started.</p>
            <button class="welcome-open" @click="repos.pickAndOpen()">
              Open Repository…
              <kbd aria-hidden="true">{{ shortcut('O') }}</kbd>
            </button>

            <section
              v-if="repos.recentRepos.length"
              class="recents"
              aria-labelledby="recents-title"
            >
              <h2 id="recents-title">Recent</h2>
              <ul>
                <li v-for="r in repos.recentRepos" :key="r.path">
                  <button class="recent-btn" :title="r.path" @click="repos.openRepo(r.path)">
                    <svg width="14" height="14" viewBox="0 0 24 24" aria-hidden="true">
                      <path
                        d="M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"
                      />
                    </svg>
                    <strong>{{ r.name }}</strong>
                    <span>{{ r.path }}</span>
                  </button>
                </li>
              </ul>
            </section>
          </div>
        </div>

        <template v-else>
          <!-- Commits view -->
          <div v-if="viewMode === 'commits'" class="repo-view">
            <template v-if="commits.selectedFileIndex !== null">
              <FileDiffView style="flex: 1" />
            </template>
            <template v-else>
              <CommitGraph style="flex: 1; min-width: 300px" />
              <template v-if="commits.selectedHash">
                <div class="resize-handle" @mousedown.prevent="startDetailResize" />
                <CommitDetail :style="{ width: detailWidth + 'px' }" />
              </template>
            </template>
          </div>

          <!-- Staging view -->
          <StagingView v-else />
        </template>
      </main>
    </div>
  </div>
</template>

<style>
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}
body {
  background: transparent;
  color: var(--text-primary);
}
</style>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.workspace {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.main-panel {
  flex: 1;
  overflow: hidden;
  background: var(--surface-base);
  display: flex;
  flex-direction: column;
}

.repo-view {
  flex: 1;
  display: flex;
  overflow: hidden;
}

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

@font-face {
  font-family: 'Burlwood Wordmark';
  src: url('./assets/fonts/bricolage-wordmark.woff') format('woff');
  font-weight: 600;
  font-display: block;
}

.welcome {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-6);
  overflow-y: auto;
}

.welcome-inner {
  width: min(440px, 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  animation: welcome-rise 520ms cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes welcome-rise {
  from {
    opacity: 0.4;
    transform: translateY(8px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .welcome-inner {
    animation: none;
  }
}

.restoring .welcome-lede {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.restoring-spinner {
  width: 14px;
  height: 14px;
  flex: none;
  border: 2px solid var(--text-tertiary);
  border-top-color: transparent;
  border-radius: 50%;
  animation: restoring-spin 0.8s linear infinite;
}

@keyframes restoring-spin {
  to {
    transform: rotate(360deg);
  }
}

.welcome-logo {
  width: 128px;
  height: 128px;
}

/* The wordmark is the only place the brand typeface appears. */
.welcome-wordmark {
  margin-top: var(--space-1);
  font-family: 'Burlwood Wordmark', var(--font-ui);
  font-size: 44px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.02em;
  color: var(--brand-sapwood);
}

.welcome-lede {
  margin-top: var(--space-3);
  font-size: var(--text-title);
  color: var(--text-secondary);
}

.welcome-open {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  height: 32px;
  margin-top: var(--space-5);
  padding: 0 var(--space-4);
  border: none;
  border-radius: 6px;
  background: var(--accent-fill);
  color: var(--text-on-accent);
  font-size: var(--text-body);
  font-weight: 500;
  cursor: default;
}
.welcome-open:hover {
  background: var(--accent-fill-hover);
}
.welcome-open kbd {
  font-family: var(--font-ui);
  font-size: var(--text-meta);
  opacity: 0.8;
}

.recents {
  width: 100%;
  margin-top: 40px;
  text-align: left;
}
.recents h2 {
  padding: 0 10px var(--space-2);
  font-size: var(--text-meta);
  font-weight: 600;
  color: var(--text-tertiary);
}
.recents ul {
  list-style: none;
  border-top: 1px solid var(--border-subtle);
  padding-top: var(--space-1);
}

.recent-btn {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr);
  column-gap: 10px;
  row-gap: 2px;
  align-items: center;
  width: 100%;
  padding: var(--space-2) 10px;
  border: none;
  border-radius: 6px;
  background: none;
  text-align: left;
  font: inherit;
  cursor: default;
}
.recent-btn:hover {
  background: var(--surface-hover);
}
.recent-btn svg {
  fill: none;
  stroke: var(--text-tertiary);
  stroke-width: 2;
  stroke-linejoin: round;
}
.recent-btn strong {
  font-size: var(--text-body);
  font-weight: 500;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.recent-btn span {
  grid-column: 2;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
