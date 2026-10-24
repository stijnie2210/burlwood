import { describe, it, expect, vi } from 'vitest';
import { createApp, nextTick } from 'vue';
import { createPinia } from 'pinia';

vi.mock('../wailsjs/go/main/App', async (importOriginal) => {
  const orig = (await importOriginal()) as Record<string, unknown>;
  const lists = [
    'GetCommitGraph',
    'GetStatus',
    'ListBranches',
    'ListStashes',
    'GetWorkingDiff',
    'GetCommitDiff',
    'GetRecentRepos',
  ];
  return Object.fromEntries(
    Object.keys(orig).map((k) => [
      k,
      async () =>
        lists.includes(k)
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
vi.mock('../wailsjs/runtime/runtime', () => ({
  EventsOn: () => () => {},
  EventsOff: () => {},
  OnFileDrop: () => {},
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
    app.use(createPinia());
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
  });
});
