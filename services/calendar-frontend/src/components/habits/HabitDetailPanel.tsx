'use client';

import type { HabitStats } from '@/types/calendar';
import HabitHeatmap from './HabitHeatmap';

/** Expanded view of one habit: streaks, 30-day rate, totals and its 16-week heatmap. */
export default function HabitDetailPanel({
  stats,
  onTogglePause,
}: {
  stats: HabitStats;
  onTogglePause: () => void;
}) {
  const quit = stats.habitKind === 'QUIT';
  const cells: { label: string; value: string }[] = [
    { label: quit ? 'Dias limpos agora' : 'Sequência atual (dias)', value: String(stats.currentStreak) },
    { label: 'Melhor sequência', value: String(stats.longestStreak) },
    {
      label: quit ? 'Dias limpos, últimos 30' : 'Conclusão, últimos 30 dias',
      value: stats.completionRate30d === null ? '–' : `${stats.completionRate30d}%`,
    },
  ];
  if (stats.totalAmount !== null) {
    cells.push({
      label: `Total ${stats.targetUnit ?? ''}`.trim(),
      value: stats.totalAmount.toLocaleString('pt-BR'),
    });
  }

  return (
    <div className="mt-3 rounded-xl bg-black/20 p-4 space-y-4">
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {cells.map((c) => (
          <div key={c.label}>
            <div className="text-xl font-bold text-white tabular-nums">{c.value}</div>
            <div className="text-xs text-white/50">{c.label}</div>
          </div>
        ))}
      </div>
      <HabitHeatmap cells={stats.heatmap} />
      <button
        type="button"
        onClick={onTogglePause}
        className="px-3 py-1.5 rounded-lg text-sm bg-white/10 text-white/80 hover:bg-white/20"
      >
        {stats.isPaused ? 'Retomar' : 'Pausar'}
      </button>
    </div>
  );
}
