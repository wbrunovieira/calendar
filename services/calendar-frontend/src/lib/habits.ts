import type { Event, HabitStats, HabitTimeOfDay } from '@/types/calendar';

export type HeatCell = { date: string; value: number | null };

export const TIME_OF_DAY: { key: HabitTimeOfDay; label: string }[] = [
  { key: 'MORNING', label: 'Manhã' },
  { key: 'ANYTIME', label: 'Qualquer hora' },
  { key: 'EVENING', label: 'Noite' },
];

/** Backend values are English; labels are what the UI shows. */
export const SLIP_TRIGGERS: { value: string; label: string }[] = [
  { value: 'Social', label: 'Social' },
  { value: 'Stress', label: 'Estresse' },
  { value: 'Boredom', label: 'Tédio' },
  { value: 'Tired', label: 'Cansaço' },
  { value: 'Celebration', label: 'Comemoração' },
  { value: 'Craving', label: 'Vontade' },
];

export const triggerLabel = (value: string | null | undefined) =>
  SLIP_TRIGGERS.find((t) => t.value === value)?.label ?? value ?? '';

export interface HabitIdea {
  title: string;
  habitKind: 'BUILD' | 'QUIT';
  dailyTarget?: number;
  targetUnit?: string;
  timeOfDay?: HabitTimeOfDay;
}

export const HABIT_IDEAS: HabitIdea[] = [
  { title: 'Meditar', habitKind: 'BUILD', dailyTarget: 10, targetUnit: 'min', timeOfDay: 'MORNING' },
  { title: 'Alongar', habitKind: 'BUILD', dailyTarget: 10, targetUnit: 'min', timeOfDay: 'EVENING' },
  { title: 'Ler', habitKind: 'BUILD', dailyTarget: 20, targetUnit: 'páginas', timeOfDay: 'EVENING' },
  { title: 'Diário', habitKind: 'BUILD', timeOfDay: 'EVENING' },
  { title: 'Sol pela manhã', habitKind: 'BUILD', dailyTarget: 10, targetUnit: 'min', timeOfDay: 'MORNING' },
  { title: 'Passos', habitKind: 'BUILD', dailyTarget: 10000, targetUnit: 'passos' },
  { title: 'Sem álcool', habitKind: 'QUIT' },
  { title: 'Sem celular na cama', habitKind: 'QUIT' },
  { title: 'Sem refrigerante', habitKind: 'QUIT' },
  { title: 'Sem fast food', habitKind: 'QUIT' },
  { title: 'Sem doomscrolling', habitKind: 'QUIT' },
];

export const isQuitHabit = (h: Pick<Event, 'habitKind'>) => h.habitKind === 'QUIT';

/**
 * Habits that belong in a daily done/not-done list. Quit habits are tracked by slips and
 * paused habits are hidden, so views built on checkboxes (day view, dashboard) skip them.
 */
export const isCheckableHabit = (h: { habitKind?: string | null; isPaused?: boolean }) =>
  h.habitKind !== 'QUIT' && !h.isPaused;

export function groupByTimeOfDay<T extends Pick<Event, 'timeOfDay'>>(habits: T[]) {
  return TIME_OF_DAY.map(({ key, label }) => ({
    key,
    label,
    habits: habits.filter((h) => (h.timeOfDay || 'ANYTIME') === key),
  })).filter((g) => g.habits.length > 0);
}

export const amountStep = (target: number) => (target >= 1000 ? 500 : target >= 10 ? 5 : 1);

const dayOf = (iso: string) => iso.split('T')[0];

export const loggedAmount = (habit: Event, date: string) =>
  habit.executions?.find((e) => dayOf(e.executionDate) === date)?.value ?? 0;

/** Share of active build habits done each day; null where none was scheduled. */
export function aggregateHeatmap(stats: HabitStats[]): HeatCell[] {
  const active = stats.filter((s) => s.habitKind !== 'QUIT' && !s.isPaused && s.heatmap?.length);
  if (!active.length) return [];
  return active[0].heatmap.map(({ date }, i) => {
    const values = active
      .map((s) => s.heatmap[i]?.value)
      .filter((v): v is number => v !== null && v !== undefined);
    return { date, value: values.length ? values.reduce((a, b) => a + b, 0) / values.length : null };
  });
}

/** Splits cells into Monday-first columns of 7, padding with null outside the range. */
export function heatmapWeeks(cells: HeatCell[]): (HeatCell | null)[][] {
  if (!cells.length) return [];
  const [y, m, d] = cells[0].date.split('-').map(Number);
  const lead = (new Date(y, m - 1, d).getDay() + 6) % 7; // Monday = 0
  const padded: (HeatCell | null)[] = [...Array(lead).fill(null), ...cells];
  while (padded.length % 7) padded.push(null);
  const weeks: (HeatCell | null)[][] = [];
  for (let i = 0; i < padded.length; i += 7) weeks.push(padded.slice(i, i + 7));
  return weeks;
}
