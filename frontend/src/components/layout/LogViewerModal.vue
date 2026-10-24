<script setup lang="ts">
import { ref } from 'vue';
import { useLogModalStore } from '../../stores/logModal';
import { useDialog } from '../../composables/useDialog';

const logModal = useLogModalStore();
const copied = ref(false);

async function onCopy() {
  await navigator.clipboard.writeText(logModal.content);
  copied.value = true;
  setTimeout(() => (copied.value = false), 1500);
}

function onOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) {
    logModal.close();
  }
}

const panelRef = ref<HTMLElement | null>(null);
const { onKeydown } = useDialog(
  () => logModal.open,
  panelRef,
  () => logModal.close()
);
</script>

<template>
  <Teleport to="body">
    <div v-if="logModal.open" class="log-overlay" @click="onOverlayClick">
      <div
        ref="panelRef"
        class="log-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="log-title"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <div class="log-header">
          <span id="log-title" class="log-title">{{ logModal.title }}</span>
          <button class="log-close" aria-label="Close" @click="logModal.close()">✕</button>
        </div>

        <pre class="log-body">{{ logModal.content }}</pre>

        <div class="log-footer">
          <button class="log-btn-copy" @click="onCopy">{{ copied ? 'Copied' : 'Copy' }}</button>
          <button class="log-btn-close" @click="logModal.close()">Close</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.log-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.log-panel {
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  border-radius: 10px;
  width: 640px;
  max-width: 90vw;
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-modal);
}

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.log-title {
  font-size: var(--text-title);
  font-weight: 600;
  color: var(--text-primary);
}

.log-close {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  font-size: var(--text-row);
  cursor: default;
  padding: 3px 7px;
  border-radius: 4px;
  line-height: 1;
}
.log-close:hover {
  color: var(--text-secondary);
  background: var(--surface-hover);
}

.log-body {
  overflow: auto;
  flex: 1;
  margin: 0;
  padding: 14px 20px;
  font-family: var(--font-mono);
  font-size: var(--text-row);
  line-height: 1.5;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
  user-select: text;
}

.log-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.log-btn-copy {
  padding: 6px 16px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-tertiary);
  font-size: var(--text-row);
  cursor: default;
}
.log-btn-copy:hover {
  color: var(--text-secondary);
  border-color: var(--border-strong);
}

.log-btn-close {
  padding: 6px 20px;
  background: var(--accent-fill);
  border: none;
  border-radius: 6px;
  color: var(--text-on-accent);
  font-size: var(--text-row);
  font-weight: 600;
  cursor: default;
}
.log-btn-close:hover {
  background: var(--accent-fill-hover);
}
</style>
