'use client';

import { useCallback, useEffect, useState } from 'react';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { CampaignPlatform, MarketingCampaign, MarketingCampaignMetrics } from '@/types/finances';
import { useToast } from '@/components/ui/Toast';
import { formatCurrency } from '@/utils/format';

interface FormData {
  name: string;
  platform: CampaignPlatform;
  startDate: string;
  endDate: string;
  budget: string;
  revenueAttributed: string;
  leads: string;
  conversions: string;
  notes: string;
}

const PLATFORM_LABELS: Record<CampaignPlatform, string> = {
  META_ADS: 'Meta Ads (Facebook/Instagram)',
  GOOGLE_ADS: 'Google Ads',
  INSTAGRAM: 'Instagram Orgânico',
  LINKEDIN: 'LinkedIn',
  OTHER: 'Outro',
};

const PLATFORM_ICONS: Record<CampaignPlatform, string> = {
  META_ADS: '📘',
  GOOGLE_ADS: '🔍',
  INSTAGRAM: '📸',
  LINKEDIN: '💼',
  OTHER: '📣',
};

function RoiBadge({ roi }: { roi: number }) {
  if (roi === 0) return <span className="text-white/40 text-xs">ROI: —</span>;
  const positive = roi >= 0;
  return (
    <span className={`text-xs font-bold px-2 py-0.5 rounded-full border ${positive ? 'bg-emerald-500/15 text-emerald-300 border-emerald-400/30' : 'bg-rose-500/15 text-rose-300 border-rose-400/30'}`}>
      ROI {roi >= 0 ? '+' : ''}{roi.toFixed(1)}%
    </span>
  );
}

export default function MarketingPage() {
  const { selectedProfileId } = useProfile();
  const { toast, confirm } = useToast();

  const [campaigns, setCampaigns] = useState<MarketingCampaignMetrics[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<MarketingCampaign | null>(null);
  const [formData, setFormData] = useState<FormData>({
    name: '', platform: 'META_ADS', startDate: '', endDate: '',
    budget: '', revenueAttributed: '0', leads: '0', conversions: '0', notes: '',
  });
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!selectedProfileId) return;
    setLoading(true);
    try {
      const data = await api.get<{ data: MarketingCampaign[] }>(`/marketing-campaigns?profileId=${selectedProfileId}`);
      const list = data?.data ?? [];
      // Fetch metrics for each campaign
      const withMetrics = await Promise.all(
        list.map(async (c) => {
          try {
            const m = await api.get<{ data: MarketingCampaignMetrics }>(`/marketing-campaigns/${c.id}/metrics`);
            return m?.data ?? { ...c, totalSpent: 0, roi: 0, costPerLead: 0, costPerConversion: 0 };
          } catch {
            return { ...c, totalSpent: 0, roi: 0, costPerLead: 0, costPerConversion: 0 };
          }
        })
      );
      setCampaigns(withMetrics);
    } catch (e) { console.warn(e); } finally { setLoading(false); }
  }, [selectedProfileId]);

  useEffect(() => { load(); }, [load]);

  const openCreate = () => {
    setEditing(null);
    const today = new Date();
    const dateStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
    setFormData({ name: '', platform: 'META_ADS', startDate: dateStr, endDate: '', budget: '', revenueAttributed: '0', leads: '0', conversions: '0', notes: '' });
    setModalOpen(true);
  };

  const openEdit = (c: MarketingCampaignMetrics) => {
    setEditing(c);
    setFormData({
      name: c.name, platform: c.platform,
      startDate: c.startDate.slice(0, 10),
      endDate: c.endDate?.slice(0, 10) || '',
      budget: String(c.budget),
      revenueAttributed: String(c.revenueAttributed),
      leads: String(c.leads),
      conversions: String(c.conversions),
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
        name: formData.name.trim(),
        platform: formData.platform,
        startDate: formData.startDate,
        budget: parseFloat(formData.budget) || 0,
        revenueAttributed: parseFloat(formData.revenueAttributed) || 0,
        leads: parseInt(formData.leads) || 0,
        conversions: parseInt(formData.conversions) || 0,
      };
      if (formData.endDate) payload.endDate = formData.endDate;
      if (formData.notes.trim()) payload.notes = formData.notes.trim();
      if (editing) payload.isActive = editing.isActive;

      if (editing) {
        await api.put(`/marketing-campaigns/${editing.id}`, payload);
        toast('Campanha atualizada');
      } else {
        await api.post('/marketing-campaigns', payload);
        toast('Campanha criada');
      }
      setModalOpen(false);
      await load();
    } catch (e) { console.warn(e); toast('Erro ao salvar', 'error'); }
    finally { setSaving(false); }
  };

  const handleDelete = async (c: MarketingCampaign) => {
    if (deletingId) return;
    if (!await confirm(`Excluir campanha "${c.name}"?`)) return;
    setDeletingId(c.id);
    try {
      await api.delete(`/marketing-campaigns/${c.id}`);
      toast('Campanha excluída');
      await load();
    } catch (e) { console.warn(e); toast('Erro ao excluir', 'error'); }
    finally { setDeletingId(null); }
  };

  const totalBudget = campaigns.reduce((s, c) => s + c.budget, 0);
  const totalSpent = campaigns.reduce((s, c) => s + c.totalSpent, 0);
  const totalRevenue = campaigns.reduce((s, c) => s + c.revenueAttributed, 0);
  const overallRoi = totalSpent > 0 ? ((totalRevenue - totalSpent) / totalSpent) * 100 : 0;

  const inputClass = 'w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-white/40';

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">Marketing & ROI</h2>
            <p className="text-white/60 text-sm">Controle campanhas de anúncios e calcule o retorno sobre investimento</p>
          </div>
          <button onClick={openCreate} className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors">
            + Nova Campanha
          </button>
        </div>

        {/* Summary */}
        {campaigns.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div className="bg-blue-500/10 border border-blue-400/30 rounded-xl p-4">
              <p className="text-blue-300 text-xs font-semibold uppercase tracking-wider mb-1">Orçamento Total</p>
              <p className="text-white font-bold text-lg">{formatCurrency(totalBudget)}</p>
            </div>
            <div className="bg-rose-500/10 border border-rose-400/30 rounded-xl p-4">
              <p className="text-rose-300 text-xs font-semibold uppercase tracking-wider mb-1">Total Investido</p>
              <p className="text-white font-bold text-lg">{formatCurrency(totalSpent)}</p>
            </div>
            <div className="bg-emerald-500/10 border border-emerald-400/30 rounded-xl p-4">
              <p className="text-emerald-300 text-xs font-semibold uppercase tracking-wider mb-1">Receita Atribuída</p>
              <p className="text-white font-bold text-lg">{formatCurrency(totalRevenue)}</p>
            </div>
            <div className={`border rounded-xl p-4 ${overallRoi >= 0 ? 'bg-emerald-500/10 border-emerald-400/30' : 'bg-rose-500/10 border-rose-400/30'}`}>
              <p className={`text-xs font-semibold uppercase tracking-wider mb-1 ${overallRoi >= 0 ? 'text-emerald-300' : 'text-rose-300'}`}>ROI Geral</p>
              <p className={`font-bold text-lg ${overallRoi >= 0 ? 'text-emerald-200' : 'text-rose-200'}`}>{overallRoi >= 0 ? '+' : ''}{overallRoi.toFixed(1)}%</p>
            </div>
          </div>
        )}

        {/* Campaigns list */}
        {loading && <p className="text-white/60">Carregando...</p>}
        {!loading && campaigns.length === 0 && (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
            <p className="text-white/60 mb-4">Nenhuma campanha. Crie a primeira!</p>
            <button onClick={openCreate} className="px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40">+ Criar Campanha</button>
          </div>
        )}
        <div className="space-y-3">
          {campaigns.map((c) => (
            <div key={c.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <span className="text-2xl">{PLATFORM_ICONS[c.platform]}</span>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-white font-semibold">{c.name}</p>
                      <RoiBadge roi={c.roi} />
                      {!c.isActive && <span className="text-xs px-2 py-0.5 rounded-full bg-slate-500/20 text-slate-400 border border-slate-400/30">Encerrada</span>}
                    </div>
                    <p className="text-white/50 text-xs mt-0.5">
                      {PLATFORM_LABELS[c.platform]} · {c.startDate.slice(0, 10)}
                      {c.endDate ? ` → ${c.endDate.slice(0, 10)}` : ' · Em andamento'}
                    </p>
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

              {/* Metrics grid */}
              <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-2 mt-2">
                {[
                  { label: 'Orçamento', value: formatCurrency(c.budget), color: 'text-blue-300' },
                  { label: 'Gasto', value: formatCurrency(c.totalSpent), color: 'text-rose-300' },
                  { label: 'Receita', value: formatCurrency(c.revenueAttributed), color: 'text-emerald-300' },
                  { label: 'Leads', value: c.leads > 0 ? `${c.leads} (${formatCurrency(c.costPerLead)}/lead)` : '—', color: 'text-amber-300' },
                  { label: 'Conversões', value: c.conversions > 0 ? `${c.conversions} (${formatCurrency(c.costPerConversion)}/conv)` : '—', color: 'text-violet-300' },
                  { label: 'ROI', value: c.totalSpent > 0 ? `${c.roi >= 0 ? '+' : ''}${c.roi.toFixed(1)}%` : '—', color: c.roi >= 0 ? 'text-emerald-300' : 'text-rose-300' },
                ].map((m) => (
                  <div key={m.label} className="bg-white/5 rounded-lg p-2">
                    <p className="text-white/40 text-xs">{m.label}</p>
                    <p className={`text-sm font-semibold ${m.color}`}>{m.value}</p>
                  </div>
                ))}
              </div>

              {c.notes && <p className="text-white/40 text-xs mt-2 italic">{c.notes}</p>}
            </div>
          ))}
        </div>
      </div>

      {/* Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-xl font-bold text-white">{editing ? 'Editar' : 'Nova'} Campanha</h3>
              <button onClick={() => setModalOpen(false)} className="text-white/50 hover:text-white">✕</button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Nome da Campanha</label>
                <input type="text" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} className={inputClass} placeholder="Ex: Google Ads — Revalida Abr/2026" required />
              </div>
              <div>
                <label className="block text-sm text-white/70 mb-1">Plataforma</label>
                <select value={formData.platform} onChange={(e) => setFormData({ ...formData, platform: e.target.value as CampaignPlatform })} className={inputClass}>
                  {(Object.keys(PLATFORM_LABELS) as CampaignPlatform[]).map((p) => (
                    <option key={p} value={p} className="bg-slate-900">{PLATFORM_ICONS[p]} {PLATFORM_LABELS[p]}</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-white/70 mb-1">Data Início</label>
                  <input type="date" value={formData.startDate} onChange={(e) => setFormData({ ...formData, startDate: e.target.value })} className={inputClass} required />
                </div>
                <div>
                  <label className="block text-sm text-white/70 mb-1">Data Fim (opcional)</label>
                  <input type="date" value={formData.endDate} onChange={(e) => setFormData({ ...formData, endDate: e.target.value })} className={inputClass} />
                </div>
              </div>
              <div>
                <label className="block text-sm text-white/70 mb-1">Orçamento (R$)</label>
                <input type="number" step="0.01" min="0" value={formData.budget} onChange={(e) => setFormData({ ...formData, budget: e.target.value })} className={inputClass} placeholder="0,00" required />
              </div>
              <div className="border border-white/10 rounded-xl p-3 space-y-3 bg-white/[0.02]">
                <p className="text-white/50 text-xs font-semibold uppercase tracking-wider">Resultados (atualizar conforme campanha progride)</p>
                <div>
                  <label className="block text-sm text-white/70 mb-1">Receita Atribuída (R$)</label>
                  <input type="number" step="0.01" min="0" value={formData.revenueAttributed} onChange={(e) => setFormData({ ...formData, revenueAttributed: e.target.value })} className={inputClass} />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-sm text-white/70 mb-1">Leads</label>
                    <input type="number" min="0" value={formData.leads} onChange={(e) => setFormData({ ...formData, leads: e.target.value })} className={inputClass} />
                  </div>
                  <div>
                    <label className="block text-sm text-white/70 mb-1">Conversões</label>
                    <input type="number" min="0" value={formData.conversions} onChange={(e) => setFormData({ ...formData, conversions: e.target.value })} className={inputClass} />
                  </div>
                </div>
              </div>
              <div>
                <label className="block text-sm text-white/70 mb-1">Observações (opcional)</label>
                <textarea value={formData.notes} onChange={(e) => setFormData({ ...formData, notes: e.target.value })} className={`${inputClass} resize-none`} rows={2} />
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
