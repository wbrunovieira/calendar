'use client';

import { formatCurrency } from '@/utils/format';
import type { BankAccount } from '@/types/finances';
import { isDebtType, countsTowardHeldTotal } from '@/utils/accountGroups';

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
  // Split, not lumped: the total above counts only what can be spent, so a debt
  // listed under it has to be visibly apart or the two disagree on screen.
  // Same rule as the total above, so the list and the number cannot disagree:
  // everything held is listed, and `countsTowardHeldTotal` decides which of them
  // the number includes.
  const heldAccounts = regularAccounts.filter((a) => !isDebtType(a));
  const debtAccounts = regularAccounts.filter((a) => a.type === 'LIABILITY');
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
          {heldAccounts.map((a) => (
            <div key={a.id} className="flex items-center justify-between text-xs">
              <span className={`truncate mr-2 ${countsTowardHeldTotal(a) ? 'text-white/40' : 'text-white/25'}`}>
                {a.name}
                {!countsTowardHeldTotal(a) && <span className="text-[10px] ml-1">(desativada, fora do total)</span>}
              </span>
              <span className="text-white/60 font-medium shrink-0">{formatCurrency(a.currentBalance, a.currency)}</span>
            </div>
          ))}
          {debtAccounts.length > 0 && (
            <div className="pt-2 mt-1 border-t border-white/10 space-y-1">
              <p className="text-white/30 text-[10px] uppercase tracking-wide">Dividas (fora do saldo)</p>
              {debtAccounts.map((a) => (
                <div key={a.id} className="flex items-center justify-between text-xs">
                  <span className="text-white/40 truncate mr-2">{a.name}</span>
                  <span className="text-rose-400 font-medium shrink-0">{formatCurrency(a.currentBalance, a.currency)}</span>
                </div>
              ))}
            </div>
          )}
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
