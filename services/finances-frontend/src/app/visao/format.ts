import type { Trend } from './types';

export const fmt = (v: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v);

export const fmtShort = (v: number) => {
  const a = Math.abs(v);
  const s = a >= 1000 ? `${(a / 1000).toFixed(1)}k` : a.toFixed(0);
  return v < 0 ? `-${s}` : s;
};

export const monthLabel = (period: string) => {
  const [, m] = period.split('-').map(Number);
  return ['', 'Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'][m] ?? period;
};

export const trendIcon = (t: Trend) => (t === 'up' ? '▲' : t === 'down' ? '▼' : '▬');

// For expenses, rising = bad (rose); for revenue we pass invert.
export const trendColor = (t: Trend, invert = false) => {
  if (t === 'flat') return 'text-white/30';
  const good = invert ? t === 'up' : t === 'down';
  return good ? 'text-emerald-400' : 'text-rose-400';
};
