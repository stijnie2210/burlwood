<script setup lang="ts">
import { useToastStore, isLongMessage } from '../../stores/toast';
import type { Toast } from '../../stores/toast';
import { useLogModalStore } from '../../stores/logModal';

const toast = useToastStore();
const logModal = useLogModalStore();

const PREVIEW_LIMIT = 160;

function preview(message: string): string {
  const firstLine = message.split('\n')[0];
  if (firstLine.length > PREVIEW_LIMIT) {
    return firstLine.slice(0, PREVIEW_LIMIT) + '…';
  }
  return message.includes('\n') ? firstLine + ' …' : firstLine;
}

function viewLog(t: { type: string; message: string }, e: MouseEvent) {
  e.stopPropagation();
  logModal.show(t.type === 'error' ? 'Error details' : 'Details', t.message);
}

async function runAction(t: Toast, e: MouseEvent) {
  e.stopPropagation();
  toast.dismiss(t.id);
  await t.action?.run();
}
</script>

<template>
  <Teleport to="body">
    <div class="toast-stack" role="status" aria-live="polite">
      <TransitionGroup name="toast">
        <div
          v-for="t in toast.toasts"
          :key="t.id"
          class="toast"
          :class="[`toast--${t.type}`, { paused: t.paused }]"
          :role="t.type === 'error' ? 'alert' : undefined"
          @click="!t.action && toast.dismiss(t.id)"
          @mouseenter="toast.pause(t.id)"
          @mouseleave="toast.resume(t.id)"
          @focusin="toast.pause(t.id)"
          @focusout="toast.resume(t.id)"
        >
          <svg class="toast-icon" width="14" height="14" viewBox="0 0 14 14" aria-hidden="true">
            <path
              v-if="t.type === 'success'"
              d="M3 7.5l2.5 2.5L11 4.5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <path
              v-else-if="t.type === 'error'"
              d="M4 4l6 6M10 4l-6 6"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
            />
            <g v-else fill="currentColor">
              <circle cx="7" cy="3.8" r="1.1" />
              <rect x="6.1" y="6" width="1.8" height="5" rx="0.9" />
            </g>
          </svg>
          <span class="toast-msg">{{
            isLongMessage(t.message) ? preview(t.message) : t.message
          }}</span>
          <button v-if="isLongMessage(t.message)" class="toast-action" @click="viewLog(t, $event)">
            View log
          </button>
          <button
            v-if="t.action"
            class="toast-action"
            :title="t.action.label === 'Undo' ? 'Undo (⌘Z)' : undefined"
            @click="runAction(t, $event)"
          >
            {{ t.action.label }}
          </button>
          <button class="toast-close" aria-label="Dismiss" @click.stop="toast.dismiss(t.id)">
            <svg width="8" height="8" viewBox="0 0 8 8" aria-hidden="true">
              <path d="M1 1l6 6M7 1L1 7" stroke="currentColor" stroke-width="1.4" />
            </svg>
          </button>
          <span
            v-if="t.action && t.duration > 0"
            class="toast-timer"
            :style="{ animationDuration: `${t.duration}ms` }"
          />
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  bottom: 20px;
  right: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 9999;
  pointer-events: none;
}

.toast {
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 14px;
  border-radius: 6px;
  font-size: var(--text-row);
  max-width: 420px;
  cursor: pointer;
  pointer-events: all;
  box-shadow: var(--shadow-popover);
  user-select: none;
}

.toast--success {
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  color: var(--text-primary);
}
.toast--error {
  background: var(--surface-overlay);
  border: 1px solid var(--danger-border);
  color: var(--text-primary);
}
.toast--info {
  background: var(--surface-overlay);
  border: 1px solid var(--border);
  color: var(--text-primary);
}

.toast--success .toast-icon {
  color: var(--add);
}
.toast--error .toast-icon {
  color: var(--danger);
}
.toast--info .toast-icon {
  color: var(--accent-text);
}

.toast-icon {
  flex-shrink: 0;
}

.toast-close {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  padding: 0;
  background: none;
  border: none;
  border-radius: 4px;
  color: var(--text-tertiary);
  cursor: pointer;
}
.toast-close:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.toast.paused .toast-timer {
  animation-play-state: paused;
}

.toast-msg {
  flex: 1;
  line-height: 1.4;
}

.toast-action {
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 4px;
  color: inherit;
  font-size: var(--text-meta);
  font-weight: 600;
  padding: 3px 8px;
  cursor: pointer;
}
.toast-action:hover {
  background: rgba(255, 255, 255, 0.16);
}

.toast-timer {
  position: absolute;
  left: 0;
  bottom: 0;
  height: 2px;
  width: 100%;
  background: currentColor;
  opacity: 0.45;
  transform-origin: left;
  animation: toast-timer linear forwards;
}
@keyframes toast-timer {
  to {
    transform: scaleX(0);
  }
}
@media (prefers-reduced-motion: reduce) {
  .toast-timer {
    display: none;
  }
}

/* Transitions */
.toast-enter-active {
  transition:
    transform 0.2s ease,
    opacity 0.2s ease;
}
.toast-leave-active {
  transition:
    transform 0.18s ease,
    opacity 0.18s ease;
}
.toast-enter-from {
  transform: translateX(20px);
  opacity: 0;
}
.toast-leave-to {
  transform: translateX(20px);
  opacity: 0;
}
.toast-move {
  transition: transform 0.2s ease;
}
</style>
