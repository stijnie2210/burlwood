import { test as base, type Page } from '@playwright/test';
import { execFileSync, spawn, type ChildProcess } from 'node:child_process';
import { mkdirSync, mkdtempSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { basename, join } from 'node:path';

export const FRONTEND_DIR = join(import.meta.dirname, '..');
export const SERVER_BIN = join(import.meta.dirname, '.server', 'burlwood-server');

// A throwaway HOME per test: Burlwood's settings folder lives under it, and git reads no
// global config from the developer's machine (signing, hooks, aliases).
export class Sandbox {
  readonly home = realpathSync(mkdtempSync(join(tmpdir(), 'burlwood-e2e-')));

  constructor() {
    writeFileSync(
      join(this.home, '.gitconfig'),
      '[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n'
    );
  }

  env(): NodeJS.ProcessEnv {
    return {
      ...process.env,
      HOME: this.home,
      XDG_CONFIG_HOME: join(this.home, '.config'),
      GIT_CONFIG_NOSYSTEM: '1',
    };
  }

  git(dir: string, ...args: string[]): string {
    return execFileSync('git', args, { cwd: dir, env: this.env(), encoding: 'utf8' }).trim();
  }

  gitOk(dir: string, ...args: string[]): boolean {
    try {
      execFileSync('git', args, { cwd: dir, env: this.env(), stdio: 'ignore' });
      return true;
    } catch {
      return false;
    }
  }

  repo(name = 'repo'): string {
    const dir = join(this.home, name);
    mkdirSync(dir);
    this.git(dir, 'init', '-q');
    this.commit(dir, 'README.md', '# test\n', 'initial commit');
    return dir;
  }

  commit(dir: string, file: string, content: string, message: string) {
    writeFileSync(join(dir, file), content);
    this.git(dir, 'add', '--', file);
    this.git(dir, 'commit', '-q', '-m', message);
  }

  bareRemote(dir: string, name = 'origin'): string {
    const remote = join(this.home, `${basename(dir)}-${name}.git`);
    this.git(this.home, 'init', '-q', '--bare', remote);
    this.git(dir, 'remote', 'add', name, remote);
    return remote;
  }

  configDir(): string {
    return process.platform === 'darwin'
      ? join(this.home, 'Library', 'Application Support', 'burlwood')
      : join(this.home, '.config', 'burlwood');
  }

  writeConfig(name: string, value: unknown) {
    mkdirSync(this.configDir(), { recursive: true });
    writeFileSync(join(this.configDir(), name), JSON.stringify(value, null, 2));
  }
}

type Fixtures = {
  sandbox: Sandbox;
  launch: (repos: string[]) => Promise<Page>;
};

export const test = base.extend<Fixtures>({
  // eslint-disable-next-line no-empty-pattern
  sandbox: async ({}, use) => {
    const sandbox = new Sandbox();
    await use(sandbox);
    rmSync(sandbox.home, { recursive: true, force: true });
  },

  launch: async ({ page, sandbox }, use, testInfo) => {
    let server: ChildProcess | undefined;
    let output = '';
    await use(async (repos) => {
      sandbox.writeConfig('session.json', {
        tabs: repos.map((path) => ({ path, name: basename(path) })),
        activeIndex: 0,
      });
      // Background fetches would change refs while a test is asserting on them.
      sandbox.writeConfig('prefs.json', { autoFetchEnabled: false });

      const port = await freePort();
      server = spawn(SERVER_BIN, [], {
        env: { ...sandbox.env(), WAILS_SERVER_PORT: String(port) },
        stdio: ['ignore', 'pipe', 'pipe'],
      });
      server.stdout?.on('data', (d) => (output += d));
      server.stderr?.on('data', (d) => (output += d));

      const url = `http://localhost:${port}/`;
      await waitForServer(url, server);
      await page.goto(url);
      // The macOS sidebar is see-through to show the window's vibrancy; a browser has none behind it.
      await page.addStyleTag({ content: 'html:root { --surface-sidebar: #211d1a; }' });
      return page;
    });
    server?.kill();
    if (testInfo.status !== testInfo.expectedStatus && output) {
      await testInfo.attach('server-output', { body: output, contentType: 'text/plain' });
    }
  },
});

export { expect } from '@playwright/test';

function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = createServer();
    srv.once('error', reject);
    srv.listen(0, '127.0.0.1', () => {
      const address = srv.address();
      srv.close(() => resolve(typeof address === 'object' && address ? address.port : 0));
    });
  });
}

async function waitForServer(url: string, server: ChildProcess) {
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    if (server.exitCode !== null) {
      throw new Error(`burlwood-server exited with code ${server.exitCode}`);
    }
    try {
      if ((await fetch(url)).ok) {
        return;
      }
    } catch {
      // not listening yet
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(`burlwood-server didn't answer on ${url}`);
}
