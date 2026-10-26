import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

vi.mock('../../wailsjs/go/main/App', () => ({
  LoadSession: vi.fn(async () => ({
    tabs: [
      { path: '/a', name: 'a' },
      { path: '/flaky', name: 'flaky' },
      { path: '/b', name: 'b' },
    ],
    activeIndex: 2,
  })),
  OpenRepository: async (path: string) => {
    if (path === '/gone' || path === '/flaky') {
      throw `${path} isn't inside a git repository`;
    }
    return path;
  },
  RepositoryGone: async (path: string) => path === '/gone',
  SaveSession: vi.fn(async () => {}),
}));

const { useReposStore } = await import('./repos');
const { useToastStore } = await import('./toast');
const { LoadSession, SaveSession } = await import('../../wailsjs/go/main/App');

describe('restoring the last session', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.mocked(SaveSession).mockClear();
  });

  it('stays in the restoring state until every repo has answered', async () => {
    const repos = useReposStore();
    expect(repos.restoring).toBe(true);
    const restore = repos.restoreSession();
    expect(repos.restoring).toBe(true);
    await restore;
    expect(repos.restoring).toBe(false);
    expect(repos.restoringNames).toEqual(['a', 'flaky', 'b']);
  });

  it('reports a repo that failed but keeps it in the saved session', async () => {
    const repos = useReposStore();
    await repos.restoreSession();
    expect(repos.tabs.map((t) => t.path)).toEqual(['/a', '/b']);
    expect(repos.activeRepo?.path).toBe('/b');
    expect(useToastStore().toasts.map((t) => t.message)).toEqual(["Couldn't reopen flaky"]);
    expect(SaveSession).not.toHaveBeenCalled();
  });

  it('closes the tab of a repo that no longer exists', async () => {
    vi.mocked(LoadSession).mockResolvedValueOnce({
      tabs: [
        { path: '/gone', name: 'gone' },
        { path: '/flaky', name: 'flaky' },
        { path: '/b', name: 'b' },
      ],
      activeIndex: 2,
    } as never);
    const repos = useReposStore();
    await repos.restoreSession();
    expect(repos.tabs.map((t) => t.path)).toEqual(['/b']);
    expect(SaveSession).toHaveBeenCalledWith(
      [
        { path: '/flaky', name: 'flaky' },
        { path: '/b', name: 'b' },
      ],
      1
    );
    expect(useToastStore().toasts.map((t) => t.message)).toEqual([
      "Couldn't reopen flaky",
      'Closed gone: the repository no longer exists',
    ]);
  });

  it('falls back to the first tab when the active one failed', async () => {
    vi.mocked(LoadSession).mockResolvedValueOnce({
      tabs: [
        { path: '/a', name: 'a' },
        { path: '/b', name: 'b' },
        { path: '/gone', name: 'gone' },
      ],
      activeIndex: 2,
    } as never);
    const repos = useReposStore();
    await repos.restoreSession();
    expect(repos.activeRepo?.path).toBe('/a');
    expect(SaveSession).toHaveBeenCalledWith(expect.any(Array), 0);
  });
});
