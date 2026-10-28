import { describe, it, expect } from 'vitest';
import { describeGitError } from './gitErrors';

describe('describeGitError', () => {
  it('translates a missing upstream', () => {
    const r = describeGitError(
      'git exited 128: fatal: The current branch feature has no upstream branch.'
    );
    expect(r.message).toMatch(/doesn't track a remote branch/);
    expect(r.detail).toContain('no upstream branch');
  });

  it('translates checkout conflicts with local changes', () => {
    const r = describeGitError(
      'git exited 1: error: Your local changes to the following files would be overwritten by checkout:\n\ta.txt'
    );
    expect(r.message).toBe(
      'Your local changes would be overwritten by the checkout. Commit or stash them first.'
    );
  });

  it('names the worktree that holds a branch, for old and new git wording', () => {
    for (const raw of [
      "git exited 128: fatal: 'feature' is already used by worktree at '/repo/.claude/worktrees/fix'",
      "git exited 1: error: cannot delete branch 'feature' checked out at '/repo/.claude/worktrees/fix'",
    ]) {
      expect(describeGitError(raw).message).toMatch(/checked out in the worktree fix\./);
    }
  });

  it('recognises a non-repo folder from the backend', () => {
    expect(describeGitError("/tmp/x isn't inside a git repository").message).toMatch(
      /isn't a git repository/
    );
  });

  it('explains dubious ownership with the safe.directory command', () => {
    const r = describeGitError(
      "git exited 128: fatal: detected dubious ownership in repository at 'D:/code/app'\nTo add an exception..."
    );
    expect(r.message).toContain('git config --global --add safe.directory "D:/code/app"');
  });

  it('explains a missing git binary', () => {
    expect(describeGitError('exec: "git": executable file not found in %PATH%').message).toMatch(
      /can't find git/
    );
  });

  it('falls back to a cleaned first line and keeps the raw text', () => {
    const r = describeGitError('git exited 128: fatal: something unexpected\nmore output');
    expect(r.message).toBe('Something unexpected');
    expect(r.detail).toBe('git exited 128: fatal: something unexpected\nmore output');
  });

  it('leaves plain app messages alone', () => {
    expect(describeGitError('HEAD is detached; check out a branch to push it')).toEqual({
      message: 'HEAD is detached; check out a branch to push it',
    });
  });
});
