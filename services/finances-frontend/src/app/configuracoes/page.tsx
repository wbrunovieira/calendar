'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import ProfileModal from '@/components/finances/ProfileModal';
import BankAccountModal from '@/components/finances/BankAccountModal';
import type { Profile, BankAccount } from '@/types/finances';

interface Calendar {
  id: string;
  name: string;
  email: string | null;
}

const API_BASE = 'http://localhost:3335/api/v1';

export default function ConfiguracoesPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [profilesLoading, setProfilesLoading] = useState(true);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);

  const [isProfileModalOpen, setIsProfileModalOpen] = useState(false);
  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);

  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);

  useEffect(() => {
    fetchProfiles();
    fetchCalendars();
    fetchBankAccounts();
  }, []);

  const fetchProfiles = async () => {
    try {
      setProfilesLoading(true);
      const response = await fetch(`${API_BASE}/profiles`);
      const data = await response.json();
      setProfiles(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar perfis:', error);
    } finally {
      setProfilesLoading(false);
    }
  };

  const fetchCalendars = async () => {
    try {
      const response = await fetch('http://localhost:3334/calendars');
      const data = await response.json();
      setCalendars(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar calendários:', error);
    }
  };

  const fetchBankAccounts = async () => {
    try {
      const response = await fetch(`${API_BASE}/bank-accounts`);
      const data = await response.json();
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Erro ao carregar contas bancárias:', error);
    }
  };

  const handleCreateProfile = async (
    profileData: Omit<Profile, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    try {
      const response = await fetch(`${API_BASE}/profiles`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(profileData),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao criar perfil');
      }

      await fetchProfiles();
    } catch (error) {
      console.error('Erro ao criar perfil:', error);
      alert('Erro ao criar perfil');
    }
  };

  const handleUpdateProfile = async (
    profileData: Omit<Profile, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (!editingProfile) return;

    try {
      const response = await fetch(`${API_BASE}/profiles/${editingProfile.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(profileData),
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao atualizar perfil');
      }

      await fetchProfiles();
      setEditingProfile(null);
    } catch (error) {
      console.error('Erro ao atualizar perfil:', error);
      alert('Erro ao atualizar perfil');
    }
  };

  const handleDeleteProfile = async (id: string) => {
    if (!confirm('Tem certeza que deseja excluir este perfil?')) return;

    try {
      const response = await fetch(`${API_BASE}/profiles/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao excluir perfil');
      }

      await fetchProfiles();
    } catch (error) {
      console.error('Erro ao excluir perfil:', error);
      alert('Erro ao excluir perfil');
    }
  };

  const handleSaveProfile = (
    profileData: Omit<Profile, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>,
  ) => {
    if (editingProfile) {
      handleUpdateProfile(profileData);
    } else {
      handleCreateProfile(profileData);
    }
  };

  const handleCreateBankAccount = useCallback(async (
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
      console.error('Erro ao criar conta bancária:', error);
      alert('Erro ao criar conta bancária');
    }
  }, []);

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
        throw new Error(errorMessage || 'Erro ao atualizar conta bancária');
      }

      await fetchBankAccounts();
      setEditingBankAccount(null);
    } catch (error) {
      console.error('Erro ao atualizar conta bancária:', error);
      alert('Erro ao atualizar conta bancária');
    }
  };

  const handleDeleteBankAccount = async (id: string) => {
    if (!confirm('Tem certeza que deseja excluir esta conta bancária?')) return;

    try {
      const response = await fetch(`${API_BASE}/bank-accounts/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage || 'Erro ao excluir conta bancária');
      }

      await fetchBankAccounts();
    } catch (error) {
      console.error('Erro ao excluir conta bancária:', error);
      alert('Erro ao excluir conta bancária');
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

  const getProfileName = (profileId: string) => {
    const profile = profiles.find((p) => p.id === profileId);
    return profile?.name || 'Sem perfil';
  };

  return (
    <AppLayout>
      <div className="py-10 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Configuracoes</h1>
            <p className="text-white/60 text-sm">Gerencie perfis financeiros e contas bancarias</p>
          </div>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">
            ← Voltar
          </Link>
        </div>

        <div className="space-y-8">
          {/* Perfis Financeiros */}
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-2xl font-bold text-white">Perfis financeiros</h2>
                <p className="text-white/60 text-sm">Organize contas pessoais e corporativas</p>
              </div>
              <button
                onClick={() => {
                  setEditingProfile(null);
                  setIsProfileModalOpen(true);
                }}
                className="flex items-center gap-2 px-5 py-2 rounded-xl bg-emerald-500/80 hover:bg-emerald-500 text-white font-semibold border border-emerald-400/40 transition-colors"
              >
                <span>➕</span>
                <span>Novo perfil</span>
              </button>
            </div>

            {profilesLoading ? (
              <div className="text-center py-12 text-white/60">Carregando...</div>
            ) : profiles.length === 0 ? (
              <div className="bg-white/10 border border-white/10 rounded-xl p-6 text-white/60 text-sm text-center">
                Nenhum perfil cadastrado ainda. Utilize o botao acima para comecar.
              </div>
            ) : (
              <div className="grid gap-4">
                {profiles.map((profile) => {
                  const calendar = calendars.find((c) => c.id === profile.calendarId);
                  const profileAccounts = bankAccounts.filter((a) => a.profileId === profile.id);
                  const totalBalance = profileAccounts
                    .filter((a) => a.type !== 'CREDIT_CARD')
                    .reduce((sum, a) => sum + a.currentBalance, 0);

                  return (
                    <div
                      key={profile.id}
                      className="bg-white/10 border border-white/15 rounded-xl p-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
                    >
                      <div className="flex items-center gap-4">
                        <div className="w-14 h-14 rounded-2xl flex items-center justify-center text-2xl bg-gradient-to-br from-white/20 to-white/5">
                          {profile.type === 'PERSONAL' ? '👤' : '🏢'}
                        </div>
                        <div>
                          <h3 className="text-white font-semibold text-lg">{profile.name}</h3>
                          <p className="text-white/60 text-sm">
                            {profile.type === 'PERSONAL' ? 'Pessoal' : 'Empresarial'}
                            {calendar ? ` • ${calendar.name}` : ''}
                          </p>
                          <p className="text-white/40 text-xs mt-1">
                            {profileAccounts.length} {profileAccounts.length === 1 ? 'conta' : 'contas'} •{' '}
                            {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(totalBalance)}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button
                          onClick={() => {
                            setEditingProfile(profile);
                            setIsProfileModalOpen(true);
                          }}
                          className="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white/80 border border-white/20 transition-colors"
                        >
                          Editar
                        </button>
                        <button
                          onClick={() => handleDeleteProfile(profile.id)}
                          className="px-4 py-2 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 text-rose-100 border border-rose-500/30 transition-colors"
                        >
                          Excluir
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Contas Bancárias */}
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-2xl font-bold text-white">Contas bancarias</h2>
                <p className="text-white/60 text-sm">Bancos, cartoes e carteiras digitais</p>
              </div>
              <button
                onClick={() => {
                  setEditingBankAccount(null);
                  setIsBankAccountModalOpen(true);
                }}
                className="flex items-center gap-2 px-5 py-2 rounded-xl bg-emerald-500/80 hover:bg-emerald-500 text-white font-semibold border border-emerald-400/40 transition-colors"
              >
                <span>➕</span>
                <span>Nova conta</span>
              </button>
            </div>

            {bankAccounts.length === 0 ? (
              <div className="bg-white/10 border border-white/10 rounded-xl p-6 text-white/60 text-sm text-center">
                Nenhuma conta cadastrada ainda. Use o botao acima para incluir um banco, cartao ou carteira.
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {bankAccounts.map((account) => {
                  const typeIcons: Record<string, string> = {
                    CHECKING: '🏦',
                    SAVINGS: '💰',
                    CREDIT_CARD: '💳',
                    INVESTMENT: '📈',
                    CASH: '💵',
                    DIGITAL_WALLET: '📱',
                  };
                  const typeLabels: Record<string, string> = {
                    CHECKING: 'Conta Corrente',
                    SAVINGS: 'Poupanca',
                    CREDIT_CARD: 'Cartao de Credito',
                    INVESTMENT: 'Investimento',
                    CASH: 'Dinheiro',
                    DIGITAL_WALLET: 'Carteira Digital',
                  };

                  return (
                    <div
                      key={account.id}
                      className="bg-white/10 border border-white/15 rounded-xl p-5 hover:bg-white/15 transition-colors cursor-pointer group"
                      onClick={() => {
                        setEditingBankAccount(account);
                        setIsBankAccountModalOpen(true);
                      }}
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <span className="text-2xl">{account.icon || typeIcons[account.type] || '🏦'}</span>
                          <div>
                            <h3 className="text-white font-semibold">{account.name}</h3>
                            <p className="text-white/50 text-xs">{typeLabels[account.type] || account.type}</p>
                          </div>
                        </div>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteBankAccount(account.id);
                          }}
                          className="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 transition-all"
                          title="Excluir"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                      <div className="space-y-1">
                        <p className={`text-lg font-bold ${account.type === 'CREDIT_CARD' ? 'text-rose-400' : 'text-emerald-400'}`}>
                          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: account.currency }).format(account.currentBalance)}
                        </p>
                        <p className="text-white/40 text-xs">
                          {getProfileName(account.profileId)}
                          {account.bankName && ` • ${account.bankName}`}
                        </p>
                      </div>
                      {account.type === 'CREDIT_CARD' && account.creditLimit && (
                        <div className="mt-3 pt-3 border-t border-white/10">
                          <div className="flex justify-between text-xs text-white/50 mb-1">
                            <span>Limite usado</span>
                            <span>
                              {Math.round((account.currentBalance / account.creditLimit) * 100)}%
                            </span>
                          </div>
                          <div className="h-1.5 bg-white/10 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-rose-500 transition-all"
                              style={{ width: `${Math.min(100, (account.currentBalance / account.creditLimit) * 100)}%` }}
                            />
                          </div>
                          <p className="text-white/40 text-xs mt-1">
                            Limite: {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: account.currency }).format(account.creditLimit)}
                          </p>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>

      <ProfileModal
        isOpen={isProfileModalOpen}
        onClose={() => {
          setIsProfileModalOpen(false);
          setEditingProfile(null);
        }}
        onSave={handleSaveProfile}
        profile={editingProfile}
        calendars={calendars}
      />

      <BankAccountModal
        isOpen={isBankAccountModalOpen}
        onClose={() => {
          setIsBankAccountModalOpen(false);
          setEditingBankAccount(null);
        }}
        onSave={handleSaveBankAccount}
        account={editingBankAccount}
        profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
        existingAccounts={bankAccounts}
      />
    </AppLayout>
  );
}
