export const repo = 'https://github.com/stijnie2210/burlwood';

const fallbackVersion = 'v0.4.0';

export const downloads = {
  mac: `${repo}/releases/latest/download/burlwood-mac-universal.dmg`,
  linux: `${repo}/releases/latest/download/burlwood-linux-amd64.AppImage`,
  linuxArm: `${repo}/releases/latest/download/burlwood-linux-arm64.AppImage`,
  windows: `${repo}/releases/latest/download/burlwood-windows-amd64.zip`,
  all: `${repo}/releases/latest`,
};

export async function latestVersion(): Promise<string> {
  try {
    const res = await fetch('https://api.github.com/repos/stijnie2210/burlwood/releases/latest', {
      headers: { Accept: 'application/vnd.github+json' },
      signal: AbortSignal.timeout(4000),
    });
    if (!res.ok) {
      return fallbackVersion;
    }
    const body = (await res.json()) as { tag_name?: string };
    return body.tag_name ?? fallbackVersion;
  } catch {
    return fallbackVersion;
  }
}
