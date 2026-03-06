'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import BankAccountModal from '@/components/finances/BankAccountModal';
import CreditCardInfo from '@/components/finances/CreditCardInfo';
import InvestmentAccountInfo from '@/components/finances/InvestmentAccountInfo';
import TransactionForm from '@/components/finances/TransactionForm';
import type { BankAccount, Invoice, Transaction, TransactionFormData, Category } from '@/types/finances';
import { API_BASE } from '@/lib/api';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

const formatDate = (dateStr: string) => {
  const raw = dateStr.includes('T') ? dateStr : dateStr + 'T12:00:00';
  const date = new Date(raw);
  return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: '2-digit' });
};

const statusConfig: Record<string, { label: string; color: string }> = {
  PLANNED: { label: 'Planejado', color: 'bg-blue-500/20 text-blue-400' },
  CONFIRMED: { label: 'Confirmado', color: 'bg-emerald-500/20 text-emerald-400' },
  CANCELLED: { label: 'Cancelado', color: 'bg-red-500/20 text-red-400' },
};

function TransactionHistory({
  accountId,
  profileId,
  categories,
  selectedInvoiceId,
  currentBalance = 0,
}: {
  accountId: string;
  profileId: string;
  categories: Category[];
  selectedInvoiceId?: string;
  currentBalance?: number;
}) {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);

  const categoryMap = useMemo(() => {
    const map: Record<string, string> = {};
    categories.forEach((c) => { map[c.id] = c.name; });
    return map;
  }, [categories]);

  useEffect(() => {
    const fetchTransactions = async () => {
      try {
        setLoading(true);
        const params = new URLSearchParams({ profileId, bankAccountId: accountId });
        if (selectedInvoiceId) {
          params.set('invoiceId', selectedInvoiceId);
        }
        const response = await fetch(`${API_BASE}/transactions?${params}`);
        if (response.ok) {
          const data = await response.json();
          setTransactions(data.data || []);
        }
      } catch (error) {
        console.warn('Erro ao carregar historico:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchTransactions();
  }, [accountId, profileId, selectedInvoiceId]);

  const groupedByDay = useMemo(() => {
    const groups: Record<string, Transaction[]> = {};
    for (const tx of transactions) {
      const dateKey = tx.occurredOn.split('T')[0];
      if (!groups[dateKey]) groups[dateKey] = [];
      groups[dateKey].push(tx);
    }
    return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b));
  }, [transactions]);

  // Running balance: derive backwards from currentBalance (only confirmed count)
  const dayBalances = useMemo(() => {
    // Only confirmed transactions affect currentBalance
    let confirmedTotal = 0;
    for (const [, dayTxs] of groupedByDay) {
      for (const tx of dayTxs) {
        if (tx.status === 'PLANNED' || tx.status === 'CANCELLED') continue;
        if (tx.type === 'INCOME') confirmedTotal += tx.amount;
        else confirmedTotal -= tx.amount;
      }
    }
    const startBalance = currentBalance - confirmedTotal;

    const balances: Record<string, number> = {};
    let running = startBalance;
    for (const [dateKey, dayTxs] of groupedByDay) {
      for (const tx of dayTxs) {
        if (tx.type === 'INCOME') running += tx.amount;
        else running -= tx.amount;
      }
      balances[dateKey] = running;
    }
    return balances;
  }, [groupedByDay, currentBalance]);

  // Display newest first
  const groupedByDayDesc = useMemo(
    () => [...groupedByDay].reverse(),
    [groupedByDay],
  );

  if (loading) {
    return (
      <div className="py-4 text-center text-white/50 text-sm">
        Carregando historico...
      </div>
    );
  }

  if (transactions.length === 0) {
    return (
      <div className="py-4 text-center text-white/40 text-sm">
        Nenhuma transacao encontrada para esta conta.
      </div>
    );
  }

  return (
    <div className="space-y-3 max-h-[28rem] overflow-y-auto scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
      {groupedByDayDesc.map(([dateKey, dayTransactions]) => {
        const dayTotal = dayTransactions.reduce((sum, tx) => {
          if (tx.type === 'INCOME') return sum + tx.amount;
          return sum - tx.amount; // EXPENSE and TRANSFER
        }, 0);
        const endOfDayBalance = dayBalances[dateKey] ?? 0;

        return (
          <div key={dateKey}>
            <div className="flex items-center justify-between px-3 py-1.5 bg-white/10 rounded-t-lg border-b border-white/10">
              <span className="text-white/70 text-xs font-semibold">{formatDate(dateKey)}</span>
              <div className="flex items-center gap-3">
                <span className={`text-xs font-semibold ${dayTotal >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
                  {dayTotal >= 0 ? '+' : ''}{formatCurrency(Math.abs(dayTotal))}
                </span>
                <span className={`text-xs font-medium ${endOfDayBalance >= 0 ? 'text-white/60' : 'text-red-300'}`}>
                  Saldo: {formatCurrency(endOfDayBalance)}
                </span>
              </div>
            </div>
            <div className="space-y-1 mt-1">
              {dayTransactions.map((tx) => {
                const status = statusConfig[tx.status] || statusConfig.PLANNED;
                const isExpense = tx.type === 'EXPENSE';
                const isIncome = tx.type === 'INCOME';
                return (
                  <div
                    key={tx.id}
                    className="flex items-center justify-between py-2 px-3 bg-white/5 rounded-lg"
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-white text-sm truncate">{tx.description}</p>
                      <div className="flex items-center gap-2 text-white/40 text-xs">
                        {tx.categoryId && categoryMap[tx.categoryId] && (
                          <span className="truncate">{categoryMap[tx.categoryId]}</span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 ml-3 shrink-0">
                      <div className="text-right">
                        <p className={`text-sm font-semibold ${isExpense ? 'text-red-400' : isIncome ? 'text-emerald-400' : 'text-blue-400'}`}>
                          {isExpense ? '-' : isIncome ? '+' : ''}{formatCurrency(tx.amount, tx.currency)}
                        </p>
                      </div>
                      <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${status.color}`}>
                        {status.label}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

type DndListeners = ReturnType<typeof useSortable>['listeners'];
type DndAttributes = ReturnType<typeof useSortable>['attributes'];

function DragHandle({ listeners, attributes }: { listeners?: DndListeners; attributes?: DndAttributes }) {
  return (
    <button
      className="touch-none p-1 cursor-grab active:cursor-grabbing text-white/30 hover:text-white/60 transition-colors"
      {...attributes}
      {...listeners}
    >
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
        <circle cx="9" cy="6" r="1.5" />
        <circle cx="15" cy="6" r="1.5" />
        <circle cx="9" cy="12" r="1.5" />
        <circle cx="15" cy="12" r="1.5" />
        <circle cx="9" cy="18" r="1.5" />
        <circle cx="15" cy="18" r="1.5" />
      </svg>
    </button>
  );
}

function SortableItem({ id, children }: { id: string; children: (props: { listeners: DndListeners; attributes: DndAttributes }) => React.ReactNode }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 50 : undefined,
    opacity: isDragging ? 0.8 : 1,
  };

  return (
    <div ref={setNodeRef} style={style}>
      {children({ listeners, attributes })}
    </div>
  );
}

export default function ContasPage() {
  const { profiles, selectedProfileId, selectedProfile, isLoading: profilesLoading } = useProfile();

  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);
  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);
  const [expandedAccountId, setExpandedAccountId] = useState<string | null>(null);
  const [selectedInvoiceByAccount, setSelectedInvoiceByAccount] = useState<Record<string, string>>({});
  const [isTransactionFormOpen, setIsTransactionFormOpen] = useState(false);
  const [preselectedAccountId, setPreselectedAccountId] = useState<string>('');

  const [invoicesByAccount, setInvoicesByAccount] = useState<Record<string, Invoice[]>>({});
  const [currentInvoices, setCurrentInvoices] = useState<Record<string, Invoice>>({});

  const filteredAccounts = useMemo(
    () => bankAccounts
      .filter((account) => account.profileId === selectedProfileId)
      .sort((a, b) => (a.displayOrder ?? 0) - (b.displayOrder ?? 0)),
    [bankAccounts, selectedProfileId],
  );

  const regularAccounts = useMemo(
    () => filteredAccounts.filter((a) => a.type !== 'INVESTMENT'),
    [filteredAccounts],
  );

  const investmentAccounts = useMemo(
    () => filteredAccounts.filter((a) => a.type === 'INVESTMENT'),
    [filteredAccounts],
  );

  const totalBalance = useMemo(
    () => regularAccounts.filter((a) => a.type !== 'CREDIT_CARD').reduce((sum, a) => sum + a.currentBalance, 0),
    [regularAccounts],
  );

  const totalInvested = useMemo(
    () => investmentAccounts.reduce((sum, a) => sum + a.currentBalance, 0),
    [investmentAccounts],
  );

  const toggleExpand = (accountId: string) => {
    setExpandedAccountId((prev) => (prev === accountId ? null : accountId));
  };

  useEffect(() => {
    fetchBankAccounts();
  }, []);

  useEffect(() => {
    if (selectedProfileId) {
      fetchCategories(selectedProfileId);
    }
  }, [selectedProfileId]);

  const fetchBankAccounts = async () => {
    try {
      const response = await fetch(`${API_BASE}/bank-accounts`);
      const data = await response.json();
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar contas bancarias:', error);
    }
  };

  const fetchCategories = async (profileId: string) => {
    try {
      const response = await fetch(`${API_BASE}/categories?profileId=${profileId}`);
      if (response.ok) {
        const data = await response.json();
        setCategories(data.data || []);
      }
    } catch (error) {
      console.warn('Erro ao carregar categorias:', error);
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
          const response = await fetch(`${API_BASE}/invoices?bankAccountId=${card.id}`);
          if (response.ok) {
            const data = await response.json();
            invoicesMap[card.id] = data.data || [];
          }

          const currentResponse = await fetch(`${API_BASE}/invoices/current?bankAccountId=${card.id}`);
          if (currentResponse.ok) {
            const currentData = await currentResponse.json();
            if (currentData.data) {
              currentMap[card.id] = currentData.data;
            }
          }
        } catch (error) {
          console.warn(`Erro ao carregar faturas do cartao ${card.name}:`, error);
        }
      }),
    );

    setInvoicesByAccount(invoicesMap);
    setCurrentInvoices(currentMap);
  }, []);

  useEffect(() => {
    if (filteredAccounts.length > 0) {
      fetchInvoicesForCreditCards(filteredAccounts);
    }
  }, [filteredAccounts, fetchInvoicesForCreditCards]);

  const handleCreateBankAccount = async (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'> & { initialInvoiceAmount?: number },
  ) => {
    try {
      const { initialInvoiceAmount, ...bankAccountData } = accountData;

      const response = await fetch(`${API_BASE}/bank-accounts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...bankAccountData, isActive: true }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao criar conta bancaria');
      }

      const createdAccount = await response.json();

      if (
        accountData.type === 'CREDIT_CARD' &&
        initialInvoiceAmount &&
        initialInvoiceAmount > 0 &&
        createdAccount.data?.id
      ) {
        const currentDate = new Date();
        const referenceDate = `${currentDate.getFullYear()}-${String(currentDate.getMonth() + 1).padStart(2, '0')}`;

        try {
          const invoiceResponse = await fetch(`${API_BASE}/invoices`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ bankAccountId: createdAccount.data.id, referenceDate }),
          });

          if (invoiceResponse.ok) {
            const invoiceData = await invoiceResponse.json();
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
        }
      }

      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao criar conta bancaria:', error);
      alert('Erro ao criar conta bancaria');
    }
  };

  const handleUpdateBankAccount = async (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (!editingBankAccount) return;

    try {
      const response = await fetch(`${API_BASE}/bank-accounts/${editingBankAccount.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...accountData, isActive: editingBankAccount.isActive }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao atualizar conta bancaria');
      }

      await fetchBankAccounts();
      setEditingBankAccount(null);
    } catch (error) {
      console.error('Erro ao atualizar conta bancaria:', error);
      alert('Erro ao atualizar conta bancaria');
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

  const handlePayInvoice = async (invoiceId: string, amount: number) => {
    try {
      const today = new Date().toISOString().split('T')[0];
      const response = await fetch(`${API_BASE}/invoices/${invoiceId}/pay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ paidAmount: amount, paidAt: today }),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao pagar fatura');
      }

      await fetchInvoicesForCreditCards(filteredAccounts);
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao pagar fatura:', error);
      alert('Erro ao pagar fatura');
    }
  };

  const handleInvoiceSelect = (accountId: string, invoiceId: string) => {
    setSelectedInvoiceByAccount((prev) => ({
      ...prev,
      [accountId]: invoiceId,
    }));
  };

  const handleAddTransaction = (accountId: string) => {
    setPreselectedAccountId(accountId);
    setIsTransactionFormOpen(true);
  };

  const handleSaveTransaction = async (payload: TransactionFormData) => {
    try {
      const response = await fetch(`${API_BASE}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao criar transacao');
      }
      setIsTransactionFormOpen(false);
      await fetchBankAccounts();
      // Re-expand to refresh transaction history
      const currentExpanded = expandedAccountId;
      setExpandedAccountId(null);
      setTimeout(() => setExpandedAccountId(currentExpanded), 50);
    } catch (error) {
      console.error('Erro ao criar transacao:', error);
      alert('Erro ao criar transacao');
    }
  };

  const openEditModal = (account: BankAccount) => {
    setEditingBankAccount(account);
    setIsBankAccountModalOpen(true);
  };

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleDragEnd = async (event: DragEndEvent, accountList: BankAccount[]) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = accountList.findIndex((a) => a.id === active.id);
    const newIndex = accountList.findIndex((a) => a.id === over.id);
    if (oldIndex === -1 || newIndex === -1) return;

    const reordered = arrayMove(accountList, oldIndex, newIndex);

    // Update local state immediately
    setBankAccounts((prev) => {
      const updated = [...prev];
      reordered.forEach((account, index) => {
        const i = updated.findIndex((a) => a.id === account.id);
        if (i !== -1) {
          updated[i] = { ...updated[i], displayOrder: index + 1 };
        }
      });
      return updated;
    });

    // Persist to backend
    try {
      await fetch(`${API_BASE}/bank-accounts/reorder`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          items: reordered.map((account, index) => ({
            id: account.id,
            displayOrder: index + 1,
          })),
        }),
      });
    } catch (error) {
      console.error('Erro ao reordenar contas:', error);
      await fetchBankAccounts();
    }
  };

  const PencilButton = ({ account }: { account: BankAccount }) => (
    <button
      onClick={(e) => { e.stopPropagation(); openEditModal(account); }}
      className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-white/40 hover:text-white/80"
      title="Editar conta"
    >
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
      </svg>
    </button>
  );

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">Contas</h2>
            <p className="text-white/60 text-sm">Gerencie suas contas bancarias, cartoes e investimentos</p>
          </div>
          <button
            onClick={() => {
              setEditingBankAccount(null);
              setIsBankAccountModalOpen(true);
            }}
            className={`flex items-center gap-2 px-5 py-2 rounded-xl text-white font-semibold transition-colors ${
              selectedProfile?.type === 'BUSINESS'
                ? 'bg-amber-500/80 hover:bg-amber-500 border border-amber-400/40'
                : 'bg-emerald-500/80 hover:bg-emerald-500 border border-emerald-400/40'
            }`}
          >
            <span>+</span>
            <span>Nova conta</span>
          </button>
        </div>

        {profilesLoading ? (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center text-white/70">
            Carregando perfis...
          </div>
        ) : profiles.length === 0 ? (
          <div className="bg-white/10 border border-white/10 rounded-2xl p-8 text-center text-white/70">
            Cadastre um perfil financeiro para comecar.
          </div>
        ) : selectedProfile && (
          <>
            {/* Summary cards */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
                <p className="text-white/50 text-sm mb-1">Saldo em contas</p>
                <p className="text-2xl font-bold text-white">{formatCurrency(totalBalance)}</p>
                <div className="mt-2 space-y-1">
                  {regularAccounts.filter((a) => a.type !== 'CREDIT_CARD').map((a) => (
                    <div key={a.id} className="flex items-center justify-between text-xs">
                      <span className="text-white/40 truncate mr-2">{a.name}</span>
                      <span className="text-white/60 font-medium shrink-0">{formatCurrency(a.currentBalance, a.currency)}</span>
                    </div>
                  ))}
                </div>
              </div>
              {investmentAccounts.length > 0 && (
                <div className="bg-gradient-to-br from-purple-900/30 to-indigo-900/30 border border-purple-500/20 rounded-2xl p-5 backdrop-blur-sm">
                  <p className="text-purple-300/70 text-sm mb-1">Total investido</p>
                  <p className="text-2xl font-bold text-white">{formatCurrency(totalInvested)}</p>
                  <p className="text-white/40 text-xs mt-1">
                    {investmentAccounts.length} {investmentAccounts.length === 1 ? 'investimento' : 'investimentos'}
                  </p>
                </div>
              )}
              {regularAccounts.filter((a) => a.type === 'CREDIT_CARD').length > 0 && (
                <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
                  <p className="text-white/50 text-sm mb-1">Cartoes de credito</p>
                  <p className="text-2xl font-bold text-white">
                    {regularAccounts.filter((a) => a.type === 'CREDIT_CARD').length}
                  </p>
                  <p className="text-white/40 text-xs mt-1">
                    Limite total: {formatCurrency(
                      regularAccounts
                        .filter((a) => a.type === 'CREDIT_CARD')
                        .reduce((sum, a) => sum + (a.creditLimit || 0), 0)
                    )}
                  </p>
                </div>
              )}
            </div>

            {/* Regular accounts */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-white">Contas do perfil</h3>
                <span className="text-white/50 text-sm">
                  {regularAccounts.length} {regularAccounts.length === 1 ? 'conta' : 'contas'}
                </span>
              </div>
              {regularAccounts.length === 0 && (
                <p className="text-white/60 text-sm">
                  Nenhuma conta cadastrada para este perfil.
                </p>
              )}
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={(event) => handleDragEnd(event, regularAccounts)}
              >
                <SortableContext items={regularAccounts.map((a) => a.id)} strategy={verticalListSortingStrategy}>
                  {regularAccounts.map((account) => {
                    const isExpanded = expandedAccountId === account.id;

                    if (account.type === 'CREDIT_CARD') {
                      return (
                        <SortableItem key={account.id} id={account.id}>
                          {({ listeners, attributes }) => (
                            <div className="space-y-0">
                              <div className="flex items-center gap-1">
                                <DragHandle listeners={listeners} attributes={attributes} />
                                <div
                                  className="cursor-pointer flex-1 rounded-xl transition-all duration-300"
                                  style={isExpanded ? {
                                    boxShadow: '0 0 20px rgba(52, 211, 153, 0.3), 0 0 0 2px #34d399',
                                  } : undefined}
                                  onClick={() => toggleExpand(account.id)}
                                >
                                  <CreditCardInfo
                                    account={account}
                                    currentInvoice={currentInvoices[account.id]}
                                    invoices={invoicesByAccount[account.id] || []}
                                    onPayInvoice={handlePayInvoice}
                                    onEdit={() => openEditModal(account)}
                                    selectedInvoiceId={selectedInvoiceByAccount[account.id] || ''}
                                    onInvoiceSelect={(invoiceId) => handleInvoiceSelect(account.id, invoiceId)}
                                  />
                                </div>
                              </div>
                              {isExpanded && selectedProfileId && (
                                <div className="bg-white/[0.03] border border-white/10 border-t-0 rounded-b-xl p-4 ml-7">
                                  <div className="flex items-center justify-between mb-3">
                                    <p className="text-white/60 text-xs font-semibold uppercase tracking-wider">Historico de transacoes</p>
                                    <button
                                      onClick={(e) => { e.stopPropagation(); handleAddTransaction(account.id); }}
                                      className="flex items-center gap-1 px-2.5 py-1 bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/30 rounded-lg text-emerald-400 text-xs font-medium transition-colors"
                                    >
                                      <span>+</span>
                                      <span>Lancamento</span>
                                    </button>
                                  </div>
                                  <TransactionHistory
                                    accountId={account.id}
                                    profileId={selectedProfileId}
                                    categories={categories}
                                    selectedInvoiceId={selectedInvoiceByAccount[account.id] || ''}
                                    currentBalance={account.currentBalance}
                                  />
                                </div>
                              )}
                            </div>
                          )}
                        </SortableItem>
                      );
                    }

                    return (
                      <SortableItem key={account.id} id={account.id}>
                        {({ listeners, attributes }) => (
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
                              onClick={() => toggleExpand(account.id)}
                            >
                              <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                  <DragHandle listeners={listeners} attributes={attributes} />
                                  <span className="text-2xl">{account.icon || '🏦'}</span>
                                  <div>
                                    <p className="text-white font-semibold text-sm">{account.name}</p>
                                    <p className="text-white/50 text-xs">{account.bankName || account.type}</p>
                                  </div>
                                </div>
                                <div className="flex items-center gap-3">
                                  <div className="text-right">
                                    <p className="text-white/80 text-sm font-semibold">
                                      {formatCurrency(account.currentBalance, account.currency)}
                                    </p>
                                    <p className="text-white/50 text-xs">Saldo atual</p>
                                  </div>
                                  <PencilButton account={account} />
                                </div>
                              </div>
                            </div>
                            {isExpanded && selectedProfileId && (
                              <div className="bg-white/[0.03] border border-white/10 border-t-0 rounded-b-xl p-4">
                                <div className="flex items-center justify-between mb-3">
                                  <p className="text-white/60 text-xs font-semibold uppercase tracking-wider">Historico de transacoes</p>
                                  <button
                                    onClick={(e) => { e.stopPropagation(); handleAddTransaction(account.id); }}
                                    className="flex items-center gap-1 px-2.5 py-1 bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/30 rounded-lg text-emerald-400 text-xs font-medium transition-colors"
                                  >
                                    <span>+</span>
                                    <span>Lancamento</span>
                                  </button>
                                </div>
                                <TransactionHistory
                                  accountId={account.id}
                                  profileId={selectedProfileId}
                                  categories={categories}
                                  currentBalance={account.currentBalance}
                                />
                              </div>
                            )}
                          </div>
                        )}
                      </SortableItem>
                    );
                  })}
                </SortableContext>
              </DndContext>
            </div>

            {/* Investments */}
            {investmentAccounts.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-xl">📈</span>
                    <h3 className="text-lg font-semibold text-white">Investimentos</h3>
                  </div>
                  <div className="text-right">
                    <p className="text-purple-300 text-sm font-semibold">
                      {formatCurrency(totalInvested)}
                    </p>
                    <span className="text-white/50 text-xs">
                      {investmentAccounts.length} {investmentAccounts.length === 1 ? 'investimento' : 'investimentos'}
                    </span>
                  </div>
                </div>
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={(event) => handleDragEnd(event, investmentAccounts)}
                >
                  <SortableContext items={investmentAccounts.map((a) => a.id)} strategy={verticalListSortingStrategy}>
                    {investmentAccounts.map((account) => {
                      const isExpanded = expandedAccountId === account.id;
                      return (
                        <SortableItem key={account.id} id={account.id}>
                          {({ listeners, attributes }) => (
                            <div>
                              <div className="flex items-center gap-1">
                                <DragHandle listeners={listeners} attributes={attributes} />
                                <div
                                  className="cursor-pointer flex-1 rounded-xl transition-all duration-300"
                                  style={isExpanded ? {
                                    boxShadow: '0 0 20px rgba(52, 211, 153, 0.3), 0 0 0 2px #34d399',
                                  } : undefined}
                                  onClick={() => toggleExpand(account.id)}
                                >
                                  <InvestmentAccountInfo
                                    account={account}
                                    linkedAccount={account.linkedAccountId
                                      ? filteredAccounts.find((a) => a.id === account.linkedAccountId)
                                      : undefined
                                    }
                                    onEdit={() => openEditModal(account)}
                                  />
                                </div>
                              </div>
                              {isExpanded && selectedProfileId && (
                                <div className="bg-white/[0.03] border border-white/10 border-t-0 rounded-b-xl p-4 ml-7">
                                  <div className="flex items-center justify-between mb-3">
                                    <p className="text-white/60 text-xs font-semibold uppercase tracking-wider">Historico de transacoes</p>
                                    <button
                                      onClick={(e) => { e.stopPropagation(); handleAddTransaction(account.id); }}
                                      className="flex items-center gap-1 px-2.5 py-1 bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/30 rounded-lg text-emerald-400 text-xs font-medium transition-colors"
                                    >
                                      <span>+</span>
                                      <span>Lancamento</span>
                                    </button>
                                  </div>
                                  <TransactionHistory
                                    accountId={account.id}
                                    profileId={selectedProfileId}
                                    categories={categories}
                                    currentBalance={account.currentBalance}
                                  />
                                </div>
                              )}
                            </div>
                          )}
                        </SortableItem>
                      );
                    })}
                  </SortableContext>
                </DndContext>
              </div>
            )}
          </>
        )}
      </div>

      {selectedProfileId && (
        <TransactionForm
          isOpen={isTransactionFormOpen}
          onClose={() => setIsTransactionFormOpen(false)}
          onSave={handleSaveTransaction}
          accounts={bankAccounts}
          categories={categories}
          defaultProfileId={selectedProfileId}
          defaultBankAccountId={preselectedAccountId}
          profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
        />
      )}

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
    </AppLayout>
  );
}
