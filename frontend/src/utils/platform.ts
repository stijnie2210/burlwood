export type Platform = 'mac' | 'windows' | 'linux';

export const platform: Platform = /Windows/.test(navigator.userAgent)
  ? 'windows'
  : /Macintosh/.test(navigator.userAgent)
    ? 'mac'
    : 'linux';

export function shortcut(key: string): string {
  return platform === 'mac' ? `⌘${key}` : `Ctrl+${key}`;
}
