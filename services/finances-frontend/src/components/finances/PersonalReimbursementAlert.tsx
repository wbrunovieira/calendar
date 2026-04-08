'use client';

import { useMemo, useState } from 'react';
import type { Transaction } from '@/types/finances';
import { formatCurrency } from '@/utils/format';

interface Props {
  transactions: Transaction[];
}

export default function PersonalReimbursementAlert({ transactions }: Props) {
  const [dismissed, setDismissed] = useState(false);

  const pending = useMemo(
    () =>
      transactions.filter(
        (t) => t.isPersonalReimbursement && t.type === 'EXPENSE' && t.status !== 'CANCELLED',
      ),
    [transactions],
  );

  if (pending.length === 0 || dismissed) return null;

  const total = pending.reduce((sum, t) => sum + t.amount, 0);

  return (
    <div className="bg-orange-500/15 border border-orange-400/40 rounded-2xl p-4 flex items-center justify-between gap-4">
      <div className="flex items-start gap-3">
        <span className="text-xl mt-0.5">💳</span>
        <div>
          <p className="text-orange-200 font-semibold text-sm">
            {pending.length} despesa{pending.length > 1 ? 's pessoais' : ' pessoal'} aguardando reembolso
          </p>
          <p className="text-orange-300/70 text-xs mt-0.5">
            Total a reembolsar: <span className="font-semibold text-orange-200">{formatCurrency(total)}</span>
          </p>
        </div>
      </div>
      <button
        onClick={() => setDismissed(true)}
        className="text-orange-300/60 hover:text-orange-200 text-lg leading-none flex-shrink-0"
        title="Dispensar"
      >
        ✕
      </button>
    </div>
  );
}
