'use client';

import { useMemo, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import BankAccountModal from '@/components/finances/BankAccountModal';
import CreditCardInfo from '@/components/finances/CreditCardInfo';
import TransactionForm from '@/components/finances/TransactionForm';
import type { BankAccount, Transaction } from '@/types/finances';
import { api } from '@/lib/api';
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
import { DragHandle, SortableItem } from '@/components/finances/SortableHelpers';
import ExpandedTransactionPanel from '@/components/finances/ExpandedTransactionPanel';
import CryptoSection from '@/components/finances/CryptoSection';
import InvestmentSection from '@/components/finances/InvestmentSection';
import SummaryCards from '@/components/finances/SummaryCards';
import BrokerAccountCard from '@/components/finances/BrokerAccountCard';
import { useContasData } from '@/hooks/useContasData';
import { useContasActions } from '@/hooks/useContasActions';


export default function ContasPage() {
  const { profiles, selectedProfileId, selectedProfile, isLoading: profilesLoading } = useProfile();

  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);
  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);
  const [expandedAccountId, setExpandedAccountId] = useState<string | null>(null);
  const [expandedFiiId, setExpandedFiiId] = useState<string | null>(null);
  const [selectedInvoiceByAccount, setSelectedInvoiceByAccount] = useState<Record<string, string>>({});
  const [isTransactionFormOpen, setIsTransactionFormOpen] = useState(false);
  const [preselectedAccountId, setPreselectedAccountId] = useState<string>('');
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null);

  const {
    bankAccounts,
    setBankAccounts,
    categories,
    filteredAccounts,
    invoicesByAccount,
    currentInvoices,
    cryptoPrices,
    cryptoPurchases,
    fetchBankAccounts,
    fetchInvoicesForCreditCards,
  } = useContasData(selectedProfileId);

  const {
    handleCreateBankAccount,
    handleUpdateBankAccount,
    handlePayInvoice,
    handleUpdateInvoice,
    handleSaveTransaction,
    handleUpdateTransaction,
    handleDeleteTransaction,
  } = useContasActions({
    fetchBankAccounts,
    fetchInvoicesForCreditCards,
    filteredAccounts,
    expandedAccountId,
    setExpandedAccountId,
    setIsTransactionFormOpen,
    setEditingTransaction: () => setEditingTransaction(null),
    setEditingBankAccount,
  });

  const cryptoAccounts = useMemo(
    () => filteredAccounts.filter((a) => a.type === 'EXCHANGE' || a.type === 'WALLET'),
    [filteredAccounts],
  );

  const cryptoAccountIds = useMemo(
    () => new Set(cryptoAccounts.map((a) => a.id)),
    [cryptoAccounts],
  );

  const getSubAssets = (parentId: string) =>
    filteredAccounts.filter((a) => a.linkedAccountId === parentId);

  const getSubInvestments = (parentId: string) =>
    filteredAccounts.filter((a) => a.linkedAccountId === parentId && a.type !== 'CREDIT_CARD');

  const getSubCreditCards = (parentId: string) =>
    filteredAccounts.filter((a) => a.linkedAccountId === parentId && a.type === 'CREDIT_CARD');

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
      || brokerAccountIds.has(a.id)
    ),
    [filteredAccounts, brokerAccountIds],
  );

  const investmentAccounts = useMemo(
    () => filteredAccounts.filter((a) =>
      a.type === 'INVESTMENT'
      && !brokerAccountIds.has(a.id)
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

  const investedByCurrency = useMemo(() => {
    const totals: Record<string, number> = {};
    investmentAccounts.forEach((a) => {
      const cur = a.currency || 'BRL';
      totals[cur] = (totals[cur] || 0) + a.currentBalance;
    });
    return totals;
  }, [investmentAccounts]);

  const toggleExpand = (accountId: string) => {
    setExpandedAccountId((prev) => (prev === accountId ? null : accountId));
  };

  const handleInvoiceSelect = (accountId: string, invoiceId: string) => {
    setSelectedInvoiceByAccount((prev) => ({ ...prev, [accountId]: invoiceId }));
    if (invoiceId) setExpandedAccountId(accountId);
  };

  const handleAddTransaction = (accountId: string) => {
    setPreselectedAccountId(accountId);
    setIsTransactionFormOpen(true);
  };

  const handleConfirmTransaction = async (tx: Transaction) => {
    await api.put(`/transactions/${tx.id}/status`, {
      status: 'CONFIRMED',
      occurredOn: tx.occurredOn,
    });
    await fetchBankAccounts();
    if (selectedProfileId) fetchInvoicesForCreditCards(filteredAccounts);
  };

  const handleEditTransaction = (tx: Transaction) => {
    setEditingTransaction(tx);
    setPreselectedAccountId(tx.bankAccountId);
    setIsTransactionFormOpen(true);
  };

  const openEditModal = (account: BankAccount) => {
    setEditingBankAccount(account);
    setIsBankAccountModalOpen(true);
  };

  const handleSaveBankAccount = (
    accountData: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (editingBankAccount) {
      handleUpdateBankAccount(editingBankAccount, accountData);
    } else {
      handleCreateBankAccount(accountData);
    }
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

    try {
      await api.put('/bank-accounts/reorder', {
        items: reordered.map((account, index) => ({ id: account.id, displayOrder: index + 1 })),
      });
    } catch (error) {
      console.error('Erro ao reordenar contas:', error);
      await fetchBankAccounts();
    }
  };

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
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
            <SummaryCards
              regularAccounts={regularAccounts}
              investmentAccounts={investmentAccounts}
              totalsByCurrency={totalsByCurrency}
              investedByCurrency={investedByCurrency}
            />

            {/* Regular accounts */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-white">Contas do perfil</h3>
                <span className="text-white/50 text-sm">
                  {regularAccounts.length} {regularAccounts.length === 1 ? 'conta' : 'contas'}
                </span>
              </div>
              {regularAccounts.length === 0 && (
                <p className="text-white/60 text-sm">Nenhuma conta cadastrada para este perfil.</p>
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
                      if (account.linkedAccountId && brokerAccountIds.has(account.linkedAccountId)) return null;
                      return (
                        <SortableItem key={account.id} id={account.id}>
                          {({ listeners, attributes }) => (
                            <div className="space-y-0">
                              <div className="flex items-center gap-1">
                                <DragHandle listeners={listeners} attributes={attributes} />
                                <div
                                  className="cursor-pointer flex-1 rounded-xl transition-all duration-300"
                                  style={isExpanded ? { boxShadow: '0 0 20px rgba(52, 211, 153, 0.3), 0 0 0 2px #34d399' } : undefined}
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
                                <ExpandedTransactionPanel
                                  accountId={account.id}
                                  profileId={selectedProfileId}
                                  categories={categories}
                                  selectedInvoiceId={selectedInvoiceByAccount[account.id] || ''}
                                  accountCurrency={account.currency}
                                  isCreditCard
                                  onAddTransaction={handleAddTransaction}
                                  onEdit={handleEditTransaction}
                                  onDelete={handleDeleteTransaction}
                                  onConfirm={handleConfirmTransaction}
                                  className="ml-7"
                                />
                              )}
                            </div>
                          )}
                        </SortableItem>
                      );
                    }

                    return (
                      <SortableItem key={account.id} id={account.id}>
                        {({ listeners, attributes }) => (
                          <BrokerAccountCard
                            account={account}
                            isExpanded={isExpanded}
                            isBroker={brokerAccountIds.has(account.id)}
                            subInvestments={brokerAccountIds.has(account.id) ? getSubInvestments(account.id) : []}
                            subCreditCards={brokerAccountIds.has(account.id) ? getSubCreditCards(account.id) : []}
                            expandedFiiId={expandedFiiId}
                            selectedProfileId={selectedProfileId}
                            categories={categories}
                            currentInvoices={currentInvoices}
                            invoicesByAccount={invoicesByAccount}
                            selectedInvoiceByAccount={selectedInvoiceByAccount}
                            listeners={listeners}
                            attributes={attributes}
                            onToggleExpand={toggleExpand}
                            onToggleFii={setExpandedFiiId}
                            onEditAccount={openEditModal}
                            onAddTransaction={handleAddTransaction}
                            onEditTransaction={handleEditTransaction}
                            onDeleteTransaction={handleDeleteTransaction}
                            onConfirmTransaction={handleConfirmTransaction}
                            onPayInvoice={handlePayInvoice}
                            onUpdateInvoice={handleUpdateInvoice}
                            onInvoiceSelect={handleInvoiceSelect}
                          />
                        )}
                      </SortableItem>
                    );
                  })}
                </SortableContext>
              </DndContext>
            </div>

            <InvestmentSection
              investmentAccounts={investmentAccounts}
              investedByCurrency={investedByCurrency}
              allAccounts={filteredAccounts}
              expandedAccountId={expandedAccountId}
              selectedProfileId={selectedProfileId}
              categories={categories}
              sensors={sensors}
              onToggleExpand={toggleExpand}
              onDragEnd={handleDragEnd}
              onEditAccount={openEditModal}
              onAddTransaction={handleAddTransaction}
              onEditTransaction={handleEditTransaction}
              onDeleteTransaction={handleDeleteTransaction}
              onConfirmTransaction={handleConfirmTransaction}
            />

            <CryptoSection
              cryptoAccounts={cryptoAccounts}
              cryptoPrices={cryptoPrices}
              cryptoPurchases={cryptoPurchases}
              expandedAccountId={expandedAccountId}
              selectedProfileId={selectedProfileId}
              categories={categories}
              sensors={sensors}
              getSubAssets={getSubAssets}
              onToggleExpand={toggleExpand}
              onDragEnd={handleDragEnd}
              onEditAccount={openEditModal}
              onAddTransaction={handleAddTransaction}
              onEditTransaction={handleEditTransaction}
              onDeleteTransaction={handleDeleteTransaction}
              onConfirmTransaction={handleConfirmTransaction}
            />
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
