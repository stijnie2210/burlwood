// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest';
import { clampMenuPosition, handleMenuKeydown } from './useContextMenu';

function makeEl(width: number, height: number): HTMLElement {
  const el = document.createElement('div');
  el.getBoundingClientRect = () =>
    ({ width, height, top: 0, left: 0, right: 0, bottom: 0 }) as DOMRect;
  return el;
}

beforeEach(() => {
  Object.defineProperty(window, 'innerWidth', { value: 1000, writable: true, configurable: true });
  Object.defineProperty(window, 'innerHeight', { value: 800, writable: true, configurable: true });
});

describe('clampMenuPosition', () => {
  it('returns the position unchanged when menu fits without clipping', () => {
    const el = makeEl(200, 150);
    const result = clampMenuPosition(el, 100, 100);
    expect(result).toEqual({ x: 100, y: 100 });
  });

  it('flips the menu to the left when it overflows the right edge', () => {
    const el = makeEl(200, 100);
    // x=850 + width=200 + padding=8 = 1058 > 1000
    const result = clampMenuPosition(el, 850, 100);
    expect(result.x).toBe(850 - 200); // 650
    expect(result.y).toBe(100);
  });

  it('flips the menu upward when it overflows the bottom edge', () => {
    const el = makeEl(100, 200);
    // y=650 + height=200 + padding=8 = 858 > 800
    const result = clampMenuPosition(el, 100, 650);
    expect(result.x).toBe(100);
    expect(result.y).toBe(650 - 200); // 450
  });

  it('clamps both axes when menu overflows both edges', () => {
    const el = makeEl(200, 200);
    const result = clampMenuPosition(el, 850, 650);
    expect(result.x).toBe(650);
    expect(result.y).toBe(450);
  });

  it('does not go negative when menu is larger than viewport', () => {
    const el = makeEl(1200, 900);
    const result = clampMenuPosition(el, 10, 10);
    expect(result.x).toBeGreaterThanOrEqual(0);
    expect(result.y).toBeGreaterThanOrEqual(0);
  });

  it('respects a custom padding value', () => {
    const el = makeEl(100, 50);
    // x=910 + 100 + 20 = 1030 > 1000 — should flip with padding=20
    const result = clampMenuPosition(el, 910, 100, 20);
    expect(result.x).toBe(910 - 100); // 810
  });
});

describe('handleMenuKeydown', () => {
  function makeMenu(): HTMLElement {
    const menu = document.createElement('div');
    menu.innerHTML = '<button>One</button><button disabled>Off</button><button>Two</button>';
    document.body.appendChild(menu);
    return menu;
  }

  function key(k: string): KeyboardEvent {
    return new KeyboardEvent('keydown', { key: k, cancelable: true });
  }

  it('moves focus down and wraps, skipping disabled items', () => {
    const menu = makeMenu();
    const [one, , two] = Array.from(menu.querySelectorAll('button'));
    one.focus();
    expect(handleMenuKeydown(key('ArrowDown'), menu, () => {})).toBe(true);
    expect(document.activeElement).toBe(two);
    handleMenuKeydown(key('ArrowDown'), menu, () => {});
    expect(document.activeElement).toBe(one);
    menu.remove();
  });

  it('moves focus up from the first item to the last', () => {
    const menu = makeMenu();
    const buttons = menu.querySelectorAll('button');
    buttons[0].focus();
    handleMenuKeydown(key('ArrowUp'), menu, () => {});
    expect(document.activeElement).toBe(buttons[2]);
    menu.remove();
  });

  it('closes on Escape and ignores other keys', () => {
    const menu = makeMenu();
    let closed = false;
    expect(handleMenuKeydown(key('a'), menu, () => (closed = true))).toBe(false);
    expect(closed).toBe(false);
    expect(handleMenuKeydown(key('Escape'), menu, () => (closed = true))).toBe(true);
    expect(closed).toBe(true);
    menu.remove();
  });
});
