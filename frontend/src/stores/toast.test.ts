import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useToastStore } from './toast';

describe('toast store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('holds a paused toast past its deadline and expires it after resuming', () => {
    const toast = useToastStore();
    const id = toast.withUndo('Discarded', () => {});
    vi.advanceTimersByTime(4000);
    toast.pause(id);
    vi.advanceTimersByTime(60000);
    expect(toast.toasts).toHaveLength(1);
    toast.resume(id);
    vi.advanceTimersByTime(6100);
    expect(toast.toasts).toHaveLength(0);
  });

  it('keeps a Working Tree hint until the Working Tree opens', () => {
    const toast = useToastStore();
    toast.workingTreeHint('Stopped on a conflict');
    toast.success('Something else');
    vi.advanceTimersByTime(60000);
    expect(toast.toasts.map((t) => t.message)).toEqual(['Stopped on a conflict']);

    toast.setWorkingTreeOpen(true);
    expect(toast.toasts).toHaveLength(0);
  });

  it('lets a hint expire by itself when the Working Tree is already open', () => {
    const toast = useToastStore();
    toast.setWorkingTreeOpen(true);
    toast.workingTreeHint('Stopped on a conflict');
    expect(toast.toasts).toHaveLength(1);
    vi.advanceTimersByTime(6100);
    expect(toast.toasts).toHaveLength(0);
  });

  it('undoLatest runs the newest undo and dismisses it', () => {
    const toast = useToastStore();
    const calls: string[] = [];
    toast.withUndo('first', () => {
      calls.push('first');
    });
    toast.success('unrelated');
    toast.withUndo('second', () => {
      calls.push('second');
    });
    expect(toast.undoLatest()).toBe(true);
    expect(calls).toEqual(['second']);
    expect(toast.toasts.map((t) => t.message)).toEqual(['first', 'unrelated']);
  });

  it('undoLatest does nothing without an undo toast', () => {
    const toast = useToastStore();
    toast.success('done');
    expect(toast.undoLatest()).toBe(false);
  });
});
