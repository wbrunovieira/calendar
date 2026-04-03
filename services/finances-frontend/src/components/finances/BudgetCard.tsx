'use client';

import type { BudgetSummaryItem, Category, RecurringTransaction, Transaction } from '@/types/finances';

interface PaceMetrics {
  percentMonth: number;
  percentSpent: number;
  projection: number;
  safePerDay: number;
  daysRemaining: number;
  status: 'on_track' | 'ahead' | 'behind';
  pendingRecurring: number;
}

interface BudgetCardProps {
  summary: BudgetSummaryItem;
  categories: Category[];
  recurrings: RecurringTransaction[];
  transactions: Transaction[];
  period: string;
  pendingRecurring: number;
  pace: PaceMetrics;
  isExpanded: boolean;
  onToggleExpand: () => void;
  onEdit: () => void;
}

export default function BudgetCard({
  summary: s,
  categories,
  recurrings,
  transactions,
  period,
  pendingRecurring: pendingRec,
  pace,
  isExpanded,
  onToggleExpand,
  onEdit,
}: BudgetCardProps) {
  const cat = categories.find((c) => c.id === s.target.categoryId);
  const catName = cat ? cat.name : s.target.categoryId;
  const fmt = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v);

  const statusColorClass = pace.status === 'ahead' ? 'text-emerald-400' : pace.status === 'behind' ? 'text-rose-400' : 'text-amber-400';
  const statusLabel = pace.status === 'ahead' ? 'Abaixo do ritmo' : pace.status === 'behind' ? 'Acima do ritmo' : 'No ritmo';
  const progressColor = pace.status === 'ahead' ? 'bg-emerald-500' : pace.status === 'behind' ? 'bg-rose-500' : 'bg-amber-500';

  // Get transactions and pending recurrings for this category
  const { completedTx, pendingRecurrings } = getTransactionsForCategory(
    s.target.categoryId, period, categories, recurrings, transactions
  );
  const hasItems = completedTx.length > 0 || pendingRecurrings.length > 0;

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <h4 className="text-white font-semibold">{catName}</h4>
          <span className={`text-xs px-2 py-0.5 rounded-full ${statusColorClass} bg-white/10`}>
            {statusLabel}
          </span>
          {s.target.isRecurring && (
            <span className="text-xs px-2 py-0.5 rounded-full text-blue-400 bg-blue-500/20">
              Recorrente
            </span>
          )}
        </div>
        <button
          onClick={onEdit}
          className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10"
        >
          Editar
        </button>
      </div>

      {/* Progress bar */}
      <div className="mb-3">
        <div className="flex justify-between text-xs text-white/60 mb-1">
          <span>{fmt(s.spent)} de {fmt(s.target.amount)}</span>
          <span>{pace.percentSpent}% gasto</span>
        </div>
        <div className="h-2 bg-white/10 rounded-full overflow-hidden flex">
          <div
            className={`h-full ${progressColor} transition-all`}
            style={{ width: `${Math.min(100, pace.percentSpent)}%` }}
          />
          {pendingRec > 0 && (
            <div
              className="h-full bg-blue-500/70 transition-all"
              style={{ width: `${Math.min(100 - pace.percentSpent, (pendingRec / s.target.amount) * 100)}%` }}
            />
          )}
        </div>
        <div className="flex justify-between text-xs text-white/40 mt-1">
          <span>{pace.percentMonth}% do mês</span>
          <span>{pace.daysRemaining} dias restantes</span>
        </div>
      </div>

      {/* Metrics grid */}
      <div className="grid grid-cols-3 gap-4 text-center">
        <div className="bg-white/5 rounded-lg p-2">
          <p className="text-xs text-white/50">Restante</p>
          <p className={`text-sm font-semibold ${s.remaining >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
            {fmt(s.remaining)}
          </p>
        </div>
        <div className="bg-white/5 rounded-lg p-2">
          <p className="text-xs text-white/50">Projeção</p>
          <p className={`text-sm font-semibold ${pace.projection <= s.target.amount ? 'text-emerald-400' : 'text-rose-400'}`}>
            {fmt(pace.projection)}
          </p>
        </div>
        <div className="bg-white/5 rounded-lg p-2">
          <p className="text-xs text-white/50">Seguro/dia</p>
          <p className={`text-sm font-semibold ${pace.safePerDay >= 0 ? 'text-white' : 'text-rose-400'}`}>
            {fmt(pace.safePerDay)}
          </p>
        </div>
      </div>

      {/* Expandable transactions section */}
      {hasItems && (
        <div className="mt-3 pt-3 border-t border-white/10">
          <button
            onClick={onToggleExpand}
            className="flex items-center gap-2 text-xs text-white/60 hover:text-white/80 transition-colors w-full"
          >
            <svg
              className={`w-3 h-3 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            <span>Ver detalhes</span>
            <span className="text-white/40">
              ({completedTx.length} concluido{completedTx.length !== 1 ? 's' : ''}, {pendingRecurrings.length} pendente{pendingRecurrings.length !== 1 ? 's' : ''})
            </span>
          </button>

          <div
            className={`overflow-hidden transition-all duration-300 ease-in-out ${
              isExpanded ? 'max-h-[500px] opacity-100 mt-3' : 'max-h-0 opacity-0'
            }`}
          >
            {completedTx.length > 0 && (
              <div className="mb-3">
                <p className="text-xs text-emerald-400 mb-1">Concluidos</p>
                <div className="space-y-1">
                  {completedTx.map((tx) => (
                    <div key={tx.id} className="flex justify-between text-xs">
                      <span className="text-white/70 truncate flex-1">
                        {tx.description}
                        <span className="text-white/40 ml-1">
                          ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })})
                        </span>
                      </span>
                      <span className="text-emerald-400 ml-2">{fmt(tx.amount)}</span>
                    </div>
                  ))}
                  <div className="flex justify-between text-xs pt-1 border-t border-white/10 mt-1">
                    <span className="text-white/50 font-medium">Total concluidos</span>
                    <span className="text-emerald-400 font-medium">{fmt(completedTx.reduce((sum, tx) => sum + tx.amount, 0))}</span>
                  </div>
                </div>
              </div>
            )}

            {pendingRecurrings.length > 0 && (
              <div className="mb-3">
                <p className="text-xs text-blue-400 mb-1">Fixas pendentes</p>
                <div className="space-y-1">
                  {pendingRecurrings.map((r, idx) => (
                    <div key={idx} className="flex justify-between text-xs">
                      <span className="text-white/70 truncate flex-1">
                        {r.description}
                        <span className="text-white/40 ml-1">(dia {r.day})</span>
                      </span>
                      <span className="text-blue-400 ml-2">{fmt(r.amount)}</span>
                    </div>
                  ))}
                  <div className="flex justify-between text-xs pt-1 border-t border-white/10 mt-1">
                    <span className="text-white/50 font-medium">Total pendentes</span>
                    <span className="text-blue-400 font-medium">{fmt(pendingRecurrings.reduce((sum, r) => sum + r.amount, 0))}</span>
                  </div>
                </div>
              </div>
            )}

            {(completedTx.length > 0 || pendingRecurrings.length > 0) && (
              <div className="flex justify-between text-xs pt-2 border-t border-white/20 mt-2">
                <span className="text-white font-semibold">Total geral</span>
                <span className="text-white font-semibold">
                  {fmt(completedTx.reduce((sum, tx) => sum + tx.amount, 0) + pendingRecurrings.reduce((sum, r) => sum + r.amount, 0))}
                </span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function getTransactionsForCategory(
  budgetCategoryId: string,
  period: string,
  categories: Category[],
  recurrings: RecurringTransaction[],
  transactions: Transaction[],
) {
  const [year, month] = period.split('-').map(Number);
  const today = new Date();
  const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year
    ? today.getDate()
    : 0;

  const getCategoryChain = (categoryId: string): string[] => {
    const chain: string[] = [categoryId];
    let current = categories.find((c) => c.id === categoryId);
    while (current?.parentId) {
      chain.push(current.parentId);
      current = categories.find((c) => c.id === current?.parentId);
    }
    return chain;
  };

  const completedTx = transactions.filter((tx) => {
    if (!tx.categoryId || tx.status !== 'CONFIRMED') return false;
    const chain = getCategoryChain(tx.categoryId);
    return chain.includes(budgetCategoryId);
  });

  const pendingRecurrings: { description: string; amount: number; day: number }[] = [];
  recurrings.forEach((r) => {
    if (r.status !== 'ACTIVE' || r.type !== 'EXPENSE' || !r.categoryId) return;
    const chain = getCategoryChain(r.categoryId);
    if (!chain.includes(budgetCategoryId)) return;

    const rule = new Map<string, string>();
    (r.recurrenceRule || '').split(';').forEach((kv) => {
      const [k, v] = kv.split('=');
      if (k && v) rule.set(k.toUpperCase(), v.toUpperCase());
    });

    const byMonthDay = rule.get('BYMONTHDAY');
    if (!byMonthDay) return;

    const dayOfMonth = parseInt(byMonthDay, 10);
    if (dayOfMonth > currentDay) {
      pendingRecurrings.push({ description: r.description, amount: r.amount, day: dayOfMonth });
    }
  });

  return { completedTx, pendingRecurrings };
}
