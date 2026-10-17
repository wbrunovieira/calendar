'use client';

import { heatmapWeeks, type HeatCell } from '@/lib/habits';

const DAY_LABELS = ['S', 'T', 'Q', 'Q', 'S', 'S', 'D'];

const fmt = (date: string) => date.split('-').reverse().slice(0, 2).join('/');

/** GitHub-style grid: one column per week (Monday on top), one square per day. */
export default function HabitHeatmap({
  cells,
  tone = 'emerald',
  label = 'Últimas 16 semanas',
}: {
  cells: HeatCell[];
  tone?: 'emerald' | 'purple';
  label?: string;
}) {
  const weeks = heatmapWeeks(cells);
  if (!weeks.length) return null;
  const rgb = tone === 'emerald' ? '16 185 129' : '168 85 247';

  return (
    <div className="overflow-x-auto" role="img" aria-label={label}>
      <div className="inline-flex gap-[3px]">
        <div className="flex flex-col gap-[3px] mr-1">
          {DAY_LABELS.map((d, i) => (
            <span key={i} className="h-[11px] text-[9px] leading-[11px] text-white/40">
              {i % 2 === 0 ? d : ''}
            </span>
          ))}
        </div>
        {weeks.map((week, wi) => (
          <div key={wi} className="flex flex-col gap-[3px]">
            {week.map((cell, di) => {
              if (!cell) return <span key={di} className="w-[11px] h-[11px]" />;
              const style =
                cell.value === null
                  ? { background: 'rgb(255 255 255 / 0.04)' }
                  : cell.value === 0
                    ? { background: 'rgb(255 255 255 / 0.12)' }
                    : { background: `rgb(${rgb} / ${0.35 + 0.65 * cell.value})` };
              const title =
                cell.value === null
                  ? `${fmt(cell.date)}: não programado`
                  : `${fmt(cell.date)}: ${Math.round(cell.value * 100)}%`;
              return <span key={di} title={title} className="w-[11px] h-[11px] rounded-[3px]" style={style} />;
            })}
          </div>
        ))}
      </div>
    </div>
  );
}
