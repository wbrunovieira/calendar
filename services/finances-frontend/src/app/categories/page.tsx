'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { Category, CategoryType } from '@/types/finances';
import { useToast } from '@/components/ui/Toast';

interface CategoryFormData {
  name: string;
  type: CategoryType;
  color: string;
  icon: string;
  parentId: string | null;
}

const CATEGORY_COLORS = [
  '#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16',
  '#22c55e', '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9',
  '#3b82f6', '#6366f1', '#8b5cf6', '#a855f7', '#d946ef',
  '#ec4899', '#f43f5e',
];

const CATEGORY_ICONS = [
  { value: 'shopping', label: '🛍️ Compras' },
  { value: 'food', label: '🍽️ Alimentacao' },
  { value: 'delivery', label: '🛵 Delivery' },
  { value: 'supermarket', label: '🛒 Supermercado' },
  { value: 'restaurant', label: '🍴 Restaurante' },
  { value: 'transport', label: '🚗 Transporte' },
  { value: 'fuel', label: '⛽ Combustivel' },
  { value: 'health', label: '🏥 Saude' },
  { value: 'education', label: '📚 Educacao' },
  { value: 'entertainment', label: '🎮 Lazer' },
  { value: 'home', label: '🏠 Casa' },
  { value: 'utilities', label: '💡 Contas' },
  { value: 'work', label: '💼 Trabalho' },
  { value: 'salary', label: '💵 Salario' },
  { value: 'investment', label: '📈 Investimento' },
  { value: 'gift', label: '🎁 Presente' },
  { value: 'subscription', label: '📺 Assinatura' },
  { value: 'travel', label: '✈️ Viagem' },
  { value: 'pet', label: '🐕 Pet' },
  { value: 'clothing', label: '👕 Roupas' },
  { value: 'beauty', label: '💄 Beleza' },
  { value: 'drinks', label: '🍺 Bebidas' },
  { value: 'coffee', label: '☕ Cafe' },
  { value: 'other', label: '📦 Outros' },
];

export default function CategoriesPage() {
  const { selectedProfileId } = useProfile();
  const { toast, confirm } = useToast();
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
    parentId: null,
  });
  const [saving, setSaving] = useState(false);

  // Filter state
  const [filterType, setFilterType] = useState<CategoryType | 'ALL'>('ALL');

  const loadCategories = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      setLoading(true);
      setError(null);
      const data = await api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}`);
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

  // Get parent categories (categories without parentId)
  const parentCategories = useMemo(
    () => categories.filter((c) => !c.parentId),
    [categories]
  );

  // Get subcategories for a parent
  const getSubcategories = useCallback(
    (parentId: string) => categories.filter((c) => c.parentId === parentId),
    [categories]
  );

  // Hierarchical structure for display
  const hierarchicalCategories = useMemo(() => {
    const filtered = filterType === 'ALL'
      ? parentCategories
      : parentCategories.filter((c) => c.type === filterType);

    return filtered.map((parent) => ({
      ...parent,
      children: getSubcategories(parent.id),
    }));
  }, [parentCategories, getSubcategories, filterType]);

  const openCreateModal = (parentId: string | null = null) => {
    const parent = parentId ? categories.find((c) => c.id === parentId) : null;
    setEditingCategory(null);
    setFormData({
      name: '',
      type: parent?.type || 'EXPENSE',
      color: parent?.color || CATEGORY_COLORS[0],
      icon: 'other',
      parentId,
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
      parentId: cat.parentId || null,
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
      const payload: Record<string, unknown> = {
        profileId: selectedProfileId,
        name: formData.name.trim(),
        type: formData.type,
        color: formData.color,
        icon: formData.icon,
      };

      // Only include parentId if it's set
      if (formData.parentId) {
        payload.parentId = formData.parentId;
      }

      if (editingCategory) {
        await api.put(`/categories/${editingCategory.id}`, payload);
      } else {
        await api.post('/categories', payload);
      }

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
    const subcats = getSubcategories(cat.id);
    const msg = subcats.length > 0
      ? `A categoria "${cat.name}" tem ${subcats.length} subcategoria(s). Deseja excluir mesmo assim?`
      : `Deseja realmente excluir a categoria "${cat.name}"?`;

    const ok = await confirm(msg);
    if (!ok) return;

    try {
      await api.delete(`/categories/${cat.id}`);
      await loadCategories();
      toast('Categoria excluida com sucesso');
    } catch (e) {
      console.warn('Erro ao excluir categoria', e);
      toast('Nao foi possivel excluir a categoria', 'error');
      setError('Nao foi possivel excluir a categoria.');
    }
  };

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

  const getIconEmoji = (iconValue: string) => {
    const icon = CATEGORY_ICONS.find((i) => i.value === iconValue);
    return icon?.label.split(' ')[0] || '📦';
  };


  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">Categorias</h2>
            <p className="text-white/60 text-sm">Organize suas receitas e despesas com categorias e subcategorias</p>
          </div>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">
            ← Voltar
          </Link>
        </div>

        {/* Actions */}
        <div className="flex justify-end">
          <button
            onClick={() => openCreateModal(null)}
            className="px-4 py-2 bg-emerald-500/80 hover:bg-emerald-500 text-white rounded-lg font-semibold border border-emerald-400/40 transition-colors"
          >
            + Nova Categoria
          </button>
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
        <div className="space-y-4">
          {loading && <p className="text-white/70">Carregando...</p>}
          {error && (
            <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3">
              {error}
            </div>
          )}

          {!loading && hierarchicalCategories.length === 0 ? (
            <div className="bg-white/5 border border-white/10 rounded-2xl p-8 text-center">
              <p className="text-white/70 mb-4">Nenhuma categoria encontrada. Crie uma nova!</p>
              <button
                onClick={() => openCreateModal(null)}
                className="px-4 py-2 bg-emerald-500/80 hover:bg-emerald-500 text-white rounded-lg font-semibold border border-emerald-400/40 transition-colors"
              >
                + Criar Primeira Categoria
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              {hierarchicalCategories.map((parent) => (
                <div
                  key={parent.id}
                  className="bg-white/5 border border-white/10 rounded-2xl overflow-hidden"
                >
                  {/* Parent category */}
                  <div className="p-4 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
                        style={{ backgroundColor: `${parent.color}20` }}
                      >
                        {getIconEmoji(parent.icon || 'other')}
                      </div>
                      <div>
                        <p className="text-white font-semibold">{parent.name}</p>
                        <div className="flex items-center gap-2">
                          <span className={`text-xs px-2 py-0.5 rounded-full border ${getTypeColor(parent.type)}`}>
                            {getTypeLabel(parent.type)}
                          </span>
                          {parent.children.length > 0 && (
                            <span className="text-white/50 text-xs">
                              {parent.children.length} subcategoria{parent.children.length > 1 ? 's' : ''}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => openCreateModal(parent.id)}
                        className="px-3 py-1.5 text-xs text-emerald-300 hover:text-emerald-200 hover:bg-emerald-500/20 rounded-lg transition-colors border border-emerald-500/30"
                        title="Adicionar subcategoria"
                      >
                        + Sub
                      </button>
                      <button
                        onClick={() => openEditModal(parent)}
                        className="p-2 text-white/60 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                        title="Editar"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => handleDelete(parent)}
                        className="p-2 text-white/60 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
                        title="Excluir"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </div>

                  {/* Subcategories */}
                  {parent.children.length > 0 && (
                    <div className="border-t border-white/10 bg-white/[0.02]">
                      {parent.children.map((sub) => (
                        <div
                          key={sub.id}
                          className="px-4 py-3 flex items-center justify-between border-b border-white/5 last:border-b-0"
                        >
                          <div className="flex items-center gap-3 pl-6">
                            <div className="w-2 h-2 rounded-full" style={{ backgroundColor: sub.color || parent.color }} />
                            <span className="text-lg">{getIconEmoji(sub.icon || 'other')}</span>
                            <span className="text-white/80">{sub.name}</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => openEditModal(sub)}
                              className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                              title="Editar"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                              </svg>
                            </button>
                            <button
                              onClick={() => handleDelete(sub)}
                              className="p-1.5 text-white/40 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
                              title="Excluir"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
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
              {editingCategory
                ? 'Editar Categoria'
                : formData.parentId
                ? 'Nova Subcategoria'
                : 'Nova Categoria'}
            </h3>

            {formData.parentId && (
              <div className="mb-4 p-3 bg-white/5 rounded-lg border border-white/10">
                <span className="text-white/60 text-sm">Subcategoria de: </span>
                <span className="text-white font-medium">
                  {categories.find((c) => c.id === formData.parentId)?.name}
                </span>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-white/70 mb-1">Nome</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  placeholder={formData.parentId ? 'Ex: iFood' : 'Ex: Alimentacao'}
                  required
                />
              </div>

              {/* Only show type selector for parent categories */}
              {!formData.parentId && (
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
              )}

              <div>
                <label className="block text-sm text-white/70 mb-1">Icone</label>
                <div className="grid grid-cols-6 gap-2 max-h-32 overflow-y-auto p-2 bg-white/5 rounded-lg">
                  {CATEGORY_ICONS.map((icon) => (
                    <button
                      key={icon.value}
                      type="button"
                      onClick={() => setFormData({ ...formData, icon: icon.value })}
                      className={`p-2 rounded-lg text-xl transition-all ${
                        formData.icon === icon.value
                          ? 'bg-white/20 ring-2 ring-emerald-400'
                          : 'hover:bg-white/10'
                      }`}
                      title={icon.label}
                    >
                      {icon.label.split(' ')[0]}
                    </button>
                  ))}
                </div>
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
