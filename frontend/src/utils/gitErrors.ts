export interface DescribedError {
  message: string;
  detail?: string;
}

type Rule = { test: RegExp; message: (m: RegExpMatchArray) => string };

// Ordered: the first match wins, so specific causes come before generic ones.
const RULES: Rule[] = [
  {
    test: /index\.lock|Another git process seems to be running/i,
    message: () =>
      'Another git process is using this repository. Wait for it to finish, or remove .git/index.lock if it crashed.',
  },
  {
    test: /detected dubious ownership in repository at '([^']+)'/i,
    message: (m) =>
      `Git doesn't trust this folder because another user owns it. Run: git config --global --add safe.directory "${m[1]}"`,
  },
  {
    test: /executable file not found|"git": file does not exist/i,
    message: () =>
      "Burlwood can't find git. Install git, or add it to your PATH, and restart the app.",
  },
  {
    test: /isn't inside a git repository|not a git repository/i,
    message: () =>
      "That folder isn't a git repository. Pick the repository's folder, or run git init.",
  },
  {
    test: /has no upstream branch|no upstream configured/i,
    message: () => "This branch doesn't track a remote branch yet. Push it once to set one.",
  },
  {
    test: /\[rejected\][\s\S]*(non-fast-forward|fetch first)|Updates were rejected/i,
    message: () => "The remote has commits you don't have yet. Pull first, then push again.",
  },
  {
    test: /would be overwritten by (checkout|merge)/i,
    message: (m) =>
      `Your local changes would be overwritten by the ${m[1]}. Commit or stash them first.`,
  },
  {
    test: /untracked working tree files would be overwritten/i,
    message: () => 'Untracked files would be overwritten. Move, commit or remove them first.',
  },
  {
    test: /patch does not apply|corrupt patch|patch failed/i,
    message: () => 'That change no longer matches the file on disk. Refresh and try again.',
  },
  {
    test: /nothing to commit|no changes added to commit/i,
    message: () => 'There is nothing staged to commit.',
  },
  {
    test: /Authentication failed|Permission denied \(publickey|could not read Username/i,
    message: () => 'The remote rejected your credentials. Check your SSH key or access token.',
  },
  {
    test: /Could not resolve host|Could not read from remote|Connection (timed out|refused)|unable to access/i,
    message: () => "Couldn't reach the remote. Check your connection and the remote URL.",
  },
  {
    test: /CONFLICT|could not apply|Merge conflict/,
    message: () => 'Stopped on a conflict. Resolve it in the Working Tree, then continue.',
  },
  {
    test: /pathspec '([^']+)' did not match/i,
    message: (m) => `${m[1]} isn't in the working tree any more. Refresh the status.`,
  },
  {
    test: /not a valid (object|branch) name|unknown revision|bad revision/i,
    message: () => "That commit or branch doesn't exist (any more). Refresh and try again.",
  },
];

// Turns raw "git exited 128: fatal: …" output into a sentence with a next step; the raw text
// stays available as detail.
export function describeGitError(err: unknown): DescribedError {
  const raw = String(err instanceof Error ? err.message : err).trim();
  for (const rule of RULES) {
    const m = raw.match(rule.test);
    if (m) {
      return { message: rule.message(m), detail: raw };
    }
  }
  const cleaned = raw
    .replace(/^Error:\s*/, '')
    .replace(/^git exited \d+:\s*/, '')
    .replace(/^(fatal|error):\s*/i, '');
  const firstLine = cleaned.split('\n').find((l) => l.trim()) ?? cleaned;
  const message = firstLine.charAt(0).toUpperCase() + firstLine.slice(1);
  return cleaned.includes('\n') || cleaned !== raw ? { message, detail: raw } : { message };
}
