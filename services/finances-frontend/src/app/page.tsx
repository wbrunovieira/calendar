'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import BankAccountModal from '@/components/finances/BankAccountModal';
import TransactionForm from '@/components/finances/TransactionForm';
import TransactionsTable from '@/components/finances/TransactionsTable';
import CashflowSummary from '@/components/finances/CashflowSummary';
import QuickExpense from '@/components/finances/QuickExpense';
import SafeToSpend from '@/components/finances/SafeToSpend';
import CreditCardInfo from '@/components/finances/CreditCardInfo';
import InvestmentAccountInfo from '@/components/finances/InvestmentAccountInfo';
import GlobalSearch from '@/components/finances/GlobalSearch';
import TodayAlerts from '@/components/finances/TodayAlerts';
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
import { API_BASE } from '@/lib/api';

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

  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [_budgetSummary, setBudgetSummary] = useState<BudgetSummaryItem[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [transactionsLoading, setTransactionsLoading] = useState(false);
  const [transactionsError, setTransactionsError] = useState<'NOT_IMPLEMENTED' | 'GENERIC' | null>(null);
  const [transactionFilters, setTransactionFilters] = useState<TransactionFilters>({ ...defaultFilters });

  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);
  const [isTransactionModalOpen, setIsTransactionModalOpen] = useState(false);

  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null);

  const [savingTransaction, setSavingTransaction] = useState(false);

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
      const response = await fetch(`${API_BASE}/bank-accounts`);
      const data = await response.json();
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
          const response = await fetch(`${API_BASE}/invoices?bankAccountId=${card.id}`);
          if (response.ok) {
            const data = await response.json();
            invoicesMap[card.id] = data.data || [];
          }

          // Fetch current invoice
          const currentResponse = await fetch(`${API_BASE}/invoices/current?bankAccountId=${card.id}`);
          if (currentResponse.ok) {
            const currentData = await currentResponse.json();
            if (currentData.data) {
              currentMap[card.id] = currentData.data;
            }
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
      const response = await fetch(`${API_BASE}/categories?profileId=${profileId}`);
      if (response.status === 404) {
        setCategories([]);
        return;
      }
      if (!response.ok) {
        console.warn('Erro ao carregar categorias:', response.status);
        setCategories([]);
        return;
      }
      const data = await response.json();
      setCategories(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar categorias:', error);
      setCategories([]);
    }
  }, []);

  const fetchBudgetSummary = useCallback(async (profileId: string) => {
    try {
      const month = new Date().toISOString().slice(0, 7);
      const response = await fetch(`${API_BASE}/budgets/summary?profileId=${profileId}&period=${month}`);
      if (!response.ok) {
        setBudgetSummary([]);
        return;
      }
      const data = await response.json();
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

      const response = await fetch(`${API_BASE}/transactions?${params.toString()}`);
      if (!response.ok) {
        if (response.status === 501) {
          setTransactionsError('NOT_IMPLEMENTED');
        } else {
          setTransactionsError('GENERIC');
        }
        setTransactions([]);
        return;
      }
      const data = await response.json();
      setTransactions(data.data || []);
    } catch (error) {
      console.warn('Erro ao carregar lançamentos:', error);
      setTransactionsError('GENERIC');
      setTransactions([]);
    } finally {
      setTransactionsLoading(false);
    }
  }, []);

  const fetchRecurringTransactions = useCallback(async (profileId: string) => {
    try {
      const response = await fetch(`${API_BASE}/recurring-transactions?profileId=${profileId}`);
      if (!response.ok) {
        setRecurringTransactions([]);
        return;
      }
      const data = await response.json();
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

  const handleCreateBankAccount = async (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'> & { initialInvoiceAmount?: number },
  ) => {
    try {
      // Extract initialInvoiceAmount before sending to API (it's not a BankAccount field)
      const { initialInvoiceAmount, ...bankAccountData } = accountData;

      const response = await fetch(`${API_BASE}/bank-accounts`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...bankAccountData,
          isActive: true, // New accounts are active by default
        }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao criar conta bancária');
      }

      const createdAccount = await response.json();

      // If it's a credit card with an initial invoice amount, create the invoice
      if (
        accountData.type === 'CREDIT_CARD' &&
        initialInvoiceAmount &&
        initialInvoiceAmount > 0 &&
        createdAccount.data?.id
      ) {
        const currentDate = new Date();
        const referenceDate = `${currentDate.getFullYear()}-${String(currentDate.getMonth() + 1).padStart(2, '0')}`;

        try {
          // Create the invoice
          const invoiceResponse = await fetch(`${API_BASE}/invoices`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              bankAccountId: createdAccount.data.id,
              referenceDate,
            }),
          });

          if (invoiceResponse.ok) {
            const invoiceData = await invoiceResponse.json();
            // Add the initial amount to the invoice
            if (invoiceData.data?.id) {
              await fetch(`${API_BASE}/invoices/${invoiceData.data.id}/add-amount`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ amount: initialInvoiceAmount }),
              });
            }
          }
        } catch (invoiceError) {
          console.warn('Erro ao criar fatura inicial:', invoiceError);
          // Don't fail the whole operation if invoice creation fails
        }
      }

      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao criar conta bancária:', error);
      alert('Erro ao criar conta bancária');
    }
  };

  const handleUpdateBankAccount = async (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (!editingBankAccount) return;

    try {
      const response = await fetch(`${API_BASE}/bank-accounts/${editingBankAccount.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...accountData,
          isActive: editingBankAccount.isActive, // Preserve isActive status
        }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao atualizar conta bancária');
      }

      await fetchBankAccounts();
      setEditingBankAccount(null);
    } catch (error) {
      console.error('Erro ao atualizar conta bancária:', error);
      alert('Erro ao atualizar conta bancária');
    }
  };

  const handleSaveBankAccount = (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (editingBankAccount) {
      handleUpdateBankAccount(accountData);
    } else {
      handleCreateBankAccount(accountData);
    }
  };

  const handleCreateTransaction = async (payload: TransactionFormData) => {
    if (!selectedProfileId) {
      alert('Selecione um perfil financeiro antes de criar lançamentos.');
      return;
    }

    try {
      setSavingTransaction(true);
      // Remove undefined/null values to avoid API validation issues
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      const response = await fetch(`${API_BASE}/transactions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(cleanPayload),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao criar lançamento');
      }

      await fetchTransactions(selectedProfileId, transactionFilters);
      setEditingTransaction(null);
    } catch (error) {
      console.error('Erro ao criar lançamento:', error);
      alert('Erro ao salvar lançamento');
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
      const response = await fetch(`${API_BASE}/transactions/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(cleanPayload),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao atualizar lançamento');
      }

      await fetchTransactions(selectedProfileId, transactionFilters);
      setEditingTransaction(null);
    } catch (error) {
      console.error('Erro ao atualizar lançamento:', error);
      alert('Erro ao atualizar lançamento');
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
      const response = await fetch(`${API_BASE}/transactions/${id}/status`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          status,
          occurredOn: transaction.occurredOn,
          reason: status === 'CANCELLED' ? 'Cancelado no painel de finanças' : undefined,
        }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao atualizar status');
      }

      if (selectedProfileId) {
        await fetchTransactions(selectedProfileId, transactionFilters);
      }
    } catch (error) {
      console.error('Erro ao atualizar status do lançamento:', error);
      alert('Erro ao atualizar status do lançamento');
    }
  };

  const handleDeleteTransaction = async (id: string) => {
    if (!confirm('Deseja realmente excluir este lançamento?')) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/transactions/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao excluir lançamento');
      }

      if (selectedProfileId) {
        await fetchTransactions(selectedProfileId, transactionFilters);
      }
    } catch (error) {
      console.error('Erro ao excluir lançamento:', error);
      alert('Erro ao excluir lançamento');
    }
  };

  const handlePayInvoice = async (invoiceId: string, amount: number) => {
    try {
      const today = new Date().toISOString().split('T')[0]; // Format: YYYY-MM-DD
      const response = await fetch(`${API_BASE}/invoices/${invoiceId}/pay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ paidAmount: amount, paidAt: today }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao pagar fatura');
      }

      // Refresh invoices after payment
      await fetchInvoicesForCreditCards(filteredAccounts);
      // Also refresh bank accounts to update balances
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao pagar fatura:', error);
      alert('Erro ao pagar fatura');
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
      alert('Crie ou selecione um perfil financeiro antes de registrar lançamentos.');
      return;
    }
    setIsTransactionModalOpen(true);
  };


  // Handle search result selection
  const handleSearchSelectTransaction = useCallback((id: string) => {
    const tx = transactions.find((t) => t.id === id);
    if (tx) {
      setEditingTransaction(tx);
      setIsTransactionModalOpen(true);
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

      const response = await fetch(`${API_BASE}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(transactionData),
      });

      if (!response.ok) {
        throw new Error('Erro ao criar transacao');
      }

      // Refresh transactions to update the alerts
      await fetchTransactions(selectedProfileId, transactionFilters);
    } catch (error) {
      console.error('Erro ao confirmar recorrente:', error);
      alert('Erro ao confirmar pagamento');
    }
  }, [selectedProfileId, filteredAccounts, transactionFilters, fetchTransactions]);

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
                <SafeToSpend accounts={filteredAccounts} transactions={transactions} />
                <CashflowSummary transactions={transactions} accounts={filteredAccounts} currentInvoices={currentInvoices} />

                <div className="grid gap-6 lg:grid-cols-3">
                  <div className="lg:col-span-2 space-y-6">
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

                  <div className="space-y-4">
                    {/* Contas bancárias (exceto investimentos) */}
                    <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-sm">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-white">Contas do perfil</h3>
                        <span className="text-white/50 text-sm">
                          {filteredAccounts.filter(a => a.type !== 'INVESTMENT').length} {filteredAccounts.filter(a => a.type !== 'INVESTMENT').length === 1 ? 'conta' : 'contas'}
                        </span>
                      </div>
                      <div className="space-y-3">
                        {filteredAccounts.filter(a => a.type !== 'INVESTMENT').length === 0 && (
                          <p className="text-white/60 text-sm">
                            Nenhuma conta cadastrada para este perfil.
                          </p>
                        )}
                        {filteredAccounts
                          .filter(account => account.type !== 'INVESTMENT')
                          .map((account) =>
                            account.type === 'CREDIT_CARD' ? (
                              <CreditCardInfo
                                key={account.id}
                                account={account}
                                currentInvoice={currentInvoices[account.id]}
                                invoices={invoicesByAccount[account.id] || []}
                                onPayInvoice={handlePayInvoice}
                                onEdit={() => {
                                  setEditingBankAccount(account);
                                  setIsBankAccountModalOpen(true);
                                }}
                              />
                            ) : (
                              <div
                                key={account.id}
                                className="border border-white/10 rounded-xl p-4 flex items-center justify-between bg-white/5 cursor-pointer hover:bg-white/10 transition-colors"
                                onClick={() => {
                                  setEditingBankAccount(account);
                                  setIsBankAccountModalOpen(true);
                                }}
                              >
                                <div className="flex items-center gap-3">
                                  <span className="text-2xl">{account.icon || '🏦'}</span>
                                  <div>
                                    <p className="text-white font-semibold text-sm">{account.name}</p>
                                    <p className="text-white/50 text-xs">{account.bankName || account.type}</p>
                                  </div>
                                </div>
                                <div className="text-right">
                                  <p className="text-white/80 text-sm font-semibold">
                                    {new Intl.NumberFormat('pt-BR', {
                                      style: 'currency',
                                      currency: account.currency,
                                    }).format(account.currentBalance)}
                                  </p>
                                  <p className="text-white/50 text-xs">Saldo atual</p>
                                </div>
                              </div>
                            ),
                          )}
                      </div>
                    </div>

                    {/* Investimentos - card separado */}
                    {filteredAccounts.filter(a => a.type === 'INVESTMENT').length > 0 && (
                      <div className="bg-gradient-to-br from-purple-900/30 to-indigo-900/30 border border-purple-500/20 rounded-2xl p-6 backdrop-blur-sm">
                        <div className="flex items-center justify-between mb-4">
                          <div className="flex items-center gap-2">
                            <span className="text-xl">📈</span>
                            <h3 className="text-lg font-semibold text-white">Investimentos</h3>
                          </div>
                          <div className="text-right">
                            <p className="text-purple-300 text-sm font-semibold">
                              {new Intl.NumberFormat('pt-BR', {
                                style: 'currency',
                                currency: 'BRL',
                              }).format(
                                filteredAccounts
                                  .filter(a => a.type === 'INVESTMENT')
                                  .reduce((sum, a) => sum + a.currentBalance, 0)
                              )}
                            </p>
                            <span className="text-white/50 text-xs">
                              {filteredAccounts.filter(a => a.type === 'INVESTMENT').length} {filteredAccounts.filter(a => a.type === 'INVESTMENT').length === 1 ? 'investimento' : 'investimentos'}
                            </span>
                          </div>
                        </div>
                        <div className="space-y-3">
                          {filteredAccounts
                            .filter(account => account.type === 'INVESTMENT')
                            .map((account) => (
                              <InvestmentAccountInfo
                                key={account.id}
                                account={account}
                                linkedAccount={account.linkedAccountId
                                  ? filteredAccounts.find((a) => a.id === account.linkedAccountId)
                                  : undefined
                                }
                                onEdit={() => {
                                  setEditingBankAccount(account);
                                  setIsBankAccountModalOpen(true);
                                }}
                              />
                            ))}
                        </div>
                      </div>
                    )}
                    <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-sm">
                      <h3 className="text-lg font-semibold text-white mb-3">Categorias disponíveis</h3>
                      {categories.length === 0 ? (
                        <p className="text-white/60 text-sm">
                          Nenhuma categoria cadastrada para o perfil selecionado.
                        </p>
                      ) : (
                        <div className="flex flex-wrap gap-2 text-xs">
                          {categories.map((category) => (
                            <span
                              key={category.id}
                              className="px-3 py-1 rounded-full border border-white/15 text-white/70"
                            >
                              {category.name}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
      </div>

      <BankAccountModal
        isOpen={isBankAccountModalOpen}
        onClose={() => {
          setIsBankAccountModalOpen(false);
          setEditingBankAccount(null);
        }}
        onSave={handleSaveBankAccount}
        account={editingBankAccount}
        profiles={profiles.map((profile) => ({ id: profile.id, name: profile.name }))}
        existingAccounts={bankAccounts}
      />

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
