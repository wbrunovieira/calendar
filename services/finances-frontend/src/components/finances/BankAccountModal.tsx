'use client';

import { useState, useEffect } from 'react';

export type AccountType = 'CHECKING' | 'SAVINGS' | 'INVESTMENT' | 'CREDIT_CARD' | 'CASH' | 'OTHER';

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

interface BankAccountModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (account: Omit<BankAccount, 'id' | 'currentBalance' | 'isActive' | 'createdAt' | 'updatedAt'>) => void;
  account: BankAccount | null;
  profiles: Array<{ id: string; name: string }>;
}

const accountTypes: { value: AccountType; label: string; icon: string }[] = [
  { value: 'CHECKING', label: 'Conta Corrente', icon: '🏦' },
  { value: 'SAVINGS', label: 'Conta Poupança', icon: '💰' },
  { value: 'INVESTMENT', label: 'Investimentos', icon: '📈' },
  { value: 'CREDIT_CARD', label: 'Cartão de Crédito', icon: '💳' },
  { value: 'CASH', label: 'Dinheiro', icon: '💵' },
  { value: 'OTHER', label: 'Outro', icon: '🔹' },
];

export default function BankAccountModal({
  isOpen,
  onClose,
  onSave,
  account,
  profiles,
}: BankAccountModalProps) {
  const [formData, setFormData] = useState({
    profileId: '',
    name: '',
    type: 'CHECKING' as AccountType,
    initialBalance: 0,
    currency: 'BRL',
    bankName: '',
    bankCode: '',
    agency: '',
    accountNumber: '',
    accountDigit: '',
    color: '#10b981',
    icon: '🏦',
    description: '',
    creditLimit: undefined as number | undefined,
    dueDay: undefined as number | undefined,
    closingDay: undefined as number | undefined,
  });

  useEffect(() => {
    if (account) {
      setFormData({
        profileId: account.profileId,
        name: account.name,
        type: account.type,
        initialBalance: account.initialBalance,
        currency: account.currency,
        bankName: account.bankName || '',
        bankCode: account.bankCode || '',
        agency: account.agency || '',
        accountNumber: account.accountNumber || '',
        accountDigit: account.accountDigit || '',
        color: account.color || '#10b981',
        icon: account.icon || '🏦',
        description: account.description || '',
        creditLimit: account.creditLimit,
        dueDay: account.dueDay,
        closingDay: account.closingDay,
      });
    } else {
      setFormData({
        profileId: profiles[0]?.id || '',
        name: '',
        type: 'CHECKING',
        initialBalance: 0,
        currency: 'BRL',
        bankName: '',
        bankCode: '',
        agency: '',
        accountNumber: '',
        accountDigit: '',
        color: '#10b981',
        icon: '🏦',
        description: '',
        creditLimit: undefined,
        dueDay: undefined,
        closingDay: undefined,
      });
    }
  }, [account, profiles, isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
    onClose();
  };

  if (!isOpen) return null;

  const isCreditCard = formData.type === 'CREDIT_CARD';

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-gradient-to-br from-emerald-900/95 via-teal-900/95 to-cyan-900/95 rounded-2xl p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto border border-white/20 shadow-2xl">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">
            {account ? 'Editar Conta Bancária' : 'Nova Conta Bancária'}
          </h2>
          <button
            onClick={onClose}
            className="text-white/70 hover:text-white transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Profile Selection */}
          <div>
            <label className="block text-white/90 font-semibold mb-2">Perfil Financeiro</label>
            <select
              value={formData.profileId}
              onChange={(e) => setFormData({ ...formData, profileId: e.target.value })}
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
              required
            >
              {profiles.map((profile) => (
                <option key={profile.id} value={profile.id} className="bg-gray-900">
                  {profile.name}
                </option>
              ))}
            </select>
          </div>

          {/* Account Type */}
          <div>
            <label className="block text-white/90 font-semibold mb-2">Tipo de Conta</label>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {accountTypes.map((type) => (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => setFormData({ ...formData, type: type.value, icon: type.icon })}
                  className={`flex items-center gap-2 px-4 py-3 rounded-lg transition-all border ${
                    formData.type === type.value
                      ? 'bg-white/20 border-white/40 text-white'
                      : 'bg-white/5 border-white/10 text-white/70 hover:bg-white/10'
                  }`}
                >
                  <span className="text-xl">{type.icon}</span>
                  <span className="text-sm font-semibold">{type.label}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Account Name */}
          <div>
            <label className="block text-white/90 font-semibold mb-2">Nome da Conta</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="Ex: Nubank, Conta Corrente BB, Investimentos XP"
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              required
            />
          </div>

          {/* Initial Balance */}
          <div>
            <label className="block text-white/90 font-semibold mb-2">Saldo Inicial</label>
            <input
              type="number"
              step="0.01"
              value={formData.initialBalance}
              onChange={(e) => setFormData({ ...formData, initialBalance: parseFloat(e.target.value) || 0 })}
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              required
            />
          </div>

          {/* Bank Details */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-white/90 font-semibold mb-2">Banco</label>
              <input
                type="text"
                value={formData.bankName}
                onChange={(e) => setFormData({ ...formData, bankName: e.target.value })}
                placeholder="Ex: Nubank, Banco do Brasil"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
            <div>
              <label className="block text-white/90 font-semibold mb-2">Código do Banco</label>
              <input
                type="text"
                value={formData.bankCode}
                onChange={(e) => setFormData({ ...formData, bankCode: e.target.value })}
                placeholder="Ex: 260, 001"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
          </div>

          {/* Account Details */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-white/90 font-semibold mb-2">Agência</label>
              <input
                type="text"
                value={formData.agency}
                onChange={(e) => setFormData({ ...formData, agency: e.target.value })}
                placeholder="0001"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
            <div>
              <label className="block text-white/90 font-semibold mb-2">Conta</label>
              <input
                type="text"
                value={formData.accountNumber}
                onChange={(e) => setFormData({ ...formData, accountNumber: e.target.value })}
                placeholder="123456"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
            <div>
              <label className="block text-white/90 font-semibold mb-2">Dígito</label>
              <input
                type="text"
                value={formData.accountDigit}
                onChange={(e) => setFormData({ ...formData, accountDigit: e.target.value })}
                placeholder="7"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
          </div>

          {/* Credit Card Specific Fields */}
          {isCreditCard && (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-white/90 font-semibold mb-2">Limite</label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.creditLimit || ''}
                  onChange={(e) => setFormData({ ...formData, creditLimit: parseFloat(e.target.value) || undefined })}
                  className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block text-white/90 font-semibold mb-2">Dia Vencimento</label>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={formData.dueDay || ''}
                  onChange={(e) => setFormData({ ...formData, dueDay: parseInt(e.target.value) || undefined })}
                  className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block text-white/90 font-semibold mb-2">Dia Fechamento</label>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={formData.closingDay || ''}
                  onChange={(e) => setFormData({ ...formData, closingDay: parseInt(e.target.value) || undefined })}
                  className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>
            </div>
          )}

          {/* Color and Description */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label className="block text-white/90 font-semibold mb-2">Cor</label>
              <input
                type="color"
                value={formData.color}
                onChange={(e) => setFormData({ ...formData, color: e.target.value })}
                className="w-full h-12 bg-white/10 border border-white/20 rounded-lg cursor-pointer"
              />
            </div>
            <div className="md:col-span-3">
              <label className="block text-white/90 font-semibold mb-2">Descrição (opcional)</label>
              <input
                type="text"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="Informações adicionais sobre a conta"
                className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
          </div>

          {/* Buttons */}
          <div className="flex gap-4 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-6 py-3 bg-white/10 hover:bg-white/20 text-white rounded-xl font-semibold transition-all duration-300 border border-white/20"
            >
              Cancelar
            </button>
            <button
              type="submit"
              className="flex-1 px-6 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl"
            >
              {account ? 'Atualizar' : 'Criar'} Conta
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
