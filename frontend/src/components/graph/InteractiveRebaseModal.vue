<script setup lang="ts">
import { nextTick, ref } from 'vue';
import { useDialog } from '../../composables/useDialog';
import { useInteractiveRebaseStore } from '../../stores/interactiveRebase';
import { useBranchesStore } from '../../stores/branches';
import { useCommitsStore } from '../../stores/commits';
import { useReposStore } from '../../stores/repos';
import { useStagingStore } from '../../stores/staging';
import { useToastStore } from '../../stores/toast';
import { shortcut } from '../../utils/platform';
import { squashTarget, type RebaseAction } from '../../utils/rebasePlan';
import { UndoInteractiveRebase } from '../../../bindings/burlwood/app';
import { InteractiveRebaseOutcome } from '../../../bindings/burlwood/internal/repo/models';

const rebase = useInteractiveRebaseStore();
const toast = useToastStore();

const ACTIONS: { value: RebaseAction; label: string }[] = [
  { value: 'pick', label: 'Pick' },
  { value: 'reword', label: 'Reword' },
  { value: 'squash', label: 'Squash' },
  { value: 'fixup', label: 'Fixup' },
  { value: 'drop', label: 'Drop' },
];

const panelRef = ref<HTMLElement | null>(null);
const { onKeydown } = useDialog(
  () => rebase.open,
  panelRef,
  () => rebase.close()
);

function onOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) {
    rebase.close();
  }
}

async function move(index: number, by: 1 | -1) {
  const hash = rebase.rows[index].hash;
  rebase.move(index, by);
  await nextTick();
  const row = panelRef.value?.querySelector<HTMLElement>(`[data-hash="${hash}"]`);
  const button = row?.querySelector<HTMLButtonElement>(
    by === -1 ? '.move-up:not(:disabled)' : '.move-down:not(:disabled)'
  );
  (button ?? row?.querySelector<HTMLElement>('button:not(:disabled)'))?.focus();
}

function note(index: number): string {
  const row = rebase.rows[index];
  if (row.action === 'drop') {
    return 'Removed from the branch';
  }
  if (row.action === 'squash' || row.action === 'fixup') {
    const target = squashTarget(rebase.rows, index);
    if (!target) {
      return 'Nothing above to combine with';
    }
    return row.action === 'squash'
      ? `Combined into ${target.shortHash}, keeping both messages`
      : `Combined into ${target.shortHash}, dropping this message`;
  }
  return '';
}

async function reloadRepo(repo: string) {
  if (useReposStore().activeRepo?.path !== repo) {
    return;
  }
  await Promise.all([
    useCommitsStore().load(repo),
    useBranchesStore().load(repo),
    useStagingStore().load(repo),
  ]);
}

// Only the latest rebase can be undone, so an earlier Undo toast is taken down.
let undoToast: number | null = null;

async function start() {
  if (rebase.problem || !rebase.changed || rebase.starting) {
    return;
  }
  const repo = rebase.repoPath;
  const branch = rebase.branch;
  try {
    const result = await rebase.start();
    rebase.close();
    await reloadRepo(repo);
    if (result.outcome === InteractiveRebaseOutcome.InteractiveRebaseStopped) {
      toast.workingTreeHint(
        `The rebase of ${branch} stopped on a conflict. Resolve it in the Working Tree (${shortcut('2')}), then continue.`
      );
      return;
    }
    if (undoToast !== null) {
      toast.dismiss(undoToast);
    }
    undoToast = toast.withUndo(
      `Rebased ${branch}`,
      async () => {
        try {
          await UndoInteractiveRebase(repo, result);
          toast.success(`Rebase undone, ${branch} is back at ${result.previousHead.slice(0, 7)}`);
        } catch (e) {
          toast.error(e);
        }
        await reloadRepo(repo);
      },
      true
    );
  } catch (e) {
    toast.error(e);
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="rebase.open" class="ir-overlay" @click="onOverlayClick">
      <div
        ref="panelRef"
        class="ir-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="ir-title"
        aria-describedby="ir-intro"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <div class="ir-header">
          <h2 id="ir-title" class="ir-title">Interactive rebase of {{ rebase.branch }}</h2>
          <p id="ir-intro" class="ir-intro">
            {{ rebase.rows.length }} {{ rebase.rows.length === 1 ? 'commit' : 'commits' }}, oldest
            first. Git replays them from top to bottom.
          </p>
        </div>

        <ol class="ir-list">
          <li
            v-for="(row, i) in rebase.rows"
            :key="row.hash"
            class="ir-row"
            :class="`ir-row--${row.action}`"
            :data-hash="row.hash"
          >
            <div class="ir-line">
              <select
                v-model="row.action"
                class="ir-select"
                :aria-label="`Action for ${row.shortHash} ${row.subject}`"
              >
                <option v-for="a in ACTIONS" :key="a.value" :value="a.value">{{ a.label }}</option>
              </select>
              <span class="ir-hash">{{ row.shortHash }}</span>
              <span class="ir-subject" :title="row.subject">{{ row.subject }}</span>
              <span v-if="row.pushed" class="ir-pushed" :title="`Already on ${rebase.upstream}`"
                >pushed</span
              >
              <button
                class="ir-move move-up"
                :disabled="i === 0"
                :aria-label="`Move ${row.shortHash} ${row.subject} up`"
                title="Move up"
                @click="move(i, -1)"
              >
                <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                  <path d="M3 7.5L6 4.5L9 7.5" />
                </svg>
              </button>
              <button
                class="ir-move move-down"
                :disabled="i === rebase.rows.length - 1"
                :aria-label="`Move ${row.shortHash} ${row.subject} down`"
                title="Move down"
                @click="move(i, 1)"
              >
                <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                  <path d="M3 4.5L6 7.5L9 4.5" />
                </svg>
              </button>
            </div>
            <p v-if="note(i)" class="ir-note">{{ note(i) }}</p>
            <textarea
              v-if="row.action === 'reword'"
              v-model="row.newMessage"
              class="ir-message"
              rows="3"
              spellcheck="true"
              :aria-label="`New message for ${row.shortHash}`"
            />
          </li>
        </ol>

        <p v-if="rebase.pushedRewritten" class="ir-warning">
          This rewrites {{ rebase.pushedRewritten }}
          {{ rebase.pushedRewritten === 1 ? 'commit that is' : 'commits that are' }} already on
          {{ rebase.upstream }}, so the next push has to be a force push.
        </p>

        <div class="ir-footer">
          <p class="ir-status" role="status">
            {{ rebase.problem || (rebase.changed ? '' : 'Nothing changed yet.') }}
          </p>
          <button class="ir-btn" @click="rebase.close()">Cancel</button>
          <button
            class="ir-btn ir-btn--primary"
            :disabled="!!rebase.problem || !rebase.changed || rebase.starting"
            @click="start"
          >
            {{ rebase.starting ? 'Rebasing…' : 'Start rebase' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.ir-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.ir-panel {
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 10px;
  width: min(680px, 92vw);
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-modal);
}

.ir-header {
  padding: 14px 20px 12px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.ir-title {
  margin: 0;
  font-size: var(--text-title);
  font-weight: 600;
  color: var(--text-primary);
}
.ir-intro {
  margin: 4px 0 0;
  font-size: var(--text-row);
  color: var(--text-secondary);
}

.ir-list {
  list-style: none;
  margin: 0;
  padding: 6px 0;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.ir-row {
  padding: 5px 20px;
}
.ir-row + .ir-row {
  border-top: 1px solid var(--border-subtle);
}

.ir-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.ir-select {
  flex-shrink: 0;
  width: 92px;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--text-primary);
  font-size: var(--text-row);
  padding: 4px 6px;
  cursor: default;
}
.ir-select:focus-visible,
.ir-move:focus-visible,
.ir-btn:focus-visible,
.ir-message:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: 1px;
}

.ir-hash {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}
.ir-subject {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--text-row);
  color: var(--text-primary);
}
.ir-row--drop .ir-subject {
  color: var(--text-tertiary);
  text-decoration: line-through;
}
.ir-row--squash .ir-subject,
.ir-row--fixup .ir-subject {
  color: var(--text-secondary);
}

.ir-pushed {
  flex-shrink: 0;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
  border: 1px solid var(--border);
  border-radius: 3px;
  padding: 0 5px;
}

.ir-move {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: default;
}
.ir-move svg {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.ir-move:hover:not(:disabled) {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.ir-move:disabled {
  opacity: 0.35;
}

.ir-note {
  margin: 3px 0 0 100px;
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.ir-message {
  display: block;
  width: calc(100% - 100px);
  margin: 6px 0 2px 100px;
  box-sizing: border-box;
  background: var(--surface-field);
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: var(--text-row);
  padding: 6px 8px;
  resize: vertical;
}

.ir-warning {
  margin: 0;
  padding: 8px 20px;
  border-top: 1px solid var(--border-subtle);
  font-size: var(--text-row);
  color: var(--warning);
  flex-shrink: 0;
}

.ir-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.ir-status {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: var(--text-row);
  color: var(--text-secondary);
}

.ir-btn {
  padding: 6px 14px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: var(--text-body);
  cursor: default;
}
.ir-btn:hover:not(:disabled) {
  border-color: var(--border-strong);
  color: var(--text-primary);
}
.ir-btn--primary {
  background: var(--accent-fill);
  border-color: var(--accent-fill);
  color: var(--text-on-accent);
  font-weight: 500;
}
.ir-btn--primary:hover:not(:disabled) {
  background: var(--accent-fill-hover);
  border-color: var(--accent-fill-hover);
  color: var(--text-on-accent);
}
.ir-btn--primary:disabled {
  opacity: 0.45;
}
</style>
