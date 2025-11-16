'use client';

import type { BudgetSummaryItem } from '@/types/finances';

interface Props {
  summary: BudgetSummaryItem[];
}

export default function SafeToSpend({ summary }: Props) {
  const remaining = summary.reduce((sum, s) => sum + (s?.remaining ?? 0), 0);
  const now = new Date();
  const daysInMonth = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0)).getUTCDate();
  const today = now.getUTCDate();
  const daysLeft = Math.max(1, daysInMonth - today + 1);
  const perDay = remaining / daysLeft;
  const perWeek = perDay * 7;

  return (
    <div className="grid gap-4 sm:grid-cols-3">
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <p className="text-white/60 text-sm">Restante no mês</p>
        <p className={`text-2xl font-bold ${remaining < 0 ? 'text-rose-200' : 'text-white'}`}>
          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(remaining)}
        </p>
      </div>
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <p className="text-white/60 text-sm">Safe-to-spend diário</p>
        <p className={`text-2xl font-bold ${perDay < 0 ? 'text-rose-200' : 'text-white'}`}>
          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(perDay)}
        </p>
      </div>
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <p className="text-white/60 text-sm">Safe-to-spend semanal</p>
        <p className={`text-2xl font-bold ${perWeek < 0 ? 'text-rose-200' : 'text-white'}`}>
          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(perWeek)}
        </p>
      </div>
    </div>
  );
}

