'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { ExpenseAnalysis, FinancialSummary, CapitalSummary, ViewMode } from './types';
import { BusinessView } from './BusinessView';
import { CostOfLivingView } from './CostOfLivingView';

export default function VisaoMensalPage() {
  const { selectedProfileId, selectedProfile } = useProfile();
  const isBusiness = selectedProfile?.type === 'BUSINESS';
  const [basePeriod, setBasePeriod] = useState<string>(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 5);
    return d.toISOString().slice(0, 7);
  });
  const [viewMode, setViewMode] = useState<ViewMode>(6);
  const [personal, setPersonal] = useState<ExpenseAnalysis | null>(null);
  const [business, setBusiness] = useState<FinancialSummary | null>(null);
  const [capital, setCapital] = useState<CapitalSummary | null>(null);
  const [loading, setLoading] = useState(true);

  const range = useMemo(() => {
    const [y, m] = basePeriod.split('-').map(Number);
    const from = new Date(y, m - 1, 1);
    const to = new Date(y, m - 1 + viewMode, 0);
    const iso = (d: Date) =>
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    return { from: iso(from), to: iso(to) };
  }, [basePeriod, viewMode]);

  useEffect(() => {
    if (!selectedProfileId) return;
    (async () => {
      try {
        setLoading(true);
        setPersonal(null);
        setBusiness(null);
        setCapital(null);
        if (isBusiness) {
          const [res, cap] = await Promise.all([
            api.get<{ data: FinancialSummary }>(
              `/transactions/financial-summary?profileId=${selectedProfileId}&from=${range.from}&to=${range.to}`,
            ),
            api
              .get<{ data: CapitalSummary }>(`/capital-contributions/summary?profileId=${selectedProfileId}`)
              .catch(() => ({ data: null as CapitalSummary | null })),
          ]);
          setBusiness(res.data);
          setCapital(cap.data);
        } else {
          const res = await api.get<{ data: ExpenseAnalysis }>(
            `/transactions/expense-analysis?profileId=${selectedProfileId}&from=${range.from}&to=${range.to}`,
          );
          setPersonal(res.data);
        }
      } catch (e) {
        console.warn('Erro ao carregar visão', e);
      } finally {
        setLoading(false);
      }
    })();
  }, [selectedProfileId, isBusiness, range]);

  return (
    <AppLayout>
      <div className="max-w-6xl mx-auto px-4 py-6">
        <div className="flex items-center justify-between flex-wrap gap-3 mb-5">
          <div>
            <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
            <h1 className="text-2xl font-bold text-white mt-1">{isBusiness ? 'Resultado (DRE)' : 'Custo de vida'}</h1>
            <p className="text-white/50 text-sm">
              {isBusiness
                ? 'Receita, despesa e resultado mês a mês — a empresa está dando lucro?'
                : 'Suas despesas mês a mês — está subindo ou descendo, e onde.'}
            </p>
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
        ) : isBusiness ? (
          <BusinessView data={business} capital={capital} />
        ) : (
          <CostOfLivingView data={personal} />
        )}
      </div>
    </AppLayout>
  );
}
