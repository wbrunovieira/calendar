'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { formatCurrency } from '@/utils/format';
import type { Profile, Transaction, Category } from '@/types/finances';

interface DasAlertProps {
  profile: Profile;
  transactions: Transaction[];
  categories: Category[];
}

export default function DasAlert({ profile, transactions, categories }: DasAlertProps) {
  const [dismissed, setDismissed] = useState(false);

  const now = new Date();
  const dayOfMonth = now.getDate();
  const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;

  // Only show from day 15 onwards
  if (!profile.simplesNacional || dayOfMonth < 15 || dismissed) return null;

  // Check if DAS was already paid this month — look for TAX transaction in current month
  const taxCategoryIds = categories
    .filter((c) => c.classificationDRE === 'TAX')
    .map((c) => c.id);

  const dasPaidThisMonth = transactions.some((tx) => {
    const txMonth = tx.occurredOn.slice(0, 7);
    return txMonth === currentMonth && tx.status === 'CONFIRMED' && tx.categoryId && taxCategoryIds.includes(tx.categoryId);
  });

  if (dasPaidThisMonth) return null;

  // Estimate DAS based on last month's revenue and profile aliquota
  const lastMonth = now.getMonth() === 0
    ? `${now.getFullYear() - 1}-12`
    : `${now.getFullYear()}-${String(now.getMonth()).padStart(2, '0')}`;

  const lastMonthRevenue = transactions
    .filter((tx) => tx.type === 'INCOME' && tx.status === 'CONFIRMED' && tx.occurredOn.slice(0, 7) === lastMonth)
    .reduce((sum, tx) => sum + tx.amount, 0);

  const dasEstimate = profile.dasAliquota && lastMonthRevenue > 0
    ? lastMonthRevenue * (profile.dasAliquota / 100)
    : null;

  const dueDay = 20;
  const daysUntilDue = dueDay - dayOfMonth;
  const isOverdue = dueDay < dayOfMonth;

  return (
    <div className={`rounded-2xl border p-4 flex items-start justify-between gap-4 ${
      isOverdue
        ? 'bg-rose-500/15 border-rose-400/40'
        : 'bg-amber-500/10 border-amber-400/30'
    }`}>
      <div className="flex items-start gap-3">
        <span className="text-2xl">🧾</span>
        <div>
          <p className={`font-semibold ${isOverdue ? 'text-rose-200' : 'text-amber-200'}`}>
            {isOverdue ? 'DAS vencido!' : `DAS vence em ${daysUntilDue} dia${daysUntilDue !== 1 ? 's' : ''} (dia ${dueDay})`}
          </p>
          <p className="text-white/60 text-sm mt-0.5">
            Simples Nacional · {currentMonth.split('-').reverse().join('/')}
            {dasEstimate !== null && (
              <> · Estimativa: <strong className="text-white/80">{formatCurrency(dasEstimate)}</strong> ({profile.dasAliquota}% s/ faturamento de {lastMonth.split('-').reverse().join('/')})</>
            )}
          </p>
        </div>
      </div>
      <button
        onClick={() => setDismissed(true)}
        className="text-white/40 hover:text-white/70 text-xs shrink-0 mt-0.5"
        title="Dispensar alerta"
      >
        ✕
      </button>
    </div>
  );
}
