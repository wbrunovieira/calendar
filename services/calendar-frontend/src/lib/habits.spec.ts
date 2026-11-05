import { describe, it, expect } from 'vitest';
import type { Event, HabitStats } from '@/types/calendar';
import {
  aggregateHeatmap,
  amountStep,
  groupByTimeOfDay,
  heatmapWeeks,
  isQuitHabit,
  isCheckableHabit,
  loggedAmount,
} from './habits';

const habit = (overrides: Partial<Event> = {}): Event =>
  ({ id: 'h', calendarId: 'c', title: 'H', startTime: '09:00', startDate: '2026-10-01', isRecurring: true, isActive: true, ...overrides }) as Event;

const stats = (overrides: Partial<HabitStats> = {}): HabitStats =>
  ({ habitId: 'h', habitKind: 'BUILD', isPaused: false, heatmap: [], ...overrides }) as HabitStats;

describe('habits helpers', () => {
  it('treats only habitKind QUIT as a quit habit', () => {
    expect(isQuitHabit(habit({ habitKind: 'QUIT' }))).toBe(true);
    expect(isQuitHabit(habit({ habitKind: null }))).toBe(false);
    expect(isQuitHabit(habit())).toBe(false);
  });

  it('groups habits by time of day in morning, any time, evening order and skips empty groups', () => {
    const groups = groupByTimeOfDay([
      habit({ id: 'a', timeOfDay: 'EVENING' }),
      habit({ id: 'b' }),
      habit({ id: 'c', timeOfDay: 'EVENING' }),
    ]);

    expect(groups.map((g) => [g.key, g.habits.map((h) => h.id)])).toEqual([
      ['ANYTIME', ['b']],
      ['EVENING', ['a', 'c']],
    ]);
    expect(groups[0].label).toBe('Qualquer hora');
  });

  it('picks a stepper increment that fits the size of the target', () => {
    expect(amountStep(8)).toBe(1);
    expect(amountStep(20)).toBe(5);
    expect(amountStep(10000)).toBe(500);
  });

  it('reads the amount logged for a given day', () => {
    const h = habit({
      executions: [
        { id: '1', eventId: 'h', executionDate: '2026-10-12T00:00:00.000Z', completed: false, value: 12 },
        { id: '2', eventId: 'h', executionDate: '2026-10-11', completed: true, value: 20 },
      ],
    });
    expect(loggedAmount(h, '2026-10-12')).toBe(12);
    expect(loggedAmount(h, '2026-10-10')).toBe(0);
  });

  it('averages the heatmap of active build habits per day, null when none was scheduled', () => {
    const merged = aggregateHeatmap([
      stats({ heatmap: [{ date: 'd1', value: 1 }, { date: 'd2', value: null }] }),
      stats({ heatmap: [{ date: 'd1', value: 0 }, { date: 'd2', value: null }] }),
      stats({ habitKind: 'QUIT', heatmap: [{ date: 'd1', value: 0 }, { date: 'd2', value: 0 }] }),
      stats({ isPaused: true, heatmap: [{ date: 'd1', value: 0 }, { date: 'd2', value: 1 }] }),
    ]);

    expect(merged).toEqual([
      { date: 'd1', value: 0.5 },
      { date: 'd2', value: null },
    ]);
  });

  it('lays the heatmap out in Monday-first week columns, padding the first week', () => {
    // 2026-10-07 is a Wednesday
    const cells = ['2026-10-07', '2026-10-08', '2026-10-09', '2026-10-10', '2026-10-11', '2026-10-12'].map(
      (date) => ({ date, value: 1 }),
    );

    const weeks = heatmapWeeks(cells);

    expect(weeks).toHaveLength(2);
    expect(weeks[0].map((c) => c?.date ?? null)).toEqual([
      null, null, '2026-10-07', '2026-10-08', '2026-10-09', '2026-10-10', '2026-10-11',
    ]);
    expect(weeks[1].map((c) => c?.date ?? null)).toEqual(['2026-10-12', null, null, null, null, null, null]);
  });

  it('only build habits that are not paused show up as daily checkboxes', () => {
    expect(isCheckableHabit(habit())).toBe(true);
    expect(isCheckableHabit(habit({ habitKind: 'QUIT' }))).toBe(false);
    expect(isCheckableHabit(habit({ isPaused: true }))).toBe(false);
    expect(isCheckableHabit({ habitKind: 'QUIT', isPaused: false })).toBe(false);
  });
});
