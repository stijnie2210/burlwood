import { describe, it, expect } from 'vitest';
import { initials } from './initials';

describe('initials', () => {
  it('takes the first and last word', () => {
    expect(initials('Stijn Mommersteeg')).toBe('SM');
    expect(initials('Ada van der Berg')).toBe('AB');
  });

  it('uses one letter for a single name', () => {
    expect(initials('stijn')).toBe('S');
  });

  it('falls back for an empty name', () => {
    expect(initials('   ')).toBe('?');
  });
});
