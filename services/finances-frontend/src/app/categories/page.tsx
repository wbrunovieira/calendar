'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import type { Profile, Category, CategoryType } from '@/types/finances';

const API_BASE = 'http://localhost:3335/api/v1';

interface CategoryFormData {
  name: string;
  type: CategoryType;
  color: string;
  icon: string;
}

const CATEGORY_COLORS = [
  '#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16',
  '#22c55e', '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9',
  '#3b82f6', '#6366f1', '#8b5cf6', '#a855f7', '#d946ef',
  '#ec4899', '#f43f5e',
];

const CATEGORY_ICONS = [
  { value: 'shopping', label: 'Compras' },
  { value: 'food', label: 'Alimentacao' },
  { value: 'transport', label: 'Transporte' },
  { value: 'health', label: 'Saude' },
  { value: 'education', label: 'Educacao' },
  { value: 'entertainment', label: 'Lazer' },
  { value: 'home', label: 'Casa' },
  { value: 'work', label: 'Trabalho' },
  { value: 'salary', label: 'Salario' },
  { value: 'investment', label: 'Investimento' },
  { value: 'gift', label: 'Presente' },
  { value: 'subscription', label: 'Assinatura' },
  { value: 'travel', label: 'Viagem' },
  { value: 'pet', label: 'Pet' },
  { value: 'other', label: 'Outros' },
];

export default function CategoriesPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [formData, setFormData] = useState<CategoryFormData>({
    name: '',
    type: 'EXPENSE',
    color: CATEGORY_COLORS[0],
    icon: 'other',
  });
  const [saving, setSaving] = useState(false);

  // Filter state
  const [filterType, setFilterType] = useState<CategoryType | 'ALL'>('ALL');

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/profiles`);
        const data = await res.json();
        const list: Profile[] = data.data || [];
        setProfiles(list);
        if (list.length > 0) setSelectedProfileId(list[0].id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  const loadCategories = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      setLoading(true);
      setError(null);
      const res = await fetch(`${API_BASE}/categories?profileId=${selectedProfileId}`);
      if (!res.ok) throw new Error(`status ${res.status}`);
      const data = await res.json();
      setCategories(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar categorias', e);
      setCategories([]);
      setError('Nao foi possivel carregar as categorias.');
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId]);

  useEffect(() => {
    if (!selectedProfileId) return;
    loadCategories();
  }, [selectedProfileId, loadCategories]);

  const openCreateModal = () => {
    setEditingCategory(null);
    setFormData({
      name: '',
      type: 'EXPENSE',
      color: CATEGORY_COLORS[0],
      icon: 'other',
    });
    setModalOpen(true);
  };

  const openEditModal = (cat: Category) => {
    setEditingCategory(cat);
    setFormData({
      name: cat.name,
      type: cat.type,
      color: cat.color || CATEGORY_COLORS[0],
      icon: cat.icon || 'other',
    });
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditingCategory(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProfileId || !formData.name.trim()) return;

    setSaving(true);
    try {
      const payload = {
        profileId: selectedProfileId,
        name: formData.name.trim(),
        type: formData.type,
        color: formData.color,
        icon: formData.icon,
      };

      let res: Response;
      if (editingCategory) {
        res = await fetch(`${API_BASE}/categories/${editingCategory.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
      } else {
        res = await fetch(`${API_BASE}/categories`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
      }

      if (!res.ok) throw new Error(`status ${res.status}`);

      await loadCategories();
      closeModal();
    } catch (e) {
      console.warn('Erro ao salvar categoria', e);
      setError('Nao foi possivel salvar a categoria.');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (cat: Category) => {
    if (!confirm(`Deseja realmente excluir a categoria "${cat.name}"?`)) return;

    try {
      const res = await fetch(`${API_BASE}/categories/${cat.id}`, {
        method: 'DELETE',
      });
      if (!res.ok) throw new Error(`status ${res.status}`);
      await loadCategories();
    } catch (e) {
      console.warn('Erro ao excluir categoria', e);
      setError('Nao foi possivel excluir a categoria.');
    }
  };

  const filteredCategories = categories.filter(
    (c) => filterType === 'ALL' || c.type === filterType
  );

  const getTypeLabel = (type: CategoryType) => {
    switch (type) {
      case 'INCOME': return 'Receita';
      case 'EXPENSE': return 'Despesa';
      case 'TRANSFER': return 'Transferencia';
    }
  };

  const getTypeColor = (type: CategoryType) => {
    switch (type) {
      case 'INCOME': return 'bg-emerald-500/20 text-emerald-200 border-emerald-400/40';
      case 'EXPENSE': return 'bg-rose-500/20 text-rose-200 border-rose-400/40';
      case 'TRANSFER': return 'bg-blue-500/20 text-blue-200 border-blue-400/40';
    }
  };

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Categorias</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">
            ← Voltar
          </Link>
        </div>

        {/* Profile selector */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Perfil:</span>
              <div className="flex flex-wrap gap-2">
                {profiles.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => setSelectedProfileId(p.id)}
                    className={`px-3 py-1.5 rounded-xl border transition-colors ${
                      selectedProfileId === p.id
                        ? 'bg-white/20 text-white border-white/40'
                        : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'
                    }`}
                  >
                    {p.name}
                  </button>
                ))}
              </div>
            </div>
            <button
              onClick={openCreateModal}
              className="px-4 py-2 bg-emerald-500/80 hover:bg-emerald-500 text-white rounded-lg font-semibold border border-emerald-400/40 transition-colors"
            >
              + Nova Categoria
            </button>
          </div>
        </div>

        {/* Type filter */}
        <div className="flex flex-wrap gap-2">
          {(['ALL', 'EXPENSE', 'INCOME', 'TRANSFER'] as const).map((type) => (
            <button
              key={type}
              onClick={() => setFilterType(type)}
              className={`px-3 py-1.5 rounded-xl text-sm border transition-colors ${
                filterType === type
                  ? 'bg-white/20 text-white border-white/40'
                  : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'
              }`}
            >
              {type === 'ALL' ? 'Todas' : getTypeLabel(type)}
            </button>
          ))}
        </div>

        {/* Categories list */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          {loading && <p className="text-white/70">Carregando...</p>}
          {error && (
            <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3 mb-3">
              {error}
            </div>
          )}

          {!loading && filteredCategories.length === 0 ? (
            <p className="text-white/70">Nenhuma categoria encontrada. Crie uma nova!</p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {filteredCategories.map((cat) => (
                <div
                  key={cat.id}
                  className="bg-white/5 border border-white/10 rounded-xl p-4 flex items-center justify-between"
                >
                  <div className="flex items-center gap-3">
                    <div
                      className="w-4 h-4 rounded-full"
                      style={{ backgroundColor: cat.color || '#6366f1' }}
                    />
                    <div>
                      <p className="text-white font-medium">{cat.name}</p>
                      <span className={`text-xs px-2 py-0.5 rounded-full border ${getTypeColor(cat.type)}`}>
                        {getTypeLabel(cat.type)}
                      </span>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => openEditModal(cat)}
                      className="p-2 text-white/60 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                      title="Editar"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => handleDelete(cat)}
                      className="p-2 text-white/60 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
                      title="Excluir"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-md mx-4">
            <h3 className="text-xl font-bold text-white mb-4">
              {editingCategory ? 'Editar Categoria' : 'Nova Categoria'}
            </h3>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Nome</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  placeholder="Ex: Alimentacao"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Tipo</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as CategoryType })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                >
                  <option value="EXPENSE" className="bg-slate-900">Despesa</option>
                  <option value="INCOME" className="bg-slate-900">Receita</option>
                  <option value="TRANSFER" className="bg-slate-900">Transferencia</option>
                </select>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-1">Icone</label>
                <select
                  value={formData.icon}
                  onChange={(e) => setFormData({ ...formData, icon: e.target.value })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                >
                  {CATEGORY_ICONS.map((icon) => (
                    <option key={icon.value} value={icon.value} className="bg-slate-900">
                      {icon.label}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm text-white/70 mb-2">Cor</label>
                <div className="flex flex-wrap gap-2">
                  {CATEGORY_COLORS.map((color) => (
                    <button
                      key={color}
                      type="button"
                      onClick={() => setFormData({ ...formData, color })}
                      className={`w-8 h-8 rounded-full border-2 transition-all ${
                        formData.color === color ? 'border-white scale-110' : 'border-transparent'
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
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
                  disabled={saving || !formData.name.trim()}
                  className={`flex-1 px-4 py-2 rounded-lg font-semibold border transition-colors ${
                    saving || !formData.name.trim()
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
