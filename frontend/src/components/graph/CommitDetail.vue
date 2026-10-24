<script setup lang="ts">
import { computed } from 'vue';
import { useCommitsStore } from '../../stores/commits';
import { usePrefsStore } from '../../stores/prefs';
import { formatDate } from '../../utils/date';
import CommitFileList from './CommitFileList.vue';

const commits = useCommitsStore();
const prefsStore = usePrefsStore();

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
</script>

<template>
  <aside v-if="commits.selectedHash" class="detail-panel">
    <div v-if="selectedRow" class="commit-meta">
      <div class="meta-row">
        <span class="meta-hash">{{ selectedRow.shortHash }}</span>
        <span
          v-for="l in selectedRow.labels"
          :key="`${l.type}:${l.remote ?? ''}:${l.name}`"
          class="label"
          :class="`label--${l.type}`"
          >{{ l.type === 'remote' ? `${l.remote}/${l.name}` : l.name }}</span
        >
      </div>
      <div class="meta-subject">{{ selectedRow.subject }}</div>
      <div class="meta-info">
        <span>{{ selectedRow.author }}</span>
        <span>{{ displayDate }}</span>
      </div>
    </div>

    <div v-if="commits.diffLoading" class="loading">Loading…</div>
    <CommitFileList v-else />
  </aside>
</template>

<style src="../../assets/labels.css" />

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
  gap: var(--space-3);
  color: var(--text-tertiary);
  font-size: var(--text-row);
}

.loading {
  padding: 16px;
  color: var(--text-tertiary);
  text-align: center;
}
</style>
