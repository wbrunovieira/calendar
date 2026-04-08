'use client';

import { useCallback, useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { BankAccount, CapitalContribution, CapitalContributionSummary, ContributionType } from '@/types/finances';
import { useToast } from '@/components/ui/Toast';

interface ContributionFormData {
  type: ContributionType;
  amount: string;
  date: string;
  description: string;
  sourceAccountId: string;
  notes: string;
}

const TYPE_LABELS: Record<ContributionType, string> = {
  CONTRIBUTION: 'Aporte',
  WITHDRAWAL: 'Retirada',
  LOAN: 'Emprestimo',
};

const TYPE_COLORS: Record<ContributionType, string> = {
  CONTRIBUTION: 'bg-emerald-500/20 text-emerald-200 border-emerald-400/40',
  WITHDRAWAL: 'bg-rose-500/20 text-rose-200 border-rose-400/40',
  LOAN: 'bg-amber-500/20 text-amber-200 border-amber-400/40',
};

function formatCurrency(value: number) {
  return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('pt-BR');
}

export default function AportesPage() {
  const { selectedProfileId } = useProfile();
  const { toast, confirm } = useToast();

  const [contributions, setContributions] = useState<CapitalContribution[]>([]);
  const [summary, setSummary] = useState<CapitalContributionSummary | null>(null);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [loading, setLoading] = useState(false);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<CapitalContribution | null>(null);
  const [formData, setFormData] = useState<ContributionFormData>({
    type: 'CONTRIBUTION',
    amount: '',
    date: '',
    description: '',
    sourceAccountId: '',
    notes: '',
  });
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    if (!selectedProfileId) return;
    setLoading(true);
    try {
      const contribData = await api.get<{ data: CapitalContribution[] }>(`/capital-contributions?profileId=${selectedProfileId}`).catch(() => null);
      const summaryData = await api.get<{ data: CapitalContributionSummary }>(`/capital-contributions/summary?profileId=${selectedProfileId}`).catch(() => null);
      const accountData = await api.get<{ data: BankAccount[] }>(`/bank-accounts?profileId=${selectedProfileId}`).catch(() => null);
      setContributions(contribData?.data ?? []);
      setSummary(summaryData?.data ?? null);
      setAccounts(accountData?.data ?? []);
    } catch (e) {
      console.warn('Erro ao carregar aportes', e);
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const openCreate = () => {
    setEditing(null);
    const today = new Date();
    const dateStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
    setFormData({
      type: 'CONTRIBUTION',
      amount: '',
      date: dateStr,
      description: '',
      sourceAccountId: '',
      notes: '',
    });
    setModalOpen(true);
  };

  const openEdit = (c: CapitalContribution) => {
    setEditing(c);
    setFormData({
      type: c.type,
      amount: String(c.amount),
      date: c.date.slice(0, 10),
      description: c.description,
      sourceAccountId: c.sourceAccountId || '',
      notes: c.notes || '',
    });
    setModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProfileId) return;
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        profileId: selectedProfileId,
        type: formData.type,
        amount: parseFloat(formData.amount),
        date: formData.date,
        description: formData.description.trim(),
      };
      if (formData.sourceAccountId) payload.sourceAccountId = formData.sourceAccountId;
      if (formData.notes.trim()) payload.notes = formData.notes.trim();

      if (editing) {
        await api.put(`/capital-contributions/${editing.id}`, payload);
        toast('Registro atualizado com sucesso');
      } else {
        await api.post('/capital-contributions', payload);
        toast('Registro criado com sucesso');
      }
      setModalOpen(false);
      await loadData();
    } catch (e) {
      console.warn('Erro ao salvar', e);
      toast('Nao foi possivel salvar', 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (c: CapitalContribution) => {
    if (deletingId) return;
    const ok = await confirm(`Excluir "${c.description}"?`);
    if (!ok) return;
    setDeletingId(c.id);
    try {
      await api.delete(`/capital-contributions/${c.id}`);
      toast('Registro excluido');
      await loadData();
    } catch (e) {
      console.warn('Erro ao excluir', e);
      toast('Nao foi possivel excluir', 'error');
    } finally {
      setDeletingId(null);
    }
  };

  const inputClass = 'w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40';

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">Aportes do Socio</h2>
            <p className="text-white/60 text-sm">Controle os aportes, retiradas e emprestimos entre voce e a empresa</p>
          </div>
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors"
          >
            + Novo Registro
          </button>
        </div>

        {/* Summary Cards */}
        {summary && (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
            <div className="bg-emerald-500/10 border border-emerald-400/30 rounded-xl p-4">
              <p className="text-emerald-300 text-xs font-semibold uppercase tracking-wider mb-1">Total Aportado</p>
              <p className="text-white font-bold text-lg">{formatCurrency(summary.totalContributed)}</p>
            </div>
            <div className="bg-rose-500/10 border border-rose-400/30 rounded-xl p-4">
              <p className="text-rose-300 text-xs font-semibold uppercase tracking-wider mb-1">Total Retirado</p>
              <p className="text-white font-bold text-lg">{formatCurrency(summary.totalWithdrawn)}</p>
            </div>
            <div className="bg-amber-500/10 border border-amber-400/30 rounded-xl p-4">
              <p className="text-amber-300 text-xs font-semibold uppercase tracking-wider mb-1">Emprestimos</p>
              <p className="text-white font-bold text-lg">{formatCurrency(summary.totalLoaned)}</p>
            </div>
            <div className="bg-blue-500/10 border border-blue-400/30 rounded-xl p-4">
              <p className="text-blue-300 text-xs font-semibold uppercase tracking-wider mb-1">Devolvido</p>
              <p className="text-white font-bold text-lg">{formatCurrency(summary.totalReturned)}</p>
            </div>
            <div className={`border rounded-xl p-4 col-span-2 md:col-span-1 ${summary.outstandingDebt > 0 ? 'bg-orange-500/15 border-orange-400/40' : 'bg-white/5 border-white/10'}`}>
              <p className={`text-xs font-semibold uppercase tracking-wider mb-1 ${summary.outstandingDebt > 0 ? 'text-orange-300' : 'text-white/50'}`}>
                Empresa deve ao socio
              </p>
              <p className={`font-bold text-lg ${summary.outstandingDebt > 0 ? 'text-orange-200' : 'text-white/60'}`}>
                {formatCurrency(summary.outstandingDebt)}
              </p>
            </div>
          </div>
        )}

        {/* List */}
        <div className="space-y-2">
          {loading && <p className="text-white/60">Carregando...</p>}
          {!loading && contributions.length === 0 && (
            <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
              <p className="text-white/60 mb-4">Nenhum registro encontrado. Crie o primeiro aporte!</p>
              <button
                onClick={openCreate}
                className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors"
              >
                + Criar Primeiro Registro
              </button>
            </div>
          )}
          {contributions.map((c) => (
            <div key={c.id} className="bg-white/5 border border-white/10 rounded-xl p-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="text-2xl">
                  {c.type === 'CONTRIBUTION' ? '💰' : c.type === 'WITHDRAWAL' ? '📤' : '🤝'}
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className={`text-xs px-2 py-0.5 rounded-full border ${TYPE_COLORS[c.type]}`}>
                      {TYPE_LABELS[c.type]}
                    </span>
                    <span className="text-white/50 text-xs">{formatDate(c.date)}</span>
                    {c.isReturned && (
                      <span className="text-xs px-2 py-0.5 rounded-full border bg-slate-500/20 text-slate-300 border-slate-400/40">
                        Devolvido
                      </span>
                    )}
                  </div>
                  <p className="text-white font-medium">{c.description}</p>
                  {c.notes && <p className="text-white/50 text-xs mt-0.5">{c.notes}</p>}
                </div>
              </div>
              <div className="flex items-center gap-4">
                <p className={`font-bold text-lg ${c.type === 'CONTRIBUTION' ? 'text-emerald-300' : c.type === 'WITHDRAWAL' ? 'text-rose-300' : 'text-amber-300'}`}>
                  {formatCurrency(c.amount)}
                </p>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => openEdit(c)}
                    className="p-2 text-white/50 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                    </svg>
                  </button>
                  <button
                    onClick={() => handleDelete(c)}
                    disabled={!!deletingId}
                    className="p-2 text-white/50 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors disabled:opacity-50"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-xl font-bold text-white">
                {editing ? 'Editar Registro' : 'Novo Registro'}
              </h3>
              <button onClick={() => setModalOpen(false)} className="text-white/50 hover:text-white">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Tipo</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as ContributionType })}
                  className={inputClass}
                >
                  <option value="CONTRIBUTION" className="bg-slate-900">Aporte (dinheiro que voce colocou)</option>
                  <option value="WITHDRAWAL" className="bg-slate-900">Retirada (pro-labore / dividendo)</option>
                  <option value="LOAN" className="bg-slate-900">Emprestimo (empresa deve devolver)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Descricao</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className={inputClass}
                  placeholder="Ex: Aporte inicial de capital"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Valor (R$)</label>
                <input
                  type="number"
                  step="0.01"
                  min="0.01"
                  value={formData.amount}
                  onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
                  className={inputClass}
                  placeholder="0,00"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Data</label>
                <input
                  type="date"
                  value={formData.date}
                  onChange={(e) => setFormData({ ...formData, date: e.target.value })}
                  className={inputClass}
                  required
                />
              </div>

              {accounts.length > 0 && (
                <div>
                  <label className="block text-sm text-white/70 mb-1">Conta de Origem (opcional)</label>
                  <select
                    value={formData.sourceAccountId}
                    onChange={(e) => setFormData({ ...formData, sourceAccountId: e.target.value })}
                    className={inputClass}
                  >
                    <option value="" className="bg-slate-900">Nenhuma</option>
                    {accounts.map((acc) => (
                      <option key={acc.id} value={acc.id} className="bg-slate-900">{acc.name}</option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="block text-sm text-white/70 mb-1">Observacoes (opcional)</label>
                <textarea
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  className={`${inputClass} resize-none`}
                  rows={2}
                  placeholder="Observacoes adicionais..."
                />
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
                  className="flex-1 px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg border border-white/20 transition-colors"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="flex-1 px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors disabled:opacity-50"
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
