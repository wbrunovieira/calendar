'use client';

import type { FinancialSummary, CapitalSummary } from './types';
import { fmt, fmtShort, monthLabel } from './format';
import { CategoryTable } from './CategoryTable';

export function BusinessView({ data, capital }: { data: FinancialSummary | null; capital: CapitalSummary | null }) {
  if (!data || data.periods.length === 0)
    return <p className="text-white/50 py-10 text-center">Sem dados no período.</p>;

  const maxAbs = Math.max(...data.resultByMonth.map((v) => Math.abs(v)), 1);
  const profit = data.totalResult >= 0;

  return (
    <div className="space-y-5">
      {/* Headline resultado */}
      <section className="bg-white/5 border border-white/10 rounded-2xl p-5">
        <div className="flex items-baseline justify-between mb-1">
          <h2 className="text-white font-semibold">Resultado por mês</h2>
          <span className="text-white/50 text-sm">margem média {data.avgMargin.toFixed(0)}%</span>
        </div>
        <p className={`text-2xl font-bold ${profit ? 'text-emerald-400' : 'text-rose-400'}`}>
          {fmt(data.totalResult)} <span className="text-sm font-normal text-white/50">no período ({profit ? 'lucro' : 'prejuízo'})</span>
        </p>
        {capital && capital.outstandingDebt > 0 && (
          <p className="mt-2 text-sm">
            <span className="text-amber-300 font-medium">Dívida ao sócio: {fmt(capital.outstandingDebt)}</span>
            <span className="text-white/50"> — a empresa deve isso a você (aportes a devolver)</span>
          </p>
        )}
        <div className="flex items-end gap-2 h-36 mt-4">
          {data.resultByMonth.map((v, i) => {
            const positive = v >= 0;
            return (
              <div key={i} className="flex-1 flex flex-col items-center justify-end gap-1 h-full">
                <span className={`text-[11px] ${positive ? 'text-emerald-400' : 'text-rose-400'}`}>{fmtShort(v)}</span>
                <div
                  className={`w-full rounded-t ${positive ? 'bg-emerald-500/60' : 'bg-rose-500/60'}`}
                  style={{ height: `${Math.max((Math.abs(v) / maxAbs) * 100, 2)}%` }}
                  title={fmt(v)}
                />
                <span className="text-[11px] text-white/50">{monthLabel(data.periods[i])}</span>
              </div>
            );
          })}
        </div>
      </section>

      {/* DRE — cascata */}
      <section className="bg-white/5 border border-white/10 rounded-2xl p-5 overflow-x-auto">
        <h2 className="text-white font-semibold mb-3">DRE — demonstração de resultado</h2>
        <table className="w-full text-sm min-w-[560px]">
          <thead>
            <tr className="text-white/50 text-xs">
              <th className="text-left font-medium pb-2">&nbsp;</th>
              {data.periods.map((p, i) => (
                <th key={i} className="text-right font-medium pb-2 px-2">{monthLabel(p)}</th>
              ))}
              <th className="text-right font-medium pb-2 px-2">total</th>
            </tr>
          </thead>
          <tbody>
            {data.dre.map((line) => {
              const rev = line.kind === 'revenue';
              return (
                <tr key={line.classification} className="border-t border-white/5">
                  <td className={`py-1.5 ${rev ? 'text-emerald-300/80' : 'text-rose-300/80'}`}>
                    {rev ? '(+)' : '(−)'} {line.label}
                  </td>
                  {line.byMonth.map((v, i) => (
                    <td key={i} className="text-right px-2 py-1.5 text-white/70 tabular-nums">{v ? fmtShort(v) : '·'}</td>
                  ))}
                  <td className="text-right px-2 py-1.5 text-white/90 font-medium tabular-nums">{fmtShort(line.total)}</td>
                </tr>
              );
            })}
            <tr className="border-t border-white/15">
              <td className="py-2 text-white font-semibold">(=) Resultado</td>
              {data.resultByMonth.map((v, i) => (
                <td key={i} className={`text-right px-2 py-2 font-medium tabular-nums ${v >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>{fmtShort(v)}</td>
              ))}
              <td className={`text-right px-2 py-2 font-bold tabular-nums ${data.totalResult >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>{fmtShort(data.totalResult)}</td>
            </tr>
            <tr>
              <td className="py-1 text-white/50 text-xs">Margem</td>
              {data.marginByMonth.map((v, i) => {
                const inc = data.revenueByMonth[i] + data.financialIncomeByMonth[i];
                const label = !inc ? '·' : Math.abs(v) >= 1000 ? '—' : `${v.toFixed(0)}%`;
                return (
                  <td key={i} className="text-right px-2 py-1 text-white/40 text-xs tabular-nums">{label}</td>
                );
              })}
              <td className="text-right px-2 py-1 text-white/50 text-xs tabular-nums">{data.avgMargin.toFixed(0)}%</td>
            </tr>
          </tbody>
        </table>
      </section>

      <CategoryTable title="Onde está a despesa" categories={data.expenseCategories} periods={data.periods} />
      <CategoryTable title="De onde vem a receita" categories={data.revenueCategories} periods={data.periods} invertTrend />
    </div>
  );
}
