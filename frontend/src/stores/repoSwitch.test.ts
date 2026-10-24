import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

type Deferred<T> = { promise: Promise<T>; resolve: (v: T) => void };
function deferred<T>(): Deferred<T> {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
}

const pending: Record<string, Deferred<unknown>> = {};
vi.mock('../../wailsjs/go/main/App', () => ({
  ListBranches: (repo: string) => (pending[`branches:${repo}`] = deferred()).promise,
  GetAheadBehind: async () => ({ ahead: 0, behind: 0 }),
  ListStashes: (repo: string) => (pending[`stash:${repo}`] = deferred()).promise,
  SwitchBranch: async () => null,
  FetchAll: async () => null,
}));

const { useReposStore } = await import('./repos');
const { useBranchesStore } = await import('./branches');
const { useStashStore } = await import('./stash');

function branch(name: string) {
  return { name, isRemote: false, isCurrent: false };
}

describe('switching repos', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    const repos = useReposStore();
    repos.tabs.push({ path: '/a', name: 'a' }, { path: '/b', name: 'b' });
    repos.activeIndex = 0;
  });

  it('drops a slow branch load for the repo you left', async () => {
    const repos = useReposStore();
    const branches = useBranchesStore();
    const loadA = branches.load('/a');
    repos.activeIndex = 1;
    const loadB = branches.load('/b');
    expect(branches.local).toEqual([]);

    pending['branches:/b'].resolve([branch('only-in-b')]);
    await loadB;
    pending['branches:/a'].resolve([branch('only-in-a')]);
    await loadA;

    expect(branches.local.map((b) => b.name)).toEqual(['only-in-b']);
  });

  // Auto-fetch for the repo you left calls load() for it seconds later; that must not touch
  // the list on screen at all, or each switch causes one more flicker.
  it('ignores a load started for a repo that is no longer active', async () => {
    const repos = useReposStore();
    const branches = useBranchesStore();
    repos.activeIndex = 1;
    const loadB = branches.load('/b');
    pending['branches:/b'].resolve([branch('only-in-b')]);
    await loadB;

    delete pending['branches:/a'];
    const late = branches.load('/a');
    expect(branches.local.map((b) => b.name)).toEqual(['only-in-b']);
    expect(pending['branches:/a']).toBeUndefined();
    await late;
    expect(branches.local.map((b) => b.name)).toEqual(['only-in-b']);
  });

  it('shows the cached list instantly when switching back', async () => {
    const repos = useReposStore();
    const branches = useBranchesStore();
    const loadA = branches.load('/a');
    pending['branches:/a'].resolve([branch('only-in-a')]);
    await loadA;
    repos.activeIndex = 1;
    const loadB = branches.load('/b');
    pending['branches:/b'].resolve([branch('only-in-b')]);
    await loadB;

    repos.activeIndex = 0;
    const back = branches.load('/a');
    expect(branches.local.map((b) => b.name)).toEqual(['only-in-a']);
    pending['branches:/a'].resolve([branch('only-in-a')]);
    await back;
  });

  it('settles on the last repo when switching back and forth quickly', async () => {
    const repos = useReposStore();
    const branches = useBranchesStore();
    const started: { repo: string; d: Deferred<unknown> }[] = [];
    const go = (index: number) => {
      repos.activeIndex = index;
      const repo = index === 0 ? '/a' : '/b';
      const p = branches.load(repo);
      if (pending[`branches:${repo}`]) {
        started.push({ repo, d: pending[`branches:${repo}`] });
        delete pending[`branches:${repo}`];
      }
      return p;
    };
    const loads = [go(1), go(0), go(1), go(0), go(1)];
    // Late auto-fetch reloads for repos already left, and answers arriving out of order.
    loads.push(branches.load('/a'));
    const seen: string[][] = [];
    for (const { repo, d } of [...started].reverse()) {
      d.resolve([branch(`only-in-${repo.slice(1)}`)]);
      await Promise.resolve();
      await Promise.resolve();
      seen.push(branches.local.map((b) => b.name));
    }
    await Promise.all(loads);
    expect(branches.local.map((b) => b.name)).toEqual(['only-in-b']);
    // Once the active repo's answer is in, the list never changes again.
    expect(seen.every((names) => names.join() === 'only-in-b')).toBe(true);
  });

  it('clears stashes immediately and ignores late results', async () => {
    const repos = useReposStore();
    const stash = useStashStore();
    const loadA = stash.load('/a');
    pending['stash:/a'].resolve([{ ref: 'stash@{0}', message: 'a' }]);
    await loadA;
    expect(stash.entries).toHaveLength(1);

    repos.activeIndex = 1;
    const loadB = stash.load('/b');
    expect(stash.entries).toEqual([]);
    const lateA = stash.load('/a');
    pending['stash:/a'].resolve([{ ref: 'stash@{0}', message: 'late a' }]);
    await lateA;
    pending['stash:/b'].resolve([]);
    await loadB;
    expect(stash.entries).toEqual([]);
  });
});
