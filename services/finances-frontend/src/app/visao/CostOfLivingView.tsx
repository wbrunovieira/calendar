'use client';

import { useMemo } from 'react';
import type { ExpenseAnalysis } from './types';
import { fmt, fmtShort, monthLabel } from './format';
import { CategoryTable } from './CategoryTable';

export function CostOfLivingView({ data }: { data: ExpenseAnalysis | null }) {
  const maxTotal = data ? Math.max(...data.totalByMonth, 1) : 1;
  const halfTrend = useMemo(() => {
    if (!data || data.totalByMonth.length < 2) return null;
    const half = Math.floor(data.totalByMonth.length / 2);
    const avg = (a: number[]) => a.reduce((x, y) => x + y, 0) / (a.length || 1);
    const f = avg(data.totalByMonth.slice(0, half));
    const l = avg(data.totalByMonth.slice(half));
    return { pct: f === 0 ? 0 : ((l - f) / f) * 100, rising: l > f };
  }, [data]);

  if (!data || data.periods.length === 0)
    return <p className="text-white/50 py-10 text-center">Sem dados no período.</p>;

  return (
    <div className="space-y-5">
      <section className="bg-white/5 border border-white/10 rounded-2xl p-5">
        <div className="flex items-baseline justify-between mb-4">
          <h2 className="text-white font-semibold">Gasto total por mês</h2>
          <span className="text-white/50 text-sm">média {fmt(data.average)}</span>
        </div>
        <div className="flex items-end gap-2 h-40">
          {data.totalByMonth.map((v, i) => {
            const isLast = i === data.totalByMonth.length - 1;
            return (
              <div key={i} className="flex-1 flex flex-col items-center justify-end gap-1 h-full">
                <span className="text-[11px] text-white/60">{fmtShort(v)}</span>
                <div
                  className={`w-full rounded-t ${isLast ? 'bg-purple-400' : 'bg-purple-500/50'}`}
                  style={{ height: `${Math.max((v / maxTotal) * 100, 2)}%` }}
                  title={fmt(v)}
                />
                <span className="text-[11px] text-white/50">{monthLabel(data.periods[i])}</span>
              </div>
            );
          })}
        </div>
        {halfTrend && (
          <p className="text-sm mt-3">
            <span className={halfTrend.rising ? 'text-rose-400' : 'text-emerald-400'}>
              {halfTrend.rising ? '▲' : '▼'} {Math.abs(halfTrend.pct).toFixed(0)}%
            </span>
            <span className="text-white/50"> {halfTrend.rising ? 'subindo' : 'caindo'} (2ª metade vs 1ª metade do período)</span>
          </p>
        )}
      </section>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <p className="text-white/60 text-sm">Custos fixos (recorrentes) / mês</p>
          <p className="text-xl font-bold text-white mt-1">{fmt(data.fixedAverage)}</p>
        </div>
        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <p className="text-white/60 text-sm">Variáveis / mês</p>
          <p className="text-xl font-bold text-white mt-1">{fmt(data.variableAverage)}</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-rose-500/5 border border-rose-500/20 rounded-2xl p-4">
          <p className="text-rose-300 text-sm font-medium mb-2">▲ Maiores altas (vs média)</p>
          {(data.topUp ?? []).length === 0 ? (
            <p className="text-white/40 text-sm">—</p>
          ) : (
            (data.topUp ?? []).map((m, i) => (
              <div key={i} className="flex justify-between text-sm py-0.5">
                <span className="text-white/70 truncate mr-2">{m.name}</span>
                <span className="text-rose-400">+{fmt(m.delta)}</span>
              </div>
            ))
          )}
        </div>
        <div className="bg-emerald-500/5 border border-emerald-500/20 rounded-2xl p-4">
          <p className="text-emerald-300 text-sm font-medium mb-2">▼ Maiores quedas (vs média)</p>
          {(data.topDown ?? []).length === 0 ? (
            <p className="text-white/40 text-sm">—</p>
          ) : (
            (data.topDown ?? []).map((m, i) => (
              <div key={i} className="flex justify-between text-sm py-0.5">
                <span className="text-white/70 truncate mr-2">{m.name}</span>
                <span className="text-emerald-400">{fmt(m.delta)}</span>
              </div>
            ))
          )}
        </div>
      </div>

      <CategoryTable title="Onde está indo" categories={data.categories ?? []} periods={data.periods} />
    </div>
  );
}
