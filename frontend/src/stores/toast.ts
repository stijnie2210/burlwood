import { acceptHMRUpdate, defineStore } from 'pinia';
import { ref } from 'vue';
import { describeGitError } from '../utils/gitErrors';

export type ToastType = 'success' | 'error' | 'info';

export interface ToastAction {
  label: string;
  run: () => void | Promise<void>;
  danger?: boolean;
}

export interface Toast {
  id: number;
  message: string;
  type: ToastType;
  duration: number;
  action?: ToastAction;
  detail?: string;
  paused?: boolean;
}

const PREVIEW_LIMIT = 160;
const UNDO_WINDOW_MS = 10000;

export function isLongMessage(message: string): boolean {
  return message.length > PREVIEW_LIMIT || message.includes('\n');
}

let nextId = 1;

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([]);
  const timers = new Map<number, { handle: ReturnType<typeof setTimeout>; deadline: number }>();
  const remaining = new Map<number, number>();

  function schedule(id: number, ms: number) {
    timers.set(id, { handle: setTimeout(() => dismiss(id), ms), deadline: Date.now() + ms });
  }

  function push(
    message: string,
    type: ToastType = 'info',
    duration?: number,
    action?: ToastAction,
    detail?: string
  ) {
    const ms = duration ?? (isLongMessage(message) ? 0 : type === 'error' ? 7000 : 3500);
    const id = nextId++;
    toasts.value.push({ id, message, type, duration: ms, action, detail });
    if (ms > 0) {
      schedule(id, ms);
    }
    return id;
  }

  // Hovering or focusing a toast holds it, so an Undo can't expire while you reach for it.
  function pause(id: number) {
    const timer = timers.get(id);
    const t = toasts.value.find((x) => x.id === id);
    if (!timer || !t) {
      return;
    }
    clearTimeout(timer.handle);
    timers.delete(id);
    remaining.set(id, Math.max(1000, timer.deadline - Date.now()));
    t.paused = true;
  }

  function resume(id: number) {
    const ms = remaining.get(id);
    const t = toasts.value.find((x) => x.id === id);
    if (ms === undefined || !t) {
      return;
    }
    remaining.delete(id);
    t.paused = false;
    schedule(id, ms);
  }

  function undoLatest(): boolean {
    const t = [...toasts.value].reverse().find((x) => x.action?.label === 'Undo');
    if (!t?.action) {
      return false;
    }
    dismiss(t.id);
    t.action.run();
    return true;
  }

  function dismiss(id: number) {
    const timer = timers.get(id);
    if (timer) {
      clearTimeout(timer.handle);
      timers.delete(id);
    }
    remaining.delete(id);
    const i = toasts.value.findIndex((t) => t.id === id);
    if (i !== -1) {
      toasts.value.splice(i, 1);
    }
  }

  function success(message: string) {
    return push(message, 'success');
  }
  // Accepts raw errors from git calls and shows them as a sentence with a next step.
  function error(err: unknown) {
    const { message, detail } = describeGitError(err);
    return push(message, 'error', undefined, undefined, detail);
  }
  function withUndo(message: string, undo: () => void | Promise<void>) {
    return push(message, 'info', UNDO_WINDOW_MS, { label: 'Undo', run: undo });
  }

  // A hint that sends the user to the Working Tree stays until they get there: left up, it
  // would cover the conflict view's Save & Stage button.
  const workingTreeHints = new Set<number>();
  let workingTreeOpen = false;

  function workingTreeHint(message: string) {
    if (workingTreeOpen) {
      return push(message, 'info', 6000);
    }
    const id = push(message, 'info', 0);
    workingTreeHints.add(id);
    return id;
  }

  function setWorkingTreeOpen(open: boolean) {
    workingTreeOpen = open;
    if (open) {
      workingTreeHints.forEach(dismiss);
      workingTreeHints.clear();
    }
  }

  return {
    toasts,
    push,
    dismiss,
    pause,
    resume,
    undoLatest,
    success,
    error,
    withUndo,
    workingTreeHint,
    setWorkingTreeOpen,
  };
});

// Without this, editing a store during `wails3 dev` keeps the old instance (and its old logic)
// running until a full reload.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useToastStore, import.meta.hot));
}
