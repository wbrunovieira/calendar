'use client';

import { useState, useEffect } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import ProfileModal from '@/components/finances/ProfileModal';
import BankAccountModal, { AccountType } from '@/components/finances/BankAccountModal';

type TabType = 'dashboard' | 'settings';

interface Profile {
  id: string;
  calendarId: string;
  name: string;
  type: 'PERSONAL' | 'BUSINESS';
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface Calendar {
  id: string;
  name: string;
  email: string | null;
}

interface BankAccount {
  id: string;
  profileId: string;
  name: string;
  type: AccountType;
  initialBalance: number;
  currentBalance: number;
  currency: string;
  isActive: boolean;
  bankName?: string;
  bankCode?: string;
  agency?: string;
  accountNumber?: string;
  accountDigit?: string;
  color?: string;
  icon?: string;
  description?: string;
  creditLimit?: number;
  dueDay?: number;
  closingDay?: number;
  createdAt: string;
  updatedAt: string;
}

export default function FinancesPage() {
  const [activeTab, setActiveTab] = useState<TabType>('dashboard');
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isBankAccountModalOpen, setIsBankAccountModalOpen] = useState(false);
  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [editingBankAccount, setEditingBankAccount] = useState<BankAccount | null>(null);

  // Fetch profiles and bank accounts
  useEffect(() => {
    if (activeTab === 'settings') {
      fetchProfiles();
      fetchCalendars();
      fetchBankAccounts();
    }
  }, [activeTab]);

  const fetchProfiles = async () => {
    try {
      const response = await fetch('http://localhost:3335/api/v1/profiles');
      const data = await response.json();
      setProfiles(data.data || []);
    } catch (error) {
      console.error('Error fetching profiles:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchCalendars = async () => {
    try {
      const response = await fetch('http://localhost:3334/calendars');
      const data = await response.json();
      setCalendars(data.data || []);
    } catch (error) {
      console.error('Error fetching calendars:', error);
    }
  };

  const handleCreateProfile = async (profileData: Omit<Profile, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>) => {
    try {
      const response = await fetch('http://localhost:3335/api/v1/profiles', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(profileData),
      });

      if (response.ok) {
        await fetchProfiles();
      } else {
        const error = await response.text();
        alert(`Erro ao criar perfil: ${error}`);
      }
    } catch (error) {
      console.error('Error creating profile:', error);
      alert('Erro ao criar perfil');
    }
  };

  const handleUpdateProfile = async (profileData: Omit<Profile, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>) => {
    if (!editingProfile) return;

    try {
      const response = await fetch(
        `http://localhost:3335/api/v1/profiles/${editingProfile.id}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(profileData),
        }
      );

      if (response.ok) {
        await fetchProfiles();
        setEditingProfile(null);
      } else {
        const error = await response.text();
        alert(`Erro ao atualizar perfil: ${error}`);
      }
    } catch (error) {
      console.error('Error updating profile:', error);
      alert('Erro ao atualizar perfil');
    }
  };

  const handleDeleteProfile = async (id: string) => {
    if (!confirm('Tem certeza que deseja excluir este perfil?')) return;

    try {
      const response = await fetch(
        `http://localhost:3335/api/v1/profiles/${id}`,
        {
          method: 'DELETE',
        }
      );

      if (response.ok) {
        await fetchProfiles();
      } else {
        const error = await response.text();
        alert(`Erro ao excluir perfil: ${error}`);
      }
    } catch (error) {
      console.error('Error deleting profile:', error);
      alert('Erro ao excluir perfil');
    }
  };

  const handleSaveProfile = (profileData: Omit<Profile, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>) => {
    if (editingProfile) {
      handleUpdateProfile(profileData);
    } else {
      handleCreateProfile(profileData);
    }
  };

  // Bank Accounts Functions
  const fetchBankAccounts = async () => {
    try {
      const response = await fetch('http://localhost:3335/api/v1/bank-accounts');
      const data = await response.json();
      setBankAccounts(data.data || []);
    } catch (error) {
      console.error('Error fetching bank accounts:', error);
    }
  };

  const handleCreateBankAccount = async (accountData: Omit<BankAccount, 'id' | 'currentBalance' | 'isActive' | 'createdAt' | 'updatedAt'>) => {
    try {
      const response = await fetch('http://localhost:3335/api/v1/bank-accounts', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(accountData),
      });

      if (response.ok) {
        await fetchBankAccounts();
      } else {
        const error = await response.text();
        alert(`Erro ao criar conta: ${error}`);
      }
    } catch (error) {
      console.error('Error creating bank account:', error);
      alert('Erro ao criar conta bancária');
    }
  };

  const handleUpdateBankAccount = async (accountData: Omit<BankAccount, 'id' | 'currentBalance' | 'isActive' | 'createdAt' | 'updatedAt'>) => {
    if (!editingBankAccount) return;

    try {
      const response = await fetch(
        `http://localhost:3335/api/v1/bank-accounts/${editingBankAccount.id}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(accountData),
        }
      );

      if (response.ok) {
        await fetchBankAccounts();
        setEditingBankAccount(null);
      } else {
        const error = await response.text();
        alert(`Erro ao atualizar conta: ${error}`);
      }
    } catch (error) {
      console.error('Error updating bank account:', error);
      alert('Erro ao atualizar conta bancária');
    }
  };

  const handleDeleteBankAccount = async (id: string) => {
    if (!confirm('Tem certeza que deseja excluir esta conta bancária?')) return;

    try {
      const response = await fetch(
        `http://localhost:3335/api/v1/bank-accounts/${id}`,
        {
          method: 'DELETE',
        }
      );

      if (response.ok) {
        await fetchBankAccounts();
      } else {
        const error = await response.text();
        alert(`Erro ao excluir conta: ${error}`);
      }
    } catch (error) {
      console.error('Error deleting bank account:', error);
      alert('Erro ao excluir conta bancária');
    }
  };

  const handleSaveBankAccount = (accountData: Omit<BankAccount, 'id' | 'currentBalance' | 'isActive' | 'createdAt' | 'updatedAt'>) => {
    if (editingBankAccount) {
      handleUpdateBankAccount(accountData);
    } else {
      handleCreateBankAccount(accountData);
    }
  };

  return (
    <AppLayout>
      <div className="flex-1 w-full py-8 relative">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-extrabold text-white drop-shadow-lg mb-2">
            💰 Finanças
          </h1>
          <p className="text-white/70 text-lg">
            Gerencie suas contas, transações e investimentos
          </p>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setActiveTab('dashboard')}
            className={`px-6 py-3 rounded-xl font-semibold transition-all duration-300 ${
              activeTab === 'dashboard'
                ? 'bg-white/20 text-white shadow-lg'
                : 'bg-white/5 text-white/70 hover:bg-white/10'
            }`}
          >
            Dashboard
          </button>
          <button
            onClick={() => setActiveTab('settings')}
            className={`px-6 py-3 rounded-xl font-semibold transition-all duration-300 ${
              activeTab === 'settings'
                ? 'bg-white/20 text-white shadow-lg'
                : 'bg-white/5 text-white/70 hover:bg-white/10'
            }`}
          >
            Configurações
          </button>
        </div>

        {/* Dashboard Tab */}
        {activeTab === 'dashboard' && (
          <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-12 border border-white/10 shadow-2xl text-center">
          <div className="max-w-2xl mx-auto">
            <div className="text-6xl mb-6">🚧</div>
            <h2 className="text-3xl font-bold text-white mb-4">
              Em Construção
            </h2>
            <p className="text-white/70 text-lg mb-8">
              O módulo financeiro está sendo desenvolvido e em breve estará disponível com as seguintes funcionalidades:
            </p>

            <div className="grid md:grid-cols-2 gap-6 text-left">
              {/* Contas */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">🏦</div>
                  <h3 className="text-xl font-bold text-white">Contas</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Contas pessoais</li>
                  <li>• Conta empresarial</li>
                  <li>• Controle de saldos</li>
                  <li>• Múltiplas moedas</li>
                </ul>
              </div>

              {/* Transações */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">💸</div>
                  <h3 className="text-xl font-bold text-white">Transações</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Receitas e despesas</li>
                  <li>• Categorização automática</li>
                  <li>• Contas recorrentes</li>
                  <li>• Tags personalizadas</li>
                </ul>
              </div>

              {/* Investimentos */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">📈</div>
                  <h3 className="text-xl font-bold text-white">Investimentos</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Ações e FIIs</li>
                  <li>• Tesouro Direto</li>
                  <li>• CDB, LCI, LCA</li>
                  <li>• Rentabilidade</li>
                </ul>
              </div>

              {/* Relatórios */}
              <div className="bg-white/10 rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-3 mb-4">
                  <div className="text-3xl">📊</div>
                  <h3 className="text-xl font-bold text-white">Relatórios</h3>
                </div>
                <ul className="space-y-2 text-white/70">
                  <li>• Fluxo de caixa</li>
                  <li>• Previsões com IA</li>
                  <li>• Análise de gastos</li>
                  <li>• Metas financeiras</li>
                </ul>
              </div>
            </div>

            {/* API Status */}
            <div className="mt-8 p-4 bg-green-500/20 rounded-lg border border-green-500/30">
              <div className="flex items-center justify-center gap-2 text-green-400">
                <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                <span className="font-semibold">API Calendar Finances rodando em http://localhost:3335</span>
              </div>
            </div>
          </div>
        </div>
        )}

        {/* Settings Tab */}
        {activeTab === 'settings' && (
          <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-8 border border-white/10 shadow-2xl">
            <div className="max-w-6xl mx-auto">
              <div className="mb-8">
                <h2 className="text-3xl font-bold text-white mb-2">Configurações Financeiras</h2>
                <p className="text-white/70">
                  Gerencie perfis financeiros e suas configurações
                </p>
              </div>

              {/* Profiles Section */}
              <div className="mb-8">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-2xl font-bold text-white">Perfis Financeiros</h3>
                  <button
                    onClick={() => {
                      setEditingProfile(null);
                      setIsModalOpen(true);
                    }}
                    className="flex items-center gap-2 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105 border border-white/20"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    <span>Novo Perfil</span>
                  </button>
                </div>

                {loading ? (
                  <div className="text-center py-12">
                    <div className="inline-block animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-white"></div>
                    <p className="text-white/70 mt-4">Carregando perfis...</p>
                  </div>
                ) : profiles.length === 0 ? (
                  <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/10">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-2xl">
                          💼
                        </div>
                        <div>
                          <h4 className="text-xl font-bold text-white">Aguardando configuração</h4>
                          <p className="text-white/60 text-sm">Crie perfis financeiros para começar</p>
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="grid gap-4">
                    {profiles.map((profile) => {
                      const calendar = calendars.find((c) => c.id === profile.calendarId);
                      return (
                        <div
                          key={profile.id}
                          className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/10 hover:bg-white/15 transition-all"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-4">
                              <div className={`w-12 h-12 rounded-full flex items-center justify-center text-2xl ${
                                profile.type === 'PERSONAL'
                                  ? 'bg-gradient-to-br from-blue-500 to-purple-600'
                                  : 'bg-gradient-to-br from-green-500 to-teal-600'
                              }`}>
                                {profile.type === 'PERSONAL' ? '👤' : '🏢'}
                              </div>
                              <div>
                                <h4 className="text-xl font-bold text-white">{profile.name}</h4>
                                <p className="text-white/60 text-sm">
                                  {profile.type === 'PERSONAL' ? 'Pessoal' : 'Empresarial'}
                                  {calendar && ` • ${calendar.name}`}
                                </p>
                                <p className="text-white/40 text-xs mt-1">
                                  ID: {profile.id}
                                </p>
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => {
                                  setEditingProfile(profile);
                                  setIsModalOpen(true);
                                }}
                                className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg font-semibold transition-all duration-300 border border-white/20"
                              >
                                Editar
                              </button>
                              <button
                                onClick={() => handleDeleteProfile(profile.id)}
                                className="px-4 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-200 rounded-lg font-semibold transition-all duration-300 border border-red-500/30"
                              >
                                Excluir
                              </button>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>

              {/* Info Box */}
              <div className="p-6 bg-blue-500/10 rounded-xl border border-blue-500/20 mb-8">
                <div className="flex items-start gap-3">
                  <svg className="w-6 h-6 text-blue-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div>
                    <h4 className="text-white font-semibold mb-1">Sobre os Perfis Financeiros</h4>
                    <p className="text-white/70 text-sm">
                      Cada perfil financeiro está vinculado a um calendário. Você pode ter perfis do tipo PESSOAL ou EMPRESARIAL.
                      Os perfis controlam contas, transações e investimentos de forma independente.
                    </p>
                  </div>
                </div>
              </div>

              {/* Bank Accounts Section */}
              <div className="mb-8">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-2xl font-bold text-white">Contas Bancárias</h3>
                  <button
                    onClick={() => {
                      setEditingBankAccount(null);
                      setIsBankAccountModalOpen(true);
                    }}
                    className="flex items-center gap-2 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105 border border-white/20"
                    disabled={profiles.length === 0}
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    <span>Nova Conta</span>
                  </button>
                </div>

                {profiles.length === 0 ? (
                  <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/10">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 bg-gradient-to-br from-yellow-500 to-orange-600 rounded-full flex items-center justify-center text-2xl">
                        ⚠️
                      </div>
                      <div>
                        <h4 className="text-xl font-bold text-white">Crie um perfil primeiro</h4>
                        <p className="text-white/60 text-sm">Você precisa criar um perfil financeiro antes de adicionar contas bancárias</p>
                      </div>
                    </div>
                  </div>
                ) : bankAccounts.length === 0 ? (
                  <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/10">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-full flex items-center justify-center text-2xl">
                        🏦
                      </div>
                      <div>
                        <h4 className="text-xl font-bold text-white">Nenhuma conta cadastrada</h4>
                        <p className="text-white/60 text-sm">Adicione suas contas bancárias, cartões e investimentos</p>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="grid gap-4">
                    {bankAccounts.map((account) => {
                      const profile = profiles.find((p) => p.id === account.profileId);
                      const formatCurrency = (value: number) => {
                        return new Intl.NumberFormat('pt-BR', {
                          style: 'currency',
                          currency: account.currency || 'BRL',
                        }).format(value);
                      };

                      return (
                        <div
                          key={account.id}
                          className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/10 hover:bg-white/15 transition-all"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-4">
                              <div
                                className="w-12 h-12 rounded-full flex items-center justify-center text-2xl"
                                style={{ backgroundColor: account.color || '#10b981' }}
                              >
                                {account.icon || '🏦'}
                              </div>
                              <div>
                                <h4 className="text-xl font-bold text-white">{account.name}</h4>
                                <p className="text-white/60 text-sm">
                                  {account.bankName && `${account.bankName}`}
                                  {account.accountNumber && ` • ${account.accountNumber}${account.accountDigit ? `-${account.accountDigit}` : ''}`}
                                  {profile && ` • ${profile.name}`}
                                </p>
                                <div className="flex items-center gap-3 mt-2">
                                  <p className="text-emerald-400 font-bold">
                                    {formatCurrency(account.currentBalance)}
                                  </p>
                                  {account.type === 'CREDIT_CARD' && account.creditLimit && (
                                    <p className="text-white/50 text-sm">
                                      Limite: {formatCurrency(account.creditLimit)}
                                    </p>
                                  )}
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => {
                                  setEditingBankAccount(account);
                                  setIsBankAccountModalOpen(true);
                                }}
                                className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg font-semibold transition-all duration-300 border border-white/20"
                              >
                                Editar
                              </button>
                              <button
                                onClick={() => handleDeleteBankAccount(account.id)}
                                className="px-4 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-200 rounded-lg font-semibold transition-all duration-300 border border-red-500/30"
                              >
                                Excluir
                              </button>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Profile Modal */}
        <ProfileModal
          isOpen={isModalOpen}
          onClose={() => {
            setIsModalOpen(false);
            setEditingProfile(null);
          }}
          onSave={handleSaveProfile}
          profile={editingProfile}
          calendars={calendars}
        />

        {/* Bank Account Modal */}
        <BankAccountModal
          isOpen={isBankAccountModalOpen}
          onClose={() => {
            setIsBankAccountModalOpen(false);
            setEditingBankAccount(null);
          }}
          onSave={handleSaveBankAccount}
          account={editingBankAccount}
          profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
        />
      </div>
    </AppLayout>
  );
}
