'use client';

import { useCallback, useEffect, useState } from 'react';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import { api } from '@/lib/api';
import { useToast } from '@/components/ui/Toast';
import { formatCurrency, parseLocalDate } from '@/utils/format';
import type { Goal, GoalPriority, GoalStatus, Category } from '@/types/finances';

const priorityLabel: Record<GoalPriority, string> = {
  HIGH: 'Alta',
  MEDIUM: 'Media',
  LOW: 'Baixa',
};

const priorityColor: Record<GoalPriority, string> = {
  HIGH: 'bg-red-500/20 text-red-400 border-red-500/30',
  MEDIUM: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  LOW: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
};

const statusLabel: Record<GoalStatus, string> = {
  ACTIVE: 'Ativa',
  COMPLETED: 'Concluida',
  CANCELLED: 'Cancelada',
};

const statusColor: Record<GoalStatus, string> = {
  ACTIVE: 'bg-emerald-500/20 text-emerald-400',
  COMPLETED: 'bg-blue-500/20 text-blue-400',
  CANCELLED: 'bg-gray-500/20 text-gray-400',
};

interface FormData {
  name: string;
  description: string;
  targetAmount: string;
  priority: GoalPriority;
  targetDate: string;
  link: string;
  categoryId: string;
}

const emptyForm: FormData = {
  name: '',
  description: '',
  targetAmount: '',
  priority: 'MEDIUM',
  targetDate: '',
  link: '',
  categoryId: '',
};

export default function GoalsPage() {
  const { selectedProfileId, selectedProfile } = useProfile();
  const { toast, confirm } = useToast();
  const [goals, setGoals] = useState<Goal[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormData>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [addAmountId, setAddAmountId] = useState<string | null>(null);
  const [addAmountValue, setAddAmountValue] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const fetchGoals = useCallback(async () => {
    if (!selectedProfileId) return;
    setLoading(true);
    try {
      const data = await api.get<{ data: Goal[] }>(`/goals?profileId=${selectedProfileId}`);
      setGoals(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar metas', e);
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId]);

  const fetchCategories = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      const data = await api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}`);
      setCategories(data.data || []);
    } catch (e) {
      console.warn('Erro ao carregar categorias', e);
    }
  }, [selectedProfileId]);

  useEffect(() => {
    fetchGoals();
    fetchCategories();
  }, [fetchGoals, fetchCategories]);

  const handleSubmit = async () => {
    if (!selectedProfileId || !form.name || !form.targetAmount) return;
    setSaving(true);
    try {
      const body = {
        profileId: selectedProfileId,
        name: form.name,
        description: form.description,
        targetAmount: parseFloat(form.targetAmount),
        priority: form.priority,
        targetDate: form.targetDate || undefined,
        link: form.link || undefined,
        categoryId: form.categoryId || undefined,
      };

      if (editingId) {
        await api.put(`/goals/${editingId}`, body);
      } else {
        await api.post('/goals', body);
      }

      setShowForm(false);
      setEditingId(null);
      setForm(emptyForm);
      fetchGoals();
    } catch (e) {
      console.warn('Erro ao salvar meta', e);
    } finally {
      setSaving(false);
    }
  };

  const handleEdit = (goal: Goal) => {
    setForm({
      name: goal.name,
      description: goal.description,
      targetAmount: String(goal.targetAmount),
      priority: goal.priority,
      targetDate: goal.targetDate ? goal.targetDate.split('T')[0] : '',
      link: goal.link || '',
      categoryId: goal.categoryId || '',
    });
    setEditingId(goal.id);
    setShowForm(true);
  };

  const handleDelete = async (id: string) => {
    if (deletingId) return;
    const ok = await confirm('Tem certeza que deseja excluir esta meta?');
    if (!ok) return;
    setDeletingId(id);
    try {
      await api.delete(`/goals/${id}`);
      fetchGoals();
      toast('Meta excluida com sucesso');
    } catch (e) {
      console.warn('Erro ao excluir', e);
      toast('Erro ao excluir meta', 'error');
    } finally {
      setDeletingId(null);
    }
  };

  const handleStatusChange = async (id: string, status: GoalStatus) => {
    try {
      await api.patch(`/goals/${id}/status`, { status });
      fetchGoals();
    } catch (e) {
      console.warn('Erro ao atualizar status', e);
    }
  };

  const handleAddAmount = async () => {
    if (!addAmountId || !addAmountValue) return;
    try {
      await api.post(`/goals/${addAmountId}/add-amount`, { amount: parseFloat(addAmountValue) });
      setAddAmountId(null);
      setAddAmountValue('');
      fetchGoals();
    } catch (e) {
      console.warn('Erro ao adicionar valor', e);
    }
  };

  const activeGoals = goals.filter((g) => g.status === 'ACTIVE');
  const completedGoals = goals.filter((g) => g.status === 'COMPLETED');
  const cancelledGoals = goals.filter((g) => g.status === 'CANCELLED');

  const totalTarget = activeGoals.reduce((sum, g) => sum + g.targetAmount, 0);
  const totalSaved = activeGoals.reduce((sum, g) => sum + g.currentAmount, 0);

  return (
    <AppLayout>
      <div className="max-w-5xl mx-auto p-4 sm:p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-white">Metas</h1>
            <p className="text-white/50 text-sm">
              {selectedProfile?.name || 'Selecione um perfil'}
            </p>
          </div>
          <button
            onClick={() => {
              setForm(emptyForm);
              setEditingId(null);
              setShowForm(true);
            }}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm font-medium transition-colors"
          >
            + Nova Meta
          </button>
        </div>

        {/* Summary Cards */}
        {activeGoals.length > 0 && (
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-white/5 border border-white/10 rounded-xl p-4">
              <p className="text-white/50 text-xs mb-1">Total Metas</p>
              <p className="text-white font-bold text-lg">{formatCurrency(totalTarget)}</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-xl p-4">
              <p className="text-white/50 text-xs mb-1">Guardado</p>
              <p className="text-emerald-400 font-bold text-lg">{formatCurrency(totalSaved)}</p>
            </div>
            <div className="bg-white/5 border border-white/10 rounded-xl p-4">
              <p className="text-white/50 text-xs mb-1">Faltam</p>
              <p className="text-yellow-400 font-bold text-lg">
                {formatCurrency(totalTarget - totalSaved)}
              </p>
            </div>
          </div>
        )}

        {/* Loading */}
        {loading && <p className="text-white/50 text-center py-8">Carregando...</p>}

        {/* Empty State */}
        {!loading && goals.length === 0 && (
          <div className="text-center py-12 bg-white/5 border border-white/10 rounded-xl">
            <p className="text-white/50 mb-2">Nenhuma meta cadastrada</p>
            <p className="text-white/30 text-sm">
              Adicione metas para acompanhar seus objetivos de compra
            </p>
          </div>
        )}

        {/* Active Goals */}
        {activeGoals.length > 0 && (
          <div className="space-y-3">
            <h2 className="text-white/70 text-sm font-medium uppercase tracking-wider">
              Ativas ({activeGoals.length})
            </h2>
            {activeGoals.map((goal) => (
              <GoalCard
                key={goal.id}
                goal={goal}
                categories={categories}
                onEdit={handleEdit}
                onDelete={handleDelete}
                deletingId={deletingId}
                onStatusChange={handleStatusChange}
                onAddAmount={(id) => {
                  setAddAmountId(id);
                  setAddAmountValue('');
                }}
              />
            ))}
          </div>
        )}

        {/* Completed Goals */}
        {completedGoals.length > 0 && (
          <div className="space-y-3">
            <h2 className="text-white/70 text-sm font-medium uppercase tracking-wider">
              Concluidas ({completedGoals.length})
            </h2>
            {completedGoals.map((goal) => (
              <GoalCard
                key={goal.id}
                goal={goal}
                categories={categories}
                onEdit={handleEdit}
                onDelete={handleDelete}
                deletingId={deletingId}
                onStatusChange={handleStatusChange}
                onAddAmount={() => {}}
              />
            ))}
          </div>
        )}

        {/* Cancelled Goals */}
        {cancelledGoals.length > 0 && (
          <div className="space-y-3">
            <h2 className="text-white/70 text-sm font-medium uppercase tracking-wider">
              Canceladas ({cancelledGoals.length})
            </h2>
            {cancelledGoals.map((goal) => (
              <GoalCard
                key={goal.id}
                goal={goal}
                categories={categories}
                onEdit={handleEdit}
                onDelete={handleDelete}
                deletingId={deletingId}
                onStatusChange={handleStatusChange}
                onAddAmount={() => {}}
              />
            ))}
          </div>
        )}

        {/* Add Amount Modal */}
        {addAmountId && (
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4"
            onClick={() => setAddAmountId(null)}
          >
            <div
              className="bg-gray-900 border border-white/10 rounded-xl p-6 w-full max-w-sm space-y-4"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-white font-semibold">Adicionar Valor</h3>
              <input
                type="number"
                step="0.01"
                min="0.01"
                value={addAmountValue}
                onChange={(e) => setAddAmountValue(e.target.value)}
                placeholder="Valor guardado"
                className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white placeholder-white/30"
                autoFocus
              />
              <div className="flex gap-2">
                <button
                  onClick={() => setAddAmountId(null)}
                  className="flex-1 px-3 py-2 bg-white/5 text-white/70 rounded-lg text-sm"
                >
                  Cancelar
                </button>
                <button
                  onClick={handleAddAmount}
                  disabled={!addAmountValue || parseFloat(addAmountValue) <= 0}
                  className="flex-1 px-3 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium"
                >
                  Adicionar
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Form Modal */}
        {showForm && (
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4"
            onClick={() => setShowForm(false)}
          >
            <div
              className="bg-gray-900 border border-white/10 rounded-xl p-6 w-full max-w-lg space-y-4 max-h-[90vh] overflow-y-auto"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-white font-semibold text-lg">
                {editingId ? 'Editar Meta' : 'Nova Meta'}
              </h3>

              <div className="space-y-3">
                <div>
                  <label className="text-white/50 text-xs mb-1 block">Nome *</label>
                  <input
                    type="text"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    placeholder="Ex: MacBook Pro, iPhone, Viagem..."
                    className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white placeholder-white/30"
                    autoFocus
                  />
                </div>

                <div>
                  <label className="text-white/50 text-xs mb-1 block">Descricao</label>
                  <textarea
                    value={form.description}
                    onChange={(e) => setForm({ ...form, description: e.target.value })}
                    placeholder="Detalhes sobre a meta..."
                    rows={2}
                    className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white placeholder-white/30 resize-none"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-white/50 text-xs mb-1 block">Valor *</label>
                    <input
                      type="number"
                      step="0.01"
                      min="0.01"
                      value={form.targetAmount}
                      onChange={(e) => setForm({ ...form, targetAmount: e.target.value })}
                      placeholder="0,00"
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white placeholder-white/30"
                    />
                  </div>
                  <div>
                    <label className="text-white/50 text-xs mb-1 block">Prioridade</label>
                    <select
                      value={form.priority}
                      onChange={(e) =>
                        setForm({ ...form, priority: e.target.value as GoalPriority })
                      }
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white"
                    >
                      <option value="HIGH">Alta</option>
                      <option value="MEDIUM">Media</option>
                      <option value="LOW">Baixa</option>
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-white/50 text-xs mb-1 block">Previsao</label>
                    <input
                      type="date"
                      value={form.targetDate}
                      onChange={(e) => setForm({ ...form, targetDate: e.target.value })}
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white"
                    />
                  </div>
                  <div>
                    <label className="text-white/50 text-xs mb-1 block">Categoria</label>
                    <select
                      value={form.categoryId}
                      onChange={(e) => setForm({ ...form, categoryId: e.target.value })}
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white"
                    >
                      <option value="">Nenhuma</option>
                      {categories.map((cat) => (
                        <option key={cat.id} value={cat.id}>
                          {cat.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div>
                  <label className="text-white/50 text-xs mb-1 block">Link do Produto</label>
                  <input
                    type="url"
                    value={form.link}
                    onChange={(e) => setForm({ ...form, link: e.target.value })}
                    placeholder="https://..."
                    className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-white placeholder-white/30"
                  />
                </div>
              </div>

              <div className="flex gap-2 pt-2">
                <button
                  onClick={() => {
                    setShowForm(false);
                    setEditingId(null);
                  }}
                  className="flex-1 px-3 py-2 bg-white/5 text-white/70 rounded-lg text-sm"
                >
                  Cancelar
                </button>
                <button
                  onClick={handleSubmit}
                  disabled={saving || !form.name || !form.targetAmount}
                  className="flex-1 px-3 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium"
                >
                  {saving ? 'Salvando...' : editingId ? 'Salvar' : 'Criar'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}

function GoalCard({
  goal,
  categories,
  onEdit,
  onDelete,
  onStatusChange,
  onAddAmount,
  deletingId,
}: {
  goal: Goal;
  categories: Category[];
  onEdit: (g: Goal) => void;
  onDelete: (id: string) => void;
  onStatusChange: (id: string, status: GoalStatus) => void;
  onAddAmount: (id: string) => void;
  deletingId?: string | null;
}) {
  const progress = goal.targetAmount > 0 ? (goal.currentAmount / goal.targetAmount) * 100 : 0;
  const remaining = goal.targetAmount - goal.currentAmount;
  const category = categories.find((c) => c.id === goal.categoryId);
  const isActive = goal.status === 'ACTIVE';

  const daysLeft =
    goal.targetDate
      ? Math.ceil(
          (parseLocalDate(goal.targetDate).getTime() - Date.now()) / (1000 * 60 * 60 * 24),
        )
      : null;

  return (
    <div
      className={`bg-white/5 border rounded-xl p-4 ${
        isActive ? 'border-white/10' : 'border-white/5 opacity-60'
      }`}
    >
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <h3 className="text-white font-medium truncate">{goal.name}</h3>
            <span
              className={`text-[10px] px-1.5 py-0.5 rounded border ${priorityColor[goal.priority]}`}
            >
              {priorityLabel[goal.priority]}
            </span>
            {goal.status !== 'ACTIVE' && (
              <span className={`text-[10px] px-1.5 py-0.5 rounded ${statusColor[goal.status]}`}>
                {statusLabel[goal.status]}
              </span>
            )}
          </div>
          {goal.description && (
            <p className="text-white/40 text-xs truncate">{goal.description}</p>
          )}
          <div className="flex items-center gap-3 mt-1 text-xs text-white/40">
            {category && <span>{category.name}</span>}
            {goal.targetDate && (
              <span>
                {daysLeft !== null && daysLeft >= 0
                  ? `${daysLeft} dias`
                  : daysLeft !== null
                    ? 'Atrasada'
                    : ''}{' '}
                - {parseLocalDate(goal.targetDate).toLocaleDateString('pt-BR')}
              </span>
            )}
            {goal.link && (
              <a
                href={goal.link}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-400 hover:underline"
              >
                Ver produto
              </a>
            )}
          </div>
        </div>
        <div className="text-right shrink-0">
          <p className="text-white font-bold">{formatCurrency(goal.targetAmount)}</p>
          {goal.currentAmount > 0 && (
            <p className="text-emerald-400 text-xs">{formatCurrency(goal.currentAmount)} guardado</p>
          )}
        </div>
      </div>

      {/* Progress bar */}
      {isActive && (
        <div className="mb-3">
          <div className="flex justify-between text-xs text-white/40 mb-1">
            <span>{progress.toFixed(0)}%</span>
            <span>Faltam {formatCurrency(remaining > 0 ? remaining : 0)}</span>
          </div>
          <div className="h-2 bg-white/5 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${
                progress >= 100
                  ? 'bg-emerald-500'
                  : progress >= 50
                    ? 'bg-yellow-500'
                    : 'bg-blue-500'
              }`}
              style={{ width: `${Math.min(progress, 100)}%` }}
            />
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2">
        {isActive && (
          <>
            <button
              onClick={() => onAddAmount(goal.id)}
              className="px-2.5 py-1 bg-emerald-600/20 text-emerald-400 hover:bg-emerald-600/30 rounded text-xs transition-colors"
            >
              + Guardar
            </button>
            <button
              onClick={() => onStatusChange(goal.id, 'COMPLETED')}
              className="px-2.5 py-1 bg-blue-600/20 text-blue-400 hover:bg-blue-600/30 rounded text-xs transition-colors"
            >
              Concluir
            </button>
            <button
              onClick={() => onStatusChange(goal.id, 'CANCELLED')}
              className="px-2.5 py-1 bg-gray-600/20 text-gray-400 hover:bg-gray-600/30 rounded text-xs transition-colors"
            >
              Cancelar
            </button>
          </>
        )}
        <div className="flex-1" />
        <button
          onClick={() => onEdit(goal)}
          className="px-2.5 py-1 bg-white/5 text-white/50 hover:text-white hover:bg-white/10 rounded text-xs transition-colors"
        >
          Editar
        </button>
        <button
          onClick={() => onDelete(goal.id)}
          disabled={!!deletingId}
          className="px-2.5 py-1 bg-red-600/10 text-red-400/50 hover:text-red-400 hover:bg-red-600/20 rounded text-xs transition-colors disabled:opacity-50"
        >
          {deletingId === goal.id ? 'Excluindo...' : 'Excluir'}
        </button>
      </div>
    </div>
  );
}
