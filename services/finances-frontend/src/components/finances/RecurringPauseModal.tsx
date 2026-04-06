'use client';

import { useState } from 'react';
import type { RecurringTransaction } from '@/types/finances';
import { getLocalDateString } from '@/utils/format';

const getDateMonthsFromNow = (months: number) => {
  const date = new Date();
  date.setMonth(date.getMonth() + months);
  return getLocalDateString(date);
};

interface RecurringPauseModalProps {
  pausingItem: RecurringTransaction;
  reviewOnDate: string;
  setReviewOnDate: (date: string) => void;
  onConfirm: () => void | Promise<void>;
  onClose: () => void;
}

export default function RecurringPauseModal({
  pausingItem,
  reviewOnDate,
  setReviewOnDate,
  onConfirm,
  onClose,
}: RecurringPauseModalProps) {
  const [loading, setLoading] = useState(false);

  const handleConfirm = async () => {
    if (loading) return;
    setLoading(true);
    try { await onConfirm(); } finally { setLoading(false); }
  };

  return (
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
            onClick={onClose}
            className="flex-1 px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg border border-white/20 transition-colors"
          >
            Cancelar
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={loading}
            className="flex-1 px-4 py-2 bg-amber-500/80 hover:bg-amber-500 text-white rounded-lg font-semibold border border-amber-400/40 transition-colors disabled:opacity-50"
          >
            {loading ? 'Pausando...' : 'Pausar'}
          </button>
        </div>
      </div>
    </div>
  );
}
