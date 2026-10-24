// Call after nextTick — element must be rendered to measure its size.
export function clampMenuPosition(
  el: HTMLElement,
  x: number,
  y: number,
  padding = 8
): { x: number; y: number } {
  const { width, height } = el.getBoundingClientRect();
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  return {
    x: x + width + padding > vw ? Math.max(0, x - width) : x,
    y: y + height + padding > vh ? Math.max(0, y - height) : y,
  };
}

const MENU_ITEMS = 'button:not(:disabled)';

export function focusFirstMenuItem(el: HTMLElement | null | undefined) {
  el?.querySelector<HTMLElement>(MENU_ITEMS)?.focus();
}

// Returns true when the key was consumed, so callers listening at document level can stop.
export function handleMenuKeydown(
  e: KeyboardEvent,
  el: HTMLElement | null | undefined,
  close: () => void
): boolean {
  if (!el) {
    return false;
  }
  if (e.key === 'Escape') {
    e.preventDefault();
    e.stopPropagation();
    close();
    return true;
  }
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') {
    return false;
  }
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return false;
  }
  const items = Array.from(el.querySelectorAll<HTMLElement>(MENU_ITEMS));
  if (items.length === 0) {
    return false;
  }
  e.preventDefault();
  e.stopPropagation();
  const i = items.indexOf(document.activeElement as HTMLElement);
  const next = e.key === 'ArrowDown' ? (i + 1) % items.length : i <= 0 ? items.length - 1 : i - 1;
  items[next].focus();
  return true;
}

// Opens a context menu from the keyboard (Shift+F10) at the focused element's position.
export function menuEventFor(e: KeyboardEvent): MouseEvent {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
  return new MouseEvent('contextmenu', {
    clientX: rect.left + 16,
    clientY: rect.bottom,
    cancelable: true,
  });
}

// Switching a menu into a confirm or input mode removes the focused button; pull focus back
// inside so Escape and the arrow keys keep working.
export function keepFocusInMenu(el: HTMLElement | null | undefined) {
  if (!el || el.contains(document.activeElement)) {
    return;
  }
  el.querySelector<HTMLElement>('input, ' + MENU_ITEMS)?.focus();
}
