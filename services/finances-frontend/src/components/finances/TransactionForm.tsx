'use client';

import { useEffect, useMemo, useState } from 'react';
import { API_BASE } from '@/lib/api';
import type {
  BankAccount,
  Category,
  CategoryType,
  Transaction,
  TransactionFormData,
  TransactionStatus,
  TransactionType,
} from '@/types/finances';

interface TransactionFormProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (payload: TransactionFormData) => Promise<void> | void;
  onUpdate?: (id: string, payload: TransactionFormData) => Promise<void> | void;
  accounts: BankAccount[];
  categories: Category[];
  defaultProfileId: string;
  loading?: boolean;
  profiles: { id: string; name: string }[];
  editingTransaction?: Transaction | null;
}

const transactionTypes: { value: TransactionType; label: string; icon: string; description: string }[] = [
  { value: 'INCOME', label: 'Receita', icon: '💰', description: 'Entradas de dinheiro' },
  { value: 'EXPENSE', label: 'Despesa', icon: '💸', description: 'Saídas e custos' },
  { value: 'TRANSFER', label: 'Transferência', icon: '🔁', description: 'Entre contas' },
];

const typeToCategory: Record<TransactionType, CategoryType> = {
  INCOME: 'INCOME',
  EXPENSE: 'EXPENSE',
  TRANSFER: 'TRANSFER',
};

// Get local date in YYYY-MM-DD format (without timezone conversion)
const getLocalDateString = () => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const defaultForm = (profileId: string, status: TransactionStatus = 'CONFIRMED'): TransactionFormData => ({
  profileId,
  bankAccountId: '',
  destinationAccountId: undefined,
  categoryId: undefined,
  destinationCategoryId: undefined,
  type: 'EXPENSE',
  status,
  amount: 0,
  currency: 'BRL',
  description: '',
  notes: undefined,
  costCenter: undefined,
  occurredOn: getLocalDateString(),
  dueOn: undefined,
  reminderOn: undefined,
  recurrenceRule: undefined,
  installmentNumber: undefined,
  installmentTotal: undefined,
  externalId: undefined,
  tags: [],
});

export default function TransactionForm({
  isOpen,
  onClose,
  onSave,
  onUpdate,
  accounts,
  categories,
  defaultProfileId,
  loading = false,
  profiles,
  editingTransaction,
}: TransactionFormProps) {
  const [formData, setFormData] = useState<TransactionFormData>(() => defaultForm(defaultProfileId));
  const [tagsInput, setTagsInput] = useState('');
  const [selectedProfileId, setSelectedProfileId] = useState<string>(defaultProfileId);
  const [destProfileId, setDestProfileId] = useState<string>(defaultProfileId);
  const [localCategories, setLocalCategories] = useState<Category[]>(categories);
  const [destCategories, setDestCategories] = useState<Category[]>([]);

  const isEditing = !!editingTransaction;

  useEffect(() => {
    if (isOpen) {
      setLocalCategories(categories);

      if (editingTransaction) {
        // Editing mode - populate form with transaction data
        setSelectedProfileId(editingTransaction.profileId);
        setFormData({
          profileId: editingTransaction.profileId,
          bankAccountId: editingTransaction.bankAccountId,
          destinationAccountId: editingTransaction.destinationAccountId,
          categoryId: editingTransaction.categoryId,
          type: editingTransaction.type,
          status: editingTransaction.status,
          amount: editingTransaction.amount,
          currency: editingTransaction.currency,
          description: editingTransaction.description,
          notes: editingTransaction.notes,
          costCenter: editingTransaction.costCenter,
          occurredOn: editingTransaction.occurredOn.slice(0, 10),
          dueOn: editingTransaction.dueOn?.slice(0, 10),
          reminderOn: editingTransaction.reminderOn?.slice(0, 10),
          recurrenceRule: editingTransaction.recurrenceRule,
          installmentNumber: editingTransaction.installmentNumber,
          installmentTotal: editingTransaction.installmentTotal,
          externalId: editingTransaction.externalId,
          tags: editingTransaction.tags || [],
        });
        setTagsInput((editingTransaction.tags || []).join(', '));
      } else {
        // Create mode - use defaults (CONFIRMED by default for quick daily transactions)
        setSelectedProfileId(defaultProfileId);
        setDestProfileId(defaultProfileId);
        setDestCategories([]);
        const initialAccount = accounts.find((account) => account.profileId === defaultProfileId)?.id || '';
        setFormData({
          ...defaultForm(defaultProfileId, 'CONFIRMED'),
          bankAccountId: initialAccount,
        });
        setTagsInput('');
      }
    }
  }, [isOpen, defaultProfileId, accounts, categories, editingTransaction]);

  const availableCategories = useMemo(() => {
    // For cross-profile transfers, source side uses EXPENSE categories
    const expectedType = (formData.type === 'TRANSFER' && destProfileId !== selectedProfileId)
      ? 'EXPENSE'
      : typeToCategory[formData.type];
    return localCategories.filter((category) => category.type === expectedType);
  }, [localCategories, formData.type, destProfileId, selectedProfileId]);

  // Organize categories hierarchically
  const hierarchicalCategories = useMemo(() => {
    const parents = availableCategories.filter((c) => !c.parentId);
    return parents.map((parent) => ({
      ...parent,
      children: availableCategories.filter((c) => c.parentId === parent.id),
    }));
  }, [availableCategories]);

  const isCrossProfile = formData.type === 'TRANSFER' && destProfileId !== selectedProfileId;

  const destinationOptions = useMemo(() => {
    if (formData.type !== 'TRANSFER') return [];
    return accounts.filter(
      (account) => account.profileId === destProfileId && account.id !== formData.bankAccountId,
    );
  }, [accounts, destProfileId, formData.type, formData.bankAccountId]);

  // Categories for destination profile (cross-profile only, INCOME type)
  const destIncomeCategories = useMemo(() => {
    if (!isCrossProfile) return [];
    return destCategories.filter((c) => c.type === 'INCOME');
  }, [isCrossProfile, destCategories]);

  const destHierarchicalCategories = useMemo(() => {
    const parents = destIncomeCategories.filter((c) => !c.parentId);
    return parents.map((parent) => ({
      ...parent,
      children: destIncomeCategories.filter((c) => c.parentId === parent.id),
    }));
  }, [destIncomeCategories]);

  const accountsForProfile = useMemo(
    () => accounts.filter((a) => a.profileId === selectedProfileId),
    [accounts, selectedProfileId],
  );

  const onChangeProfile = async (profileId: string) => {
    setSelectedProfileId(profileId);
    const newInitialAccount = accounts.find((a) => a.profileId === profileId)?.id || '';
    setFormData((prev) => ({
      ...prev,
      profileId,
      bankAccountId: newInitialAccount,
      destinationAccountId: undefined,
      categoryId: undefined,
    }));
    // fetch categories for selected profile
    try {
      const res = await fetch(`${API_BASE}/categories?profileId=${profileId}`);
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

  const onChangeDestProfile = async (profileId: string) => {
    setDestProfileId(profileId);
    setFormData((prev) => ({
      ...prev,
      destinationAccountId: undefined,
      destinationCategoryId: undefined,
    }));
    if (profileId !== selectedProfileId) {
      try {
        const res = await fetch(`${API_BASE}/categories?profileId=${profileId}`);
        if (res.ok) {
          const data = await res.json();
          setDestCategories(data.data || []);
        } else {
          setDestCategories([]);
        }
      } catch {
        setDestCategories([]);
      }
    } else {
      setDestCategories([]);
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const payload: TransactionFormData = {
      ...formData,
      amount: Number(formData.amount),
      tags: tagsInput
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean),
      destinationAccountId:
        formData.type === 'TRANSFER' ? formData.destinationAccountId : undefined,
      destinationCategoryId:
        isCrossProfile ? formData.destinationCategoryId : undefined,
      categoryId:
        isCrossProfile ? formData.categoryId : (formData.type === 'TRANSFER' ? formData.categoryId ?? availableCategories[0]?.id : formData.categoryId),
      // Only include installment fields if they have valid positive values
      installmentNumber: formData.installmentNumber && formData.installmentNumber > 0 ? formData.installmentNumber : undefined,
      installmentTotal: formData.installmentTotal && formData.installmentTotal > 0 ? formData.installmentTotal : undefined,
    };

    if (isEditing && onUpdate && editingTransaction) {
      await Promise.resolve(onUpdate(editingTransaction.id, payload));
    } else {
      await Promise.resolve(onSave(payload));
    }
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      <div className="relative w-full max-w-5xl bg-gradient-to-br from-emerald-900/95 via-teal-900/95 to-cyan-900/95 border border-white/10 rounded-2xl shadow-2xl p-8 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-white">
              {isEditing ? 'Editar Lançamento' : 'Novo Lançamento'}
            </h2>
            <p className="text-white/60 text-sm">
              {isEditing ? 'Altere os dados do lançamento' : 'Registre entradas, despesas e transferências'}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-white/70 hover:text-white transition-colors"
            type="button"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="grid gap-6 lg:grid-cols-2">
            <div className="space-y-4">
              <div>
                <label className="block text-white/80 text-sm font-semibold mb-2">Perfil</label>
                <select
                  value={selectedProfileId}
                  onChange={(e) => onChangeProfile(e.target.value)}
                  className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  {profiles.map((p) => (
                    <option key={p.id} value={p.id} className="bg-slate-900">{p.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-white/80 text-sm font-semibold mb-2">Tipo de lancamento</label>
                <div className="grid grid-cols-3 gap-3">
                  {transactionTypes.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() => {
                        setFormData((prev) => ({
                          ...prev,
                          type: option.value,
                          destinationAccountId: undefined,
                          categoryId: undefined,
                          destinationCategoryId: undefined,
                        }));
                        setDestProfileId(selectedProfileId);
                        setDestCategories([]);
                      }
                      }
                      className={`flex items-center gap-3 px-4 py-3 rounded-xl border transition-all ${
                        formData.type === option.value
                          ? 'bg-white/15 border-white/40 text-white'
                          : 'bg-white/5 border-white/10 text-white/70 hover:bg-white/10'
                      }`}
                    >
                      <span className="text-xl">{option.icon}</span>
                      <div className="text-left">
                        <p className="text-sm font-semibold">{option.label}</p>
                        <p className="text-xs text-white/50">{option.description}</p>
                      </div>
                    </button>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Conta de origem</label>
                  <select
                    value={formData.bankAccountId}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        bankAccountId: event.target.value,
                      }))
                    }
                    required
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  >
                    <option value="">Selecione uma conta</option>
                    {accountsForProfile.map((account) => (
                        <option key={account.id} value={account.id} className="bg-slate-900">
                          {account.name}
                        </option>
                      ))}
                  </select>
                </div>

                {formData.type === 'TRANSFER' && (
                  <>
                    <div>
                      <label className="block text-white/80 text-sm font-semibold mb-2">Perfil de destino</label>
                      <select
                        value={destProfileId}
                        onChange={(e) => onChangeDestProfile(e.target.value)}
                        className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                      >
                        {profiles.map((p) => (
                          <option key={p.id} value={p.id} className="bg-slate-900">{p.name}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-white/80 text-sm font-semibold mb-2">Conta de destino</label>
                      <select
                        value={formData.destinationAccountId ?? ''}
                        onChange={(event) =>
                          setFormData((prev) => ({
                            ...prev,
                            destinationAccountId: event.target.value || undefined,
                          }))
                        }
                        required
                        className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                      >
                        <option value="">Selecione uma conta</option>
                        {destinationOptions.map((account) => (
                          <option key={account.id} value={account.id} className="bg-slate-900">
                            {account.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  </>
                )}
              </div>

              {formData.type !== 'TRANSFER' && (
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Categoria</label>
                  <select
                    value={formData.categoryId ?? ''}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        categoryId: event.target.value || undefined,
                      }))
                    }
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  >
                    <option value="">Selecione uma categoria</option>
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
              )}

              {isCrossProfile && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-white/80 text-sm font-semibold mb-2">
                      Categoria de saída <span className="text-white/50 text-xs">(origem)</span>
                    </label>
                    <select
                      value={formData.categoryId ?? ''}
                      onChange={(event) =>
                        setFormData((prev) => ({
                          ...prev,
                          categoryId: event.target.value || undefined,
                        }))
                      }
                      required
                      className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    >
                      <option value="">Selecione uma categoria</option>
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
                    <label className="block text-white/80 text-sm font-semibold mb-2">
                      Categoria de entrada <span className="text-white/50 text-xs">(destino)</span>
                    </label>
                    <select
                      value={formData.destinationCategoryId ?? ''}
                      onChange={(event) =>
                        setFormData((prev) => ({
                          ...prev,
                          destinationCategoryId: event.target.value || undefined,
                        }))
                      }
                      required
                      className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    >
                      <option value="">Selecione uma categoria</option>
                      {destHierarchicalCategories.map((parent) => (
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
                </div>
              )}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Valor</label>
                  <input
                    type="number"
                    inputMode="decimal"
                    step="0.01"
                    min="0"
                    value={formData.amount || ''}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        amount: Number(event.target.value) || 0,
                      }))
                    }
                    required
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="0,00"
                  />
                </div>
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Data</label>
                  <input
                    type="date"
                    value={formData.occurredOn}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        occurredOn: event.target.value,
                      }))
                    }
                    required
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-white/80 text-sm font-semibold mb-2">Descrição</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(event) =>
                    setFormData((prev) => ({
                      ...prev,
                      description: event.target.value,
                    }))
                  }
                  required
                  className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="Descreva o lançamento"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Centro de custo</label>
                  <input
                    type="text"
                    value={formData.costCenter ?? ''}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        costCenter: event.target.value || undefined,
                      }))
                    }
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="Opcional"
                  />
                </div>
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Vencimento</label>
                  <input
                    type="date"
                    value={formData.dueOn ?? ''}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        dueOn: event.target.value || undefined,
                      }))
                    }
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>
              </div>

              {/* Reminder field - only show for PLANNED transactions */}
              {formData.status === 'PLANNED' && (
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">
                    🔔 Lembrar em
                  </label>
                  <input
                    type="date"
                    value={formData.reminderOn ?? ''}
                    onChange={(event) =>
                      setFormData((prev) => ({
                        ...prev,
                        reminderOn: event.target.value || undefined,
                      }))
                    }
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                  <p className="text-white/50 text-xs mt-1">
                    Alertas serao exibidos 10, 5, 1 dia antes e no dia do lembrete
                  </p>
                </div>
              )}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Parcelas</label>
                  <div className="grid grid-cols-2 gap-3">
                    <input
                      type="number"
                      min="1"
                      value={formData.installmentNumber ?? ''}
                      onChange={(event) =>
                        setFormData((prev) => ({
                          ...prev,
                          installmentNumber: event.target.value
                            ? Number(event.target.value)
                            : undefined,
                        }))
                      }
                      className="px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                      placeholder="Parcela"
                    />
                    <input
                      type="number"
                      min="1"
                      value={formData.installmentTotal ?? ''}
                      onChange={(event) =>
                        setFormData((prev) => ({
                          ...prev,
                          installmentTotal: event.target.value
                            ? Number(event.target.value)
                            : undefined,
                        }))
                      }
                      className="px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                      placeholder="Total"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-white/80 text-sm font-semibold mb-2">Tags</label>
                  <input
                    type="text"
                    value={tagsInput}
                    onChange={(event) => setTagsInput(event.target.value)}
                    className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="Separadas por vírgula"
                  />
                </div>
              </div>

              <div>
                <label className="block text-white/80 text-sm font-semibold mb-2">Observações</label>
                <textarea
                  value={formData.notes ?? ''}
                  onChange={(event) =>
                    setFormData((prev) => ({
                      ...prev,
                      notes: event.target.value || undefined,
                    }))
                  }
                  rows={4}
                  className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="Detalhes adicionais"
                />
              </div>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row gap-4 items-center justify-between pt-4 border-t border-white/10">
            <label className="flex items-center gap-3 cursor-pointer group">
              <input
                type="checkbox"
                checked={formData.status === 'CONFIRMED'}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    status: e.target.checked ? 'CONFIRMED' : 'PLANNED',
                  }))
                }
                className="w-5 h-5 rounded border-2 border-white/30 bg-white/10 text-emerald-500 focus:ring-emerald-500 focus:ring-offset-0 cursor-pointer"
              />
              <div>
                <span className="text-white/90 text-sm font-medium group-hover:text-white transition-colors">
                  {formData.status === 'CONFIRMED' ? '✅ Já confirmado' : '🗓️ Planejado (aguardando confirmação)'}
                </span>
                <p className="text-white/50 text-xs">
                  {formData.status === 'CONFIRMED'
                    ? 'O lançamento já ocorreu e será registrado como efetivado'
                    : 'O lançamento será salvo como pendente para confirmar depois'}
                </p>
              </div>
            </label>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="px-6 py-3 rounded-lg border border-white/20 text-white/80 hover:text-white hover:bg-white/10 transition-colors"
              >
                Cancelar
              </button>
              <button
                type="submit"
                disabled={loading}
                className="px-6 py-3 rounded-lg bg-emerald-500/80 hover:bg-emerald-500 text-white font-semibold transition-colors disabled:opacity-50"
              >
                {loading ? 'Salvando...' : 'Salvar lançamento'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
