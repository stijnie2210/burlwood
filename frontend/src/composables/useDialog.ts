import { nextTick, watch } from 'vue';
import type { Ref } from 'vue';

const FOCUSABLE =
  'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])';

export function useDialog(
  isOpen: () => boolean,
  panel: Ref<HTMLElement | null>,
  close: () => void
) {
  let returnFocus: HTMLElement | null = null;

  watch(isOpen, async (open) => {
    if (open) {
      returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      await nextTick();
      const first = panel.value?.querySelector<HTMLElement>(FOCUSABLE);
      (first ?? panel.value)?.focus();
      return;
    }
    const el = returnFocus;
    returnFocus = null;
    if (el?.isConnected) {
      el.focus();
    }
  });

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      close();
      return;
    }
    if (e.key !== 'Tab' || !panel.value) {
      return;
    }
    const items = Array.from(panel.value.querySelectorAll<HTMLElement>(FOCUSABLE));
    if (items.length === 0) {
      return;
    }
    const first = items[0];
    const last = items[items.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  return { onKeydown };
}
