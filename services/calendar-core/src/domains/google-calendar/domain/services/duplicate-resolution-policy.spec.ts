import { describe, expect, it } from 'vitest';
import { DuplicateResolutionPolicy, DuplicateCandidate } from './duplicate-resolution-policy';

const candidate = (
  id: string,
  createdAt: string,
  hasUserData = false,
): DuplicateCandidate => ({ id, createdAt: new Date(createdAt), hasUserData });

describe('DuplicateResolutionPolicy', () => {
  it('keeps the only copy when there is nothing to choose between', () => {
    const only = candidate('evt-a', '2026-07-11T13:10:00Z');

    const result = DuplicateResolutionPolicy.resolve([only]);

    expect(result).toEqual({ kind: 'resolved', survivor: only, losers: [] });
  });

  it('keeps the oldest copy when none carries history', () => {
    const older = candidate('evt-older', '2026-07-11T13:10:00Z');
    const newer = candidate('evt-newer', '2026-07-12T17:50:00Z');

    const result = DuplicateResolutionPolicy.resolve([newer, older]);

    expect(result).toEqual({ kind: 'resolved', survivor: older, losers: [newer] });
  });

  it('keeps the copy carrying history even when it is the newest', () => {
    const bare = candidate('evt-bare', '2026-07-11T13:10:00Z');
    const withHistory = candidate('evt-history', '2026-07-12T17:50:00Z', true);

    const result = DuplicateResolutionPolicy.resolve([bare, withHistory]);

    expect(result).toEqual({ kind: 'resolved', survivor: withHistory, losers: [bare] });
  });

  it('refuses to choose when two copies both carry history', () => {
    const result = DuplicateResolutionPolicy.resolve([
      candidate('evt-a', '2026-07-11T13:10:00Z', true),
      candidate('evt-b', '2026-07-12T17:50:00Z', true),
    ]);

    expect(result.kind).toBe('conflict');
  });

  it('reports every loser when more than two copies exist', () => {
    const survivor = candidate('evt-1', '2026-07-10T10:00:00Z');
    const result = DuplicateResolutionPolicy.resolve([
      candidate('evt-3', '2026-07-12T10:00:00Z'),
      survivor,
      candidate('evt-2', '2026-07-11T10:00:00Z'),
    ]);

    expect(result).toMatchObject({ kind: 'resolved', survivor });
    expect(result.kind === 'resolved' && result.losers.map((l) => l.id)).toEqual([
      'evt-3',
      'evt-2',
    ]);
  });
});
