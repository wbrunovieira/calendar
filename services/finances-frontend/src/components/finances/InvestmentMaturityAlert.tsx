'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { MaturityAlert } from '@/types/finances';
import { formatCurrency } from '@/utils/format';

interface Props {
  profileId: string | null;
  withinDays?: number;
}

const fmtDate = (iso: string) =>
  new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });

const whenLabel = (a: MaturityAlert): string => {
  if (a.isMatured) {
    const d = Math.abs(a.daysToMaturity);
    return d === 0 ? 'vence hoje' : `venceu há ${d} dia${d > 1 ? 's' : ''}`;
  }
  if (a.daysToMaturity === 0) return 'vence hoje';
  return `vence em ${a.daysToMaturity} dia${a.daysToMaturity > 1 ? 's' : ''}`;
};

export default function InvestmentMaturityAlert({ profileId, withinDays }: Props) {
  const [alerts, setAlerts] = useState<MaturityAlert[]>([]);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    if (!profileId) {
      setAlerts([]);
      return;
    }
    let active = true;
    const params = new URLSearchParams({ profileId });
    if (withinDays) params.set('withinDays', String(withinDays));
    api
      .get<{ data: MaturityAlert[] }>(`/bank-accounts/maturities?${params.toString()}`)
      .then((res) => {
        if (active) setAlerts(res.data ?? []);
      })
      .catch(() => {
        if (active) setAlerts([]);
      });
    return () => {
      active = false;
    };
  }, [profileId, withinDays]);

  if (dismissed || alerts.length === 0) return null;

  const hasMatured = alerts.some((a) => a.isMatured);
  const tone = hasMatured
    ? { box: 'bg-rose-500/15 border-rose-400/40', title: 'text-rose-200', sub: 'text-rose-300/70', x: 'text-rose-300/60 hover:text-rose-200' }
    : { box: 'bg-amber-500/15 border-amber-400/40', title: 'text-amber-200', sub: 'text-amber-300/70', x: 'text-amber-300/60 hover:text-amber-200' };

  return (
    <div className={`${tone.box} border rounded-2xl p-4`}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <span className="text-xl mt-0.5">⏰</span>
          <div>
            <p className={`${tone.title} font-semibold text-sm`}>
              {alerts.length} investimento{alerts.length > 1 ? 's' : ''}{' '}
              {hasMatured ? 'precisam de atenção (vencimento)' : 'com vencimento próximo'}
            </p>
            <ul className="mt-1.5 space-y-1">
              {alerts.map((a) => (
                <li key={a.accountId} className={`${tone.sub} text-xs`}>
                  <span className={`font-semibold ${tone.title}`}>{a.name}</span>
                  {' — '}
                  {whenLabel(a)} ({fmtDate(a.maturityDate)}) ·{' '}
                  <span className={`font-semibold ${tone.title}`}>
                    {formatCurrency(a.currentBalance, a.currency)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>
        <button
          onClick={() => setDismissed(true)}
          className={`${tone.x} text-lg leading-none flex-shrink-0`}
          title="Dispensar"
        >
          ✕
        </button>
      </div>
    </div>
  );
}
