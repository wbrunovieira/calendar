export const transactionStatusConfig: Record<string, { label: string; color: string }> = {
  PLANNED: { label: 'Planejado', color: 'bg-blue-500/20 text-blue-400' },
  CONFIRMED: { label: 'Confirmado', color: 'bg-emerald-500/20 text-emerald-400' },
  CANCELLED: { label: 'Cancelado', color: 'bg-red-500/20 text-red-400' },
};

export const transactionStatusStyles: Record<string, string> = {
  PLANNED: 'bg-white/10 text-white',
  CONFIRMED: 'bg-emerald-500/20 text-emerald-200 border border-emerald-500/50',
  CANCELLED: 'bg-rose-500/20 text-rose-200 border border-rose-500/30',
};

export const CHART_COLORS = [
  '#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1',
  '#14b8a6', '#a855f7', '#eab308', '#22c55e', '#0ea5e9',
];
