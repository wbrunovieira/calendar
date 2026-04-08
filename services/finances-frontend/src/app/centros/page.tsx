'use client';

import { useCallback, useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { CostCenter, CostCenterType } from '@/types/finances';
import { useToast } from '@/components/ui/Toast';

interface FormData {
  name: string;
  type: CostCenterType;
  color: string;
}

const TYPE_LABELS: Record<CostCenterType, string> = {
  CLIENT: 'Cliente',
  PROJECT: 'Projeto',
  DEPARTMENT: 'Departamento',
};

const TYPE_ICONS: Record<CostCenterType, string> = {
  CLIENT: '🤝',
  PROJECT: '📋',
  DEPARTMENT: '🏢',
};

const TYPE_COLORS: Record<CostCenterType, string> = {
  CLIENT: 'bg-blue-500/20 text-blue-200 border-blue-400/40',
  PROJECT: 'bg-violet-500/20 text-violet-200 border-violet-400/40',
  DEPARTMENT: 'bg-amber-500/20 text-amber-200 border-amber-400/40',
};

const PRESET_COLORS = [
  '#3b82f6', '#8b5cf6', '#f59e0b', '#10b981', '#ef4444',
  '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#6366f1',
];

function formatCurrency(v: number) {
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

export default function CentrosPage() {
  const { selectedProfileId } = useProfile();
  const { toast, confirm } = useToast();

  const [centers, setCenters] = useState<CostCenter[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<CostCenter | null>(null);
  const [formData, setFormData] = useState<FormData>({ name: '', type: 'CLIENT', color: PRESET_COLORS[0] });
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [filterType, setFilterType] = useState<CostCenterType | 'ALL'>('ALL');

  const load = useCallback(async () => {
    if (!selectedProfileId) return;
    setLoading(true);
    try {
      const data = await api.get<{ data: CostCenter[] }>(`/cost-centers?profileId=${selectedProfileId}`);
      setCenters(data?.data ?? []);
    } catch (e) { console.warn(e); } finally { setLoading(false); }
  }, [selectedProfileId]);

  useEffect(() => { load(); }, [load]);

  const openCreate = () => {
    setEditing(null);
    setFormData({ name: '', type: 'CLIENT', color: PRESET_COLORS[0] });
    setModalOpen(true);
  };

  const openEdit = (c: CostCenter) => {
    setEditing(c);
    setFormData({ name: c.name, type: c.type, color: c.color || PRESET_COLORS[0] });
    setModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProfileId) return;
    setSaving(true);
    try {
      const payload = { profileId: selectedProfileId, ...formData };
      if (editing) {
        await api.put(`/cost-centers/${editing.id}`, { ...payload, isActive: editing.isActive });
        toast('Centro de custo atualizado');
      } else {
        await api.post('/cost-centers', payload);
        toast('Centro de custo criado');
      }
      setModalOpen(false);
      await load();
    } catch (e) { console.warn(e); toast('Erro ao salvar', 'error'); }
    finally { setSaving(false); }
  };

  const handleDelete = async (c: CostCenter) => {
    if (deletingId) return;
    if (!await confirm(`Excluir "${c.name}"?`)) return;
    setDeletingId(c.id);
    try {
      await api.delete(`/cost-centers/${c.id}`);
      toast('Excluído');
      await load();
    } catch (e) { console.warn(e); toast('Erro ao excluir', 'error'); }
    finally { setDeletingId(null); }
  };

  const filtered = filterType === 'ALL' ? centers : centers.filter((c) => c.type === filterType);
  const inputClass = 'w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40';

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">Centros de Custo</h2>
            <p className="text-white/60 text-sm">Organize transações por cliente, projeto ou departamento</p>
          </div>
          <button onClick={openCreate} className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors">
            + Novo Centro
          </button>
        </div>

        {/* Filter */}
        <div className="flex flex-wrap gap-2">
          {(['ALL', 'CLIENT', 'PROJECT', 'DEPARTMENT'] as const).map((t) => (
            <button key={t} onClick={() => setFilterType(t)}
              className={`px-3 py-1.5 rounded-xl text-sm border transition-colors ${filterType === t ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}>
              {t === 'ALL' ? 'Todos' : `${TYPE_ICONS[t]} ${TYPE_LABELS[t]}s`}
            </button>
          ))}
        </div>

        {/* List */}
        {loading && <p className="text-white/60">Carregando...</p>}
        {!loading && filtered.length === 0 && (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
            <p className="text-white/60 mb-4">Nenhum centro de custo. Crie o primeiro!</p>
            <button onClick={openCreate} className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40">+ Criar</button>
          </div>
        )}
        <div className="space-y-2">
          {filtered.map((c) => (
            <div key={c.id} className="bg-white/5 border border-white/10 rounded-xl p-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl flex items-center justify-center text-xl" style={{ backgroundColor: `${c.color || '#6366f1'}20` }}>
                  {TYPE_ICONS[c.type]}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <p className="text-white font-semibold">{c.name}</p>
                    {!c.isActive && <span className="text-xs px-2 py-0.5 rounded-full bg-slate-500/20 text-slate-400 border border-slate-400/30">Inativo</span>}
                  </div>
                  <span className={`text-xs px-2 py-0.5 rounded-full border ${TYPE_COLORS[c.type]}`}>{TYPE_LABELS[c.type]}</span>
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button onClick={() => openEdit(c)} className="p-2 text-white/50 hover:text-white hover:bg-white/10 rounded-lg">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
                </button>
                <button onClick={() => handleDelete(c)} disabled={!!deletingId} className="p-2 text-white/50 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg disabled:opacity-50">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-md mx-4">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-xl font-bold text-white">{editing ? 'Editar' : 'Novo'} Centro de Custo</h3>
              <button onClick={() => setModalOpen(false)} className="text-white/50 hover:text-white">✕</button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Nome</label>
                <input type="text" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} className={inputClass} placeholder="Ex: Cliente XYZ" required />
              </div>
              <div>
                <label className="block text-sm text-white/70 mb-1">Tipo</label>
                <select value={formData.type} onChange={(e) => setFormData({ ...formData, type: e.target.value as CostCenterType })} className={inputClass}>
                  {(Object.keys(TYPE_LABELS) as CostCenterType[]).map((t) => (
                    <option key={t} value={t} className="bg-slate-900">{TYPE_ICONS[t]} {TYPE_LABELS[t]}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm text-white/70 mb-2">Cor</label>
                <div className="flex flex-wrap gap-2">
                  {PRESET_COLORS.map((color) => (
                    <button key={color} type="button" onClick={() => setFormData({ ...formData, color })}
                      className={`w-8 h-8 rounded-full border-2 transition-all ${formData.color === color ? 'border-white scale-110' : 'border-transparent'}`}
                      style={{ backgroundColor: color }} />
                  ))}
                </div>
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => setModalOpen(false)} className="flex-1 px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg border border-white/20">Cancelar</button>
                <button type="submit" disabled={saving} className="flex-1 px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 disabled:opacity-50">
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
