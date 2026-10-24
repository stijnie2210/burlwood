<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue';
import { useReposStore } from '../../stores/repos';
import { useCommitsStore } from '../../stores/commits';
import { useStagingStore } from '../../stores/staging';
import { useBranchesStore } from '../../stores/branches';
import { useToastStore } from '../../stores/toast';
import { useStashStore } from '../../stores/stash';
import {
  PushBranch,
  Stash,
  StashPop,
  CreateBranch,
  OpenTerminal,
} from '../../../wailsjs/go/main/App';
import { usePrefsStore } from '../../stores/prefs';

const repos = useReposStore();
const prefsStore = usePrefsStore();
const commits = useCommitsStore();
const staging = useStagingStore();
const branches = useBranchesStore();
const toast = useToastStore();
const stash = useStashStore();

const repoPath = computed(() => repos.activeRepo?.path ?? '');
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
    toast.error(String(e));
  } finally {
    busy.value = null;
  }
}

async function onFetch() {
  await run(
    'fetch',
    async () => {
      await staging.fetchAll(repoPath.value);
      await commits.load(repoPath.value);
      branches.load(repoPath.value);
    },
    'Fetched successfully'
  );
}

const PULL_MODES: { value: string; label: string }[] = [
  { value: 'fetch', label: 'Fetch All' },
  { value: 'ff', label: 'Pull (fast-forward if possible)' },
  { value: 'ff-only', label: 'Pull (fast-forward only)' },
  { value: 'rebase', label: 'Pull (rebase)' },
];

const pullMode = computed(() => prefsStore.prefs.pullMode || 'ff');
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

onMounted(() => window.addEventListener('mousedown', onPullMenuMouseDown));
onUnmounted(() => window.removeEventListener('mousedown', onPullMenuMouseDown));

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
      await commits.load(repoPath.value);
      await staging.load(repoPath.value);
      await branches.load(repoPath.value);
    },
    mode === 'fetch' ? 'Fetched successfully' : 'Pulled successfully'
  );
}

async function onPush() {
  await run(
    'push',
    async () => {
      await PushBranch(repoPath.value);
      branches.load(repoPath.value);
    },
    'Pushed successfully'
  );
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
    <div class="toolbar-inner">
      <div class="toolbar-side" />
      <!-- Remote ops group -->
      <div class="btn-group">
        <button
          class="tbtn"
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
          <span class="tbtn-label">Fetch</span>
        </button>
        <div ref="pullMenuEl" class="pull-btn-wrap">
          <button
            class="tbtn"
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
            <span class="tbtn-label">{{ pullMode === 'fetch' ? 'Fetch' : 'Pull' }}</span>
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
            <p class="pull-menu-desc">
              Select a default pull/fetch operation to execute when clicking this button
            </p>
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
          <span class="tbtn-label">Push</span>
        </button>
      </div>

      <div class="separator" />

      <!-- Branch group -->
      <div class="btn-group branch-group">
        <button
          class="tbtn"
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
          <span class="tbtn-label">Branch</span>
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
          <button class="branch-cancel" @click="cancelBranch">✕</button>
        </div>
      </div>

      <div class="separator" />

      <!-- Stash group -->
      <div class="btn-group">
        <button
          class="tbtn"
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
          <span class="tbtn-label">Stash</span>
        </button>
        <button
          class="tbtn"
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
          <span class="tbtn-label">Pop</span>
        </button>
      </div>

      <div class="separator" />

      <!-- Terminal -->
      <div class="btn-group">
        <button class="tbtn" :disabled="disabled" @click="onTerminal">
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
          <span class="tbtn-label">Terminal</span>
        </button>
      </div>

      <div class="toolbar-side toolbar-side--right">
        <div class="btn-group">
          <button class="tbtn" @click="prefsStore.openModal()">
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
            <span class="tbtn-label">Settings</span>
          </button>
        </div>
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
  user-select: none;
}

.toolbar-inner {
  display: flex;
  align-items: center;
  height: 48px;
  padding: 0 8px;
  gap: 2px;
}

.toolbar-side {
  flex: 1;
}

.toolbar-side--right {
  display: flex;
  justify-content: flex-end;
}

.btn-group {
  display: flex;
  align-items: center;
  position: relative;
}

.separator {
  width: 1px;
  height: 24px;
  background: var(--border);
  margin: 0 6px;
  flex-shrink: 0;
}

.tbtn {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  padding: 5px 10px;
  border: none;
  background: transparent;
  border-radius: 5px;
  color: var(--text-secondary);
  cursor: pointer;
  min-width: 52px;
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

.tbtn-label {
  font-size: var(--text-meta);
  font-weight: 500;
  letter-spacing: 0.02em;
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
  cursor: pointer;
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
  cursor: pointer;
}

.pull-menu-item:hover {
  background: var(--surface-hover);
}

.pull-menu-item.selected {
  background: var(--accent-fill);
  color: var(--text-on-accent);
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
  border-color: var(--text-on-accent);
}

.pull-menu-item.selected .pull-menu-radio::after {
  content: '';
  position: absolute;
  inset: 2px;
  border-radius: 50%;
  background: var(--text-on-accent);
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
  cursor: pointer;
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
  cursor: pointer;
}
.branch-cancel:hover {
  color: var(--text-primary);
  border-color: var(--border-strong);
}
</style>
