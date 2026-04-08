'use client';

import { useCallback, useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { AssetCategory, CompanyAsset } from '@/types/finances';
import { useToast } from '@/components/ui/Toast';

interface AssetFormData {
  name: string;
  category: AssetCategory;
  purchaseDate: string;
  purchaseAmount: string;
  currentValue: string;
  depreciationRate: string;
  notes: string;
}

const CATEGORY_LABELS: Record<AssetCategory, string> = {
  HARDWARE: 'Hardware (Computadores, etc)',
  SOFTWARE: 'Software / Licencas',
  FURNITURE: 'Moveis e Utensilios',
  VEHICLE: 'Veiculo',
  REAL_ESTATE: 'Imovel',
  OTHER: 'Outros',
};

const CATEGORY_ICONS: Record<AssetCategory, string> = {
  HARDWARE: '💻',
  SOFTWARE: '🖥️',
  FURNITURE: '🪑',
  VEHICLE: '🚗',
  REAL_ESTATE: '🏠',
  OTHER: '📦',
};

function formatCurrency(value: number) {
  return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('pt-BR');
}

export default function AtivosPage() {
  const { selectedProfileId } = useProfile();
  const { toast, confirm } = useToast();

  const [assets, setAssets] = useState<CompanyAsset[]>([]);
  const [loading, setLoading] = useState(false);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<CompanyAsset | null>(null);
  const [formData, setFormData] = useState<AssetFormData>({
    name: '',
    category: 'HARDWARE',
    purchaseDate: new Date().toISOString().slice(0, 10),
    purchaseAmount: '',
    currentValue: '',
    depreciationRate: '20',
    notes: '',
  });
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadAssets = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      setLoading(true);
      const data = await api.get<{ data: CompanyAsset[] }>(`/company-assets?profileId=${selectedProfileId}`);
      setAssets(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar ativos', e);
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId]);

  useEffect(() => {
    loadAssets();
  }, [loadAssets]);

  const openCreate = () => {
    setEditing(null);
    setFormData({
      name: '',
      category: 'HARDWARE',
      purchaseDate: new Date().toISOString().slice(0, 10),
      purchaseAmount: '',
      currentValue: '',
      depreciationRate: '20',
      notes: '',
    });
    setModalOpen(true);
  };

  const openEdit = (a: CompanyAsset) => {
    setEditing(a);
    setFormData({
      name: a.name,
      category: a.category,
      purchaseDate: a.purchaseDate.slice(0, 10),
      purchaseAmount: String(a.purchaseAmount),
      currentValue: String(a.currentValue),
      depreciationRate: String(a.depreciationRate),
      notes: a.notes || '',
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
        name: formData.name.trim(),
        category: formData.category,
        purchaseDate: formData.purchaseDate,
        purchaseAmount: parseFloat(formData.purchaseAmount),
        currentValue: formData.currentValue ? parseFloat(formData.currentValue) : parseFloat(formData.purchaseAmount),
        depreciationRate: parseFloat(formData.depreciationRate) || 0,
      };
      if (formData.notes.trim()) payload.notes = formData.notes.trim();

      if (editing) {
        await api.put(`/company-assets/${editing.id}`, payload);
        toast('Ativo atualizado com sucesso');
      } else {
        await api.post('/company-assets', payload);
        toast('Ativo registrado com sucesso');
      }
      setModalOpen(false);
      await loadAssets();
    } catch (e) {
      console.warn('Erro ao salvar ativo', e);
      toast('Nao foi possivel salvar', 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (a: CompanyAsset) => {
    if (deletingId) return;
    const ok = await confirm(`Excluir o ativo "${a.name}"?`);
    if (!ok) return;
    setDeletingId(a.id);
    try {
      await api.delete(`/company-assets/${a.id}`);
      toast('Ativo excluido');
      await loadAssets();
    } catch (e) {
      console.warn('Erro ao excluir ativo', e);
      toast('Nao foi possivel excluir', 'error');
    } finally {
      setDeletingId(null);
    }
  };

  const totalPurchaseValue = assets.filter((a) => a.isActive).reduce((s, a) => s + a.purchaseAmount, 0);
  const totalCurrentValue = assets.filter((a) => a.isActive).reduce((s, a) => s + a.currentValue, 0);
  const depreciation = totalPurchaseValue - totalCurrentValue;

  const inputClass = 'w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40';

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">Patrimonio da Empresa</h2>
            <p className="text-white/60 text-sm">Controle os ativos fixos e patrimoniais da empresa</p>
          </div>
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors"
          >
            + Novo Ativo
          </button>
        </div>

        {/* Summary */}
        {assets.length > 0 && (
          <div className="grid grid-cols-3 gap-3">
            <div className="bg-blue-500/10 border border-blue-400/30 rounded-xl p-4">
              <p className="text-blue-300 text-xs font-semibold uppercase tracking-wider mb-1">Valor de Compra</p>
              <p className="text-white font-bold text-lg">{formatCurrency(totalPurchaseValue)}</p>
            </div>
            <div className="bg-emerald-500/10 border border-emerald-400/30 rounded-xl p-4">
              <p className="text-emerald-300 text-xs font-semibold uppercase tracking-wider mb-1">Valor Atual</p>
              <p className="text-white font-bold text-lg">{formatCurrency(totalCurrentValue)}</p>
            </div>
            <div className="bg-rose-500/10 border border-rose-400/30 rounded-xl p-4">
              <p className="text-rose-300 text-xs font-semibold uppercase tracking-wider mb-1">Depreciacao Total</p>
              <p className="text-white font-bold text-lg">{formatCurrency(depreciation)}</p>
            </div>
          </div>
        )}

        {/* List */}
        <div className="space-y-2">
          {loading && <p className="text-white/60">Carregando...</p>}
          {!loading && assets.length === 0 && (
            <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
              <p className="text-white/60 mb-4">Nenhum ativo registrado. Adicione o primeiro!</p>
              <button
                onClick={openCreate}
                className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors"
              >
                + Adicionar Primeiro Ativo
              </button>
            </div>
          )}
          {assets.map((a) => {
            const depreciationAmt = a.purchaseAmount - a.currentValue;
            const depreciationPct = a.purchaseAmount > 0 ? (depreciationAmt / a.purchaseAmount) * 100 : 0;
            return (
              <div key={a.id} className="bg-white/5 border border-white/10 rounded-xl p-4 flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="text-3xl">{CATEGORY_ICONS[a.category]}</div>
                  <div>
                    <div className="flex items-center gap-2 mb-0.5">
                      <p className="text-white font-semibold">{a.name}</p>
                      {!a.isActive && (
                        <span className="text-xs px-2 py-0.5 rounded-full border bg-slate-500/20 text-slate-300 border-slate-400/40">
                          Baixado
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 text-white/50 text-xs">
                      <span className="bg-white/10 px-2 py-0.5 rounded">{CATEGORY_LABELS[a.category].split(' ')[0]}</span>
                      <span>Compra: {formatDate(a.purchaseDate)}</span>
                      {a.depreciationRate > 0 && <span>Deprec.: {a.depreciationRate}% a.a.</span>}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-6">
                  <div className="text-right">
                    <p className="text-white/50 text-xs">Compra</p>
                    <p className="text-white/80 font-medium">{formatCurrency(a.purchaseAmount)}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-white/50 text-xs">Atual</p>
                    <p className="text-white font-bold">{formatCurrency(a.currentValue)}</p>
                  </div>
                  {depreciationPct > 0 && (
                    <div className="text-right hidden sm:block">
                      <p className="text-white/50 text-xs">Deprec.</p>
                      <p className="text-rose-300 font-medium">{depreciationPct.toFixed(0)}%</p>
                    </div>
                  )}
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => openEdit(a)}
                      className="p-2 text-white/50 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => handleDelete(a)}
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
            );
          })}
        </div>
      </div>

      {/* Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-xl font-bold text-white">
                {editing ? 'Editar Ativo' : 'Novo Ativo'}
              </h3>
              <button onClick={() => setModalOpen(false)} className="text-white/50 hover:text-white">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Nome do Ativo</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className={inputClass}
                  placeholder="Ex: MacBook Pro M3"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Categoria</label>
                <select
                  value={formData.category}
                  onChange={(e) => setFormData({ ...formData, category: e.target.value as AssetCategory })}
                  className={inputClass}
                >
                  {(Object.keys(CATEGORY_LABELS) as AssetCategory[]).map((k) => (
                    <option key={k} value={k} className="bg-slate-900">{CATEGORY_LABELS[k]}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Data de Compra</label>
                <input
                  type="date"
                  value={formData.purchaseDate}
                  onChange={(e) => setFormData({ ...formData, purchaseDate: e.target.value })}
                  className={inputClass}
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-white/70 mb-1">Valor de Compra (R$)</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={formData.purchaseAmount}
                    onChange={(e) => setFormData({ ...formData, purchaseAmount: e.target.value })}
                    className={inputClass}
                    placeholder="0,00"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-white/70 mb-1">Valor Atual (R$)</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={formData.currentValue}
                    onChange={(e) => setFormData({ ...formData, currentValue: e.target.value })}
                    className={inputClass}
                    placeholder="Deixe vazio = valor de compra"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Taxa de Depreciacao Anual (%)</label>
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  max="100"
                  value={formData.depreciationRate}
                  onChange={(e) => setFormData({ ...formData, depreciationRate: e.target.value })}
                  className={inputClass}
                  placeholder="Ex: 20 para 20% ao ano"
                />
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Observacoes (opcional)</label>
                <textarea
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  className={`${inputClass} resize-none`}
                  rows={2}
                  placeholder="Numero de serie, garantia, etc..."
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
