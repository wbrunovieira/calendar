'use client';

import { useState, useEffect } from 'react';
import { AccountType, BankAccount, InvestmentType, YieldType } from '@/types/finances';

export type { AccountType } from '@/types/finances';

const investmentTypes: { value: InvestmentType; label: string; supportsQuotas: boolean }[] = [
  { value: 'SAVINGS_BOX', label: 'Caixinha', supportsQuotas: false },
  { value: 'CDB', label: 'CDB', supportsQuotas: false },
  { value: 'LCI', label: 'LCI', supportsQuotas: false },
  { value: 'LCA', label: 'LCA', supportsQuotas: false },
  { value: 'TREASURY', label: 'Tesouro Direto', supportsQuotas: false },
  { value: 'STOCKS', label: 'Ações', supportsQuotas: true },
  { value: 'FII', label: 'FII', supportsQuotas: true },
  { value: 'FUNDS', label: 'Fundos', supportsQuotas: true },
  { value: 'CRYPTO', label: 'Crypto', supportsQuotas: true },
  { value: 'OTHER', label: 'Outro', supportsQuotas: false },
];

const yieldTypes: { value: YieldType; label: string; description: string }[] = [
  { value: 'CDI_PERCENTAGE', label: '% do CDI', description: 'Ex: 100% do CDI' },
  { value: 'FIXED', label: 'Taxa Fixa', description: 'Ex: 12% a.a.' },
  { value: 'IPCA_PLUS', label: 'IPCA +', description: 'Ex: IPCA + 5%' },
  { value: 'VARIABLE', label: 'Variável', description: 'Ações, fundos, crypto' },
];

interface BankAccountModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (account: Omit<BankAccount, 'id' | 'isActive' | 'createdAt' | 'updatedAt'>) => void;
  account: BankAccount | null;
  profiles: Array<{ id: string; name: string }>;
  existingAccounts?: BankAccount[];
}

const accountTypes: { value: AccountType; label: string; icon: string }[] = [
  { value: 'CHECKING', label: 'Conta Corrente', icon: '🏦' },
  { value: 'SAVINGS', label: 'Conta Poupança', icon: '💰' },
  { value: 'INVESTMENT', label: 'Investimentos', icon: '📈' },
  { value: 'CREDIT_CARD', label: 'Cartão de Crédito', icon: '💳' },
  { value: 'EXCHANGE', label: 'Corretora Cripto', icon: '🪙' },
  { value: 'WALLET', label: 'Carteira Cripto', icon: '🔐' },
  { value: 'CASH', label: 'Dinheiro', icon: '💵' },
  { value: 'OTHER', label: 'Outro', icon: '🔹' },
];

export default function BankAccountModal({
  isOpen,
  onClose,
  onSave,
  account,
  profiles,
  existingAccounts = [],
}: BankAccountModalProps) {
  const [formData, setFormData] = useState({
    profileId: '',
    name: '',
    type: 'CHECKING' as AccountType,
    initialBalance: 0,
    currentBalance: 0,
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
    linkedAccountId: undefined as string | undefined,
    initialInvoiceAmount: undefined as number | undefined,
    // Investment fields
    investmentType: undefined as InvestmentType | undefined,
    yieldType: undefined as YieldType | undefined,
    yieldRate: undefined as number | undefined,
    maturityDate: undefined as string | undefined,
    broker: '',
    numberOfQuotas: undefined as number | undefined,
    quotaPrice: undefined as number | undefined,
  });

  // Quota input mode: 'total' = enter total value, 'price' = enter price per quota
  const [quotaInputMode, setQuotaInputMode] = useState<'total' | 'price'>('total');

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    if (account) {
      setFormData({
        profileId: account.profileId,
        name: account.name,
        type: account.type,
        initialBalance: account.initialBalance,
        currentBalance: account.currentBalance,
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
        linkedAccountId: account.linkedAccountId,
        initialInvoiceAmount: undefined,
        investmentType: account.investmentType,
        yieldType: account.yieldType,
        yieldRate: account.yieldRate,
        maturityDate: account.maturityDate,
        broker: account.broker || '',
        numberOfQuotas: account.numberOfQuotas,
        quotaPrice: account.quotaPrice,
      });
      // If editing and has quotas, default to price mode
      if (account.numberOfQuotas && account.quotaPrice) {
        setQuotaInputMode('price');
      }
    } else {
      setFormData({
        profileId: profiles[0]?.id || '',
        name: '',
        type: 'CHECKING',
        initialBalance: 0,
        currentBalance: 0,
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
        linkedAccountId: undefined,
        initialInvoiceAmount: undefined,
        investmentType: undefined,
        yieldType: undefined,
        yieldRate: undefined,
        maturityDate: undefined,
        broker: '',
        numberOfQuotas: undefined,
        quotaPrice: undefined,
      });
      setQuotaInputMode('total');
    }
  }, [account, profiles, isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
    onClose();
  };

  if (!isOpen) return null;

  const isCreditCard = formData.type === 'CREDIT_CARD';
  const isInvestment = formData.type === 'INVESTMENT';

  // Filter linkable accounts: same profile, valid link targets (CHECKING, SAVINGS, CASH), exclude self
  const linkableAccounts = existingAccounts.filter(
    (acc) =>
      acc.profileId === formData.profileId &&
      (acc.type === 'CHECKING' || acc.type === 'SAVINGS' || acc.type === 'CASH') &&
      acc.id !== account?.id
  );

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-gradient-to-br from-emerald-900/95 via-teal-900/95 to-cyan-900/95 rounded-2xl p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto border border-white/20 shadow-2xl">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">
            {account
              ? (isCreditCard ? 'Editar Cartão de Crédito' : isInvestment ? 'Editar Investimento' : 'Editar Conta Bancária')
              : (isCreditCard ? 'Novo Cartão de Crédito' : isInvestment ? 'Novo Investimento' : 'Nova Conta Bancária')}
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
            <label className="block text-white/90 font-semibold mb-2">
              {isCreditCard ? 'Nome do Cartão' : 'Nome da Conta'}
            </label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder={isCreditCard ? 'Ex: Nubank Ultravioleta, Cartão Mercado Pago' : 'Ex: Nubank, Conta Corrente BB'}
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              required
            />
          </div>

          {/* Balance Fields - Only for non-credit cards */}
          {!isCreditCard && (
            <>
              {account ? (
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Saldo Atual</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.currentBalance || ''}
                    onChange={(e) => setFormData({ ...formData, currentBalance: parseFloat(e.target.value) || 0 })}
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="0,00"
                  />
                  <p className="text-white/50 text-sm mt-1">Saldo inicial: R$ {formData.initialBalance.toFixed(2)}</p>
                </div>
              ) : (
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Saldo Inicial</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.initialBalance || ''}
                    onChange={(e) => {
                      const value = parseFloat(e.target.value) || 0;
                      setFormData({ ...formData, initialBalance: value, currentBalance: value });
                    }}
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="0,00"
                  />
                </div>
              )}
            </>
          )}

          {/* Bank Details - Only for non-credit cards */}
          {!isCreditCard && (
            <>
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
            </>
          )}

          {/* Credit Card Specific Fields */}
          {isCreditCard && (
            <>
              {/* Linked Account Selector */}
              {linkableAccounts.length > 0 && (
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Conta Vinculada (opcional)</label>
                  <select
                    value={formData.linkedAccountId || ''}
                    onChange={(e) => setFormData({ ...formData, linkedAccountId: e.target.value || undefined })}
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  >
                    <option value="" className="bg-gray-900">Nenhuma (cartão independente)</option>
                    {linkableAccounts.map((acc) => (
                      <option key={acc.id} value={acc.id} className="bg-gray-900">
                        {acc.icon} {acc.name} {acc.bankName ? `(${acc.bankName})` : ''}
                      </option>
                    ))}
                  </select>
                  <p className="text-white/50 text-sm mt-1">Vincule este cartão a uma conta bancária para agrupá-los</p>
                </div>
              )}

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Limite</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.creditLimit || ''}
                    onChange={(e) => setFormData({ ...formData, creditLimit: parseFloat(e.target.value) || undefined })}
                    placeholder="Ex: 2600.00"
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>
                {!account && (
                  <div>
                    <label className="block text-white/90 font-semibold mb-2">Fatura Atual (opcional)</label>
                    <input
                      type="number"
                      step="0.01"
                      value={formData.initialInvoiceAmount || ''}
                      onChange={(e) => setFormData({ ...formData, initialInvoiceAmount: parseFloat(e.target.value) || undefined })}
                      placeholder="Ex: 760.09"
                      className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    />
                    <p className="text-white/50 text-sm mt-1">Valor da fatura atual para importar</p>
                  </div>
                )}
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Dia Fechamento</label>
                  <input
                    type="number"
                    min="1"
                    max="31"
                    value={formData.closingDay || ''}
                    onChange={(e) => setFormData({ ...formData, closingDay: parseInt(e.target.value) || undefined })}
                    placeholder="Ex: 9"
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
                    placeholder="Ex: 14"
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>
              </div>
            </>
          )}

          {/* Investment Specific Fields */}
          {isInvestment && (
            <>
              {/* Linked Account Selector for Investments */}
              {linkableAccounts.length > 0 && (
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Conta de Origem</label>
                  <select
                    value={formData.linkedAccountId || ''}
                    onChange={(e) => setFormData({ ...formData, linkedAccountId: e.target.value || undefined })}
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  >
                    <option value="" className="bg-gray-900">Selecione a conta de origem</option>
                    {linkableAccounts.map((acc) => (
                      <option key={acc.id} value={acc.id} className="bg-gray-900">
                        {acc.icon} {acc.name} {acc.bankName ? `(${acc.bankName})` : ''}
                      </option>
                    ))}
                  </select>
                  <p className="text-white/50 text-sm mt-1">Conta de onde saiu o dinheiro para este investimento</p>
                </div>
              )}

              {/* Investment Type */}
              <div>
                <label className="block text-white/90 font-semibold mb-2">Tipo de Investimento</label>
                <div className="grid grid-cols-3 md:grid-cols-5 gap-2">
                  {investmentTypes.map((type) => (
                    <button
                      key={type.value}
                      type="button"
                      onClick={() => setFormData({ ...formData, investmentType: type.value })}
                      className={`px-3 py-2 rounded-lg transition-all border text-sm ${
                        formData.investmentType === type.value
                          ? 'bg-white/20 border-white/40 text-white'
                          : 'bg-white/5 border-white/10 text-white/70 hover:bg-white/10'
                      }`}
                    >
                      {type.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Yield Type */}
              <div>
                <label className="block text-white/90 font-semibold mb-2">Tipo de Rendimento</label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                  {yieldTypes.map((type) => (
                    <button
                      key={type.value}
                      type="button"
                      onClick={() => setFormData({ ...formData, yieldType: type.value })}
                      className={`px-3 py-2 rounded-lg transition-all border text-sm ${
                        formData.yieldType === type.value
                          ? 'bg-white/20 border-white/40 text-white'
                          : 'bg-white/5 border-white/10 text-white/70 hover:bg-white/10'
                      }`}
                    >
                      <div className="font-semibold">{type.label}</div>
                      <div className="text-xs text-white/50">{type.description}</div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Yield Rate - only for non-variable */}
              {formData.yieldType && formData.yieldType !== 'VARIABLE' && (
                <div>
                  <label className="block text-white/90 font-semibold mb-2">
                    {formData.yieldType === 'CDI_PERCENTAGE' ? 'Percentual do CDI' :
                     formData.yieldType === 'IPCA_PLUS' ? 'Taxa adicional (IPCA +)' : 'Taxa anual (% a.a.)'}
                  </label>
                  <div className="relative">
                    <input
                      type="number"
                      step="0.01"
                      value={formData.yieldRate || ''}
                      onChange={(e) => setFormData({ ...formData, yieldRate: parseFloat(e.target.value) || undefined })}
                      placeholder={formData.yieldType === 'CDI_PERCENTAGE' ? 'Ex: 100' : 'Ex: 12.5'}
                      className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    />
                    <span className="absolute right-4 top-1/2 -translate-y-1/2 text-white/50">%</span>
                  </div>
                </div>
              )}

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Broker/Platform */}
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Corretora/Plataforma</label>
                  <input
                    type="text"
                    value={formData.broker}
                    onChange={(e) => setFormData({ ...formData, broker: e.target.value })}
                    placeholder="Ex: Nubank, XP, Clear"
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>

                {/* Maturity Date - for fixed-term investments */}
                <div>
                  <label className="block text-white/90 font-semibold mb-2">Vencimento (opcional)</label>
                  <input
                    type="date"
                    value={formData.maturityDate || ''}
                    onChange={(e) => setFormData({ ...formData, maturityDate: e.target.value || undefined })}
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                  <p className="text-white/50 text-sm mt-1">Para investimentos com prazo definido (CDB, LCI, etc.)</p>
                </div>
              </div>

              {/* Quota fields - only for stocks, FII, funds, crypto */}
              {formData.investmentType && investmentTypes.find(t => t.value === formData.investmentType)?.supportsQuotas && (
                <div className="space-y-4 p-4 bg-white/5 rounded-lg border border-white/10">
                  <div className="flex items-center justify-between">
                    <label className="text-white/90 font-semibold">Cotas/Ações</label>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setQuotaInputMode('total')}
                        className={`px-3 py-1 rounded-lg text-xs transition-colors ${
                          quotaInputMode === 'total'
                            ? 'bg-emerald-500/30 text-emerald-300 border border-emerald-500/50'
                            : 'bg-white/5 text-white/50 border border-white/10 hover:bg-white/10'
                        }`}
                      >
                        Valor Total
                      </button>
                      <button
                        type="button"
                        onClick={() => setQuotaInputMode('price')}
                        className={`px-3 py-1 rounded-lg text-xs transition-colors ${
                          quotaInputMode === 'price'
                            ? 'bg-emerald-500/30 text-emerald-300 border border-emerald-500/50'
                            : 'bg-white/5 text-white/50 border border-white/10 hover:bg-white/10'
                        }`}
                      >
                        Preço por Cota
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-white/70 text-sm mb-1">Quantidade</label>
                      <input
                        type="number"
                        step="0.000001"
                        value={formData.numberOfQuotas || ''}
                        onChange={(e) => {
                          const qty = parseFloat(e.target.value) || undefined;
                          if (quotaInputMode === 'total' && qty && formData.initialBalance > 0) {
                            // Calculate price from total
                            const price = formData.initialBalance / qty;
                            setFormData({ ...formData, numberOfQuotas: qty, quotaPrice: price });
                          } else if (quotaInputMode === 'price' && qty && formData.quotaPrice) {
                            // Calculate total from price
                            const total = qty * formData.quotaPrice;
                            setFormData({ ...formData, numberOfQuotas: qty, initialBalance: total, currentBalance: total });
                          } else {
                            setFormData({ ...formData, numberOfQuotas: qty });
                          }
                        }}
                        placeholder="Ex: 100"
                        className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                      />
                    </div>

                    {quotaInputMode === 'total' ? (
                      <>
                        <div>
                          <label className="block text-white/70 text-sm mb-1">Valor Total</label>
                          <input
                            type="number"
                            step="0.01"
                            value={formData.initialBalance || ''}
                            onChange={(e) => {
                              const total = parseFloat(e.target.value) || 0;
                              if (formData.numberOfQuotas && formData.numberOfQuotas > 0) {
                                const price = total / formData.numberOfQuotas;
                                setFormData({ ...formData, initialBalance: total, currentBalance: total, quotaPrice: price });
                              } else {
                                setFormData({ ...formData, initialBalance: total, currentBalance: total });
                              }
                            }}
                            placeholder="Ex: 10000.00"
                            className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                          />
                        </div>
                        <div>
                          <label className="block text-white/70 text-sm mb-1">Preço/Cota (calculado)</label>
                          <div className="px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white/70">
                            {formData.quotaPrice
                              ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(formData.quotaPrice)
                              : '-'}
                          </div>
                        </div>
                      </>
                    ) : (
                      <>
                        <div>
                          <label className="block text-white/70 text-sm mb-1">Preço por Cota</label>
                          <input
                            type="number"
                            step="0.01"
                            value={formData.quotaPrice || ''}
                            onChange={(e) => {
                              const price = parseFloat(e.target.value) || undefined;
                              if (formData.numberOfQuotas && price) {
                                const total = formData.numberOfQuotas * price;
                                setFormData({ ...formData, quotaPrice: price, initialBalance: total, currentBalance: total });
                              } else {
                                setFormData({ ...formData, quotaPrice: price });
                              }
                            }}
                            placeholder="Ex: 100.00"
                            className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                          />
                        </div>
                        <div>
                          <label className="block text-white/70 text-sm mb-1">Valor Total (calculado)</label>
                          <div className="px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white/70">
                            {formData.initialBalance > 0
                              ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(formData.initialBalance)
                              : '-'}
                          </div>
                        </div>
                      </>
                    )}
                  </div>
                  <p className="text-white/40 text-xs">
                    {quotaInputMode === 'total'
                      ? 'Informe o valor total e a quantidade de cotas. O preço por cota será calculado automaticamente.'
                      : 'Informe o preço por cota e a quantidade. O valor total será calculado automaticamente.'}
                  </p>
                </div>
              )}
            </>
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
              {account ? 'Atualizar' : 'Criar'} {isCreditCard ? 'Cartão' : isInvestment ? 'Investimento' : 'Conta'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
