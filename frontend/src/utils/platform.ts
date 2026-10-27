export type Platform = 'mac' | 'windows' | 'linux';

export const platform: Platform = /Windows/.test(navigator.userAgent)
  ? 'windows'
  : /Macintosh/.test(navigator.userAgent)
    ? 'mac'
    : 'linux';
