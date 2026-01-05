'use client';

import { useMemo } from 'react';
import type {
  BankAccount,
  Category,
  Transaction,
  TransactionFilters,
  TransactionStatus,
  TransactionType,
} from '@/types/finances';

interface TransactionsTableProps {
  transactions: Transaction[];
  categories: Category[];
  accounts: BankAccount[];
  filters: TransactionFilters;
  onFilterChange: (filters: TransactionFilters) => void;
  onConfirm: (id: string) => Promise<void> | void;
  onCancel: (id: string) => Promise<void> | void;
  onDelete: (id: string) => Promise<void> | void;
  onEdit: (transaction: Transaction) => void;
  loading?: boolean;
}

const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

const formatDate = (value: string) =>
  new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(new Date(value));

const statusStyles: Record<TransactionStatus, string> = {
  PLANNED: 'bg-white/10 text-white',
  CONFIRMED: 'bg-emerald-500/20 text-emerald-200 border border-emerald-500/50',
  CANCELLED: 'bg-rose-500/20 text-rose-200 border border-rose-500/30',
};

const typeLabels: Record<TransactionType, string> = {
  INCOME: 'Receita',
  EXPENSE: 'Despesa',
  TRANSFER: 'Transferência',
};

export default function TransactionsTable({
  transactions,
  categories,
  accounts,
  filters,
  onFilterChange,
  onConfirm,
  onCancel,
  onDelete,
  onEdit,
  loading = false,
}: TransactionsTableProps) {
  const categoryMap = useMemo(() => {
    const map = new Map<string, Category>();
    categories.forEach((category) => map.set(category.id, category));
    return map;
  }, [categories]);

  const accountMap = useMemo(() => {
    const map = new Map<string, BankAccount>();
    accounts.forEach((account) => map.set(account.id, account));
    return map;
  }, [accounts]);

  const handleFilterChange = (partial: Partial<TransactionFilters>) => {
    onFilterChange({ ...filters, ...partial });
  };

  return (
    <div className="bg-white/5 border border-white/10 rounded-2xl overflow-hidden backdrop-blur-sm">
      <div className="p-6 border-b border-white/10 space-y-4">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <h3 className="text-xl font-semibold text-white">Movimentações</h3>
            <p className="text-white/60 text-sm">Controle completo de receitas, despesas e transferências</p>
          </div>
          <div className="flex flex-wrap gap-3">
            <select
              value={filters.bankAccountId ?? ''}
              onChange={(event) => handleFilterChange({ bankAccountId: event.target.value || null })}
              className="px-4 py-2 rounded-lg bg-white/10 border border-white/20 text-white text-sm"
            >
              <option value="" className="bg-slate-900">
                Todas as contas
              </option>
              {accounts.map((account) => (
                <option key={account.id} value={account.id} className="bg-slate-900">
                  {account.name}
                </option>
              ))}
            </select>
            <select
              value={filters.type}
              onChange={(event) =>
                handleFilterChange({ type: event.target.value as TransactionType | 'ALL' })
              }
              className="px-4 py-2 rounded-lg bg-white/10 border border-white/20 text-white text-sm"
            >
              <option value="ALL" className="bg-slate-900">
                Todos os tipos
              </option>
              <option value="INCOME" className="bg-slate-900">
                Receitas
              </option>
              <option value="EXPENSE" className="bg-slate-900">
                Despesas
              </option>
              <option value="TRANSFER" className="bg-slate-900">
                Transferências
              </option>
            </select>
            <select
              value={filters.status}
              onChange={(event) =>
                handleFilterChange({ status: event.target.value as TransactionStatus | 'ALL' })
              }
              className="px-4 py-2 rounded-lg bg-white/10 border border-white/20 text-white text-sm"
            >
              <option value="ALL" className="bg-slate-900">
                Todos os status
              </option>
              <option value="PLANNED" className="bg-slate-900">
                Planejado
              </option>
              <option value="CONFIRMED" className="bg-slate-900">
                Confirmado
              </option>
              <option value="CANCELLED" className="bg-slate-900">
                Cancelado
              </option>
            </select>
            <input
              type="date"
              value={filters.from ?? ''}
              onChange={(event) => handleFilterChange({ from: event.target.value || undefined })}
              className="px-4 py-2 rounded-lg bg-white/10 border border-white/20 text-white text-sm"
              placeholder="De"
            />
            <input
              type="date"
              value={filters.to ?? ''}
              onChange={(event) => handleFilterChange({ to: event.target.value || undefined })}
              className="px-4 py-2 rounded-lg bg-white/10 border border-white/20 text-white text-sm"
              placeholder="Até"
            />
          </div>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-white/10">
          <thead className="bg-white/5 text-white/70 text-sm uppercase tracking-wide">
            <tr>
              <th className="px-6 py-3 text-left">Data</th>
              <th className="px-6 py-3 text-left">Descrição</th>
              <th className="px-6 py-3 text-left">Conta</th>
              <th className="px-6 py-3 text-left">Categoria</th>
              <th className="px-6 py-3 text-left">Valor</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Ações</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {transactions.length === 0 && !loading && (
              <tr>
                <td colSpan={7} className="px-6 py-12 text-center text-white/60">
                  Nenhum lançamento encontrado com os filtros selecionados.
                </td>
              </tr>
            )}

            {transactions.map((transaction) => {
              const account = accountMap.get(transaction.bankAccountId);
              const category = transaction.categoryId
                ? categoryMap.get(transaction.categoryId)
                : undefined;
              const destination = transaction.destinationAccountId
                ? accountMap.get(transaction.destinationAccountId)
                : undefined;

              return (
                <tr key={transaction.id} className="hover:bg-white/5 transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap text-white/80 text-sm">
                    {formatDate(transaction.occurredOn)}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex flex-col">
                      <span className="text-white font-semibold text-sm">
                        {transaction.description}
                      </span>
                      <span className="text-white/50 text-xs">
                        {typeLabels[transaction.type]}
                        {transaction.type === 'TRANSFER' && destination
                          ? ` para ${destination.name}`
                          : ''}
                      </span>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-white/70 text-sm">
                    {account ? account.name : 'Conta removida'}
                  </td>
                  <td className="px-6 py-4 text-white/70 text-sm">
                    {category ? category.name : transaction.type === 'TRANSFER' ? 'Transferência' : '-'}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`font-semibold ${
                        transaction.type === 'INCOME' ? 'text-emerald-300' : 'text-rose-300'
                      }`}
                    >
                      {transaction.type === 'EXPENSE' ? '-' : ''}
                      {formatCurrency(transaction.amount, transaction.currency)}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`px-3 py-1 rounded-full text-xs font-semibold inline-flex items-center gap-2 ${
                        statusStyles[transaction.status]
                      }`}
                    >
                      {transaction.status === 'CONFIRMED' && '✅'}
                      {transaction.status === 'PLANNED' && '🗓️'}
                      {transaction.status === 'CANCELLED' && '🛑'}
                      {transaction.status === 'CONFIRMED'
                        ? 'Confirmado'
                        : transaction.status === 'PLANNED'
                        ? 'Planejado'
                        : 'Cancelado'}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex flex-wrap gap-2 text-xs">
                      <button
                        onClick={() => onEdit(transaction)}
                        className="px-3 py-1 rounded-lg bg-white/10 text-white/80 border border-white/20 hover:bg-white/20"
                        title="Editar"
                      >
                        Editar
                      </button>
                      {transaction.status === 'PLANNED' && (
                        <button
                          onClick={() => onConfirm(transaction.id)}
                          className="px-3 py-1 rounded-lg bg-emerald-500/20 text-emerald-200 border border-emerald-500/40 hover:bg-emerald-500/30"
                        >
                          Confirmar
                        </button>
                      )}
                      {transaction.status === 'CONFIRMED' && (
                        <button
                          onClick={() => onCancel(transaction.id)}
                          className="px-3 py-1 rounded-lg bg-amber-500/20 text-amber-200 border border-amber-500/40 hover:bg-amber-500/30"
                        >
                          Cancelar
                        </button>
                      )}
                      <button
                        onClick={() => onDelete(transaction.id)}
                        className="px-3 py-1 rounded-lg bg-rose-500/20 text-rose-200 border border-rose-500/40 hover:bg-rose-500/30"
                      >
                        Excluir
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}

            {loading && (
              <tr>
                <td colSpan={7} className="px-6 py-6 text-center text-white/60 text-sm">
                  Carregando lançamentos...
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
