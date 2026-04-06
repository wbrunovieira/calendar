'use client';

import TransactionHistory from '@/components/finances/TransactionHistory';
import type { Category, Transaction } from '@/types/finances';

interface ExpandedTransactionPanelProps {
  accountId: string;
  profileId: string;
  categories: Category[];
  accountCurrency?: string;
  isCreditCard?: boolean;
  includeAsDestination?: boolean;
  selectedInvoiceId?: string;
  botFilter?: string;
  onAddTransaction: (accountId: string) => void;
  onEdit: (tx: Transaction) => void;
  onDelete: (tx: Transaction) => void;
  onConfirm?: (tx: Transaction) => Promise<void> | void;
  refreshKey?: number;
  className?: string;
}

export default function ExpandedTransactionPanel({
  accountId,
  profileId,
  categories,
  accountCurrency,
  isCreditCard,
  includeAsDestination,
  selectedInvoiceId,
  botFilter,
  onAddTransaction,
  onEdit,
  onDelete,
  onConfirm,
  refreshKey,
  className = '',
}: ExpandedTransactionPanelProps) {
  return (
    <div className={`bg-white/[0.03] border border-white/10 border-t-0 rounded-b-xl p-4 ${className}`}>
      <div className="flex items-center justify-between mb-3">
        <p className="text-white/60 text-xs font-semibold uppercase tracking-wider">Historico de transacoes</p>
        <button
          onClick={(e) => { e.stopPropagation(); onAddTransaction(accountId); }}
          className="flex items-center gap-1 px-2.5 py-1 bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/30 rounded-lg text-emerald-400 text-xs font-medium transition-colors"
        >
          <span>+</span>
          <span>Lancamento</span>
        </button>
      </div>
      <TransactionHistory
        accountId={accountId}
        profileId={profileId}
        categories={categories}
        selectedInvoiceId={selectedInvoiceId}
        accountCurrency={accountCurrency}
        isCreditCard={isCreditCard}
        includeAsDestination={includeAsDestination}
        botFilter={botFilter}
        onEdit={onEdit}
        onDelete={onDelete}
        onConfirm={onConfirm}
        refreshKey={refreshKey}
      />
    </div>
  );
}
