'use client';

import { amountStep } from '@/lib/habits';

export default function HabitAmountStepper({
  value,
  target,
  unit,
  onChange,
}: {
  value: number;
  target: number;
  unit?: string | null;
  onChange: (value: number) => void;
}) {
  const step = amountStep(target);
  const btn =
    'w-7 h-7 rounded-full bg-white/10 hover:bg-white/20 text-white font-bold leading-none disabled:opacity-30';
  return (
    <div className="inline-flex items-center gap-1 border border-white/20 rounded-full p-0.5">
      <button type="button" className={btn} disabled={value <= 0} onClick={() => onChange(Math.max(0, value - step))} aria-label="Menos">
        −
      </button>
      <span className="text-sm text-white/80 px-1 whitespace-nowrap tabular-nums">
        <b className={value >= target ? 'text-emerald-300' : 'text-white'}>{value.toLocaleString('pt-BR')}</b> /{' '}
        {target.toLocaleString('pt-BR')} {unit}
      </span>
      <button type="button" className={btn} onClick={() => onChange(value + step)} aria-label="Mais">
        +
      </button>
    </div>
  );
}
