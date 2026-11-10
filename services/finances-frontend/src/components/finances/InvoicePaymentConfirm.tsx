'use client';

import type { BankAccount, Invoice } from '@/types/finances';
import { describeInvoicePayment } from '@/utils/invoicePayment';
import { formatCurrency } from '@/utils/format';

interface Props {
  invoice: Invoice;
  card: BankAccount;
  accounts: BankAccount[];
  paidAt: string;
  onPaidAtChange: (value: string) => void;
  onConfirm: (amount: number) => void;
  onCancel: () => void;
  busy: boolean;
  accent: 'red' | 'emerald';
}

/**
 * The confirmation step for paying a card bill, in one place.
 *
 * It existed twice — once in the overdue list, once in the due-today list — and
 * only the first got the preview and the refusals. Two copies of a flow that
 * writes to the ledger is one copy that will be hardened and one that will not,
 * and there is no way to tell them apart from the screen.
 */
export default function InvoicePaymentConfirm({
  invoice,
  card,
  accounts,
  paidAt,
  onPaidAtChange,
  onConfirm,
  onCancel,
  busy,
  accent,
}: Props) {
  const preview = describeInvoicePayment(invoice, card, accounts);

  if (preview.blockedReason) {
    return (
      <div className="pt-1 border-t border-white/10 space-y-1">
        <p className="text-amber-300 text-xs">{preview.blockedReason}</p>
        <button onClick={onCancel} className="text-white/40 hover:text-white/70 text-xs">
          fechar
        </button>
      </div>
    );
  }

  const ring = accent === 'red' ? 'focus:ring-red-400' : 'focus:ring-emerald-400';
  const button =
    accent === 'red' ? 'bg-red-600 hover:bg-red-700' : 'bg-emerald-600 hover:bg-emerald-700';

  return (
    <div className="space-y-1 pt-1 border-t border-white/10">
      {/* Say what this will write, before it is written. */}
      <p className="text-white/50 text-xs">
        {preview.noFundingAccount ? (
          <>
            Vai marcar <span className="text-white/80">{formatCurrency(preview.amount)}</span> como paga.
            Este cartão não tem conta vinculada, então nenhum lançamento de saída será criado.
          </>
        ) : (
          <>
            Vai lançar <span className="text-white/80">{formatCurrency(preview.amount)}</span> saindo de{' '}
            <span className="text-white/80">{preview.fundingAccountName}</span>.
          </>
        )}
      </p>
      <div className="flex items-center gap-2">
        <span className="text-white/60 text-xs">Pago em</span>
        <input
          type="date"
          value={paidAt}
          onChange={(e) => onPaidAtChange(e.target.value)}
          className={`px-2 py-1 bg-white/10 border border-white/20 rounded-lg text-white text-xs focus:outline-none focus:ring-1 ${ring}`}
        />
        <button
          onClick={() => onConfirm(preview.amount)}
          disabled={busy}
          className={`px-3 py-1 ${button} disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors`}
        >
          {busy ? 'Pagando...' : 'Confirmar'}
        </button>
        <button onClick={onCancel} className="text-white/40 hover:text-white/70 text-xs">
          cancelar
        </button>
      </div>
    </div>
  );
}
