import { execFileSync } from 'node:child_process';
import { join } from 'node:path';
import { FRONTEND_DIR, SERVER_BIN } from './harness';

// The server binary embeds frontend/dist, so the frontend is built first. Set
// BURLWOOD_E2E_SKIP_BUILD=1 to rerun tests against the last build.
export default function globalSetup() {
  if (process.env.BURLWOOD_E2E_SKIP_BUILD) {
    return;
  }
  run('npx', ['vite', 'build'], FRONTEND_DIR);
  // With cgo on, Wails' Linux packages still compile against GTK and WebKitGTK under the server
  // tag; server mode needs neither, so CI runners don't have to install them.
  run('go', ['build', '-tags', 'server', '-o', SERVER_BIN, '.'], join(FRONTEND_DIR, '..'), {
    CGO_ENABLED: '0',
  });
}

function run(cmd: string, args: string[], cwd: string, env: NodeJS.ProcessEnv = {}) {
  try {
    execFileSync(cmd, args, { cwd, stdio: 'pipe', env: { ...process.env, ...env } });
  } catch (e) {
    const err = e as { stdout?: Buffer; stderr?: Buffer };
    throw new Error(
      `${cmd} ${args.join(' ')} failed:\n${err.stdout?.toString() ?? ''}${err.stderr?.toString() ?? ''}`
    );
  }
}
