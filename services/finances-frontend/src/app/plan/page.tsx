'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import type { Profile, BankAccount, RecurringTransaction, BudgetSummaryItem, Category } from '@/types/finances';

const API_BASE = 'http://localhost:3335/api/v1';

type Range = '30' | '90';

export default function PlanPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [summary, setSummary] = useState<BudgetSummaryItem[]>([]);
  const [range, setRange] = useState<Range>('30');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const endDate = useMemo(() => {
    const d = new Date();
    d.setDate(d.getDate() + (range === '30' ? 30 : 90));
    return d;
  }, [range]);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/profiles`);
        const data = await res.json();
        const list: Profile[] = data.data || [];
        setProfiles(list);
        if (list.length > 0) setSelectedProfileId(list[0].id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  useEffect(() => {
    if (!selectedProfileId) return;
    (async () => {
      try {
        setLoading(true);
        setError(null);
        const month = new Date().toISOString().slice(0, 7);
        const [accRes, recRes, sumRes, catRes] = await Promise.all([
          fetch(`${API_BASE}/bank-accounts`),
          fetch(`${API_BASE}/recurring-transactions?profileId=${selectedProfileId}`),
          fetch(`${API_BASE}/budgets/summary?profileId=${selectedProfileId}&period=${month}`),
          fetch(`${API_BASE}/categories?profileId=${selectedProfileId}`),
        ]);
        if (!accRes.ok) throw new Error(`accounts ${accRes.status}`);
        if (!recRes.ok) throw new Error(`recurring ${recRes.status}`);
        if (!sumRes.ok) throw new Error(`summary ${sumRes.status}`);
        if (!catRes.ok) throw new Error(`categories ${catRes.status}`);
        const accData = await accRes.json();
        const recData = await recRes.json();
        const sumData = await sumRes.json();
        const catData = await catRes.json();
        setAccounts(accData.data || []);
        setRecurrings(recData.data || []);
        setSummary(sumData.data || []);
        setCategories(catData.data || []);
      } catch (e) {
        console.warn('Erro ao carregar dados do planejamento', e);
        setError('Não foi possível carregar dados do planejamento.');
        setAccounts([]);
        setRecurrings([]);
        setSummary([]);
        setCategories([]);
      } finally {
        setLoading(false);
      }
    })();
  }, [selectedProfileId]);

  const forecast = useMemo(() => {
    const now = new Date();
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const end = endDate;
    type Item = { date: string; description: string; amount: number; type: 'INCOME' | 'EXPENSE' };
    const items: Item[] = [];

    const clamp = (d: Date, min: Date, max?: Date) => {
      if (d < min) return new Date(min);
      if (max && d > max) return new Date(max);
      return d;
    };

    const pushIfInRange = (d: Date, desc: string, amount: number, type: 'INCOME' | 'EXPENSE', startOn?: Date, endOn?: Date) => {
      const sd = startOn ? clamp(d, startOn, endOn) : d;
      if (sd >= start && sd <= end) {
        items.push({ date: sd.toISOString().slice(0, 10), description: desc, amount, type });
      }
    };

    const parseRule = (rule: string) => {
      const map = new Map<string, string>();
      rule.split(';').forEach((kv) => {
        const [k, v] = kv.split('=');
        if (k && v) map.set(k.toUpperCase(), v.toUpperCase());
      });
      return map;
    };

    recurrings.forEach((r) => {
      const rule = parseRule(r.recurrenceRule || '');
      const freq = rule.get('FREQ') || 'MONTHLY';
      const byMonthDay = rule.get('BYMONTHDAY');
      const startOn = new Date(r.startOn);
      const endOn = r.endOn ? new Date(r.endOn) : undefined;
      let cur = new Date(r.nextOccurrence);

      const addDays = (d: Date, n: number) => {
        const c = new Date(d);
        c.setDate(c.getDate() + n);
        return c;
      };
      const addMonths = (d: Date, n: number) => {
        const c = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + n, 1));
        const day = byMonthDay ? Number(byMonthDay) : d.getUTCDate();
        const target = new Date(Date.UTC(c.getUTCFullYear(), c.getUTCMonth(), day));
        return target;
      };

      // Generate occurrences up to end
      while (cur <= end) {
        pushIfInRange(cur, r.description, r.amount, r.type as 'INCOME' | 'EXPENSE', startOn, endOn);
        if (freq === 'DAILY') {
          cur = addDays(cur, 1);
        } else if (freq === 'WEEKLY') {
          cur = addDays(cur, 7);
        } else {
          cur = addMonths(cur, 1);
        }
        if (endOn && cur > endOn) break;
      }
    });

    items.sort((a, b) => a.date.localeCompare(b.date));
    const totals = items.reduce(
      (acc, it) => {
        const sign = it.type === 'EXPENSE' ? -1 : 1;
        return { ...acc, total: acc.total + sign * it.amount, out: acc.out + (it.type === 'EXPENSE' ? it.amount : 0), inc: acc.inc + (it.type === 'INCOME' ? it.amount : 0) };
      },
      { total: 0, out: 0, inc: 0 },
    );

    return { items, totals };
  }, [recurrings, endDate]);

  const totalBalance = useMemo(() => accounts.filter((a) => a.profileId === selectedProfileId).reduce((sum, a) => sum + a.currentBalance, 0), [accounts, selectedProfileId]);

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Planejamento</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Perfil:</span>
              <div className="flex flex-wrap gap-2">
                {profiles.map((p) => (
                  <button key={p.id} onClick={() => setSelectedProfileId(p.id)} className={`px-3 py-1.5 rounded-xl border ${selectedProfileId === p.id ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}>{p.name}</button>
                ))}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Horizonte:</span>
              <button onClick={() => setRange('30')} className={`px-3 py-1.5 rounded-xl text-sm border ${range === '30' ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}>30 dias</button>
              <button onClick={() => setRange('90')} className={`px-3 py-1.5 rounded-xl text-sm border ${range === '90' ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}>90 dias</button>
            </div>
          </div>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              {loading && <p className="text-white/70">Carregando...</p>}
              {error && <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3 mb-3">{error}</div>}
              <h3 className="text-white font-semibold mb-3">Agenda de fixas</h3>
              {forecast.items.length === 0 ? (
                <p className="text-white/70">Sem itens recorrentes no período.</p>
              ) : (
                <div className="space-y-2">
                  {forecast.items.map((it, idx) => (
                    <div key={`${it.date}-${idx}`} className="flex items-center justify-between border border-white/10 bg-white/5 rounded-xl px-3 py-2">
                      <div className="text-white/80 text-sm flex items-center gap-3">
                        <span className="text-white/60 w-24">{new Date(it.date).toLocaleDateString('pt-BR')}</span>
                        <span className="font-medium">{it.description}</span>
                      </div>
                      <div className={`text-sm font-semibold ${it.type === 'EXPENSE' ? 'text-rose-200' : 'text-emerald-200'}`}>
                        {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(it.amount * (it.type === 'EXPENSE' ? -1 : 1))}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
          <div className="space-y-6">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-white font-semibold mb-3">Resumo do período</h3>
              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between text-white/80"><span>Saldo atual</span><span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(totalBalance)}</span></div>
                <div className="flex items-center justify-between text-emerald-200"><span>Fixas (receitas)</span><span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(forecast.totals.inc)}</span></div>
                <div className="flex items-center justify-between text-rose-200"><span>Fixas (despesas)</span><span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(forecast.totals.out)}</span></div>
                <div className="flex items-center justify-between text-white"><span>Variação projetada</span><span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(forecast.totals.inc - forecast.totals.out)}</span></div>
              </div>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-white font-semibold mb-3">Orçamentos do mês</h3>
              {summary.length === 0 ? (
                <p className="text-white/70 text-sm">Nenhuma meta cadastrada.</p>
              ) : (
                <div className="space-y-2 text-sm">
                  {summary.map((s) => {
                    const cat = categories.find((c) => c.id === s.target.categoryId);
                    const name = cat ? cat.name : s.target.categoryId;
                    return (
                      <div key={s.target.id} className="flex items-center justify-between text-white/80">
                        <span>{name}</span>
                        <span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(s.remaining)} restantes</span>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </AppLayout>
  );
}

