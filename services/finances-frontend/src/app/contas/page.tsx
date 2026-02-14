'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import BankAccountModal from '@/components/finances/BankAccountModal';
import CreditCardInfo from '@/components/finances/CreditCardInfo';
import InvestmentAccountInfo from '@/components/finances/InvestmentAccountInfo';
import type { BankAccount, Invoice } from '@/types/finances';
import { API_BASE } from '@/lib/api';

export default function ContasPage() {
  const { profiles, selectedProfileId, selectedProfile, isLoading: profilesLoading } = useProfile();

  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);
  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);

  const [invoicesByAccount, setInvoicesByAccount] = useState<Record<string, Invoice[]>>({});
  const [currentInvoices, setCurrentInvoices] = useState<Record<string, Invoice>>({});

  const filteredAccounts = useMemo(
    () => bankAccounts.filter((account) => account.profileId === selectedProfileId),
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

  useEffect(() => {
    fetchBankAccounts();
  }, []);

  const fetchBankAccounts = async () => {
    try {
      const response = await fetch(`${API_BASE}/bank-accounts`);
      const data = await response.json();
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar contas bancarias:', error);
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

  const formatCurrency = (value: number, currency = 'BRL') =>
    new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

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
                <p className="text-white/40 text-xs mt-1">
                  {regularAccounts.filter((a) => a.type !== 'CREDIT_CARD').length} {regularAccounts.filter((a) => a.type !== 'CREDIT_CARD').length === 1 ? 'conta' : 'contas'}
                </p>
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
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-sm">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Contas do perfil</h3>
                <span className="text-white/50 text-sm">
                  {regularAccounts.length} {regularAccounts.length === 1 ? 'conta' : 'contas'}
                </span>
              </div>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {regularAccounts.length === 0 && (
                  <p className="text-white/60 text-sm col-span-full">
                    Nenhuma conta cadastrada para este perfil.
                  </p>
                )}
                {regularAccounts.map((account) =>
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
                          {formatCurrency(account.currentBalance, account.currency)}
                        </p>
                        <p className="text-white/50 text-xs">Saldo atual</p>
                      </div>
                    </div>
                  ),
                )}
              </div>
            </div>

            {/* Investments */}
            {investmentAccounts.length > 0 && (
              <div className="bg-gradient-to-br from-purple-900/30 to-indigo-900/30 border border-purple-500/20 rounded-2xl p-6 backdrop-blur-sm">
                <div className="flex items-center justify-between mb-4">
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
                <div className="grid gap-3 sm:grid-cols-2">
                  {investmentAccounts.map((account) => (
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
          </>
        )}
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
    </AppLayout>
  );
}
