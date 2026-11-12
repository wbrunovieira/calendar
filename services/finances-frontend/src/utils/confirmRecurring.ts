/**
 * The two dates a payment confirmed from a recurrence carries.
 *
 * `occurredOn` is when the money actually moved — today, because that is when
 * the button was pressed — and `dueOn` keeps when it was owed. They used to be
 * the same field: the write was dated with the recurrence's next occurrence,
 * which the API always reported as today or later, so it was today by accident.
 *
 * With overdue recurrences now visible and confirmable, dating the write by the
 * obligation would post into a past month, and there is no closed-month guard on
 * the way in.
 */
export function datesForConfirmedPayment(
  nextOccurrence: string,
  today: string,
): { occurredOn: string; dueOn: string } {
  return { occurredOn: today, dueOn: nextOccurrence.slice(0, 10) };
}
