'use client';

import type { TransactionType, BankAccount, Category } from '@/types/finances';

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

interface RecurringFormModalProps {
  editingId: string | null;
  formData: FormData;
  setFormData: (data: FormData) => void;
  filteredAccounts: BankAccount[];
  filteredCategories: Category[];
  saving: boolean;
  onSubmit: (e: React.FormEvent) => void;
  onClose: () => void;
}

export default function RecurringFormModal({
  editingId,
  formData,
  setFormData,
  filteredAccounts,
  filteredCategories,
  saving,
  onSubmit,
  onClose,
}: RecurringFormModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-slate-900 border border-white/10 rounded-2xl p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
        <h3 className="text-xl font-bold text-white mb-4">
          {editingId ? 'Editar Recorrente' : 'Nova Transacao Recorrente'}
        </h3>

        <form onSubmit={onSubmit} className="space-y-4">
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
              onClick={onClose}
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
  );
}
