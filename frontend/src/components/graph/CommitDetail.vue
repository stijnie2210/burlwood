<script setup lang="ts">
import { computed } from 'vue';
import { useCommitsStore } from '../../stores/commits';
import { useReposStore } from '../../stores/repos';
import { usePrefsStore } from '../../stores/prefs';
import { useToastStore } from '../../stores/toast';
import { formatDate } from '../../utils/date';
import { initials } from '../../utils/initials';
import CommitFileList from './CommitFileList.vue';
import RefLabel from './RefLabel.vue';

const commits = useCommitsStore();
const repos = useReposStore();
const prefsStore = usePrefsStore();
const toast = useToastStore();

const selectedRow = computed(
  () => commits.rows.find((r) => r.hash === commits.selectedHash) ?? null
);

const displayDate = computed(() => {
  const row = selectedRow.value;
  if (!row) {
    return '';
  }
  if (!row.timestamp) {
    return row.date;
  }
  return formatDate(row.timestamp, prefsStore.prefs.dateFormat, prefsStore.now);
});

async function copyHash() {
  if (!commits.selectedHash) {
    return;
  }
  await navigator.clipboard.writeText(commits.selectedHash);
  toast.success('Hash copied');
}

function showParent(hash: string) {
  if (repos.activeRepo) {
    commits.selectCommit(repos.activeRepo.path, hash);
  }
}
</script>

<template>
  <aside v-if="commits.selectedHash" class="detail-panel">
    <div v-if="selectedRow" class="commit-meta">
      <div class="meta-row">
        <span class="meta-hash">{{ selectedRow.shortHash }}</span>
        <button
          class="copy-btn"
          title="Copy full hash (Y)"
          aria-label="Copy hash"
          @click="copyHash"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" aria-hidden="true">
            <rect x="9" y="9" width="13" height="13" rx="2" />
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
          </svg>
        </button>
        <RefLabel
          v-for="l in selectedRow.labels"
          :key="`${l.type}:${l.remote ?? ''}:${l.name}`"
          :label="l"
        />
      </div>
      <div class="meta-subject">{{ selectedRow.subject }}</div>
      <div class="meta-info">
        <span class="avatar" aria-hidden="true">{{ initials(selectedRow.author) }}</span>
        <span class="meta-author">{{ selectedRow.author }}</span>
        <span class="meta-date">· {{ displayDate }}</span>
      </div>
      <div v-if="selectedRow.parentHashes?.length" class="meta-parents">
        {{ selectedRow.parentHashes.length > 1 ? 'Parents' : 'Parent' }}
        <button
          v-for="p in selectedRow.parentHashes"
          :key="p"
          class="parent-link"
          :title="`Show ${p.slice(0, 7)}`"
          @click="showParent(p)"
        >
          {{ p.slice(0, 7) }}
        </button>
      </div>
    </div>

    <div v-if="commits.diffLoading" class="loading">Loading…</div>
    <CommitFileList v-else />
  </aside>
</template>

<style scoped>
.detail-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-size: var(--text-row);
  background: var(--surface-chrome);
  border-left: 1px solid var(--border-subtle);
  min-width: 200px;
  height: 100%;
}

.commit-meta {
  padding: var(--space-3) var(--inset-panel) var(--space-4);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-wrap: wrap;
  margin-bottom: var(--space-2);
}

.meta-hash {
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.meta-subject {
  font-size: var(--text-title);
  color: var(--text-primary);
  font-weight: 600;
  margin-bottom: var(--space-2);
  line-height: 1.35;
  overflow-wrap: anywhere;
}
.meta-info {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--text-secondary);
  font-size: var(--text-row);
}
.meta-author {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.meta-date {
  flex-shrink: 0;
  color: var(--text-tertiary);
}

.avatar {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--avatar);
  color: var(--text-primary);
  font-size: var(--text-meta);
  font-weight: 600;
  letter-spacing: -0.3px;
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 5px;
  background: none;
  color: var(--text-tertiary);
  cursor: default;
}
.copy-btn:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.copy-btn svg {
  fill: none;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.meta-parents {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: var(--space-2);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}
.parent-link {
  padding: 0;
  border: none;
  background: none;
  color: var(--accent-text);
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  cursor: default;
}
.parent-link:hover {
  text-decoration: underline;
}

.loading {
  padding: 16px;
  color: var(--text-tertiary);
  text-align: center;
}
</style>
