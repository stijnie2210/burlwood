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
          :key="l.name"
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
  background: var(--surface-sunken);
  border-left: 1px solid var(--border-subtle);
  min-width: 200px;
  height: 100%;
}

.commit-meta {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 5px;
}

.meta-hash {
  font-family: var(--font-mono);
  font-size: var(--text-meta);
  color: var(--text-tertiary);
}

.meta-subject {
  font-size: var(--text-body);
  color: var(--text-primary);
  font-weight: 500;
  margin-bottom: 3px;
}
.meta-info {
  display: flex;
  gap: 12px;
  color: var(--text-secondary);
  font-size: var(--text-meta);
}

.loading {
  padding: 16px;
  color: var(--text-tertiary);
  text-align: center;
}
</style>
