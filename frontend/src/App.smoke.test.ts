// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { createApp, nextTick } from 'vue';
import { createPinia } from 'pinia';

vi.mock('../bindings/burlwood/app', async (importOriginal) => {
  const orig = (await importOriginal()) as Record<string, unknown>;
  const lists = [
    'GetCommitGraph',
    'GetStatus',
    'ListBranches',
    'ListTags',
    'ListStashes',
    'GetWorkingDiff',
    'GetCommitDiff',
    'GetRecentRepos',
  ];
  return Object.fromEntries(
    Object.keys(orig).map((k) => [
      k,
      async (arg?: unknown) =>
        k === 'OpenRepository'
          ? arg
          : lists.includes(k)
            ? []
            : k === 'GetAheadBehind'
              ? { ahead: 0, behind: 0 }
              : k === 'GetPrefs'
                ? {}
                : k === 'LoadSession'
                  ? {
                      tabs: [
                        { path: '/tmp/a', name: 'a' },
                        { path: '/tmp/b', name: 'b' },
                      ],
                      activeIndex: 0,
                    }
                  : k === 'GetRebaseState'
                    ? { step: 0, total: 0, message: '', onto: '' }
                    : null,
    ])
  );
});
vi.mock('@wailsio/runtime', () => ({
  Events: { On: () => () => {}, Off: () => {} },
}));

// Mounts the whole app against stubbed Wails bindings, so setup-time errors (like reading a
// ref before its declaration) fail a test instead of blanking the window.
describe('App smoke test', () => {
  it('mounts App without throwing', async () => {
    const errors: unknown[] = [];
    const el = document.createElement('div');
    document.body.appendChild(el);
    const { default: App } = await import('./App.vue');
    const app = createApp(App);
    app.config.errorHandler = (e) => errors.push(e);
    const pinia = createPinia();
    app.use(pinia);
    app.mount(el);
    await nextTick();
    await new Promise((r) => setTimeout(r, 50));
    expect(errors).toEqual([]);
    expect(el.innerHTML).toContain('Working Tree');
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '2', metaKey: true }));
    await nextTick();
    await new Promise((r) => setTimeout(r, 50));
    expect(errors).toEqual([]);
    expect(el.innerHTML).toContain('Unstaged');

    // An open commit-file diff must not survive a repo switch; the graph comes back.
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '1', metaKey: true }));
    await nextTick();
    const { useCommitsStore } = await import('./stores/commits');
    const commits = useCommitsStore(pinia);
    commits.diff = [{ oldPath: 'a.txt', newPath: 'a.txt', isBinary: false, hunks: [] }] as never;
    commits.selectedHash = 'abc';
    commits.selectedFileIndex = 0;
    await nextTick();
    expect(el.querySelector('.fdv')).not.toBeNull();

    window.dispatchEvent(new KeyboardEvent('keydown', { key: ']', metaKey: true }));
    await nextTick();
    await new Promise((r) => setTimeout(r, 50));
    expect(errors).toEqual([]);
    expect(el.querySelector('.fdv')).toBeNull();
    expect(el.querySelector('.graph-scroller')).not.toBeNull();
  });
});
