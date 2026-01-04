'use client';

import { useState } from 'react';
import { BankAccount, Invoice } from '@/types/finances';

interface CreditCardInfoProps {
  account: BankAccount;
  currentInvoice?: Invoice;
  invoices: Invoice[];
  onPayInvoice?: (invoiceId: string, amount: number) => Promise<void>;
  onEdit?: () => void;
}

const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr);
  return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' });
};

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
}: CreditCardInfoProps) {
  const [showHistory, setShowHistory] = useState(false);
  const [payingInvoice, setPayingInvoice] = useState<string | null>(null);

  const limit = account.creditLimit || 0;
  const currentAmount = currentInvoice?.amount || 0;
  const available = limit - currentAmount;
  const usagePercent = limit > 0 ? (currentAmount / limit) * 100 : 0;

  const handlePayInvoice = async (invoice: Invoice) => {
    if (!onPayInvoice || payingInvoice) return;
    setPayingInvoice(invoice.id);
    try {
      await onPayInvoice(invoice.id, invoice.amount);
    } finally {
      setPayingInvoice(null);
    }
  };

  return (
    <div className="border border-white/10 rounded-xl overflow-hidden bg-gradient-to-br from-white/5 to-white/[0.02]">
      {/* Header */}
      <div
        className="p-4 flex items-center justify-between cursor-pointer hover:bg-white/5 transition-colors"
        onClick={onEdit}
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
        <div className="text-right">
          <p className="text-white/50 text-xs">Limite</p>
          <p className="text-white font-semibold">{formatCurrency(limit)}</p>
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
            <p className="text-white font-bold text-lg">{formatCurrency(currentAmount)}</p>
            {currentInvoice && (
              <p className="text-white/50 text-xs">
                Vence {formatDate(currentInvoice.dueDate)}
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
        {currentInvoice && currentInvoice.status !== 'PAID' && currentInvoice.amount > 0 && onPayInvoice && (
          <button
            onClick={() => handlePayInvoice(currentInvoice)}
            disabled={payingInvoice === currentInvoice.id}
            className="w-full py-2 px-4 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white rounded-lg font-semibold text-sm transition-colors flex items-center justify-center gap-2"
          >
            {payingInvoice === currentInvoice.id ? (
              <>
                <span className="animate-spin">⏳</span>
                Pagando...
              </>
            ) : (
              <>
                <span>💳</span>
                Pagar Fatura
              </>
            )}
          </button>
        )}

        {/* Invoice History Toggle */}
        {invoices.length > 0 && (
          <button
            onClick={() => setShowHistory(!showHistory)}
            className="w-full text-center text-white/50 text-xs hover:text-white/80 transition-colors py-1"
          >
            {showHistory ? '▲ Ocultar histórico' : `▼ Ver histórico (${invoices.length} faturas)`}
          </button>
        )}

        {/* Invoice History */}
        {showHistory && invoices.length > 0 && (
          <div className="space-y-2 pt-2 border-t border-white/10">
            {invoices.map((invoice) => {
              const status = getStatusLabel(invoice.status);
              return (
                <div
                  key={invoice.id}
                  className="flex items-center justify-between py-2 px-3 bg-white/5 rounded-lg"
                >
                  <div>
                    <p className="text-white text-sm capitalize">{getMonthName(invoice.referenceDate)}</p>
                    <p className="text-white/50 text-xs">Venceu {formatDate(invoice.dueDate)}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-white font-semibold text-sm">{formatCurrency(invoice.amount)}</p>
                    <span className={`text-xs px-2 py-0.5 rounded-full ${status.color}`}>
                      {status.label}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
