export type Platform = 'mac' | 'windows' | 'linux';

export const platform: Platform = /Windows/.test(navigator.userAgent)
  ? 'windows'
  : /Macintosh/.test(navigator.userAgent)
    ? 'mac'
    : 'linux';

export function shortcut(key: string, shift = false): string {
  if (platform === 'mac') {
    return `${shift ? '⇧' : ''}⌘${key}`;
  }
  return `Ctrl+${shift ? 'Shift+' : ''}${key}`;
}
