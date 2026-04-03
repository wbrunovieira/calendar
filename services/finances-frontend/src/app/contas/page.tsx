'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import BankAccountModal from '@/components/finances/BankAccountModal';
import CreditCardInfo from '@/components/finances/CreditCardInfo';
import InvestmentAccountInfo from '@/components/finances/InvestmentAccountInfo';
import TransactionForm from '@/components/finances/TransactionForm';
import type { BankAccount, Invoice, Transaction, TransactionFormData, Category } from '@/types/finances';
import { api } from '@/lib/api';
import { useToast } from '@/components/ui/Toast';
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
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { formatCurrency } from '@/utils/format';
import FiiDetailPanel from '@/components/finances/FiiDetailPanel';
import TransactionHistory from '@/components/finances/TransactionHistory';
import { DragHandle, SortableItem } from '@/components/finances/SortableHelpers';

interface CryptoPurchaseWithGains {
  id: string;
  asset: string;
  quantity: number;
  priceUsd: number;
  exchangeRate: number;
  investedBrl: number;
  investedUsd: number;
  occurredOn: string;
  currentPriceUsd: number;
  currentExchangeRate: number;
  currentValueUsd: number;
  currentValueBrl: number;
  gainCryptoUsd: number;
  gainCryptoPercent: number;
  gainExchangeBrl: number;
  gainExchangePercent: number;
  gainTotalBrl: number;
  gainTotalPercent: number;
}


export default function ContasPage() {
  const { profiles, selectedProfileId, selectedProfile, isLoading: profilesLoading } = useProfile();
  const { toast, confirm } = useToast();

  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);
  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);
  const [expandedAccountId, setExpandedAccountId] = useState<string | null>(null);
  const [expandedFiiId, setExpandedFiiId] = useState<string | null>(null);
  const [selectedInvoiceByAccount, setSelectedInvoiceByAccount] = useState<Record<string, string>>({});
  const [isTransactionFormOpen, setIsTransactionFormOpen] = useState(false);
  const [preselectedAccountId, setPreselectedAccountId] = useState<string>('');
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null);

  const [invoicesByAccount, setInvoicesByAccount] = useState<Record<string, Invoice[]>>({});
  const [currentInvoices, setCurrentInvoices] = useState<Record<string, Invoice>>({});
  const [cryptoPrices, setCryptoPrices] = useState<{ usdBrl: number; prices: Record<string, { priceUsd: number; priceBrl: number }> } | null>(null);
  const [cryptoPurchases, setCryptoPurchases] = useState<Record<string, CryptoPurchaseWithGains[]>>({});

  const filteredAccounts = useMemo(
    () => bankAccounts
      .filter((account) => account.profileId === selectedProfileId)
      .sort((a, b) => (a.displayOrder ?? 0) - (b.displayOrder ?? 0)),
    [bankAccounts, selectedProfileId],
  );

  const cryptoAccounts = useMemo(
    () => filteredAccounts.filter((a) => a.type === 'EXCHANGE' || a.type === 'WALLET'),
    [filteredAccounts],
  );

  const cryptoAccountIds = useMemo(
    () => new Set(cryptoAccounts.map((a) => a.id)),
    [cryptoAccounts],
  );

  // Sub-assets linked to a parent account (shown inside parent card)
  const getSubAssets = (parentId: string) =>
    filteredAccounts.filter((a) => a.linkedAccountId === parentId);

  // Sub-investments (exclude credit cards)
  const getSubInvestments = (parentId: string) =>
    filteredAccounts.filter((a) => a.linkedAccountId === parentId && a.type !== 'CREDIT_CARD');

  // Credit cards linked to a parent account
  const getSubCreditCards = (parentId: string) =>
    filteredAccounts.filter((a) => a.linkedAccountId === parentId && a.type === 'CREDIT_CARD');

  // IDs of accounts that have linked investments (broker accounts like Clear)
  const brokerAccountIds = useMemo(() => {
    const parentIds = new Set<string>();
    filteredAccounts.forEach((a) => {
      if (a.type === 'INVESTMENT' && a.linkedAccountId) {
        parentIds.add(a.linkedAccountId);
      }
    });
    return parentIds;
  }, [filteredAccounts]);

  const regularAccounts = useMemo(
    () => filteredAccounts.filter((a) =>
      (a.type !== 'INVESTMENT' && a.type !== 'EXCHANGE' && a.type !== 'WALLET')
      || brokerAccountIds.has(a.id) // Broker accounts (e.g., Clear) shown here even if INVESTMENT
    ),
    [filteredAccounts, brokerAccountIds],
  );

  const investmentAccounts = useMemo(
    () => filteredAccounts.filter((a) =>
      a.type === 'INVESTMENT'
      && !brokerAccountIds.has(a.id) // Exclude broker parents (shown in regularAccounts)
      && (!a.linkedAccountId || (!cryptoAccountIds.has(a.linkedAccountId) && !brokerAccountIds.has(a.linkedAccountId)))
    ),
    [filteredAccounts, cryptoAccountIds, brokerAccountIds],
  );

  const totalsByCurrency = useMemo(() => {
    const totals: Record<string, number> = {};
    regularAccounts.filter((a) => a.type !== 'CREDIT_CARD').forEach((a) => {
      const cur = a.currency || 'BRL';
      totals[cur] = (totals[cur] || 0) + a.currentBalance;
    });
    return totals;
  }, [regularAccounts]);

  const totalBalance = useMemo(
    () => regularAccounts.filter((a) => a.type !== 'CREDIT_CARD').reduce((sum, a) => sum + a.currentBalance, 0),
    [regularAccounts],
  );

  const investedByCurrency = useMemo(() => {
    const totals: Record<string, number> = {};
    investmentAccounts.forEach((a) => {
      const cur = a.currency || 'BRL';
      totals[cur] = (totals[cur] || 0) + a.currentBalance;
    });
    return totals;
  }, [investmentAccounts]);

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
      syncCryptoPrices(selectedProfileId);
    }
  }, [selectedProfileId]);

  const fetchBankAccounts = async () => {
    try {
      const data = await api.get<{ data: BankAccount[] }>('/bank-accounts');
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar contas bancarias:', error);
    }
  };

  const syncCryptoPrices = async (profileId: string) => {
    try {
      const data = await api.post<{ data: { usdBrl: number; prices: { symbol: string; priceUsd: number; priceBrl: number }[] } }>(`/crypto/sync?profileId=${profileId}`, {});
      const syncData = data.data;
      const pricesMap: Record<string, { priceUsd: number; priceBrl: number }> = {};
      if (syncData.prices) {
        for (const p of syncData.prices) {
          pricesMap[p.symbol] = { priceUsd: p.priceUsd, priceBrl: p.priceBrl };
        }
      }
      setCryptoPrices({ usdBrl: syncData.usdBrl, prices: pricesMap });
      await fetchBankAccounts();

      // Fetch structured purchases with gains
      const pData = await api.get<{ data: CryptoPurchaseWithGains[] }>(`/crypto/purchases?profileId=${profileId}`);
      const byAsset: Record<string, CryptoPurchaseWithGains[]> = {};
      for (const p of (pData.data || [])) {
        if (!byAsset[p.asset]) byAsset[p.asset] = [];
        byAsset[p.asset].push(p);
      }
      setCryptoPurchases(byAsset);
    } catch (error) {
      console.warn('Erro ao sincronizar precos crypto:', error);
    }
  };

  const fetchCategories = async (profileId: string) => {
    try {
      const data = await api.get<{ data: Category[] }>(`/categories?profileId=${profileId}`);
      setCategories(data.data || []);
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
          const data = await api.get<{ data: Invoice[] }>(`/invoices?bankAccountId=${card.id}`);
          invoicesMap[card.id] = data.data || [];

          const currentData = await api.get<{ data: Invoice }>(`/invoices/current?bankAccountId=${card.id}`);
          if (currentData.data) {
            currentMap[card.id] = currentData.data;
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

      const createdAccount = await api.post<{ data: BankAccount }>('/bank-accounts', { ...bankAccountData, isActive: true });

      if (
        accountData.type === 'CREDIT_CARD' &&
        initialInvoiceAmount &&
        initialInvoiceAmount > 0 &&
        createdAccount.data?.id
      ) {
        const currentDate = new Date();
        const referenceDate = `${currentDate.getFullYear()}-${String(currentDate.getMonth() + 1).padStart(2, '0')}`;

        try {
          const invoiceData = await api.post<{ data: Invoice }>('/invoices', { bankAccountId: createdAccount.data.id, referenceDate });
          if (invoiceData.data?.id) {
            await api.post(`/invoices/${invoiceData.data.id}/add-amount`, { amount: initialInvoiceAmount });
          }
        } catch (invoiceError) {
          console.warn('Erro ao criar fatura inicial:', invoiceError);
        }
      }

      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao criar conta bancaria:', error);
      toast('Erro ao criar conta bancaria', 'error');
    }
  };

  const handleUpdateBankAccount = async (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (!editingBankAccount) return;

    try {
      await api.put(`/bank-accounts/${editingBankAccount.id}`, { ...accountData, isActive: editingBankAccount.isActive });

      await fetchBankAccounts();
      setEditingBankAccount(null);
    } catch (error) {
      console.error('Erro ao atualizar conta bancaria:', error);
      toast('Erro ao atualizar conta bancaria', 'error');
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
      await api.post(`/invoices/${invoiceId}/pay`, { paidAmount: amount, paidAt: today });

      await fetchInvoicesForCreditCards(filteredAccounts);
      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao pagar fatura:', error);
      toast('Erro ao pagar fatura', 'error');
    }
  };

  const handleUpdateInvoice = async (invoiceId: string, data: { closingDate?: string; dueDate?: string }) => {
    try {
      await api.put(`/invoices/${invoiceId}`, data);

      await fetchInvoicesForCreditCards(filteredAccounts);
    } catch (error) {
      console.error('Erro ao atualizar fatura:', error);
      toast('Erro ao atualizar fatura', 'error');
    }
  };

  const handleInvoiceSelect = (accountId: string, invoiceId: string) => {
    setSelectedInvoiceByAccount((prev) => ({
      ...prev,
      [accountId]: invoiceId,
    }));
    if (invoiceId) {
      setExpandedAccountId(accountId);
    }
  };

  const handleAddTransaction = (accountId: string) => {
    setPreselectedAccountId(accountId);
    setIsTransactionFormOpen(true);
  };

  const handleSaveTransaction = async (payload: TransactionFormData) => {
    try {
      await api.post('/transactions', payload);
      setIsTransactionFormOpen(false);
      await fetchBankAccounts();
      // Re-expand to refresh transaction history
      const currentExpanded = expandedAccountId;
      setExpandedAccountId(null);
      setTimeout(() => setExpandedAccountId(currentExpanded), 50);
    } catch (error) {
      console.error('Erro ao criar transacao:', error);
      toast('Erro ao criar transacao', 'error');
    }
  };

  const handleUpdateTransaction = async (id: string, payload: TransactionFormData) => {
    try {
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      await api.put(`/transactions/${id}`, cleanPayload);
      setIsTransactionFormOpen(false);
      setEditingTransaction(null);
      await fetchBankAccounts();
      const currentExpanded = expandedAccountId;
      setExpandedAccountId(null);
      setTimeout(() => setExpandedAccountId(currentExpanded), 50);
    } catch (error) {
      console.error('Erro ao atualizar transacao:', error);
      toast('Erro ao atualizar transacao', 'error');
    }
  };

  const handleEditTransaction = (tx: Transaction) => {
    setEditingTransaction(tx);
    setPreselectedAccountId(tx.bankAccountId);
    setIsTransactionFormOpen(true);
  };

  const handleDeleteTransaction = async (tx: Transaction) => {
    try {
      await api.delete(`/transactions/${tx.id}`);
      await fetchBankAccounts();
      const currentExpanded = expandedAccountId;
      setExpandedAccountId(null);
      setTimeout(() => setExpandedAccountId(currentExpanded), 50);
      toast('Transacao excluida com sucesso');
    } catch (error) {
      console.error('Erro ao excluir transacao:', error);
      toast('Erro ao excluir transacao', 'error');
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
      await api.put('/bank-accounts/reorder', {
        items: reordered.map((account, index) => ({
          id: account.id,
          displayOrder: index + 1,
        })),
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
                <div className="space-y-0.5">
                  {Object.entries(totalsByCurrency).map(([cur, total]) => (
                    <p key={cur} className="text-2xl font-bold text-white">{formatCurrency(total, cur)}</p>
                  ))}
                  {Object.keys(totalsByCurrency).length === 0 && (
                    <p className="text-2xl font-bold text-white">{formatCurrency(0)}</p>
                  )}
                </div>
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
                  <div className="space-y-0.5">
                    {Object.entries(investedByCurrency).map(([cur, total]) => (
                      <p key={cur} className="text-2xl font-bold text-white">{formatCurrency(total, cur)}</p>
                    ))}
                  </div>
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
                      // Skip credit cards linked to a broker — they render inside the broker card
                      if (account.linkedAccountId && brokerAccountIds.has(account.linkedAccountId)) return null;
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
                                    onUpdateInvoice={handleUpdateInvoice}
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
                                    accountCurrency={account.currency}
                                    isCreditCard
                                    onEdit={handleEditTransaction}
                                    onDelete={handleDeleteTransaction}
                                  />
                                </div>
                              )}
                            </div>
                          )}
                        </SortableItem>
                      );
                    }

                    const isBroker = brokerAccountIds.has(account.id);
                    const subInvestments = isBroker ? getSubInvestments(account.id) : [];
                    const subCreditCards = isBroker ? getSubCreditCards(account.id) : [];
                    const brokerTotalBrl = isBroker
                      ? account.currentBalance + subInvestments.reduce((sum, s) => sum + s.currentBalance, 0)
                      : account.currentBalance;

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
                                  <PencilButton account={account} />
                                </div>
                              </div>
                              {/* Sub-investments inside broker card */}
                              {isBroker && subInvestments.length > 0 && (
                                <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
                                  {/* Saldo em conta */}
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
                                        onClick={(e) => { e.stopPropagation(); setExpandedFiiId(isFiiExpanded ? null : inv.id); }}
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
                                              onClick={(e) => { e.stopPropagation(); openEditModal(inv); }}
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
                                        onPayInvoice={handlePayInvoice}
                                        onEdit={() => openEditModal(cc)}
                                        onUpdateInvoice={handleUpdateInvoice}
                                        selectedInvoiceId={selectedInvoiceByAccount[cc.id] || ''}
                                        onInvoiceSelect={(invoiceId) => handleInvoiceSelect(cc.id, invoiceId)}
                                      />
                                    </div>
                                  ))}
                                </div>
                              )}
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
                                  accountCurrency={account.currency}
                                  includeAsDestination={isBroker}
                                  onEdit={handleEditTransaction}
                                  onDelete={handleDeleteTransaction}
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
                    {Object.entries(investedByCurrency).map(([cur, total]) => (
                      <p key={cur} className="text-purple-300 text-sm font-semibold">
                        {formatCurrency(total, cur)}
                      </p>
                    ))}
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
                                    accountCurrency={account.currency}
                                    onEdit={handleEditTransaction}
                                    onDelete={handleDeleteTransaction}
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

            {cryptoAccounts.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-xl">🪙</span>
                    <h3 className="text-lg font-semibold text-white">Cripto</h3>
                  </div>
                  <div className="flex items-center gap-3">
                    {cryptoPrices && (
                      <span className="text-white/40 text-xs">
                        USD/BRL: {cryptoPrices.usdBrl.toFixed(2)}
                      </span>
                    )}
                    <span className="text-white/50 text-xs">
                      {cryptoAccounts.length} {cryptoAccounts.length === 1 ? 'conta' : 'contas'}
                    </span>
                  </div>
                </div>
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={(event) => handleDragEnd(event, cryptoAccounts)}
                >
                  <SortableContext items={cryptoAccounts.map((a) => a.id)} strategy={verticalListSortingStrategy}>
                    {cryptoAccounts.map((account) => {
                      const isExpanded = expandedAccountId === account.id;
                      const typeLabel = account.type === 'EXCHANGE' ? 'Corretora' : 'Carteira';
                      const subAssets = getSubAssets(account.id);
                      // Calculate total value: BRL balance + all sub-account values in BRL
                      const subAssetsTotalBrl = subAssets.reduce((sum, sub) => {
                        const sym = sub.name.match(/\(([A-Z]+)\)/)?.[1];
                        const pi = sym && cryptoPrices?.prices[sym];
                        if (pi && sub.numberOfQuotas != null) return sum + sub.numberOfQuotas * pi.priceBrl;
                        return sum + (sub.currentBalance || 0);
                      }, 0);
                      const totalBrl = account.currentBalance + subAssetsTotalBrl;
                      const totalUsd = cryptoPrices?.usdBrl ? totalBrl / cryptoPrices.usdBrl : null;
                      return (
                        <SortableItem key={account.id} id={account.id}>
                          {({ listeners, attributes }) => (
                            <div>
                              <div className="flex items-center gap-1">
                                <DragHandle listeners={listeners} attributes={attributes} />
                                <div
                                  className="cursor-pointer flex-1 rounded-xl bg-gradient-to-r from-white/[0.06] to-white/[0.02] border border-white/10 p-4 transition-all duration-300 hover:border-white/20"
                                  style={isExpanded ? {
                                    boxShadow: '0 0 20px rgba(52, 211, 153, 0.3), 0 0 0 2px #34d399',
                                  } : undefined}
                                  onClick={() => toggleExpand(account.id)}
                                >
                                  <div className="flex items-center justify-between">
                                    <div>
                                      <div className="flex items-center gap-2">
                                        <p className="text-white font-semibold">{account.name}</p>
                                        <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-400">
                                          {typeLabel}
                                        </span>
                                        {subAssets.length > 0 && (
                                          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-white/10 text-white/50">
                                            {subAssets.length} {subAssets.length === 1 ? 'ativo' : 'ativos'}
                                          </span>
                                        )}
                                      </div>
                                      {account.broker && (
                                        <p className="text-white/40 text-xs">{account.broker}</p>
                                      )}
                                    </div>
                                    <div className="text-right">
                                      <p className="text-white font-bold">{formatCurrency(totalBrl)}</p>
                                      {totalUsd != null && (
                                        <p className="text-white/40 text-xs">{formatCurrency(totalUsd, 'USD')}</p>
                                      )}
                                      <button
                                        onClick={(e) => { e.stopPropagation(); openEditModal(account); }}
                                        className="text-white/30 hover:text-white/70 text-xs transition-colors"
                                      >
                                        Editar
                                      </button>
                                    </div>
                                  </div>
                                  {/* Sub-assets inside the exchange card */}
                                  {(subAssets.length > 0 || account.currentBalance > 0) && (
                                    <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
                                      {/* BRL balance card */}
                                      {account.currentBalance > 0 && (
                                        <div className="px-2 py-2 rounded-lg bg-white/[0.04]">
                                          <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-2">
                                              <span className="text-emerald-400 text-xs">●</span>
                                              <span className="text-white/80 text-sm font-medium">Saldo em Reais</span>
                                              <span className="text-white/40 text-xs">BRL</span>
                                            </div>
                                            <div className="text-right">
                                              <span className="text-white/80 text-sm font-medium">{formatCurrency(account.currentBalance)}</span>
                                              {cryptoPrices?.usdBrl && (
                                                <p className="text-white/40 text-xs">{formatCurrency(account.currentBalance / cryptoPrices.usdBrl, 'USD')}</p>
                                              )}
                                            </div>
                                          </div>
                                        </div>
                                      )}
                                      {subAssets.map((sub) => {
                                        const symbolMatch = sub.name.match(/\(([A-Z]+)\)/);
                                        const symbol = symbolMatch ? symbolMatch[1] : '';
                                        const priceInfo = symbol && cryptoPrices?.prices[symbol];
                                        const purchases = symbol ? (cryptoPurchases[symbol] || []) : [];
                                        const totalInvestedBrl = purchases.reduce((sum, p) => sum + p.investedBrl, 0);
                                        const totalGainBrl = purchases.reduce((sum, p) => sum + p.gainTotalBrl, 0);
                                        const totalGainPercent = totalInvestedBrl > 0 ? (totalGainBrl / totalInvestedBrl) * 100 : 0;
                                        const valueBrl = priceInfo && sub.numberOfQuotas != null
                                          ? sub.numberOfQuotas * priceInfo.priceBrl
                                          : sub.currency !== 'BRL' && cryptoPrices?.usdBrl
                                            ? sub.currentBalance * cryptoPrices.usdBrl
                                            : null;
                                        const gainCryptoPercent = purchases.length > 0 ? purchases[0].gainCryptoPercent : null;
                                        const gainExchangePercent = purchases.length > 0 ? purchases[0].gainExchangePercent : null;
                                        return (
                                          <div key={sub.id} className="px-2 py-2 rounded-lg bg-white/[0.04]">
                                            <div className="flex items-center justify-between">
                                              <div className="flex items-center gap-2">
                                                <span className="text-amber-400 text-xs">●</span>
                                                <span className="text-white/80 text-sm font-medium">{sub.name.replace(account.name + ' - ', '')}</span>
                                                {sub.numberOfQuotas != null && (
                                                  <span className="text-white/40 text-xs">{sub.numberOfQuotas} quotas</span>
                                                )}
                                                {priceInfo && (
                                                  <span className="text-white/30 text-xs">@ {formatCurrency(priceInfo.priceUsd, 'USD')}</span>
                                                )}
                                              </div>
                                              <div className="flex items-center gap-3">
                                                <div className="text-right">
                                                  {priceInfo && sub.numberOfQuotas != null ? (
                                                    <>
                                                      <span className="text-white/80 text-sm font-medium">
                                                        {formatCurrency(sub.numberOfQuotas * priceInfo.priceUsd, 'USD')}
                                                      </span>
                                                      <p className="text-white/40 text-xs">
                                                        {formatCurrency(sub.numberOfQuotas * priceInfo.priceBrl)}
                                                      </p>
                                                    </>
                                                  ) : (
                                                    <span className="text-white/80 text-sm font-medium">{formatCurrency(sub.currentBalance, sub.currency)}</span>
                                                  )}
                                                </div>
                                                <button
                                                  onClick={(e) => { e.stopPropagation(); openEditModal(sub); }}
                                                  className="text-white/30 hover:text-white/70 text-xs transition-colors"
                                                >
                                                  Editar
                                                </button>
                                              </div>
                                            </div>
                                            {/* Gains breakdown */}
                                            {purchases.length > 0 && (
                                              <div className="mt-2 pt-2 border-t border-white/[0.06] grid grid-cols-3 gap-2 text-[10px]">
                                                <div>
                                                  <p className="text-white/30 uppercase">Cripto</p>
                                                  <p className={gainCryptoPercent != null && gainCryptoPercent >= 0 ? 'text-emerald-400' : 'text-red-400'}>
                                                    {gainCryptoPercent != null ? `${gainCryptoPercent >= 0 ? '+' : ''}${gainCryptoPercent.toFixed(1)}%` : '-'}
                                                  </p>
                                                  <p className="text-white/25">${purchases[0].priceUsd.toFixed(2)} → ${purchases[0].currentPriceUsd.toFixed(2)}</p>
                                                </div>
                                                <div>
                                                  <p className="text-white/30 uppercase">Cambio</p>
                                                  <p className={gainExchangePercent != null && gainExchangePercent >= 0 ? 'text-emerald-400' : 'text-red-400'}>
                                                    {gainExchangePercent != null ? `${gainExchangePercent >= 0 ? '+' : ''}${gainExchangePercent.toFixed(1)}%` : '-'}
                                                  </p>
                                                  <p className="text-white/25">R${purchases[0].exchangeRate.toFixed(2)} → R${purchases[0].currentExchangeRate.toFixed(2)}</p>
                                                </div>
                                                <div>
                                                  <p className="text-white/30 uppercase">Total BRL</p>
                                                  <p className={totalGainBrl >= 0 ? 'text-emerald-400 font-semibold' : 'text-red-400 font-semibold'}>
                                                    {totalGainBrl >= 0 ? '+' : ''}{formatCurrency(totalGainBrl)}
                                                  </p>
                                                  <p className={totalGainPercent >= 0 ? 'text-emerald-400/60' : 'text-red-400/60'}>
                                                    {totalGainPercent >= 0 ? '+' : ''}{totalGainPercent.toFixed(1)}%
                                                  </p>
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
                                    accountCurrency={account.currency}
                                    includeAsDestination
                                    onEdit={handleEditTransaction}
                                    onDelete={handleDeleteTransaction}
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
          onClose={() => {
            setIsTransactionFormOpen(false);
            setEditingTransaction(null);
          }}
          onSave={handleSaveTransaction}
          onUpdate={handleUpdateTransaction}
          accounts={bankAccounts}
          categories={categories}
          defaultProfileId={selectedProfileId}
          defaultBankAccountId={preselectedAccountId}
          profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
          editingTransaction={editingTransaction}
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
