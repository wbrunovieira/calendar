import type { RecurringTransaction } from '@/types/finances';

export type Frequency = 'DAILY' | 'WEEKLY' | 'MONTHLY';

interface ScheduleForm {
  startOn: string;
  frequency: Frequency;
}

interface Schedule {
  nextOccurrence: string;
  recurrenceRule: string;
}

const frequencyOf = (rule: string): Frequency | null => {
  if (rule.includes('DAILY')) return 'DAILY';
  if (rule.includes('WEEKLY')) return 'WEEKLY';
  if (rule.includes('MONTHLY')) return 'MONTHLY';
  return null;
};

const dayOnly = (date: string) => date.slice(0, 10);

/**
 * The schedule half of the recurrence payload, which the save path must not
 * invent.
 *
 * `PUT /recurring-transactions/{id}` is a FULL REPLACE, and the form used to
 * send `nextOccurrence: startOn` with the rule rebuilt as `FREQ=<frequency>`.
 * Opening a row only to correct an amount therefore rewrote the schedule: on Tim
 * Celular — stored next run 2026-12-20, startOn 2026-01-20, rule
 * FREQ=MONTHLY;BYMONTHDAY=20 — saving moved the next run eleven months into the
 * past and dropped the day-of-month, silently relocating the recurrence to the
 * day of startOn.
 *
 * The API used to conceal that by recomputing the date before answering. It
 * reports the stored state now, so the damage would be visible on the screen and
 * real in the generator.
 */
export function scheduleForSave(form: ScheduleForm, editing?: RecurringTransaction): Schedule {
  if (!editing) {
    return { nextOccurrence: form.startOn, recurrenceRule: `FREQ=${form.frequency}` };
  }

  // Changing the frequency is a deliberate reschedule, so the rule is rebuilt.
  // Anything else leaves the stored rule alone, qualifiers included.
  //
  // A rule whose frequency this does not RECOGNISE is left alone too. The edit
  // form shows only DAILY/WEEKLY/MONTHLY and collapses anything else to MONTHLY,
  // so treating "unknown" as "changed" would convert a FREQ=YEARLY recurrence to
  // monthly the moment someone opened it to fix a typo.
  const storedFrequency = frequencyOf(editing.recurrenceRule);
  const frequencyChanged = storedFrequency !== null && storedFrequency !== form.frequency;

  return {
    nextOccurrence: dayOnly(editing.nextOccurrence),
    recurrenceRule: frequencyChanged ? `FREQ=${form.frequency}` : editing.recurrenceRule,
  };
}
