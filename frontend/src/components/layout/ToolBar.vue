<script setup lang="ts">
import { shortcut } from '../../utils/platform';
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue';
import { useReposStore } from '../../stores/repos';
import { useCommitsStore } from '../../stores/commits';
import { useStagingStore } from '../../stores/staging';
import { useBranchesStore } from '../../stores/branches';
import { useToastStore } from '../../stores/toast';
import { useStashStore } from '../../stores/stash';
import { Stash, StashPop, CreateBranch, OpenTerminal } from '../../../wailsjs/go/main/App';
import { usePrefsStore } from '../../stores/prefs';
import { pushWithFeedback } from '../../composables/usePush';
import { useRepoPickerStore } from '../../stores/repoPicker';

const viewMode = defineModel<'commits' | 'staging'>('viewMode', { required: true });
const picker = useRepoPickerStore();

const repos = useReposStore();
const prefsStore = usePrefsStore();
const commits = useCommitsStore();
const staging = useStagingStore();
const branches = useBranchesStore();
const toast = useToastStore();
const stash = useStashStore();

const repoPath = computed(() => repos.activeRepo?.path ?? '');
const currentBranch = computed(() => branches.local.find((b) => b.isCurrent)?.name ?? null);
const disabled = computed(() => !repos.activeRepo);
const pullDisabled = computed(() => disabled.value || staging.isInMerge || staging.isInRebase);

const busy = ref<string | null>(null);

async function run(op: string, fn: () => Promise<void>, successMsg?: string) {
  if (busy.value) {
    return;
  }
  busy.value = op;
  try {
    await fn();
    if (successMsg) {
      toast.success(successMsg);
    }
  } catch (e: unknown) {
    toast.error(e);
  } finally {
    busy.value = null;
  }
}

async function onFetch() {
  await run('fetch', async () => {
    await staging.fetchAll(repoPath.value);
    branches.markFetched(repoPath.value);
    await commits.load(repoPath.value);
    branches.load(repoPath.value);
  });
}

const PULL_MODES: { value: string; label: string }[] = [
  { value: 'fetch', label: 'Fetch All' },
  { value: 'ff', label: 'Pull (fast-forward if possible)' },
  { value: 'ff-only', label: 'Pull (fast-forward only)' },
  { value: 'rebase', label: 'Pull (rebase)' },
];

const pullMode = computed(() => prefsStore.prefs.pullMode || 'ff');
const pullTitle = computed(
  () => PULL_MODES.find((m) => m.value === pullMode.value)?.label ?? 'Pull'
);
const showPullMenu = ref(false);
const pullMenuEl = ref<HTMLElement | null>(null);

function togglePullMenu() {
  showPullMenu.value = !showPullMenu.value;
}

async function selectPullMode(mode: string) {
  showPullMenu.value = false;
  if (mode === pullMode.value) {
    return;
  }
  await prefsStore.save({ pullMode: mode });
}

function onPullMenuMouseDown(e: MouseEvent) {
  if (pullMenuEl.value && !pullMenuEl.value.contains(e.target as Node)) {
    showPullMenu.value = false;
  }
}

const searchInputEl = ref<HTMLInputElement | null>(null);

function focusSearch() {
  viewMode.value = 'commits';
  nextTick(() => searchInputEl.value?.select());
}

function onSearchInput() {
  viewMode.value = 'commits';
}

function onSearchEscape() {
  commits.searchQuery = '';
  searchInputEl.value?.blur();
}

onMounted(() => {
  window.addEventListener('mousedown', onPullMenuMouseDown);
  window.addEventListener('burlwood:find', focusSearch);
});
onUnmounted(() => {
  window.removeEventListener('mousedown', onPullMenuMouseDown);
  window.removeEventListener('burlwood:find', focusSearch);
});

async function onPull() {
  const mode = pullMode.value;
  await run(
    'pull',
    async () => {
      if (mode === 'fetch') {
        await staging.fetchAll(repoPath.value);
      } else {
        await staging.pullBranch(repoPath.value, mode);
      }
      branches.markFetched(repoPath.value);
      await commits.load(repoPath.value);
      await staging.load(repoPath.value);
      await branches.load(repoPath.value);
    },
    mode === 'fetch' ? undefined : 'Pulled successfully'
  );
}

async function onPush() {
  await run('push', () => pushWithFeedback(repoPath.value));
}

async function onStash() {
  await run(
    'stash',
    async () => {
      await Stash(repoPath.value);
      await Promise.all([
        staging.load(repoPath.value),
        stash.load(repoPath.value),
        commits.load(repoPath.value),
      ]);
    },
    'Changes stashed'
  );
}

async function onPop() {
  await run(
    'pop',
    async () => {
      await StashPop(repoPath.value);
      await Promise.all([
        staging.load(repoPath.value),
        stash.load(repoPath.value),
        commits.load(repoPath.value),
      ]);
    },
    'Stash applied'
  );
}

async function onTerminal() {
  await run('terminal', () => OpenTerminal(repoPath.value));
}

// Branch creation popover
const showBranchInput = ref(false);
const branchName = ref('');
const branchInputEl = ref<HTMLInputElement | null>(null);

async function openBranchInput() {
  if (disabled.value) {
    return;
  }
  showBranchInput.value = true;
  await nextTick();
  branchInputEl.value?.focus();
}

function cancelBranch() {
  showBranchInput.value = false;
  branchName.value = '';
}

async function confirmBranch() {
  const name = branchName.value.trim();
  if (!name) {
    return;
  }
  const captured = name;
  await run(
    'branch',
    async () => {
      await CreateBranch(repoPath.value, captured);
      branches.load(repoPath.value);
      commits.load(repoPath.value);
    },
    `Branch '${name}' created`
  );
  showBranchInput.value = false;
  branchName.value = '';
}
</script>

<template>
  <div class="toolbar">
    <div class="toolbar-inner" :class="{ 'no-repo': !repos.activeRepo }">
      <div class="toolbar-side toolbar-side--left">
        <button
          class="repo-btn"
          data-repo-picker-anchor
          :title="repos.activeRepo?.path ?? 'Open a repository'"
          aria-haspopup="menu"
          :aria-expanded="picker.open"
          @click="picker.toggle($event.currentTarget as HTMLElement)"
        >
          <span class="repo-btn-name">{{ repos.activeRepo?.name ?? 'Open Repository' }}</span>
          <svg width="8" height="5" viewBox="0 0 8 5" aria-hidden="true">
            <path d="M0.5 0.5L4 4L7.5 0.5" fill="none" stroke="currentColor" stroke-width="1.3" />
          </svg>
        </button>
        <div
          v-if="repos.activeRepo && branches.local.length"
          class="branch-chip"
          role="status"
          :title="currentBranch ? `On ${currentBranch}` : 'HEAD is detached'"
        >
          <svg
            width="13"
            height="13"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M6 3v12" />
            <circle cx="18" cy="6" r="3" />
            <circle cx="6" cy="18" r="3" />
            <path d="M18 9a9 9 0 0 1-9 9" />
          </svg>
          <span class="branch-chip-name">{{ currentBranch ?? 'detached HEAD' }}</span>
          <span
            v-if="branches.aheadBehind.ahead > 0"
            class="branch-chip-sync"
            :aria-label="`${branches.aheadBehind.ahead} to push`"
            >↑{{ branches.aheadBehind.ahead }}</span
          >
          <span
            v-if="branches.aheadBehind.behind > 0"
            class="branch-chip-sync"
            :aria-label="`${branches.aheadBehind.behind} to pull`"
            >↓{{ branches.aheadBehind.behind }}</span
          >
        </div>
        <div v-if="repos.activeRepo" class="view-switch" role="tablist" aria-label="View">
          <button
            class="view-switch-btn"
            role="tab"
            :class="{ active: viewMode === 'commits' }"
            :aria-selected="viewMode === 'commits'"
            :title="`Commits (${shortcut('1')})`"
            @click="viewMode = 'commits'"
          >
            Commits
          </button>
          <button
            class="view-switch-btn"
            role="tab"
            :class="{ active: viewMode === 'staging' }"
            :aria-selected="viewMode === 'staging'"
            :title="`Working Tree (${shortcut('2')})`"
            @click="viewMode = 'staging'"
          >
            Working Tree
            <span v-if="staging.totalChanges" class="view-switch-badge">
              {{ staging.totalChanges }}
            </span>
          </button>
        </div>
      </div>
      <!-- Remote ops group -->
      <div class="btn-group">
        <button
          class="tbtn"
          :title="`Fetch (${shortcut('R')})`"
          aria-label="Fetch"
          :disabled="disabled"
          :class="{ loading: busy === 'fetch' }"
          @click="onFetch"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="4" y1="3" x2="14" y2="3" />
              <line x1="9" y1="5" x2="9" y2="13" />
              <polyline points="5,10 9,14 13,10" />
            </svg>
          </span>
        </button>
        <div ref="pullMenuEl" class="pull-btn-wrap">
          <button
            class="tbtn"
            :title="pullTitle"
            :aria-label="pullMode === 'fetch' ? 'Fetch' : 'Pull'"
            :disabled="pullDisabled"
            :class="{ loading: busy === 'pull' }"
            @click="onPull"
          >
            <span class="tbtn-icon">
              <svg
                v-if="pullMode === 'fetch'"
                width="18"
                height="18"
                viewBox="0 0 18 18"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <line x1="4" y1="3" x2="14" y2="3" />
                <line x1="9" y1="5" x2="9" y2="13" />
                <polyline points="5,10 9,14 13,10" />
              </svg>
              <svg
                v-else
                width="18"
                height="18"
                viewBox="0 0 18 18"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <line x1="9" y1="3" x2="9" y2="13" />
                <polyline points="5,9 9,14 13,9" />
              </svg>
            </span>
          </button>
          <button
            class="pull-mode-toggle"
            :disabled="pullDisabled"
            :class="{ active: showPullMenu }"
            title="Choose default pull/fetch operation"
            @click.stop="togglePullMenu"
          >
            <svg width="8" height="8" viewBox="0 0 8 8" fill="currentColor">
              <path d="M0 1 L8 1 L4 6 Z" />
            </svg>
          </button>

          <div v-if="showPullMenu" class="pull-menu">
            <p class="pull-menu-desc">Default action for the Pull button</p>
            <button
              v-for="opt in PULL_MODES"
              :key="opt.value"
              class="pull-menu-item"
              :class="{ selected: opt.value === pullMode }"
              @click="selectPullMode(opt.value)"
            >
              <span class="pull-menu-radio" />
              <span>{{ opt.label }}</span>
            </button>
          </div>
        </div>
        <button
          class="tbtn"
          title="Push"
          aria-label="Push"
          :disabled="disabled"
          :class="{ loading: busy === 'push' }"
          @click="onPush"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="9" y1="15" x2="9" y2="5" />
              <polyline points="5,9 9,4 13,9" />
            </svg>
          </span>
        </button>
      </div>

      <!-- Branch group -->
      <div class="btn-group branch-group">
        <button
          class="tbtn"
          title="New branch"
          aria-label="New branch"
          :disabled="disabled"
          :class="{ loading: busy === 'branch', active: showBranchInput }"
          @click="openBranchInput"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            >
              <circle cx="5" cy="4" r="1.8" />
              <circle cx="5" cy="14" r="1.8" />
              <circle cx="13" cy="4" r="1.8" />
              <line x1="5" y1="6" x2="5" y2="12" />
              <path d="M13 6 C13 10 5 10 5 12" />
            </svg>
          </span>
        </button>

        <!-- Branch name popover -->
        <div v-if="showBranchInput" class="branch-popover">
          <input
            ref="branchInputEl"
            v-model="branchName"
            class="branch-input"
            placeholder="branch-name"
            @keydown.enter="confirmBranch"
            @keydown.escape="cancelBranch"
          />
          <button class="branch-confirm" @click="confirmBranch">Create</button>
          <button class="branch-cancel" aria-label="Cancel" @click="cancelBranch">✕</button>
        </div>
      </div>

      <!-- Stash group -->
      <div class="btn-group">
        <button
          class="tbtn"
          title="Stash changes"
          aria-label="Stash changes"
          :disabled="disabled"
          :class="{ loading: busy === 'stash' }"
          @click="onStash"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <rect x="3" y="11" width="12" height="4" rx="1.5" />
              <line x1="9" y1="3" x2="9" y2="10" />
              <polyline points="6,7 9,10 12,7" />
            </svg>
          </span>
        </button>
        <button
          class="tbtn"
          title="Pop latest stash"
          aria-label="Pop latest stash"
          :disabled="disabled"
          :class="{ loading: busy === 'pop' }"
          @click="onPop"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <rect x="3" y="11" width="12" height="4" rx="1.5" />
              <line x1="9" y1="9" x2="9" y2="2" />
              <polyline points="6,5 9,2 12,5" />
            </svg>
          </span>
        </button>
      </div>

      <!-- Terminal -->
      <div class="btn-group">
        <button
          class="tbtn"
          title="Open in Terminal"
          aria-label="Open in Terminal"
          :disabled="disabled"
          @click="onTerminal"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <polyline points="3,6 8,9 3,12" />
              <line x1="10" y1="12" x2="15" y2="12" />
            </svg>
          </span>
        </button>
      </div>

      <div class="toolbar-side toolbar-side--right">
        <label v-if="repos.activeRepo" class="find-field">
          <svg
            width="13"
            height="13"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            aria-hidden="true"
          >
            <circle cx="11" cy="11" r="7" />
            <path d="M21 21l-4.5-4.5" />
          </svg>
          <input
            ref="searchInputEl"
            v-model="commits.searchQuery"
            type="search"
            placeholder="Find commit"
            aria-label="Find commit by message, author or hash"
            @input="onSearchInput"
            @keydown.esc.prevent="onSearchEscape"
          />
          <kbd aria-hidden="true">{{ shortcut('F') }}</kbd>
        </label>
        <button
          class="icon-btn"
          :title="`Settings (${shortcut(',')})`"
          aria-label="Settings"
          @click="prefsStore.openModal()"
        >
          <span class="tbtn-icon">
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path
                d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"
              />
              <circle cx="12" cy="12" r="3" />
            </svg>
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  user-select: none;
  -webkit-user-select: none;
  flex-shrink: 0;
  background: var(--surface-chrome);
  border-bottom: 1px solid var(--border-subtle);
  /* This row is the window's title bar (hidden-inset), so empty space drags the window. */
  --wails-draggable: drag;
}

.toolbar button,
.toolbar input,
.pull-menu,
.branch-popover {
  --wails-draggable: no-drag;
}

.toolbar-inner {
  display: flex;
  align-items: center;
  height: var(--toolbar-h);
  padding: 0 var(--inset-panel) 0 var(--traffic-light-inset);
  gap: 2px;
}

/* Without a repo there is nothing to act on; show only the repo button and Settings. */
.toolbar-inner.no-repo > .btn-group {
  display: none;
}

.toolbar-side {
  flex: 1;
  min-width: 0;
}

.toolbar-side--left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.repo-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  min-width: 0;
  max-width: 200px;
  height: 28px;
  padding: 0 var(--space-2);
  background: none;
  border: none;
  border-radius: 5px;
  color: var(--text-primary);
  font-size: var(--text-body);
  font-weight: 600;
  cursor: default;
}

.repo-btn:hover {
  background: var(--surface-hover);
}

.repo-btn-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.repo-btn svg {
  flex-shrink: 0;
  color: var(--text-tertiary);
}

/* macOS-style segmented control */
.view-switch {
  display: flex;
  flex-shrink: 0;
  padding: 2px;
  gap: 2px;
  background: var(--surface-sunken);
  border: 1px solid var(--border-subtle);
  border-radius: 7px;
}

.view-switch-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 var(--space-3);
  background: none;
  border: none;
  border-radius: 5px;
  color: var(--text-secondary);
  font-size: var(--text-row);
  font-weight: 500;
  cursor: default;
  white-space: nowrap;
}

.view-switch-btn:hover:not(.active) {
  color: var(--text-primary);
}

.view-switch-btn.active {
  background: var(--surface-control-hover);
  color: var(--text-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.view-switch-badge {
  min-width: 16px;
  padding: 0 5px;
  border-radius: 8px;
  background: var(--accent-fill);
  color: var(--text-on-accent);
  font-size: var(--text-meta);
  font-weight: 600;
  line-height: 16px;
  text-align: center;
}

.toolbar-side--right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
}

.btn-group {
  display: flex;
  align-items: center;
  position: relative;
  flex-shrink: 0;
  gap: 2px;
  padding: 2px;
  margin: 0 3px;
  border-radius: 8px;
  background: var(--surface-raised);
}

.branch-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  max-width: 240px;
  height: 26px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 13px;
  background: var(--surface-raised);
  color: var(--text-tertiary);
}
.branch-chip svg {
  flex-shrink: 0;
}
.branch-chip-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: var(--text-row);
  color: var(--text-primary);
}
.branch-chip-sync {
  flex-shrink: 0;
  font-size: var(--text-meta);
  font-variant-numeric: tabular-nums;
  color: var(--accent-text);
}

.find-field {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 0 1 200px;
  min-width: 120px;
  height: 28px;
  padding: 0 var(--space-2);
  border: 1px solid var(--border-subtle);
  border-radius: 7px;
  background: var(--surface-field);
  color: var(--text-tertiary);
  --wails-draggable: no-drag;
}
.find-field:focus-within {
  border-color: var(--focus-ring);
}
.find-field svg {
  flex-shrink: 0;
}
.find-field input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-row);
  outline: none;
}
.find-field input::placeholder {
  color: var(--text-tertiary);
}
.find-field input::-webkit-search-cancel-button {
  display: none;
}
.find-field kbd {
  flex-shrink: 0;
  font-family: var(--font-ui);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}
.find-field:focus-within kbd {
  display: none;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  cursor: default;
}
.icon-btn:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.tbtn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 28px;
  padding: 0;
  border: none;
  background: transparent;
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: default;
  transition:
    background 0.1s,
    color 0.1s;
}

.tbtn:hover:not(:disabled) {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.tbtn.active {
  background: var(--surface-control);
  color: var(--accent-text);
}

.tbtn:disabled {
  opacity: 0.35;
  cursor: default;
}

.tbtn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

/* Spin animation for loading state */
.tbtn.loading .tbtn-icon {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Pull mode dropdown */
.pull-btn-wrap {
  position: relative;
  display: flex;
  align-items: stretch;
}

.pull-mode-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  align-self: stretch;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-tertiary);
  cursor: default;
  border-radius: 4px;
  margin-left: 1px;
}

.pull-mode-toggle:hover:not(:disabled),
.pull-mode-toggle.active {
  color: var(--text-primary);
  background: var(--surface-control);
}

.pull-mode-toggle:disabled {
  opacity: 0.35;
  cursor: default;
}

.pull-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  width: 320px;
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  z-index: 100;
  box-shadow: var(--shadow-popover);
}

.pull-menu-desc {
  margin: 0 0 10px;
  font-size: var(--text-row);
  line-height: 1.4;
  color: var(--text-secondary);
}

.pull-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-row);
  text-align: left;
  cursor: default;
}

.pull-menu-item:hover {
  background: var(--surface-hover);
}

.pull-menu-item.selected {
  background: var(--surface-control);
  color: var(--text-primary);
}

.pull-menu-radio {
  flex-shrink: 0;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1.5px solid var(--border-strong);
  box-sizing: border-box;
  position: relative;
}

.pull-menu-item.selected .pull-menu-radio {
  border-color: var(--accent-text);
}

.pull-menu-item.selected .pull-menu-radio::after {
  content: '';
  position: absolute;
  inset: 2px;
  border-radius: 50%;
  background: var(--accent-text);
}

/* Branch popover */
.branch-popover {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 6px 8px;
  z-index: 100;
  box-shadow: var(--shadow-popover);
  min-width: 220px;
}

.branch-input {
  flex: 1;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: var(--text-row);
  padding: 4px 8px;
  outline: none;
  font-family: var(--font-mono);
}
.branch-input:focus {
  border-color: var(--focus-ring);
}
.branch-input::placeholder {
  color: var(--text-tertiary);
}

.branch-confirm {
  padding: 4px 10px;
  background: var(--accent-fill);
  border: none;
  border-radius: 4px;
  color: var(--text-on-accent);
  font-size: var(--text-meta);
  cursor: default;
  white-space: nowrap;
}
.branch-confirm:hover {
  background: var(--accent-fill-hover);
}

.branch-cancel {
  padding: 4px 6px;
  background: transparent;
  border: 1px solid var(--border-strong);
  border-radius: 4px;
  color: var(--text-tertiary);
  font-size: var(--text-meta);
  cursor: default;
}
.branch-cancel:hover {
  color: var(--text-primary);
  border-color: var(--border-strong);
}
</style>
