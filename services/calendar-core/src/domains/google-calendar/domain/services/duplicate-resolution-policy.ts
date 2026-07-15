export interface DuplicateCandidate {
  id: string;
  createdAt: Date;
  hasUserData: boolean;
}

export type Resolution =
  | { kind: 'resolved'; survivor: DuplicateCandidate; losers: DuplicateCandidate[] }
  | { kind: 'conflict'; reason: string };

/**
 * Decides which copy of a duplicated event to keep, among copies none of which is linked to Google
 * yet. (When one copy already carries the googleEventId, the UNIQUE index has already decided for
 * us and this policy is not consulted.)
 *
 * Deleting an event cascades to its completions, reminders, recurrence overrides and derived
 * instances, and the sync cannot reconstruct any of it. So the rule is conservative: a copy
 * carrying user data wins over an older but bare one, and if two copies both carry user data we
 * refuse to choose and hand the decision to a human.
 */
export class DuplicateResolutionPolicy {
  static resolve(candidates: DuplicateCandidate[]): Resolution {
    const curated = candidates.filter((c) => c.hasUserData);

    if (curated.length > 1) {
      return {
        kind: 'conflict',
        reason: `${curated.length} copies carry user data (category/label/reminders/completions/overrides); a human must pick the survivor`,
      };
    }

    const oldestFirst = [...candidates].sort(
      (a, b) => a.createdAt.getTime() - b.createdAt.getTime(),
    );
    const survivor = curated[0] ?? oldestFirst[0];

    return {
      kind: 'resolved',
      survivor,
      losers: candidates.filter((c) => c.id !== survivor.id),
    };
  }
}
