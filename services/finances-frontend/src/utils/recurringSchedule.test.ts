import { describe, expect, it } from 'vitest';
import { scheduleForSave } from './recurringSchedule';
import type { RecurringTransaction } from '@/types/finances';

const stored = (over: Partial<RecurringTransaction> = {}): RecurringTransaction =>
  ({
    id: 'r1',
    description: 'Tim Celular',
    recurrenceRule: 'FREQ=MONTHLY;BYMONTHDAY=20',
    startOn: '2026-01-20T00:00:00Z',
    nextOccurrence: '2026-12-20T00:00:00Z',
    amount: 99.39,
    status: 'ACTIVE',
    ...over,
  }) as RecurringTransaction;

// PUT /recurring-transactions is a FULL REPLACE, and the edit form was sending
// `nextOccurrence: startOn`. Opening a row just to fix the amount rewrote the
// schedule to the start date — on Tim Celular, stored 2026-12-20 with startOn
// 2026-01-20, saving a new amount would move the next run eleven months into the
// past. The API used to hide it by recomputing the date before answering; it no
// longer does, so the damage is now visible on screen and in the generator.
describe('scheduleForSave on an edit', () => {
  it('keeps the stored next occurrence when only the amount changed', () => {
    const schedule = scheduleForSave({ startOn: '2026-01-20', frequency: 'MONTHLY' }, stored());

    expect(schedule.nextOccurrence).toBe('2026-12-20');
  });

  it('keeps the stored rule, so a day-of-month survives the edit', () => {
    const schedule = scheduleForSave({ startOn: '2026-01-20', frequency: 'MONTHLY' }, stored());

    expect(schedule.recurrenceRule).toBe('FREQ=MONTHLY;BYMONTHDAY=20');
  });

  // Rebuilding the rule as FREQ=MONTHLY dropped BYMONTHDAY, which silently moved
  // the recurrence to the day of startOn.
  it('rebuilds the rule only when the frequency itself changed', () => {
    const schedule = scheduleForSave({ startOn: '2026-01-20', frequency: 'WEEKLY' }, stored());

    expect(schedule.recurrenceRule).toBe('FREQ=WEEKLY');
  });

  // A rule this helper does not recognise is the strongest possible reason to
  // leave it alone. The edit form collapses an unknown rule to MONTHLY in its
  // dropdown, so rebuilding would turn a FREQ=YEARLY recurrence into a monthly
  // one on the way out — the form would have "changed the frequency" without
  // anyone touching it.
  it('leaves a rule it does not understand exactly as it was', () => {
    const schedule = scheduleForSave(
      { startOn: '2026-01-20', frequency: 'MONTHLY' },
      stored({ recurrenceRule: 'FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=20' }),
    );

    expect(schedule.recurrenceRule).toBe('FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=20');
  });

  it('survives a stored date that is a bare day with no time', () => {
    const schedule = scheduleForSave(
      { startOn: '2026-01-20', frequency: 'MONTHLY' },
      stored({ nextOccurrence: '2026-12-20' }),
    );

    expect(schedule.nextOccurrence).toBe('2026-12-20');
  });
});

describe('scheduleForSave on a new recurrence', () => {
  it('starts on the date given', () => {
    const schedule = scheduleForSave({ startOn: '2026-11-15', frequency: 'MONTHLY' });

    expect(schedule).toEqual({ nextOccurrence: '2026-11-15', recurrenceRule: 'FREQ=MONTHLY' });
  });

  it('does not borrow a rule from nowhere', () => {
    expect(scheduleForSave({ startOn: '2026-11-15', frequency: 'DAILY' }).recurrenceRule).toBe('FREQ=DAILY');
  });
});
