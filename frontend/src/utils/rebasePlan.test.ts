import { describe, it, expect } from 'vitest';
import {
  firstChangedIndex,
  planProblem,
  rewrittenPushedCount,
  squashTarget,
  type RebaseAction,
  type RebaseRow,
} from './rebasePlan';

function rows(...actions: RebaseAction[]): RebaseRow[] {
  return actions.map((action, i) => ({
    hash: `h${i}`,
    shortHash: `h${i}`,
    subject: `commit ${i}`,
    message: `commit ${i}`,
    author: 'a',
    pushed: i < 2,
    action,
    newMessage: `commit ${i}`,
  }));
}

describe('planProblem', () => {
  it('accepts a plan that keeps a commit before every squash', () => {
    expect(planProblem(rows('pick', 'squash', 'drop', 'fixup', 'reword'))).toBe('');
  });

  it('rejects a squash or fixup with nothing kept above it', () => {
    expect(planProblem(rows('squash', 'pick'))).toContain('h0 has no commit above it');
    expect(planProblem(rows('drop', 'fixup', 'pick'))).toContain('h1 has no commit above it');
  });

  it('rejects dropping everything', () => {
    expect(planProblem(rows('drop', 'drop'))).toContain('Keep at least one commit');
  });

  it('rejects a reword without a message', () => {
    const plan = rows('reword', 'pick');
    plan[0].newMessage = '  ';
    expect(planProblem(plan)).toContain('h0 needs a message');
  });
});

describe('squashTarget', () => {
  it('skips dropped and squashed rows above', () => {
    const plan = rows('pick', 'drop', 'squash', 'fixup');
    expect(squashTarget(plan, 3)?.hash).toBe('h0');
  });
});

describe('what gets rewritten', () => {
  const original = ['h0', 'h1', 'h2', 'h3'];

  it('is nothing while every commit is picked in place', () => {
    const plan = rows('pick', 'pick', 'pick', 'pick');
    expect(firstChangedIndex(plan, original)).toBe(4);
    expect(rewrittenPushedCount(plan, original)).toBe(0);
  });

  it('starts at the first changed row, so earlier pushed commits are safe', () => {
    const plan = rows('pick', 'reword', 'pick', 'pick');
    expect(firstChangedIndex(plan, original)).toBe(1);
    expect(rewrittenPushedCount(plan, original)).toBe(1);
  });

  it('counts a reorder as a change', () => {
    const plan = rows('pick', 'pick', 'pick', 'pick');
    [plan[0], plan[1]] = [plan[1], plan[0]];
    expect(rewrittenPushedCount(plan, original)).toBe(2);
  });
});
