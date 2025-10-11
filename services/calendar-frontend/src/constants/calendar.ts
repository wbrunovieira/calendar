/**
 * Calendar constants
 * Centralized constants for calendar functionality
 */

export const MONTH_NAMES = [
  'Janeiro',
  'Fevereiro',
  'Março',
  'Abril',
  'Maio',
  'Junho',
  'Julho',
  'Agosto',
  'Setembro',
  'Outubro',
  'Novembro',
  'Dezembro',
] as const;

export const DAYS_OF_WEEK_SHORT = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'] as const;

export const DAYS_OF_WEEK_FULL = [
  'Domingo',
  'Segunda',
  'Terça',
  'Quarta',
  'Quinta',
  'Sexta',
  'Sábado',
] as const;

export const VIEW_MODES = {
  MONTH: 'month',
  WEEK: 'week',
  THREE_DAYS: '3days',
  DAY: 'day',
} as const;

export type ViewMode = (typeof VIEW_MODES)[keyof typeof VIEW_MODES];

export const DEFAULT_EVENT_TIME = '09:00';

export const DATE_RANGE_CONFIG = {
  PAST_MONTHS: 3,
  FUTURE_YEARS: 1,
} as const;

export const SEARCH_CONFIG = {
  MIN_QUERY_LENGTH: 2,
  DEBOUNCE_MS: 300,
} as const;
