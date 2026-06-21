'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';

type Trend = 'up' | 'down' | 'flat';

interface CategoryTrend {
  categoryId: string;
  name: string;
  byMonth: number[];
  total: number;
  average: number;
  deltaVsAvg: number;
  trend: Trend;
  children?: CategoryTrend[];
}
interface Mover {
  name: string;
  delta: number;
}
interface ExpenseAnalysis {
  periods: string[];
  totalByMonth: number[];
  average: number;
  fixedByMonth: number[];
  variableByMonth: number[];
  fixedAverage: number;
  variableAverage: number;
  categories: CategoryTrend[];
  topUp: Mover[];
  topDown: Mover[];
}

type ViewMode = 3 | 6 | 12;

const fmt = (v: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v);
const fmtShort = (v: number) => (v >= 1000 ? `${(v / 1000).toFixed(1)}k` : v.toFixed(0));

const monthLabel = (period: string) => {
  const [, m] = period.split('-').map(Number);
  return ['', 'Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'][m] ?? period;
};

// For EXPENSES: rising cost is bad (rose), falling is good (emerald).
const trendIcon = (t: Trend) => (t === 'up' ? '▲' : t === 'down' ? '▼' : '▬');
const trendColor = (t: Trend) => (t === 'up' ? 'text-rose-400' : t === 'down' ? 'text-emerald-400' : 'text-white/30');

export default function VisaoMensalPage() {
  const { selectedProfileId } = useProfile();
  const [basePeriod, setBasePeriod] = useState<string>(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 5);
    return d.toISOString().slice(0, 7);
  });
  const [viewMode, setViewMode] = useState<ViewMode>(6);
  const [data, setData] = useState<ExpenseAnalysis | null>(null);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const range = useMemo(() => {
    const [y, m] = basePeriod.split('-').map(Number);
    const from = new Date(y, m - 1, 1);
    const to = new Date(y, m - 1 + viewMode, 0); // last day of the last month
    const iso = (d: Date) =>
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    return { from: iso(from), to: iso(to) };
  }, [basePeriod, viewMode]);

  useEffect(() => {
    if (!selectedProfileId) return;
    (async () => {
      try {
        setLoading(true);
        const res = await api.get<{ data: ExpenseAnalysis }>(
          `/transactions/expense-analysis?profileId=${selectedProfileId}&from=${range.from}&to=${range.to}`,
        );
        setData(res.data);
      } catch (e) {
        console.warn('Erro ao carregar análise de despesas', e);
        setData(null);
      } finally {
        setLoading(false);
      }
    })();
  }, [selectedProfileId, range]);

  const toggle = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const maxTotal = data ? Math.max(...data.totalByMonth, 1) : 1;
  const halfTrend = useMemo(() => {
    if (!data || data.totalByMonth.length < 2) return null;
    const half = Math.floor(data.totalByMonth.length / 2);
    const avg = (a: number[]) => a.reduce((x, y) => x + y, 0) / (a.length || 1);
    const f = avg(data.totalByMonth.slice(0, half));
    const l = avg(data.totalByMonth.slice(half));
    return { pct: f === 0 ? 0 : ((l - f) / f) * 100, rising: l > f };
  }, [data]);

  return (
    <AppLayout>
      <div className="max-w-6xl mx-auto px-4 py-6">
        <div className="flex items-center justify-between flex-wrap gap-3 mb-5">
          <div>
            <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
            <h1 className="text-2xl font-bold text-white mt-1">Custo de vida</h1>
            <p className="text-white/50 text-sm">Suas despesas mês a mês — está subindo ou descendo, e onde.</p>
          </div>
          <div className="flex items-center gap-3">
            <input
              type="month"
              value={basePeriod}
              onChange={(e) => setBasePeriod(e.target.value)}
              className="bg-white/5 border border-white/10 rounded-lg px-3 py-1.5 text-white text-sm"
            />
            <div className="flex rounded-lg overflow-hidden border border-white/10">
              {([3, 6, 12] as ViewMode[]).map((v) => (
                <button
                  key={v}
                  onClick={() => setViewMode(v)}
                  className={`px-3 py-1.5 text-sm ${viewMode === v ? 'bg-purple-600 text-white' : 'bg-white/5 text-white/60 hover:text-white'}`}
                >
                  {v}m
                </button>
              ))}
            </div>
          </div>
        </div>

        {loading ? (
          <p className="text-white/50 py-10 text-center">Carregando…</p>
        ) : !data || data.periods.length === 0 ? (
          <p className="text-white/50 py-10 text-center">Sem dados no período.</p>
        ) : (
          <div className="space-y-5">
            {/* Tendência do gasto total */}
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

            {/* Fixo vs variável */}
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

            {/* Maiores movimentos */}
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

            {/* Onde está indo (categoria × mês) */}
            <section className="bg-white/5 border border-white/10 rounded-2xl p-5 overflow-x-auto">
              <h2 className="text-white font-semibold mb-3">Onde está indo</h2>
              <table className="w-full text-sm min-w-[640px]">
                <thead>
                  <tr className="text-white/50 text-xs">
                    <th className="text-left font-medium pb-2">Categoria</th>
                    {data.periods.map((p, i) => (
                      <th key={i} className="text-right font-medium pb-2 px-2">{monthLabel(p)}</th>
                    ))}
                    <th className="text-right font-medium pb-2 px-2">média</th>
                    <th className="text-center font-medium pb-2 pl-2">tend.</th>
                  </tr>
                </thead>
                <tbody>
                  {(data.categories ?? []).map((cat) => (
                    <CategoryRows key={cat.categoryId || cat.name} cat={cat} expanded={expanded} toggle={toggle} />
                  ))}
                </tbody>
              </table>
            </section>
          </div>
        )}
      </div>
    </AppLayout>
  );
}

function CategoryRows({
  cat,
  expanded,
  toggle,
}: {
  cat: CategoryTrend;
  expanded: Set<string>;
  toggle: (id: string) => void;
}) {
  const id = cat.categoryId || cat.name;
  const hasChildren = (cat.children?.length ?? 0) > 0;
  const isOpen = expanded.has(id);
  return (
    <>
      <tr className="border-t border-white/5 hover:bg-white/5">
        <td className="py-2 pr-2">
          <button
            onClick={() => hasChildren && toggle(id)}
            className={`flex items-center gap-1.5 text-left ${hasChildren ? 'text-white hover:text-purple-300' : 'text-white/90'}`}
          >
            {hasChildren ? <span className="text-white/40 text-xs w-3">{isOpen ? '▾' : '▸'}</span> : <span className="w-3" />}
            <span className="truncate">{cat.name}</span>
          </button>
        </td>
        {cat.byMonth.map((v, i) => (
          <td key={i} className="text-right px-2 py-2 text-white/70 tabular-nums">{v ? fmtShort(v) : '·'}</td>
        ))}
        <td className="text-right px-2 py-2 text-white/90 font-medium tabular-nums">{fmtShort(cat.average)}</td>
        <td className={`text-center pl-2 py-2 ${trendColor(cat.trend)}`}>{trendIcon(cat.trend)}</td>
      </tr>
      {isOpen &&
        cat.children?.map((child) => (
          <tr key={child.categoryId || child.name} className="bg-black/20">
            <td className="py-1.5 pr-2 pl-8 text-white/60 truncate">{child.name}</td>
            {child.byMonth.map((v, i) => (
              <td key={i} className="text-right px-2 py-1.5 text-white/50 tabular-nums">{v ? fmtShort(v) : '·'}</td>
            ))}
            <td className="text-right px-2 py-1.5 text-white/60 tabular-nums">{fmtShort(child.average)}</td>
            <td className={`text-center pl-2 py-1.5 ${trendColor(child.trend)}`}>{trendIcon(child.trend)}</td>
          </tr>
        ))}
    </>
  );
}
