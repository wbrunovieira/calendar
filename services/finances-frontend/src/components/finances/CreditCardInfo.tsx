'use client';

import { useState, useMemo } from 'react';
import { BankAccount, Invoice } from '@/types/finances';
import { formatCurrency, formatDate } from '@/utils/format';

interface InvoiceUpdateData {
  closingDate?: string;
  dueDate?: string;
}

interface CreditCardInfoProps {
  account: BankAccount;
  currentInvoice?: Invoice;
  invoices: Invoice[];
  onPayInvoice?: (invoiceId: string, amount: number) => Promise<void>;
  onEdit?: () => void;
  onUpdateInvoice?: (invoiceId: string, data: InvoiceUpdateData) => Promise<void>;
  selectedInvoiceId?: string;
  onInvoiceSelect?: (invoiceId: string) => void;
}

const getMonthName = (dateStr: string) => {
  const date = new Date(dateStr);
  return date.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' });
};

const getStatusLabel = (status: string) => {
  switch (status) {
    case 'OPEN':
      return { label: 'Aberta', color: 'bg-blue-500/20 text-blue-400' };
    case 'CLOSED':
      return { label: 'Fechada', color: 'bg-yellow-500/20 text-yellow-400' };
    case 'PAID':
      return { label: 'Paga', color: 'bg-green-500/20 text-green-400' };
    default:
      return { label: status, color: 'bg-gray-500/20 text-gray-400' };
  }
};

export default function CreditCardInfo({
  account,
  currentInvoice,
  invoices,
  onPayInvoice,
  onEdit,
  onUpdateInvoice,
  selectedInvoiceId,
  onInvoiceSelect,
}: CreditCardInfoProps) {
  const [showHistory, setShowHistory] = useState(false);
  const [payingInvoice, setPayingInvoice] = useState<string | null>(null);
  const [showPaymentOptions, setShowPaymentOptions] = useState(false);
  const [editingInvoiceId, setEditingInvoiceId] = useState<string | null>(null);
  const [editClosingDate, setEditClosingDate] = useState('');
  const [editDueDate, setEditDueDate] = useState('');
  const [savingInvoice, setSavingInvoice] = useState(false);

  const startEditInvoice = (invoice: Invoice) => {
    setEditingInvoiceId(invoice.id);
    setEditClosingDate(invoice.closingDate.split('T')[0]);
    setEditDueDate(invoice.dueDate.split('T')[0]);
  };

  const cancelEditInvoice = () => {
    setEditingInvoiceId(null);
    setEditClosingDate('');
    setEditDueDate('');
  };

  const saveEditInvoice = async () => {
    if (!editingInvoiceId || !onUpdateInvoice) return;
    setSavingInvoice(true);
    try {
      await onUpdateInvoice(editingInvoiceId, {
        closingDate: editClosingDate,
        dueDate: editDueDate,
      });
      setEditingInvoiceId(null);
    } finally {
      setSavingInvoice(false);
    }
  };

  const limit = account.creditLimit || 0;
  const currentAmount = currentInvoice?.amount || 0;
  const confirmedAmount = currentInvoice?.confirmedAmount || 0;
  const plannedAmount = currentInvoice?.plannedAmount || 0;
  const available = limit - currentAmount;
  const usagePercent = limit > 0 ? (currentAmount / limit) * 100 : 0;

  // Get all unpaid invoices (OPEN or CLOSED with amount > 0)
  const unpaidInvoices = useMemo(() => {
    return invoices.filter(
      (inv) => inv.status !== 'PAID' && inv.amount > 0
    ).sort((a, b) => new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime());
  }, [invoices]);

  const handlePayInvoice = async (invoice: Invoice) => {
    if (!onPayInvoice || payingInvoice) return;
    setPayingInvoice(invoice.id);
    try {
      await onPayInvoice(invoice.id, invoice.amount);
      setShowPaymentOptions(false);
    } finally {
      setPayingInvoice(null);
    }
  };

  const handlePayButtonClick = () => {
    if (unpaidInvoices.length === 1) {
      // Only one unpaid invoice, pay it directly
      handlePayInvoice(unpaidInvoices[0]);
    } else if (unpaidInvoices.length > 1) {
      // Multiple unpaid invoices, show selection
      setShowPaymentOptions(!showPaymentOptions);
    }
  };

  return (
    <div className="border border-white/10 rounded-xl overflow-hidden bg-gradient-to-br from-white/5 to-white/[0.02]">
      {/* Header */}
      <div
        className="p-4 flex items-center justify-between"
        style={{ borderLeft: `4px solid ${account.color || '#10b981'}` }}
      >
        <div className="flex items-center gap-3">
          <span className="text-2xl">{account.icon || '💳'}</span>
          <div>
            <p className="text-white font-semibold">{account.name}</p>
            <p className="text-white/50 text-xs">
              Fecha dia {account.closingDay} | Vence dia {account.dueDay}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <div className="text-right">
            <p className="text-white/50 text-xs">Limite</p>
            <p className="text-white font-semibold">{formatCurrency(limit)}</p>
          </div>
          {onEdit && (
            <button
              onClick={(e) => { e.stopPropagation(); onEdit(); }}
              className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-white/40 hover:text-white/80"
              title="Editar conta"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Current Invoice */}
      <div className="px-4 pb-4 space-y-3">
        {/* Usage Bar */}
        <div className="space-y-1">
          <div className="flex justify-between text-xs">
            <span className="text-white/60">Utilizado</span>
            <span className="text-white/60">{usagePercent.toFixed(0)}%</span>
          </div>
          <div className="h-2 bg-white/10 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${
                usagePercent > 80 ? 'bg-red-500' : usagePercent > 50 ? 'bg-yellow-500' : 'bg-emerald-500'
              }`}
              style={{ width: `${Math.min(usagePercent, 100)}%` }}
            />
          </div>
        </div>

        {/* Invoice Info */}
        <div className="grid grid-cols-2 gap-3">
          <div className="bg-white/5 rounded-lg p-3">
            <p className="text-white/50 text-xs mb-1">Fatura Atual</p>
            <p className="text-white font-bold text-lg">{formatCurrency(confirmedAmount || currentAmount)}</p>
            {plannedAmount > 0 && (
              <p className="text-yellow-400/70 text-xs">
                + {formatCurrency(plannedAmount)} previsto
              </p>
            )}
            {currentInvoice && (
              <p className="text-white/50 text-xs">
                Fecha {formatDate(currentInvoice.closingDate, 'short')} | Vence {formatDate(currentInvoice.dueDate, 'short')}
              </p>
            )}
          </div>
          <div className="bg-white/5 rounded-lg p-3">
            <p className="text-white/50 text-xs mb-1">Disponivel</p>
            <p className={`font-bold text-lg ${available >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
              {formatCurrency(available)}
            </p>
            <p className="text-white/50 text-xs">
              {available >= 0 ? 'para usar' : 'acima do limite'}
            </p>
          </div>
        </div>

        {/* Pay Button */}
        {unpaidInvoices.length > 0 && onPayInvoice && (
          <div className="space-y-2">
            <button
              onClick={handlePayButtonClick}
              disabled={!!payingInvoice}
              className="w-full py-2 px-4 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white rounded-lg font-semibold text-sm transition-colors flex items-center justify-center gap-2"
            >
              {payingInvoice ? (
                <>
                  <span className="animate-spin">⏳</span>
                  Pagando...
                </>
              ) : (
                <>
                  <span>💳</span>
                  Pagar Fatura {unpaidInvoices.length > 1 && `(${unpaidInvoices.length})`}
                </>
              )}
            </button>

            {/* Payment Options - Multiple Invoices */}
            {showPaymentOptions && unpaidInvoices.length > 1 && (
              <div className="bg-white/5 rounded-lg p-3 space-y-2 animate-in fade-in slide-in-from-top-2 duration-200">
                <p className="text-white/60 text-xs mb-2">Selecione a fatura para pagar:</p>
                {unpaidInvoices.map((invoice) => {
                  const status = getStatusLabel(invoice.status);
                  return (
                    <button
                      key={invoice.id}
                      onClick={() => handlePayInvoice(invoice)}
                      disabled={payingInvoice === invoice.id}
                      className="w-full flex items-center justify-between p-3 bg-white/5 hover:bg-white/10 rounded-lg transition-colors disabled:opacity-50"
                    >
                      <div className="text-left">
                        <p className="text-white text-sm capitalize">{getMonthName(invoice.referenceDate)}</p>
                        <p className="text-white/50 text-xs">
                          Fecha {formatDate(invoice.closingDate, 'short')} | Vence {formatDate(invoice.dueDate, 'short')}
                          <span className={`ml-2 px-1.5 py-0.5 rounded text-[10px] ${status.color}`}>
                            {status.label}
                          </span>
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-white font-semibold">{formatCurrency(invoice.amount)}</p>
                        {payingInvoice === invoice.id ? (
                          <span className="text-xs text-white/50">Pagando...</span>
                        ) : (
                          <span className="text-xs text-emerald-400">Pagar</span>
                        )}
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* Invoice History Toggle */}
        {invoices.length > 0 && (
          <button
            onClick={() => setShowHistory(!showHistory)}
            className="w-full text-center text-white/50 text-xs hover:text-white/80 transition-colors py-1"
          >
            {showHistory ? '▲ Ocultar historico' : `▼ Ver historico (${invoices.length} faturas)`}
          </button>
        )}

        {/* Invoice History */}
        {showHistory && invoices.length > 0 && (
          <div className="space-y-2 pt-2 border-t border-white/10">
            {invoices.map((invoice) => {
              const status = getStatusLabel(invoice.status);
              const canPay = invoice.status !== 'PAID' && invoice.amount > 0;
              const isSelected = selectedInvoiceId === invoice.id;
              const isEditing = editingInvoiceId === invoice.id;
              return (
                <div key={invoice.id}>
                  <div
                    className={`flex items-center justify-between py-2 px-3 rounded-lg cursor-pointer transition-all duration-200 ${
                      isSelected
                        ? 'bg-emerald-500/15'
                        : 'bg-white/5 hover:bg-white/10'
                    }`}
                    style={{
                      border: isSelected ? '2px solid #34d399' : '1px solid rgba(255,255,255,0.06)',
                      boxShadow: isSelected ? '0 0 20px rgba(52, 211, 153, 0.4), inset 0 0 12px rgba(52, 211, 153, 0.08)' : 'none',
                      borderRadius: isEditing ? '0.5rem 0.5rem 0 0' : undefined,
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      if (!isEditing) onInvoiceSelect?.(isSelected ? '' : invoice.id);
                    }}
                  >
                    <div className="flex items-center gap-2">
                      {isSelected && (
                        <span className="text-emerald-400 text-sm">&#10003;</span>
                      )}
                      <div>
                        <p className={`text-sm capitalize ${isSelected ? 'text-emerald-300 font-semibold' : 'text-white'}`}>{getMonthName(invoice.referenceDate)}</p>
                        <p className="text-white/50 text-xs">Fecha {formatDate(invoice.closingDate, 'short')} | Vence {formatDate(invoice.dueDate, 'short')}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="text-right">
                        <p className="text-white font-semibold text-sm">{formatCurrency(invoice.amount)}</p>
                        <span className={`text-xs px-2 py-0.5 rounded-full ${status.color}`}>
                          {status.label}
                        </span>
                      </div>
                      {onUpdateInvoice && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            if (isEditing) cancelEditInvoice();
                            else startEditInvoice(invoice);
                          }}
                          className="p-1 rounded-lg hover:bg-white/10 transition-colors text-white/30 hover:text-white/70"
                          title="Editar fatura"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                          </svg>
                        </button>
                      )}
                      {canPay && onPayInvoice && (
                        <button
                          onClick={(e) => { e.stopPropagation(); handlePayInvoice(invoice); }}
                          disabled={payingInvoice === invoice.id}
                          className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                        >
                          {payingInvoice === invoice.id ? '...' : 'Pagar'}
                        </button>
                      )}
                    </div>
                  </div>
                  {isEditing && (
                    <div
                      className="bg-white/5 border border-white/10 border-t-0 rounded-b-lg p-3 space-y-2"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="text-white/50 text-[10px] block mb-0.5">Fechamento</label>
                          <input
                            type="date"
                            value={editClosingDate}
                            onChange={(e) => setEditClosingDate(e.target.value)}
                            className="w-full bg-white/10 border border-white/20 rounded px-2 py-1 text-white text-xs focus:outline-none focus:border-emerald-400"
                          />
                        </div>
                        <div>
                          <label className="text-white/50 text-[10px] block mb-0.5">Vencimento</label>
                          <input
                            type="date"
                            value={editDueDate}
                            onChange={(e) => setEditDueDate(e.target.value)}
                            className="w-full bg-white/10 border border-white/20 rounded px-2 py-1 text-white text-xs focus:outline-none focus:border-emerald-400"
                          />
                        </div>
                      </div>
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={cancelEditInvoice}
                          className="px-3 py-1 text-white/50 hover:text-white/80 text-xs transition-colors"
                        >
                          Cancelar
                        </button>
                        <button
                          onClick={saveEditInvoice}
                          disabled={savingInvoice}
                          className="px-3 py-1 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                        >
                          {savingInvoice ? 'Salvando...' : 'Salvar'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
