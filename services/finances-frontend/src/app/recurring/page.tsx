'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import { useToast } from '@/components/ui/Toast';
import { api } from '@/lib/api';
import type { RecurringTransaction, BankAccount, Category, TransactionType } from '@/types/finances';
import { parseLocalDate, getLocalDateString } from '@/utils/format';

// Get date X months from now
const getDateMonthsFromNow = (months: number) => {
  const date = new Date();
  date.setMonth(date.getMonth() + months);
  return getLocalDateString(date);
};

type Frequency = 'DAILY' | 'WEEKLY' | 'MONTHLY';

interface FormData {
  type: TransactionType;
  bankAccountId: string;
  categoryId: string;
  amount: string;
  description: string;
  frequency: Frequency;
  startOn: string;
  endOn: string;
  notes: string;
}

const defaultForm = (): FormData => ({
  type: 'EXPENSE',
  bankAccountId: '',
  categoryId: '',
  amount: '',
  description: '',
  frequency: 'MONTHLY',
  startOn: getLocalDateString(),
  endOn: '',
  notes: '',
});

export default function RecurringPage() {
  // Use shared profile context
  const { selectedProfileId, selectedProfile } = useProfile();
  const { toast, confirm } = useToast();

  const [items, setItems] = useState<RecurringTransaction[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [formData, setFormData] = useState<FormData>(defaultForm());
  const [saving, setSaving] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  // Pause modal state
  const [pauseModalOpen, setPauseModalOpen] = useState(false);
  const [pausingItem, setPausingItem] = useState<RecurringTransaction | null>(null);
  const [reviewOnDate, setReviewOnDate] = useState('');

  const filteredAccounts = useMemo(
    () => accounts.filter((a) => a.profileId === selectedProfileId),
    [accounts, selectedProfileId],
  );

  const filteredCategories = useMemo(
    () => categories.filter((c) => c.type === formData.type),
    [categories, formData.type],
  );

  const loadRecurring = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      setLoading(true);
      setError(null);
      const data = await api.get<{ data: RecurringTransaction[] }>(`/recurring-transactions?profileId=${selectedProfileId}`);
      setItems(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar recorrentes', e);
      setItems([]);
      setError('Nao foi possivel carregar as transacoes recorrentes.');
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId]);

  const loadAccounts = useCallback(async () => {
    try {
      const data = await api.get<{ data: BankAccount[] }>('/bank-accounts');
      setAccounts(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar contas', e);
    }
  }, []);

  const loadCategories = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      const data = await api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}`);
      setCategories(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar categorias', e);
    }
  }, [selectedProfileId]);

  useEffect(() => {
    loadRecurring();
    loadAccounts();
    loadCategories();
  }, [loadRecurring, loadAccounts, loadCategories]);

  const openCreateModal = () => {
    setEditingId(null);
    setFormData(defaultForm());
    setModalOpen(true);
  };

  const openEditModal = (item: RecurringTransaction) => {
    setEditingId(item.id);
    const freq = item.recurrenceRule.includes('DAILY') ? 'DAILY'
      : item.recurrenceRule.includes('WEEKLY') ? 'WEEKLY' : 'MONTHLY';
    setFormData({
      type: item.type,
      bankAccountId: item.bankAccountId || '',
      categoryId: item.categoryId || '',
      amount: String(item.amount),
      description: item.description,
      frequency: freq,
      startOn: item.startOn.slice(0, 10),
      endOn: item.endOn ? item.endOn.slice(0, 10) : '',
      notes: item.notes || '',
    });
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditingId(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProfileId || !formData.description.trim() || !formData.amount) return;

    setSaving(true);
    try {
      const recurrenceRule = `FREQ=${formData.frequency}`;
      const payload = {
        profileId: selectedProfileId,
        bankAccountId: formData.bankAccountId || undefined,
        categoryId: formData.categoryId || undefined,
        type: formData.type,
        amount: Number(formData.amount),
        currency: 'BRL',
        description: formData.description.trim(),
        recurrenceRule,
        startOn: formData.startOn,
        endOn: formData.endOn || undefined,
        nextOccurrence: formData.startOn,
        notes: formData.notes || undefined,
      };

      // Remove undefined values
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined)
      );

      if (editingId) {
        await api.put(`/recurring-transactions/${editingId}`, cleanPayload);
      } else {
        await api.post('/recurring-transactions', cleanPayload);
      }

      await loadRecurring();
      closeModal();
    } catch (e) {
      console.warn('Erro ao salvar', e);
      setError('Nao foi possivel salvar a transacao recorrente.');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    const ok = await confirm('Deseja realmente excluir esta transacao recorrente?');
    if (!ok) return;
    try {
      await api.delete(`/recurring-transactions/${id}`);
      await loadRecurring();
      toast('Transacao recorrente excluida');
    } catch (e) {
      console.warn('Erro ao excluir', e);
      toast('Nao foi possivel excluir', 'error');
    }
  };

  const handleToggleStatus = async (item: RecurringTransaction) => {
    if (item.status === 'ACTIVE') {
      // Open pause modal to set review date
      setPausingItem(item);
      setReviewOnDate(getDateMonthsFromNow(3)); // Default to 3 months
      setPauseModalOpen(true);
    } else {
      // Resume directly
      setTogglingId(item.id);
      try {
        await api.patch(`/recurring-transactions/${item.id}/status`, { status: 'ACTIVE' });
        await loadRecurring();
      } catch (e) {
        console.warn('Erro ao alterar status', e);
        setError('Nao foi possivel alterar o status.');
      } finally {
        setTogglingId(null);
      }
    }
  };

  const handlePauseConfirm = async () => {
    if (!pausingItem) return;
    setTogglingId(pausingItem.id);
    setPauseModalOpen(false);
    try {
      // First update status to PAUSED
      await api.patch(`/recurring-transactions/${pausingItem.id}/status`, { status: 'PAUSED' });

      // Then update with reviewOn date via PUT
      if (reviewOnDate) {
        const freq = pausingItem.recurrenceRule.includes('DAILY') ? 'DAILY'
          : pausingItem.recurrenceRule.includes('WEEKLY') ? 'WEEKLY' : 'MONTHLY';
        const updatePayload = {
          profileId: pausingItem.profileId,
          bankAccountId: pausingItem.bankAccountId || undefined,
          categoryId: pausingItem.categoryId || undefined,
          type: pausingItem.type,
          amount: pausingItem.amount,
          currency: pausingItem.currency,
          description: pausingItem.description,
          recurrenceRule: `FREQ=${freq}`,
          startOn: pausingItem.startOn.slice(0, 10),
          endOn: pausingItem.endOn ? pausingItem.endOn.slice(0, 10) : undefined,
          nextOccurrence: pausingItem.nextOccurrence.slice(0, 10),
          status: 'PAUSED',
          reviewOn: reviewOnDate,
          notes: pausingItem.notes || undefined,
        };
        const cleanPayload = Object.fromEntries(
          Object.entries(updatePayload).filter(([, v]) => v !== undefined)
        );
        await api.put(`/recurring-transactions/${pausingItem.id}`, cleanPayload);
      }

      await loadRecurring();
    } catch (e) {
      console.warn('Erro ao pausar', e);
      setError('Nao foi possivel pausar a transacao.');
    } finally {
      setTogglingId(null);
      setPausingItem(null);
      setReviewOnDate('');
    }
  };

  const closePauseModal = () => {
    setPauseModalOpen(false);
    setPausingItem(null);
    setReviewOnDate('');
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'ACTIVE': return 'Ativo';
      case 'PAUSED': return 'Pausado';
      case 'CANCELLED': return 'Cancelado';
      default: return status;
    }
  };

  const getTypeLabel = (type: TransactionType) => {
    switch (type) {
      case 'INCOME': return 'Receita';
      case 'EXPENSE': return 'Despesa';
      case 'TRANSFER': return 'Transferencia';
    }
  };

  const getFrequencyLabel = (rule: string) => {
    if (rule.includes('DAILY')) return 'Diaria';
    if (rule.includes('WEEKLY')) return 'Semanal';
    return 'Mensal';
  };


  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Transacoes Recorrentes</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        <div className="flex justify-end">
          <button
            onClick={openCreateModal}
            className="px-4 py-2 bg-emerald-500/80 hover:bg-emerald-500 text-white rounded-lg font-semibold border border-emerald-400/40 transition-colors"
          >
            + Nova Recorrente
          </button>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          {!selectedProfile && (
            <p className="text-white/70">Selecione um perfil para visualizar.</p>
          )}
          {selectedProfile && (
            <>
              {loading && <p className="text-white/70">Carregando...</p>}
              {error && (
                <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3 mb-3">
                  {error}
                </div>
              )}
              {items.length === 0 && !loading ? (
                <p className="text-white/70">Nenhuma transacao recorrente encontrada.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead>
                      <tr className="text-white/60">
                        <th className="text-left py-2 pr-4">Descricao</th>
                        <th className="text-left py-2 pr-4">Tipo</th>
                        <th className="text-left py-2 pr-4">Valor</th>
                        <th className="text-left py-2 pr-4">Frequencia</th>
                        <th className="text-left py-2 pr-4">Proxima</th>
                        <th className="text-left py-2 pr-4">Status</th>
                        <th className="text-left py-2">Acoes</th>
                      </tr>
                    </thead>
                    <tbody>
                      {items.map((it) => (
                        <tr key={it.id} className={`border-t border-white/10 ${it.status === 'PAUSED' ? 'text-white/50' : 'text-white/90'}`}>
                          <td className="py-3 pr-4">
                            <span className={it.status === 'PAUSED' ? 'line-through' : ''}>
                              {it.description}
                            </span>
                          </td>
                          <td className="py-3 pr-4">
                            <span className={`text-xs px-2 py-1 rounded-full ${
                              it.type === 'INCOME' ? 'bg-emerald-500/20 text-emerald-200' :
                              it.type === 'EXPENSE' ? 'bg-rose-500/20 text-rose-200' :
                              'bg-blue-500/20 text-blue-200'
                            } ${it.status === 'PAUSED' ? 'opacity-50' : ''}`}>
                              {getTypeLabel(it.type)}
                            </span>
                          </td>
                          <td className={`py-3 pr-4 font-medium ${it.type === 'INCOME' ? 'text-emerald-300' : 'text-rose-300'} ${it.status === 'PAUSED' ? 'opacity-50' : ''}`}>
                            {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: it.currency }).format(it.amount)}
                          </td>
                          <td className="py-3 pr-4">{getFrequencyLabel(it.recurrenceRule)}</td>
                          <td className="py-3 pr-4">
                            {it.status === 'PAUSED' ? (
                              <span className="text-white/40">-</span>
                            ) : (
                              parseLocalDate(it.nextOccurrence).toLocaleDateString('pt-BR')
                            )}
                          </td>
                          <td className="py-3 pr-4">
                            <div className="flex flex-col gap-1">
                              <span className={`text-xs px-2 py-1 rounded-full w-fit ${
                                it.status === 'ACTIVE' ? 'bg-emerald-500/20 text-emerald-200' :
                                it.status === 'PAUSED' ? 'bg-amber-500/20 text-amber-200' :
                                'bg-zinc-500/20 text-zinc-200'
                              }`}>
                                {getStatusLabel(it.status)}
                              </span>
                              {it.status === 'PAUSED' && it.reviewOn && (
                                <span className="text-xs text-amber-300/70">
                                  Revisar: {parseLocalDate(it.reviewOn).toLocaleDateString('pt-BR')}
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="py-3">
                            <div className="flex gap-1">
                              <button
                                onClick={() => handleToggleStatus(it)}
                                disabled={togglingId === it.id}
                                className={`p-1.5 rounded-lg transition-colors ${
                                  togglingId === it.id
                                    ? 'text-white/30 bg-white/5'
                                    : it.status === 'ACTIVE'
                                      ? 'text-amber-400/70 hover:text-amber-400 hover:bg-amber-500/10'
                                      : 'text-emerald-400/70 hover:text-emerald-400 hover:bg-emerald-500/10'
                                }`}
                                title={it.status === 'ACTIVE' ? 'Pausar' : 'Retomar'}
                              >
                                {togglingId === it.id ? (
                                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                                  </svg>
                                ) : it.status === 'ACTIVE' ? (
                                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                                  </svg>
                                ) : (
                                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                  </svg>
                                )}
                              </button>
                              <button
                                onClick={() => openEditModal(it)}
                                className="p-1.5 text-white/60 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                                title="Editar"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                </svg>
                              </button>
                              <button
                                onClick={() => handleDelete(it.id)}
                                className="p-1.5 text-white/60 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
                                title="Excluir"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Pause Modal */}
      {pauseModalOpen && pausingItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-md mx-4">
            <h3 className="text-xl font-bold text-white mb-2">Pausar Transacao</h3>
            <p className="text-white/70 mb-4">
              Pausando: <span className="text-white font-medium">{pausingItem.description}</span>
            </p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Lembrar de revisar em:</label>
                <input
                  type="date"
                  value={reviewOnDate}
                  onChange={(e) => setReviewOnDate(e.target.value)}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                />
                <p className="text-xs text-white/50 mt-1">
                  Voce sera alertado nesta data para revisar a pausa.
                </p>
              </div>

              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setReviewOnDate(getDateMonthsFromNow(1))}
                  className="px-3 py-1.5 text-xs bg-white/10 hover:bg-white/20 text-white/70 rounded-lg transition-colors"
                >
                  1 mes
                </button>
                <button
                  type="button"
                  onClick={() => setReviewOnDate(getDateMonthsFromNow(3))}
                  className="px-3 py-1.5 text-xs bg-white/10 hover:bg-white/20 text-white/70 rounded-lg transition-colors"
                >
                  3 meses
                </button>
                <button
                  type="button"
                  onClick={() => setReviewOnDate(getDateMonthsFromNow(6))}
                  className="px-3 py-1.5 text-xs bg-white/10 hover:bg-white/20 text-white/70 rounded-lg transition-colors"
                >
                  6 meses
                </button>
                <button
                  type="button"
                  onClick={() => setReviewOnDate('')}
                  className="px-3 py-1.5 text-xs bg-white/10 hover:bg-white/20 text-white/70 rounded-lg transition-colors"
                >
                  Sem lembrete
                </button>
              </div>
            </div>

            <div className="flex gap-3 pt-6">
              <button
                type="button"
                onClick={closePauseModal}
                className="flex-1 px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg border border-white/20 transition-colors"
              >
                Cancelar
              </button>
              <button
                type="button"
                onClick={handlePauseConfirm}
                className="flex-1 px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors"
              >
                Pausar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
            <h3 className="text-xl font-bold text-white mb-4">
              {editingId ? 'Editar Recorrente' : 'Nova Transacao Recorrente'}
            </h3>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-2">Tipo</label>
                <div className="grid grid-cols-3 gap-2">
                  {(['EXPENSE', 'INCOME', 'TRANSFER'] as TransactionType[]).map((t) => (
                    <button
                      key={t}
                      type="button"
                      onClick={() => setFormData({ ...formData, type: t, categoryId: '' })}
                      className={`px-3 py-2 rounded-lg border text-sm transition-colors ${
                        formData.type === t
                          ? 'bg-white/15 border-white/40 text-white'
                          : 'bg-white/5 border-white/10 text-white/60 hover:bg-white/10'
                      }`}
                    >
                      {t === 'EXPENSE' ? 'Despesa' : t === 'INCOME' ? 'Receita' : 'Transferencia'}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Descricao</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  placeholder="Ex: Aluguel, Salario..."
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-white/70 mb-1">Valor</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.amount}
                    onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                    placeholder="0,00"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-white/70 mb-1">Frequencia</label>
                  <select
                    value={formData.frequency}
                    onChange={(e) => setFormData({ ...formData, frequency: e.target.value as Frequency })}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  >
                    <option value="DAILY" className="bg-slate-900">Diaria</option>
                    <option value="WEEKLY" className="bg-slate-900">Semanal</option>
                    <option value="MONTHLY" className="bg-slate-900">Mensal</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-white/70 mb-1">Conta</label>
                  <select
                    value={formData.bankAccountId}
                    onChange={(e) => setFormData({ ...formData, bankAccountId: e.target.value })}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  >
                    <option value="" className="bg-slate-900">Selecione</option>
                    {filteredAccounts.map((a) => (
                      <option key={a.id} value={a.id} className="bg-slate-900">{a.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm text-white/70 mb-1">Categoria</label>
                  <select
                    value={formData.categoryId}
                    onChange={(e) => setFormData({ ...formData, categoryId: e.target.value })}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  >
                    <option value="" className="bg-slate-900">Selecione</option>
                    {filteredCategories.map((c) => (
                      <option key={c.id} value={c.id} className="bg-slate-900">{c.name}</option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-white/70 mb-1">Inicio</label>
                  <input
                    type="date"
                    value={formData.startOn}
                    onChange={(e) => setFormData({ ...formData, startOn: e.target.value })}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-white/70 mb-1">Fim (opcional)</label>
                  <input
                    type="date"
                    value={formData.endOn}
                    onChange={(e) => setFormData({ ...formData, endOn: e.target.value })}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Observacoes</label>
                <textarea
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  rows={2}
                  placeholder="Opcional..."
                />
              </div>

              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={closeModal}
                  className="flex-1 px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg border border-white/20 transition-colors"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={saving || !formData.description.trim() || !formData.amount}
                  className={`flex-1 px-4 py-2 rounded-lg font-semibold border transition-colors ${
                    saving || !formData.description.trim() || !formData.amount
                      ? 'bg-white/10 text-white/40 border-white/10'
                      : 'bg-emerald-500/80 hover:bg-emerald-500 text-white border-emerald-400/40'
                  }`}
                >
                  {saving ? 'Salvando...' : 'Salvar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
