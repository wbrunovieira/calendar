'use client';

import { useEffect, useMemo, useState } from 'react';
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

const formatDate = (value: string) => {
  // Extract date part to avoid timezone conversion (UTC -> local)
  const datePart = value.split('T')[0];
  const [year, month, day] = datePart.split('-').map(Number);
  const date = new Date(year, month - 1, day);

  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(date);
};

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

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

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
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

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

  // Pagination calculations
  const totalItems = transactions.length;
  const totalPages = Math.ceil(totalItems / pageSize);

  const paginatedTransactions = useMemo(() => {
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return transactions.slice(startIndex, endIndex);
  }, [transactions, currentPage, pageSize]);

  // Reset to page 1 when filters change (transactions array changes)
  useEffect(() => {
    setCurrentPage(1);
  }, [transactions.length]);

  const handleFilterChange = (partial: Partial<TransactionFilters>) => {
    setCurrentPage(1);
    onFilterChange({ ...filters, ...partial });
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setCurrentPage(1);
  };

  const goToPage = (page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page);
    }
  };

  // Generate page numbers to display
  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    const maxVisible = 5;

    if (totalPages <= maxVisible + 2) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
    } else {
      pages.push(1);

      if (currentPage > 3) pages.push('...');

      const start = Math.max(2, currentPage - 1);
      const end = Math.min(totalPages - 1, currentPage + 1);

      for (let i = start; i <= end; i++) pages.push(i);

      if (currentPage < totalPages - 2) pages.push('...');

      pages.push(totalPages);
    }

    return pages;
  };

  return (
    <div className="bg-white/5 border border-white/10 rounded-2xl overflow-hidden backdrop-blur-sm">
      <div className="p-6 border-b border-white/10 space-y-4">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <h3 className="text-xl font-semibold text-white">Movimentações</h3>
            <p className="text-white/60 text-sm">
              {totalItems} {totalItems === 1 ? 'lançamento' : 'lançamentos'}
              {totalPages > 1 && ` • Página ${currentPage} de ${totalPages}`}
            </p>
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
            {paginatedTransactions.length === 0 && !loading && (
              <tr>
                <td colSpan={7} className="px-6 py-12 text-center text-white/60">
                  Nenhum lançamento encontrado com os filtros selecionados.
                </td>
              </tr>
            )}

            {paginatedTransactions.map((transaction) => {
              const account = accountMap.get(transaction.bankAccountId);
              const category = transaction.categoryId
                ? categoryMap.get(transaction.categoryId)
                : undefined;
              const parentCategory = category?.parentId
                ? categoryMap.get(category.parentId)
                : undefined;
              const destination = transaction.destinationAccountId
                ? accountMap.get(transaction.destinationAccountId)
                : undefined;

              // Build category display with hierarchy
              const categoryDisplay = (() => {
                if (!category) {
                  return transaction.type === 'TRANSFER' ? 'Transferência' : '-';
                }
                if (parentCategory) {
                  return (
                    <span className="flex flex-col">
                      <span className="text-white/50 text-xs">{parentCategory.name}</span>
                      <span>↳ {category.name}</span>
                    </span>
                  );
                }
                return category.name;
              })();

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
                    {categoryDisplay}
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

      {/* Pagination Controls */}
      {totalPages > 1 && (
        <div className="p-4 border-t border-white/10 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-sm text-white/60">
            <span>Exibir</span>
            <select
              value={pageSize}
              onChange={(e) => handlePageSizeChange(Number(e.target.value))}
              className="px-2 py-1 rounded bg-white/10 border border-white/20 text-white text-sm"
            >
              {PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size} className="bg-slate-900">
                  {size}
                </option>
              ))}
            </select>
            <span>por página</span>
          </div>

          <div className="flex items-center gap-1">
            <button
              onClick={() => goToPage(1)}
              disabled={currentPage === 1}
              className="px-3 py-1.5 rounded-lg text-sm border border-white/20 text-white/80 hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed"
              title="Primeira página"
            >
              ««
            </button>
            <button
              onClick={() => goToPage(currentPage - 1)}
              disabled={currentPage === 1}
              className="px-3 py-1.5 rounded-lg text-sm border border-white/20 text-white/80 hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed"
              title="Página anterior"
            >
              «
            </button>

            {getPageNumbers().map((page, index) =>
              typeof page === 'number' ? (
                <button
                  key={index}
                  onClick={() => goToPage(page)}
                  className={`px-3 py-1.5 rounded-lg text-sm border transition-colors ${
                    page === currentPage
                      ? 'bg-white/20 border-white/40 text-white font-semibold'
                      : 'border-white/20 text-white/80 hover:bg-white/10'
                  }`}
                >
                  {page}
                </button>
              ) : (
                <span key={index} className="px-2 text-white/40">
                  {page}
                </span>
              )
            )}

            <button
              onClick={() => goToPage(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="px-3 py-1.5 rounded-lg text-sm border border-white/20 text-white/80 hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed"
              title="Próxima página"
            >
              »
            </button>
            <button
              onClick={() => goToPage(totalPages)}
              disabled={currentPage === totalPages}
              className="px-3 py-1.5 rounded-lg text-sm border border-white/20 text-white/80 hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed"
              title="Última página"
            >
              »»
            </button>
          </div>

          <div className="text-sm text-white/60">
            {((currentPage - 1) * pageSize) + 1}-{Math.min(currentPage * pageSize, totalItems)} de {totalItems}
          </div>
        </div>
      )}
    </div>
  );
}
