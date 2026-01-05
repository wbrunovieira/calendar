'use client';

import { useEffect, useMemo, useState } from 'react';
import type { BankAccount, Category, TransactionFormData } from '@/types/finances';

interface QuickExpenseProps {
  accounts: BankAccount[]; // all accounts
  categories?: Category[]; // optional initial categories for default profile
  defaultProfileId: string;
  profiles: { id: string; name: string }[];
  onSave: (payload: TransactionFormData) => Promise<void> | void;
}

export default function QuickExpense({ accounts, categories = [], defaultProfileId, profiles, onSave }: QuickExpenseProps) {
  const API_BASE = 'http://localhost:3335/api/v1';
  const [selectedProfileId, setSelectedProfileId] = useState<string>(defaultProfileId);
  const [localCategories, setLocalCategories] = useState<Category[]>(categories);
  const expenseCategories = useMemo(() => localCategories.filter((c) => c.type === 'EXPENSE'), [localCategories]);

  // Organize categories hierarchically
  const hierarchicalCategories = useMemo(() => {
    const parents = expenseCategories.filter((c) => !c.parentId);
    return parents.map((parent) => ({
      ...parent,
      children: expenseCategories.filter((c) => c.parentId === parent.id),
    }));
  }, [expenseCategories]);
  const defaultAccountId = useMemo(() => accounts.find((a) => a.profileId === selectedProfileId)?.id ?? '', [accounts, selectedProfileId]);

  const [accountId, setAccountId] = useState<string>(defaultAccountId);
  const [categoryId, setCategoryId] = useState<string>(expenseCategories[0]?.id ?? '');
  const [amount, setAmount] = useState<string>('');
  const [description, setDescription] = useState<string>('');
  const [date, setDate] = useState<string>(() => new Date().toISOString().slice(0, 10));
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setSelectedProfileId(defaultProfileId);
    setLocalCategories(categories);
  }, [defaultProfileId, categories]);

  useEffect(() => {
    setAccountId(defaultAccountId);
  }, [defaultAccountId]);

  const accountsForProfile = useMemo(
    () => accounts.filter((a) => a.profileId === selectedProfileId),
    [accounts, selectedProfileId],
  );

  const changeProfile = async (profileId: string) => {
    setSelectedProfileId(profileId);
    try {
      const res = await fetch(`${API_BASE}/categories?profileId=${profileId}&type=EXPENSE`);
      if (res.ok) {
        const data = await res.json();
        setLocalCategories(data.data || []);
      } else {
        setLocalCategories([]);
      }
    } catch {
      setLocalCategories([]);
    }
  };

  const disabled = !defaultProfileId || !accountId || !categoryId || !amount || Number(amount) <= 0 || saving;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (disabled) return;
    setSaving(true);
    try {
      const payload: TransactionFormData = {
        profileId: selectedProfileId,
        bankAccountId: accountId,
        categoryId,
        type: 'EXPENSE',
        status: 'CONFIRMED', // Quick expenses are already confirmed (happened now)
        amount: Number(amount),
        currency: 'BRL',
        description: description || 'Despesa',
        occurredOn: date,
        notes: undefined,
        costCenter: undefined,
        dueOn: undefined,
        recurrenceRule: undefined,
        installmentNumber: undefined,
        installmentTotal: undefined,
        externalId: undefined,
        destinationAccountId: undefined,
        tags: [],
      };
      await Promise.resolve(onSave(payload));
      setAmount('');
      setDescription('');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="bg-white/5 border border-white/10 rounded-2xl p-4 flex flex-col lg:flex-row gap-3 items-stretch">
      <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-3">
        <div>
          <label className="block text-xs text-white/60 mb-1">Perfil</label>
          <select
            value={selectedProfileId}
            onChange={(e) => changeProfile(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
          >
            {profiles.map((p) => (
              <option key={p.id} value={p.id} className="bg-slate-900">{p.name}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-white/60 mb-1">Conta</label>
          <select
            value={accountId}
            onChange={(e) => setAccountId(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
          >
            <option value="">Selecione</option>
            {accountsForProfile.map((a) => (
              <option key={a.id} value={a.id} className="bg-slate-900">
                {a.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-white/60 mb-1">Categoria</label>
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
          >
            <option value="">Selecione</option>
            {hierarchicalCategories.map((parent) => (
              parent.children.length > 0 ? (
                <optgroup key={parent.id} label={parent.name} className="bg-slate-900">
                  <option value={parent.id} className="bg-slate-900">
                    {parent.name} (geral)
                  </option>
                  {parent.children.map((sub) => (
                    <option key={sub.id} value={sub.id} className="bg-slate-900">
                      ↳ {sub.name}
                    </option>
                  ))}
                </optgroup>
              ) : (
                <option key={parent.id} value={parent.id} className="bg-slate-900">
                  {parent.name}
                </option>
              )
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-white/60 mb-1">Valor</label>
          <input
            type="number"
            step="0.01"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
          />
        </div>
        <div>
          <label className="block text-xs text-white/60 mb-1">Data</label>
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
          />
        </div>
        <div className="sm:col-span-2 lg:col-span-1">
          <label className="block text-xs text-white/60 mb-1">Descrição</label>
          <input
            type="text"
            placeholder="Ex.: Almoço"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
          />
        </div>
      </div>
      <div className="flex items-end">
        <button
          type="submit"
          disabled={disabled}
          className={`px-4 py-2 rounded-lg font-semibold border transition-colors ${
            disabled ? 'bg-white/10 text-white/40 border-white/10' : 'bg-emerald-500/80 hover:bg-emerald-500 text-white border-emerald-400/40'
          }`}
        >
          Lançar
        </button>
      </div>
    </form>
  );
}
