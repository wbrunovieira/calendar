'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import TransactionForm from '@/components/finances/TransactionForm';
import TransactionsTable from '@/components/finances/TransactionsTable';
import CashflowSummary from '@/components/finances/CashflowSummary';
import QuickExpense from '@/components/finances/QuickExpense';
import GlobalSearch from '@/components/finances/GlobalSearch';
import TodayAlerts from '@/components/finances/TodayAlerts';
import { useToast } from '@/components/ui/Toast';
import type {
  BankAccount,
  Category,
  Transaction,
  TransactionFilters,
  TransactionFormData,
  BudgetSummaryItem,
  Invoice,
  RecurringTransaction,
} from '@/types/finances';
import { api, ApiError } from '@/lib/api';

// Format date to YYYY-MM-DD without timezone conversion
const formatLocalDate = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

// Get current month's first and last day
const getCurrentMonthRange = () => {
  const now = new Date();
  const firstDay = new Date(now.getFullYear(), now.getMonth(), 1);
  const lastDay = new Date(now.getFullYear(), now.getMonth() + 1, 0);
  return {
    from: formatLocalDate(firstDay),
    to: formatLocalDate(lastDay),
  };
};

const defaultFilters: TransactionFilters = {
  bankAccountId: null,
  type: 'ALL',
  status: 'ALL',
  ...getCurrentMonthRange(),
};

export default function FinancesPage() {
  // Use shared profile context
  const { profiles, selectedProfileId, selectedProfile, isLoading: profilesLoading } = useProfile();
  const { toast, confirm } = useToast();

  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [, setBudgetSummary] = useState<BudgetSummaryItem[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [transactionsLoading, setTransactionsLoading] = useState(false);
  const [transactionsError, setTransactionsError] = useState<'NOT_IMPLEMENTED' | 'GENERIC' | null>(null);
  const [transactionFilters, setTransactionFilters] = useState<TransactionFilters>({ ...defaultFilters });

  const [isTransactionModalOpen, setIsTransactionModalOpen] = useState(false);

  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null);

  const [savingTransaction, setSavingTransaction] = useState(false);
  const [searchRefreshKey, setSearchRefreshKey] = useState(0);

  // Invoice state for credit cards
  const [invoicesByAccount, setInvoicesByAccount] = useState<Record<string, Invoice[]>>({});
  const [currentInvoices, setCurrentInvoices] = useState<Record<string, Invoice>>({});

  // Recurring transactions for alerts
  const [recurringTransactions, setRecurringTransactions] = useState<RecurringTransaction[]>([]);

  const filteredAccounts = useMemo(
    () => bankAccounts.filter((account) => account.profileId === selectedProfileId),
    [bankAccounts, selectedProfileId],
  );

  useEffect(() => {
    fetchBankAccounts();
  }, []);

  const fetchBankAccounts = async () => {
    try {
      const data = await api.get<{ data: BankAccount[] }>('/bank-accounts');
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar contas bancárias:', error);
    }
  };

  const fetchInvoicesForCreditCards = useCallback(async (accounts: BankAccount[]) => {
    const creditCards = accounts.filter((acc) => acc.type === 'CREDIT_CARD');
    if (creditCards.length === 0) return;

    const invoicesMap: Record<string, Invoice[]> = {};
    const currentMap: Record<string, Invoice> = {};

    await Promise.all(
      creditCards.map(async (card) => {
        try {
          // Fetch all invoices for the card
          const data = await api.get<{ data: Invoice[] }>(`/invoices?bankAccountId=${card.id}`);
          invoicesMap[card.id] = data.data || [];

          // Fetch current invoice
          const currentData = await api.get<{ data: Invoice }>(`/invoices/current?bankAccountId=${card.id}`);
          if (currentData.data) {
            currentMap[card.id] = currentData.data;
          }
        } catch (error) {
          console.warn(`Erro ao carregar faturas do cartão ${card.name}:`, error);
        }
      }),
    );

    setInvoicesByAccount(invoicesMap);
    setCurrentInvoices(currentMap);
  }, []);

  const fetchCategories = useCallback(async (profileId: string) => {
    try {
      const data = await api.get<{ data: Category[] }>(`/categories?profileId=${profileId}`);
      setCategories(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar categorias:', error);
      setCategories([]);
    }
  }, []);

  const fetchBudgetSummary = useCallback(async (profileId: string) => {
    try {
      const month = new Date().toISOString().slice(0, 7);
      const data = await api.get<{ data: BudgetSummaryItem[] }>(`/budgets/summary?profileId=${profileId}&period=${month}`);
      setBudgetSummary(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar summary de orçamentos:', error);
      setBudgetSummary([]);
    }
  }, []);

  const fetchTransactions = useCallback(async (profileId: string, filters: TransactionFilters) => {
    try {
      setTransactionsLoading(true);
      setTransactionsError(null);
      const params = new URLSearchParams({ profileId });

      if (filters.bankAccountId) params.append('bankAccountId', filters.bankAccountId);
      if (filters.type && filters.type !== 'ALL') params.append('type', filters.type);
      if (filters.status && filters.status !== 'ALL') params.append('status', filters.status);
      if (filters.from) params.append('occurredFrom', filters.from);
      if (filters.to) params.append('occurredTo', filters.to);

      const data = await api.get<{ data: Transaction[] }>(`/transactions?${params.toString()}`);
      setTransactions(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar lançamentos:', error);
      if (error instanceof ApiError && error.status === 501) {
        setTransactionsError('NOT_IMPLEMENTED');
      } else {
        setTransactionsError('GENERIC');
      }
      setTransactions([]);
    } finally {
      setTransactionsLoading(false);
    }
  }, []);

  const fetchRecurringTransactions = useCallback(async (profileId: string) => {
    try {
      const data = await api.get<{ data: RecurringTransaction[] }>(`/recurring-transactions?profileId=${profileId}`);
      setRecurringTransactions(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar transacoes recorrentes:', error);
      setRecurringTransactions([]);
    }
  }, []);

  useEffect(() => {
    if (!selectedProfileId) {
      setCategories([]);
      setTransactions([]);
      setRecurringTransactions([]);
      return;
    }

    const baseFilters: TransactionFilters = { ...defaultFilters };
    setTransactionFilters(baseFilters);
    fetchCategories(selectedProfileId);
    fetchTransactions(selectedProfileId, baseFilters);
    fetchBudgetSummary(selectedProfileId);
    fetchRecurringTransactions(selectedProfileId);
  }, [selectedProfileId, fetchCategories, fetchTransactions, fetchBudgetSummary, fetchRecurringTransactions]);

  // Fetch invoices when filtered accounts change
  useEffect(() => {
    if (filteredAccounts.length > 0) {
      fetchInvoicesForCreditCards(filteredAccounts);
    }
  }, [filteredAccounts, fetchInvoicesForCreditCards]);

  const handleCreateTransaction = async (payload: TransactionFormData) => {
    if (!selectedProfileId) {
      toast('Selecione um perfil financeiro antes de criar lancamentos.', 'warning');
      return;
    }

    try {
      setSavingTransaction(true);
      // Remove undefined/null values to avoid API validation issues
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      await api.post('/transactions', cleanPayload);

      await fetchTransactions(selectedProfileId, transactionFilters);
      await fetchBankAccounts();
      setEditingTransaction(null);
      setSearchRefreshKey((k) => k + 1);
    } catch (error) {
      console.error('Erro ao criar lançamento:', error);
      toast('Erro ao salvar lancamento', 'error');
    } finally {
      setSavingTransaction(false);
    }
  };

  const handleUpdateTransaction = async (id: string, payload: TransactionFormData) => {
    if (!selectedProfileId) return;

    try {
      setSavingTransaction(true);
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      await api.put(`/transactions/${id}`, cleanPayload);

      await fetchTransactions(selectedProfileId, transactionFilters);
      await fetchBankAccounts();
      setEditingTransaction(null);
      setSearchRefreshKey((k) => k + 1);
    } catch (error) {
      console.error('Erro ao atualizar lançamento:', error);
      toast('Erro ao atualizar lancamento', 'error');
    } finally {
      setSavingTransaction(false);
    }
  };

  const handleEditTransaction = (transaction: Transaction) => {
    setEditingTransaction(transaction);
    setIsTransactionModalOpen(true);
  };

  const updateTransactionStatus = async (
    id: string,
    status: 'CONFIRMED' | 'CANCELLED',
  ) => {
    const transaction = transactions.find((item) => item.id === id);
    if (!transaction) return;

    try {
      await api.put(`/transactions/${id}/status`, {
        status,
        occurredOn: transaction.occurredOn,
        reason: status === 'CANCELLED' ? 'Cancelado no painel de finanças' : undefined,
      });

      if (selectedProfileId) {
        await fetchTransactions(selectedProfileId, transactionFilters);
        await fetchBankAccounts();
        setSearchRefreshKey((k) => k + 1);
      }
    } catch (error) {
      console.error('Erro ao atualizar status do lançamento:', error);
      toast('Erro ao atualizar status do lancamento', 'error');
    }
  };

  const handleDeleteTransaction = async (id: string) => {
    const ok = await confirm('Deseja realmente excluir este lancamento?');
    if (!ok) return;

    try {
      await api.delete(`/transactions/${id}`);

      if (selectedProfileId) {
        await fetchTransactions(selectedProfileId, transactionFilters);
        await fetchBankAccounts();
      }
      toast('Lancamento excluido com sucesso');
    } catch (error) {
      console.error('Erro ao excluir lancamento:', error);
      toast('Erro ao excluir lancamento', 'error');
    }
  };

  const handlePayInvoice = async (invoiceId: string, amount: number) => {
    try {
      const today = new Date().toISOString().split('T')[0]; // Format: YYYY-MM-DD
      await api.post(`/invoices/${invoiceId}/pay`, { paidAmount: amount, paidAt: today });

      // Refresh invoices after payment
      await fetchInvoicesForCreditCards(filteredAccounts);
      // Also refresh bank accounts to update balances
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao pagar fatura:', error);
      toast('Erro ao pagar fatura', 'error');
    }
  };

  const handleFilterChange = (filters: TransactionFilters) => {
    setTransactionFilters(filters);
    if (selectedProfileId) {
      fetchTransactions(selectedProfileId, filters);
    }
  };

  const openTransactionModal = () => {
    if (!selectedProfileId) {
      toast('Crie ou selecione um perfil financeiro antes de registrar lancamentos.', 'warning');
      return;
    }
    setIsTransactionModalOpen(true);
  };


  // Handle search result selection
  const handleSearchSelectTransaction = useCallback(async (id: string) => {
    // Try local list first
    const tx = transactions.find((t) => t.id === id);
    if (tx) {
      setEditingTransaction(tx);
      setIsTransactionModalOpen(true);
      return;
    }
    // Fetch from API if not in current filtered list
    try {
      const data = await api.get<{ data: Transaction }>(`/transactions/${id}`);
      const remoteTx = data.data || data;
      setEditingTransaction(remoteTx as Transaction);
      setIsTransactionModalOpen(true);
    } catch (error) {
      console.error('Erro ao buscar transacao:', error);
    }
  }, [transactions]);

  const handleSearchSelectRecurring = useCallback(() => {
    // Navigate to recurring page
    window.location.href = '/recurring';
  }, []);

  // Handler to confirm a recurring transaction (create a confirmed transaction from it)
  const handleConfirmRecurring = useCallback(async (recurring: RecurringTransaction) => {
    if (!selectedProfileId) return;

    try {
      // Create a confirmed transaction based on the recurring
      const transactionData = {
        profileId: recurring.profileId,
        bankAccountId: recurring.bankAccountId || filteredAccounts[0]?.id,
        categoryId: recurring.categoryId,
        type: recurring.type,
        status: 'CONFIRMED',
        amount: recurring.amount,
        currency: recurring.currency,
        description: recurring.description,
        notes: recurring.notes,
        occurredOn: recurring.nextOccurrence.includes('T')
          ? recurring.nextOccurrence.split('T')[0]
          : recurring.nextOccurrence,
      };

      await api.post('/transactions', transactionData);

      // Refresh transactions and balances to update the alerts
      await fetchTransactions(selectedProfileId, transactionFilters);
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao confirmar recorrente:', error);
      toast('Erro ao confirmar pagamento', 'error');
    }
  }, [selectedProfileId, filteredAccounts, transactionFilters, fetchTransactions, fetchBankAccounts]);

  return (
    <AppLayout>
      <div className="py-10 space-y-8">
        {/* Global Search */}
        <div className="mb-10">
          <GlobalSearch
            profileId={selectedProfileId}
            categories={categories}
            accounts={filteredAccounts}
            onSelectTransaction={handleSearchSelectTransaction}
            onSelectRecurring={handleSearchSelectRecurring}
            refreshKey={searchRefreshKey}
          />
        </div>

        {/* Today Alerts */}
        {selectedProfileId && (
          <TodayAlerts
            transactions={transactions}
            recurringTransactions={recurringTransactions}
            invoices={invoicesByAccount}
            accounts={filteredAccounts}
            categories={categories}
            onPayInvoice={handlePayInvoice}
            onConfirmTransaction={(id) => updateTransactionStatus(id, 'CONFIRMED')}
            onConfirmRecurring={handleConfirmRecurring}
            onEditTransaction={handleEditTransaction}
          />
        )}

        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold text-white">Despesas e planejamento</h1>
            <p className="text-white/60 text-sm">Registre gastos diários, semanais e mensais; programe fixas e controle orçamentos por categoria.</p>
          </div>
          <button
            onClick={openTransactionModal}
            className={`flex items-center gap-2 px-5 py-2 rounded-xl text-white font-semibold transition-colors ${
              selectedProfile?.type === 'BUSINESS'
                ? 'bg-amber-500/80 hover:bg-amber-500 border border-amber-400/40'
                : 'bg-emerald-500/80 hover:bg-emerald-500 border border-emerald-400/40'
            }`}
          >
            <span>➕</span>
            <span>Novo lançamento</span>
          </button>
        </div>

        <div className="space-y-6">
            {profilesLoading ? (
              <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center text-white/70">
                Carregando perfis...
              </div>
            ) : profiles.length === 0 ? (
              <div className="bg-white/10 border border-white/10 rounded-2xl p-8 text-center text-white/70">
                Cadastre um perfil financeiro para começar a registrar lançamentos.
              </div>
            ) : selectedProfile && (
              <>
<CashflowSummary transactions={transactions} accounts={filteredAccounts} currentInvoices={currentInvoices} />

                <div className="space-y-6">
                    <QuickExpense
                      accounts={bankAccounts}
                      categories={categories}
                      defaultProfileId={selectedProfileId ?? ''}
                      profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
                      onSave={handleCreateTransaction}
                    />
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="flex gap-2">
                        <button
                          onClick={() => {
                            const today = formatLocalDate(new Date());
                            const next: TransactionFilters = { ...transactionFilters, from: today, to: today, type: 'EXPENSE' };
                            handleFilterChange(next);
                          }}
                          className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10"
                        >
                          Hoje
                        </button>
                        <button
                          onClick={() => {
                            const now = new Date();
                            const day = (now.getDay() + 6) % 7; // 0 = Monday
                            const start = new Date(now);
                            start.setDate(now.getDate() - day);
                            const end = new Date(start);
                            end.setDate(start.getDate() + 6);
                            const from = formatLocalDate(start);
                            const to = formatLocalDate(end);
                            const next: TransactionFilters = { ...transactionFilters, from, to, type: 'EXPENSE' };
                            handleFilterChange(next);
                          }}
                          className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10"
                        >
                          Semana
                        </button>
                        <button
                          onClick={() => {
                            const now = new Date();
                            const start = new Date(now.getFullYear(), now.getMonth(), 1);
                            const end = new Date(now.getFullYear(), now.getMonth() + 1, 0);
                            const from = formatLocalDate(start);
                            const to = formatLocalDate(end);
                            const next: TransactionFilters = { ...transactionFilters, from, to, type: 'EXPENSE' };
                            handleFilterChange(next);
                          }}
                          className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10"
                        >
                          Mês
                        </button>
                      </div>
                      <div className="flex gap-2">
                        <a href="/visao" className="px-3 py-1.5 rounded-lg text-xs border border-blue-500/40 text-blue-400 hover:bg-blue-500/10">Visao Mensal</a>
                        <a href="/recurring" className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10">Fixas</a>
                        <a href="/budgets" className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10">Orçamentos</a>
                      </div>
                    </div>
                    {transactionsError === 'NOT_IMPLEMENTED' && (
                      <div className="bg-amber-500/20 border border-amber-400/40 text-amber-100 rounded-2xl p-4">
                        O backend de lançamentos (GET /transactions) ainda não está disponível.
                        Assim que for ativado, esta visão mostrará os registros automaticamente.
                      </div>
                    )}
                    {transactionsError === 'GENERIC' && (
                      <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-2xl p-4">
                        Não foi possível carregar os lançamentos agora. Tente novamente mais tarde.
                      </div>
                    )}
                    <TransactionsTable
                      transactions={transactions}
                      categories={categories}
                      accounts={filteredAccounts}
                      filters={transactionFilters}
                      onFilterChange={handleFilterChange}
                      onConfirm={(id) => updateTransactionStatus(id, 'CONFIRMED')}
                      onCancel={(id) => updateTransactionStatus(id, 'CANCELLED')}
                      onDelete={handleDeleteTransaction}
                      onEdit={handleEditTransaction}
                      loading={transactionsLoading}
                    />
                </div>
              </>
            )}
          </div>
      </div>

      <TransactionForm
        isOpen={isTransactionModalOpen}
        onClose={() => {
          setIsTransactionModalOpen(false);
          setEditingTransaction(null);
        }}
        onSave={handleCreateTransaction}
        onUpdate={handleUpdateTransaction}
        accounts={bankAccounts}
        categories={categories}
        defaultProfileId={selectedProfileId ?? ''}
        loading={savingTransaction}
        profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
        editingTransaction={editingTransaction}
      />
    </AppLayout>
  );
}
