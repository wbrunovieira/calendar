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
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);

  // Fetch profiles and bank accounts
  useEffect(() => {
    if (activeTab === 'settings') {
      fetchProfiles();
      fetchCalendars();
      fetchBankAccounts();
    }
    if (activeTab === 'dashboard') {
      fetchProfiles();
    }
  }, [activeTab]);

  // Auto-select first profile when profiles load
  useEffect(() => {
    if (profiles.length > 0 && !selectedProfileId) {
      setSelectedProfileId(profiles[0].id);
    }
  }, [profiles, selectedProfileId]);

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
          <div className="space-y-6">
            {/* Profile Selector */}
            {profiles.length > 0 && (
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-4 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <svg className="w-5 h-5 text-white/70" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                    <span className="text-white/70 text-sm font-medium">Visualizando perfil:</span>
                  </div>
                  <div className="flex gap-2">
                    {profiles.map((profile) => {
                      const isSelected = selectedProfileId === profile.id;
                      return (
                        <button
                          key={profile.id}
                          onClick={() => setSelectedProfileId(profile.id)}
                          className={`flex items-center gap-2 px-4 py-2 rounded-xl font-semibold transition-all duration-300 ${
                            isSelected
                              ? 'bg-white/20 text-white shadow-lg border border-white/30'
                              : 'bg-white/5 text-white/60 hover:bg-white/10 hover:text-white/80 border border-white/10'
                          }`}
                        >
                          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-lg ${
                            profile.type === 'PERSONAL'
                              ? 'bg-gradient-to-br from-blue-500 to-purple-600'
                              : 'bg-gradient-to-br from-green-500 to-teal-600'
                          }`}>
                            {profile.type === 'PERSONAL' ? '👤' : '🏢'}
                          </div>
                          <span>{profile.name}</span>
                        </button>
                      );
                    })}
                  </div>
                </div>
              </div>
            )}

            {/* Profile Selector & Quick Stats */}
            <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
              {/* Total Balance Card */}
              <div className="lg:col-span-2 bg-gradient-to-br from-emerald-600 to-teal-700 rounded-2xl p-6 border border-white/20 shadow-2xl">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <p className="text-emerald-100 text-sm font-medium">Saldo Total</p>
                    <h2 className="text-4xl font-bold text-white mt-1">R$ 47.582,50</h2>
                  </div>
                  <div className="w-16 h-16 bg-white/20 rounded-full flex items-center justify-center">
                    <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-emerald-100">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                  </svg>
                  <span className="text-sm">+12.5% vs mês anterior</span>
                </div>
              </div>

              {/* Income Card */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-3">
                  <p className="text-white/70 text-sm font-medium">Receitas</p>
                  <div className="w-10 h-10 bg-green-500/20 rounded-lg flex items-center justify-center">
                    <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 11l5-5m0 0l5 5m-5-5v12" />
                    </svg>
                  </div>
                </div>
                <h3 className="text-2xl font-bold text-white">R$ 18.500,00</h3>
                <p className="text-green-400 text-sm mt-2">Este mês</p>
              </div>

              {/* Expenses Card */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-3">
                  <p className="text-white/70 text-sm font-medium">Despesas</p>
                  <div className="w-10 h-10 bg-red-500/20 rounded-lg flex items-center justify-center">
                    <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 13l-5 5m0 0l-5-5m5 5V6" />
                    </svg>
                  </div>
                </div>
                <h3 className="text-2xl font-bold text-white">R$ 12.340,00</h3>
                <p className="text-red-400 text-sm mt-2">Este mês</p>
              </div>
            </div>

            {/* Charts & Analysis */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Cash Flow Chart */}
              <div className="lg:col-span-2 bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xl font-bold text-white">Fluxo de Caixa</h3>
                  <select className="px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500">
                    <option className="bg-gray-900">Últimos 6 meses</option>
                    <option className="bg-gray-900">Últimos 3 meses</option>
                    <option className="bg-gray-900">Este ano</option>
                  </select>
                </div>

                {/* Simplified Bar Chart Visualization */}
                <div className="space-y-4">
                  {[
                    { month: 'Jun', income: 15000, expense: 11000 },
                    { month: 'Jul', income: 17500, expense: 13200 },
                    { month: 'Ago', income: 16200, expense: 10800 },
                    { month: 'Set', income: 19000, expense: 14500 },
                    { month: 'Out', income: 18200, expense: 11900 },
                    { month: 'Nov', income: 18500, expense: 12340 },
                  ].map((data) => (
                    <div key={data.month} className="flex items-center gap-4">
                      <div className="w-12 text-white/70 text-sm font-medium">{data.month}</div>
                      <div className="flex-1 flex gap-2">
                        <div
                          className="bg-green-500/50 h-8 rounded flex items-center justify-end pr-2"
                          style={{ width: `${(data.income / 20000) * 100}%` }}
                        >
                          <span className="text-white text-xs font-semibold">{(data.income / 1000).toFixed(0)}k</span>
                        </div>
                        <div
                          className="bg-red-500/50 h-8 rounded flex items-center justify-end pr-2"
                          style={{ width: `${(data.expense / 20000) * 100}%` }}
                        >
                          <span className="text-white text-xs font-semibold">{(data.expense / 1000).toFixed(0)}k</span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                <div className="flex items-center gap-6 mt-6 pt-4 border-t border-white/10">
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 bg-green-500 rounded"></div>
                    <span className="text-white/70 text-sm">Receitas</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 bg-red-500 rounded"></div>
                    <span className="text-white/70 text-sm">Despesas</span>
                  </div>
                </div>
              </div>

              {/* Expense Categories */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <h3 className="text-xl font-bold text-white mb-6">Despesas por Categoria</h3>
                <div className="space-y-4">
                  {[
                    { name: 'Moradia', value: 3500, icon: '🏠', color: 'bg-blue-500' },
                    { name: 'Alimentação', value: 2800, icon: '🍔', color: 'bg-emerald-500' },
                    { name: 'Transporte', value: 1200, icon: '🚗', color: 'bg-yellow-500' },
                    { name: 'Lazer', value: 1500, icon: '🎮', color: 'bg-purple-500' },
                    { name: 'Serviços', value: 2340, icon: '💼', color: 'bg-orange-500' },
                    { name: 'Outros', value: 1000, icon: '📦', color: 'bg-gray-500' },
                  ].map((category) => (
                    <div key={category.name}>
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <span className="text-xl">{category.icon}</span>
                          <span className="text-white text-sm font-medium">{category.name}</span>
                        </div>
                        <span className="text-white/70 text-sm">R$ {category.value.toLocaleString('pt-BR')}</span>
                      </div>
                      <div className="w-full bg-white/10 rounded-full h-2">
                        <div
                          className={`${category.color} h-2 rounded-full transition-all duration-500`}
                          style={{ width: `${(category.value / 3500) * 100}%` }}
                        ></div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Accounts & Transactions Row */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Accounts Overview */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xl font-bold text-white">Minhas Contas</h3>
                  <button className="text-emerald-400 text-sm font-semibold hover:text-emerald-300 transition-colors">
                    Ver todas
                  </button>
                </div>
                <div className="space-y-3">
                  {[
                    { name: 'Nubank', balance: 15420.50, icon: '💜', change: '+5.2%' },
                    { name: 'Banco Inter', balance: 8750.00, icon: '🧡', change: '+2.1%' },
                    { name: 'Investimentos XP', balance: 23412.00, icon: '📈', change: '+8.4%' },
                  ].map((account) => (
                    <div key={account.name} className="flex items-center justify-between p-4 bg-white/5 rounded-xl border border-white/10 hover:bg-white/10 transition-all">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-white/10 rounded-full flex items-center justify-center text-xl">
                          {account.icon}
                        </div>
                        <div>
                          <p className="text-white font-semibold">{account.name}</p>
                          <p className="text-white/60 text-sm">R$ {account.balance.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}</p>
                        </div>
                      </div>
                      <div className="text-green-400 text-sm font-semibold">{account.change}</div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Recent Transactions */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xl font-bold text-white">Transações Recentes</h3>
                  <button className="text-emerald-400 text-sm font-semibold hover:text-emerald-300 transition-colors">
                    Ver todas
                  </button>
                </div>
                <div className="space-y-3">
                  {[
                    { name: 'Salário WB Digital', value: 18500, type: 'income', date: '01/12', category: '💼' },
                    { name: 'Aluguel', value: -3500, type: 'expense', date: '05/12', category: '🏠' },
                    { name: 'Mercado', value: -487.50, type: 'expense', date: '10/12', category: '🛒' },
                    { name: 'Freela Design', value: 2500, type: 'income', date: '12/12', category: '💻' },
                    { name: 'Netflix', value: -55.90, type: 'expense', date: '15/12', category: '🎬' },
                  ].map((transaction, idx) => (
                    <div key={idx} className="flex items-center justify-between p-3 bg-white/5 rounded-xl border border-white/10">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-white/10 rounded-full flex items-center justify-center text-xl">
                          {transaction.category}
                        </div>
                        <div>
                          <p className="text-white font-medium text-sm">{transaction.name}</p>
                          <p className="text-white/50 text-xs">{transaction.date}/2024</p>
                        </div>
                      </div>
                      <div className={`font-bold ${transaction.type === 'income' ? 'text-green-400' : 'text-red-400'}`}>
                        {transaction.type === 'income' ? '+' : ''}R$ {Math.abs(transaction.value).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Upcoming Bills & Goals */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Upcoming Bills */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xl font-bold text-white">Contas a Pagar</h3>
                  <span className="px-3 py-1 bg-yellow-500/20 text-yellow-400 rounded-full text-xs font-semibold">
                    3 pendentes
                  </span>
                </div>
                <div className="space-y-3">
                  {[
                    { name: 'Cartão de Crédito', value: 2847.50, dueDate: '18/12', status: 'pending' },
                    { name: 'Conta de Luz', value: 284.50, dueDate: '20/12', status: 'pending' },
                    { name: 'Internet', value: 119.90, dueDate: '25/12', status: 'pending' },
                  ].map((bill) => (
                    <div key={bill.name} className="flex items-center justify-between p-4 bg-white/5 rounded-xl border border-white/10">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-yellow-500/20 rounded-full flex items-center justify-center">
                          <svg className="w-5 h-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                        </div>
                        <div>
                          <p className="text-white font-medium text-sm">{bill.name}</p>
                          <p className="text-white/50 text-xs">Venc: {bill.dueDate}/2024</p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="text-white font-bold text-sm">R$ {bill.value.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}</p>
                        <button className="text-emerald-400 text-xs font-semibold hover:text-emerald-300 mt-1">
                          Pagar agora
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Financial Goals */}
              <div className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-xl">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xl font-bold text-white">Metas Financeiras</h3>
                  <button className="text-emerald-400 text-sm font-semibold hover:text-emerald-300 transition-colors">
                    + Nova meta
                  </button>
                </div>
                <div className="space-y-4">
                  {[
                    { name: 'Fundo de Emergência', current: 28500, target: 50000, icon: '🛡️' },
                    { name: 'Viagem Europa', current: 8200, target: 20000, icon: '✈️' },
                    { name: 'Novo MacBook', current: 6500, target: 15000, icon: '💻' },
                  ].map((goal) => {
                    const progress = (goal.current / goal.target) * 100;
                    return (
                      <div key={goal.name}>
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <span className="text-xl">{goal.icon}</span>
                            <span className="text-white font-medium text-sm">{goal.name}</span>
                          </div>
                          <span className="text-white/70 text-sm">{progress.toFixed(0)}%</span>
                        </div>
                        <div className="w-full bg-white/10 rounded-full h-3 mb-1">
                          <div
                            className="bg-gradient-to-r from-emerald-500 to-teal-500 h-3 rounded-full transition-all duration-500"
                            style={{ width: `${progress}%` }}
                          ></div>
                        </div>
                        <div className="flex items-center justify-between text-xs text-white/60">
                          <span>R$ {goal.current.toLocaleString('pt-BR')}</span>
                          <span>R$ {goal.target.toLocaleString('pt-BR')}</span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* AI Insights */}
            <div className="bg-gradient-to-r from-purple-600/20 to-pink-600/20 backdrop-blur-sm rounded-2xl p-6 border border-purple-500/30 shadow-xl">
              <div className="flex items-start gap-4">
                <div className="w-12 h-12 bg-purple-500/30 rounded-full flex items-center justify-center flex-shrink-0">
                  <svg className="w-6 h-6 text-purple-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                  </svg>
                </div>
                <div className="flex-1">
                  <h3 className="text-xl font-bold text-white mb-2">💡 Insights com IA</h3>
                  <p className="text-white/80 mb-4">
                    Baseado nos seus hábitos de consumo, você pode economizar até <span className="text-emerald-400 font-bold">R$ 1.250/mês</span> reduzindo gastos com delivery e assinaturas não utilizadas.
                  </p>
                  <div className="flex gap-3">
                    <button className="px-4 py-2 bg-purple-500/30 hover:bg-purple-500/40 text-white rounded-lg font-semibold transition-all text-sm">
                      Ver análise completa
                    </button>
                    <button className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg font-semibold transition-all text-sm">
                      Ignorar
                    </button>
                  </div>
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
