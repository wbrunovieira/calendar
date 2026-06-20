'use client';

import { useMemo, useState } from 'react';
import type { BankAccount, Transaction, Invoice } from '@/types/finances';
import { formatCurrency } from '@/utils/format';
import { summarizeCashflow } from '@/utils/cashflow';

interface CashflowSummaryProps {
  transactions: Transaction[];
  accounts: BankAccount[];
  currentInvoices?: Record<string, Invoice>;
}

const getCurrentMonthYear = () => {
  const now = new Date();
  const month = now.toLocaleDateString('pt-BR', { month: 'short' }).replace('.', '');
  const year = now.getFullYear();
  return `${month.charAt(0).toUpperCase() + month.slice(1)}/${year}`;
};

// Investment prices/balances older than this are flagged as possibly outdated.
const STALE_DAYS = 7;
const daysSince = (dateStr?: string): number | null => {
  if (!dateStr) return null;
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return null;
  return Math.floor((Date.now() - d.getTime()) / (1000 * 60 * 60 * 24));
};
const formatShortDate = (dateStr?: string) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '';
  return `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}`;
};

export default function CashflowSummary({ transactions, accounts, currentInvoices = {} }: CashflowSummaryProps) {
  const currentPeriod = getCurrentMonthYear();
  const [showPatrimonio, setShowPatrimonio] = useState(false);

  const cf = useMemo(() => summarizeCashflow(transactions, accounts), [transactions, accounts]);

  // Separate accounts by type
  const regularAccounts = accounts.filter(
    (acc) => acc.type !== 'CREDIT_CARD' && acc.type !== 'INVESTMENT'
  );
  const creditCards = accounts.filter((acc) => acc.type === 'CREDIT_CARD');
  const investments = accounts.filter((acc) => acc.type === 'INVESTMENT');

  const availableBalance = regularAccounts.reduce((total, acc) => total + acc.currentBalance, 0);
  const creditCardDebt = creditCards.reduce(
    (total, acc) => total + (currentInvoices[acc.id]?.amount || 0),
    0,
  );
  const investmentTotal = investments.reduce((total, acc) => total + acc.currentBalance, 0);
  const netWorth = availableBalance + investmentTotal - creditCardDebt;

  const staleInvestments = investments.filter((acc) => {
    const d = daysSince(acc.updatedAt);
    return d !== null && d > STALE_DAYS && acc.currentBalance > 0;
  }).length;

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
      {/* Entradas */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">⬆️</span>
          <span className="text-white/70 text-sm">Entradas</span>
          <span className="text-white/40 text-xs ml-auto">{currentPeriod}</span>
        </div>
        <div className="text-2xl font-bold text-emerald-400">{formatCurrency(cf.income)}</div>
        <div className="mt-2 space-y-0.5 text-xs">
          <div className="flex justify-between">
            <span className="text-white/50">Outras entradas</span>
            <span className="text-white/70">{formatCurrency(cf.incomeOther)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-white/50">Rendimentos</span>
            <span className="text-white/70">{formatCurrency(cf.incomeYield)}</span>
          </div>
        </div>
        <p className="text-white/40 text-[11px] mt-1">{cf.incomeCount} confirmadas</p>
      </div>

      {/* Saidas */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">⬇️</span>
          <span className="text-white/70 text-sm">Saidas</span>
          <span className="text-white/40 text-xs ml-auto">{currentPeriod}</span>
        </div>
        <div className="text-2xl font-bold text-rose-400">{formatCurrency(cf.expense)}</div>
        <p className="text-white/50 text-xs mt-1">{cf.expenseCount} confirmadas</p>
      </div>

      {/* Saldo Disponivel (contas correntes, poupanca, dinheiro) */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">🏦</span>
          <span className="text-white/70 text-sm">Disponivel</span>
        </div>
        <div className="space-y-1 mb-2">
          {regularAccounts.map((acc) => (
            <div key={acc.id} className="flex justify-between text-sm">
              <span className="text-white/60 truncate mr-2">{acc.name}</span>
              <span className={acc.currentBalance >= 0 ? 'text-white/80' : 'text-rose-400'}>
                {formatCurrency(acc.currentBalance)}
              </span>
            </div>
          ))}
        </div>
        <div className="border-t border-white/10 pt-2">
          <div className="flex justify-between">
            <span className="text-white/70 text-sm font-medium">Total</span>
            <span className={`text-lg font-bold ${availableBalance >= 0 ? 'text-white' : 'text-rose-400'}`}>
              {formatCurrency(availableBalance)}
            </span>
          </div>
        </div>
      </div>

      {/* Cartoes de Credito */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">💳</span>
          <span className="text-white/70 text-sm">Faturas</span>
        </div>
        <div className={`text-2xl font-bold ${creditCardDebt > 0 ? 'text-amber-400' : 'text-white'}`}>
          {formatCurrency(creditCardDebt)}
        </div>
        <p className="text-white/50 text-xs mt-1">
          {creditCards.length} {creditCards.length === 1 ? 'cartao' : 'cartoes'}
        </p>
      </div>

      {/* Patrimonio Liquido - clicavel */}
      <button
        type="button"
        onClick={() => setShowPatrimonio(true)}
        className="text-left bg-gradient-to-br from-purple-900/30 to-indigo-900/30 border border-purple-500/20 rounded-2xl p-5 backdrop-blur-sm hover:border-purple-400/40 transition-colors"
      >
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">💰</span>
          <span className="text-white/70 text-sm">Patrimonio</span>
          <span className="text-purple-300/60 text-xs ml-auto">ver detalhes →</span>
        </div>
        <div className={`text-2xl font-bold ${netWorth >= 0 ? 'text-purple-300' : 'text-rose-400'}`}>
          {formatCurrency(netWorth)}
        </div>
        <p className="text-white/50 text-xs mt-1">
          {investments.length > 0 ? `+ ${formatCurrency(investmentTotal)} investidos` : 'Disponivel - Faturas'}
          {staleInvestments > 0 && <span className="text-amber-400"> · ⚠️ {staleInvestments} desatualizados</span>}
        </p>
      </button>

      {showPatrimonio && (
        <PatrimonioModal
          onClose={() => setShowPatrimonio(false)}
          regularAccounts={regularAccounts}
          investments={investments}
          creditCards={creditCards}
          currentInvoices={currentInvoices}
          availableBalance={availableBalance}
          investmentTotal={investmentTotal}
          creditCardDebt={creditCardDebt}
          netWorth={netWorth}
        />
      )}
    </div>
  );
}

interface PatrimonioModalProps {
  onClose: () => void;
  regularAccounts: BankAccount[];
  investments: BankAccount[];
  creditCards: BankAccount[];
  currentInvoices: Record<string, Invoice>;
  availableBalance: number;
  investmentTotal: number;
  creditCardDebt: number;
  netWorth: number;
}

function PatrimonioModal({
  onClose,
  regularAccounts,
  investments,
  creditCards,
  currentInvoices,
  availableBalance,
  investmentTotal,
  creditCardDebt,
  netWorth,
}: PatrimonioModalProps) {
  const sortedInvestments = [...investments].sort((a, b) => b.currentBalance - a.currentBalance);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 col-span-full"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg max-h-[85vh] overflow-y-auto bg-gradient-to-br from-slate-900 to-indigo-950 border border-purple-500/30 rounded-2xl p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-white font-semibold text-lg">💰 Patrimonio</h3>
            <p className={`text-2xl font-bold ${netWorth >= 0 ? 'text-purple-300' : 'text-rose-400'}`}>
              {formatCurrency(netWorth)}
            </p>
          </div>
          <button onClick={onClose} className="text-white/50 hover:text-white text-xl px-2">✕</button>
        </div>

        {/* Disponivel */}
        <Section title="Disponivel" total={availableBalance} color="text-white">
          {regularAccounts.map((acc) => (
            <Row key={acc.id} label={acc.name} value={acc.currentBalance} />
          ))}
        </Section>

        {/* Investido */}
        <Section title="Investido" total={investmentTotal} color="text-purple-300">
          {sortedInvestments.map((acc) => {
            const d = daysSince(acc.updatedAt);
            const stale = d !== null && d > STALE_DAYS && acc.currentBalance > 0;
            return (
              <Row
                key={acc.id}
                label={acc.name}
                value={acc.currentBalance}
                hint={
                  <span className={stale ? 'text-amber-400' : 'text-white/30'}>
                    {stale ? '⚠️ ' : ''}{formatShortDate(acc.updatedAt)}
                  </span>
                }
              />
            );
          })}
        </Section>

        {/* Faturas */}
        <Section title="Faturas a pagar" total={-creditCardDebt} color="text-amber-400">
          {creditCards.map((acc) => (
            <Row key={acc.id} label={acc.name} value={-(currentInvoices[acc.id]?.amount || 0)} />
          ))}
        </Section>

        <div className="border-t border-white/10 mt-3 pt-3 text-xs text-white/50">
          Disponivel {formatCurrency(availableBalance)} + Investido {formatCurrency(investmentTotal)} −
          Faturas {formatCurrency(creditCardDebt)} = <span className="text-white/80 font-medium">{formatCurrency(netWorth)}</span>
        </div>
      </div>
    </div>
  );
}

function Section({ title, total, color, children }: { title: string; total: number; color: string; children: React.ReactNode }) {
  return (
    <div className="mb-4">
      <div className="flex justify-between items-baseline mb-1.5">
        <span className="text-white/70 text-sm font-medium">{title}</span>
        <span className={`text-sm font-semibold ${color}`}>{formatCurrency(total)}</span>
      </div>
      <div className="space-y-1 pl-1">{children}</div>
    </div>
  );
}

function Row({ label, value, hint }: { label: string; value: number; hint?: React.ReactNode }) {
  return (
    <div className="flex justify-between items-center text-sm">
      <span className="text-white/60 truncate mr-2 flex items-center gap-2">
        {label}
        {hint && <span className="text-[10px]">{hint}</span>}
      </span>
      <span className={value >= 0 ? 'text-white/80' : 'text-rose-400'}>{formatCurrency(value)}</span>
    </div>
  );
}
