'use client';

import { formatCurrency } from '@/utils/format';
import type { BankAccount } from '@/types/finances';

interface SummaryCardsProps {
  regularAccounts: BankAccount[];
  investmentAccounts: BankAccount[];
  totalsByCurrency: Record<string, number>;
  investedByCurrency: Record<string, number>;
}

export default function SummaryCards({
  regularAccounts,
  investmentAccounts,
  totalsByCurrency,
  investedByCurrency,
}: SummaryCardsProps) {
  const nonCreditCards = regularAccounts.filter((a) => a.type !== 'CREDIT_CARD');
  const creditCards = regularAccounts.filter((a) => a.type === 'CREDIT_CARD');

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <p className="text-white/50 text-sm mb-1">Saldo em contas</p>
        <div className="space-y-0.5">
          {Object.entries(totalsByCurrency).map(([cur, total]) => (
            <p key={cur} className="text-2xl font-bold text-white">{formatCurrency(total, cur)}</p>
          ))}
          {Object.keys(totalsByCurrency).length === 0 && (
            <p className="text-2xl font-bold text-white">{formatCurrency(0)}</p>
          )}
        </div>
        <div className="mt-2 space-y-1">
          {nonCreditCards.map((a) => (
            <div key={a.id} className="flex items-center justify-between text-xs">
              <span className="text-white/40 truncate mr-2">{a.name}</span>
              <span className="text-white/60 font-medium shrink-0">{formatCurrency(a.currentBalance, a.currency)}</span>
            </div>
          ))}
        </div>
      </div>
      {investmentAccounts.length > 0 && (
        <div className="bg-gradient-to-br from-purple-900/30 to-indigo-900/30 border border-purple-500/20 rounded-2xl p-5 backdrop-blur-sm">
          <p className="text-purple-300/70 text-sm mb-1">Total investido</p>
          <div className="space-y-0.5">
            {Object.entries(investedByCurrency).map(([cur, total]) => (
              <p key={cur} className="text-2xl font-bold text-white">{formatCurrency(total, cur)}</p>
            ))}
          </div>
          <p className="text-white/40 text-xs mt-1">
            {investmentAccounts.length} {investmentAccounts.length === 1 ? 'investimento' : 'investimentos'}
          </p>
        </div>
      )}
      {creditCards.length > 0 && (
        <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
          <p className="text-white/50 text-sm mb-1">Cartoes de credito</p>
          <p className="text-2xl font-bold text-white">{creditCards.length}</p>
          <p className="text-white/40 text-xs mt-1">
            Limite total: {formatCurrency(creditCards.reduce((sum, a) => sum + (a.creditLimit || 0), 0))}
          </p>
        </div>
      )}
    </div>
  );
}
