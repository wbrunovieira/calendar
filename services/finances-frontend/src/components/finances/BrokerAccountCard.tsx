'use client';

import { useState } from 'react';
import { formatCurrency } from '@/utils/format';
import { DragHandle } from '@/components/finances/SortableHelpers';
import type { DndListeners, DndAttributes } from '@/components/finances/SortableHelpers';
import FiiDetailPanel from '@/components/finances/FiiDetailPanel';
import ExpandedTransactionPanel from '@/components/finances/ExpandedTransactionPanel';
import CreditCardInfo from '@/components/finances/CreditCardInfo';
import type { BankAccount, Category, Invoice, Transaction } from '@/types/finances';

type BotTab = 'all' | 'MACross1' | 'MemeCoin1';
const botTabs: { key: BotTab; label: string }[] = [
  { key: 'all', label: 'Todas' },
  { key: 'MACross1', label: 'MACross1 (BRL)' },
  { key: 'MemeCoin1', label: 'MemeCoin1 (USD)' },
];

interface BrokerAccountCardProps {
  account: BankAccount;
  isExpanded: boolean;
  isBroker: boolean;
  subInvestments: BankAccount[];
  subCreditCards: BankAccount[];
  expandedFiiId: string | null;
  selectedProfileId: string | null;
  categories: Category[];
  currentInvoices: Record<string, Invoice>;
  invoicesByAccount: Record<string, Invoice[]>;
  selectedInvoiceByAccount: Record<string, string>;
  listeners: DndListeners;
  attributes: DndAttributes;
  onToggleExpand: (id: string) => void;
  onToggleFii: (id: string | null) => void;
  onEditAccount: (account: BankAccount) => void;
  onAddTransaction: (accountId: string) => void;
  onEditTransaction: (tx: Transaction) => void;
  onDeleteTransaction: (tx: Transaction) => void;
  onPayInvoice: (invoiceId: string, amount: number) => Promise<void>;
  onUpdateInvoice: (invoiceId: string, data: { closingDate?: string; dueDate?: string }) => Promise<void>;
  onInvoiceSelect: (accountId: string, invoiceId: string) => void;
}

export default function BrokerAccountCard({
  account,
  isExpanded,
  isBroker,
  subInvestments,
  subCreditCards,
  expandedFiiId,
  selectedProfileId,
  categories,
  currentInvoices,
  invoicesByAccount,
  selectedInvoiceByAccount,
  listeners,
  attributes,
  onToggleExpand,
  onToggleFii,
  onEditAccount,
  onAddTransaction,
  onEditTransaction,
  onDeleteTransaction,
  onPayInvoice,
  onUpdateInvoice,
  onInvoiceSelect,
}: BrokerAccountCardProps) {
  const [activeBot, setActiveBot] = useState<BotTab>('all');
  const isExchange = account.type === 'EXCHANGE';

  const brokerTotalBrl = isBroker
    ? account.currentBalance + subInvestments.reduce((sum, s) => sum + s.currentBalance, 0)
    : account.currentBalance;

  return (
    <div>
      <div
        className={`p-4 bg-white/5 cursor-pointer hover:bg-white/10 transition-all duration-300 ${
          isExpanded ? 'rounded-t-xl' : 'rounded-xl'
        }`}
        style={isExpanded ? {
          border: '2px solid #34d399',
          boxShadow: '0 0 20px rgba(52, 211, 153, 0.3)',
        } : {
          border: '1px solid rgba(255, 255, 255, 0.1)',
        }}
        onClick={() => onToggleExpand(account.id)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <DragHandle listeners={listeners} attributes={attributes} />
            <span className="text-2xl">{account.icon || '🏦'}</span>
            <div>
              <p className="text-white font-semibold text-sm">{account.name}</p>
              <div className="flex items-center gap-2">
                <p className="text-white/50 text-xs">{account.bankName || account.type}</p>
                {isBroker && subInvestments.length > 0 && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-purple-500/20 text-purple-400">
                    {subInvestments.length} {subInvestments.length === 1 ? 'ativo' : 'ativos'}
                  </span>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-right">
              <p className="text-white/80 text-sm font-semibold">
                {formatCurrency(isBroker ? brokerTotalBrl : account.currentBalance, account.currency)}
              </p>
              <p className="text-white/50 text-xs">
                {isBroker ? 'Total' : 'Saldo atual'}
              </p>
            </div>
            <button
              onClick={(e) => { e.stopPropagation(); onEditAccount(account); }}
              className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-white/40 hover:text-white/80"
              title="Editar conta"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
          </div>
        </div>

        {/* Sub-investments inside broker card */}
        {isBroker && subInvestments.length > 0 && (
          <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
            {account.currentBalance > 0 && (
              <div className="px-2 py-2 rounded-lg bg-white/[0.04]">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-emerald-400 text-xs">●</span>
                    <span className="text-white/80 text-sm font-medium">Saldo em conta</span>
                  </div>
                  <span className="text-white/80 text-sm font-medium">{formatCurrency(account.currentBalance)}</span>
                </div>
              </div>
            )}
            {subInvestments.map((inv) => {
              const valorization = inv.currentBalance - inv.initialBalance;
              const valorizationPct = inv.initialBalance > 0
                ? (valorization / inv.initialBalance) * 100
                : 0;
              const hasValorization = inv.initialBalance > 0 && inv.numberOfQuotas != null;
              const isFiiExpanded = expandedFiiId === inv.id;
              return (
                <div key={inv.id}>
                  <div
                    className="px-2 py-2 rounded-lg bg-white/[0.04] cursor-pointer hover:bg-white/[0.07] transition-colors"
                    onClick={(e) => { e.stopPropagation(); onToggleFii(isFiiExpanded ? null : inv.id); }}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className={`text-xs transition-transform ${isFiiExpanded ? 'rotate-90' : ''}`}>▶</span>
                        <span className="text-white/80 text-sm font-medium">{inv.name}</span>
                        {inv.numberOfQuotas != null && (
                          <span className="text-white/40 text-xs">{inv.numberOfQuotas} cotas</span>
                        )}
                        {inv.quotaPrice != null && (
                          <span className="text-white/30 text-xs">@ {formatCurrency(inv.quotaPrice)}</span>
                        )}
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-white/80 text-sm font-medium">{formatCurrency(inv.currentBalance, inv.currency)}</span>
                        <button
                          onClick={(e) => { e.stopPropagation(); onEditAccount(inv); }}
                          className="text-white/30 hover:text-white/70 text-xs transition-colors"
                        >
                          Editar
                        </button>
                      </div>
                    </div>
                    {hasValorization && (
                      <div className="flex items-center justify-between mt-1 px-6">
                        <span className="text-white/30 text-xs">
                          Investido: {formatCurrency(inv.initialBalance)}
                        </span>
                        <span className={`text-xs font-medium ${valorization >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
                          {valorization >= 0 ? '+' : ''}{formatCurrency(valorization)} ({valorizationPct >= 0 ? '+' : ''}{valorizationPct.toFixed(2)}%)
                        </span>
                      </div>
                    )}
                  </div>
                  {isFiiExpanded && selectedProfileId && (
                    <FiiDetailPanel
                      account={inv}
                      clearAccountId={account.id}
                      profileId={selectedProfileId}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* Credit cards linked to this broker account */}
        {subCreditCards.length > 0 && (
          <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
            <p className="text-white/40 text-[10px] uppercase tracking-wider font-semibold px-2">Cartões de crédito</p>
            {subCreditCards.map((cc) => (
              <div key={cc.id} className="px-1">
                <CreditCardInfo
                  account={cc}
                  currentInvoice={currentInvoices[cc.id]}
                  invoices={invoicesByAccount[cc.id] || []}
                  onPayInvoice={onPayInvoice}
                  onEdit={() => onEditAccount(cc)}
                  onUpdateInvoice={onUpdateInvoice}
                  selectedInvoiceId={selectedInvoiceByAccount[cc.id] || ''}
                  onInvoiceSelect={(invoiceId) => onInvoiceSelect(cc.id, invoiceId)}
                />
              </div>
            ))}
          </div>
        )}
      </div>

      {isExpanded && selectedProfileId && (
        isExchange ? (
          <div className="bg-white/[0.03] border border-white/10 border-t-0 rounded-b-xl p-4">
            <div className="flex items-center gap-2 mb-3">
              {botTabs.map((tab) => (
                <button
                  key={tab.key}
                  onClick={(e) => { e.stopPropagation(); setActiveBot(tab.key); }}
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                    activeBot === tab.key
                      ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                      : 'bg-white/5 text-white/50 border border-white/10 hover:bg-white/10'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            <ExpandedTransactionPanel
              accountId={account.id}
              profileId={selectedProfileId}
              categories={categories}
              accountCurrency={account.currency}
              includeAsDestination={isBroker}
              botFilter={activeBot === 'all' ? undefined : activeBot}
              onAddTransaction={onAddTransaction}
              onEdit={onEditTransaction}
              onDelete={onDeleteTransaction}
            />
          </div>
        ) : (
          <ExpandedTransactionPanel
            accountId={account.id}
            profileId={selectedProfileId}
            categories={categories}
            accountCurrency={account.currency}
            includeAsDestination={isBroker}
            onAddTransaction={onAddTransaction}
            onEdit={onEditTransaction}
            onDelete={onDeleteTransaction}
          />
        )
      )}
    </div>
  );
}
