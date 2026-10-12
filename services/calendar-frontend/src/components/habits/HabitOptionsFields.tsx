'use client';

import type { HabitKind, HabitTimeOfDay } from '@/types/calendar';
import { TIME_OF_DAY } from '@/lib/habits';

interface HabitOptionsFieldsProps {
  habitKind: HabitKind;
  dailyTarget: string;
  targetUnit: string;
  timeOfDay: HabitTimeOfDay;
  onDailyTargetChange: (value: string) => void;
  onTargetUnitChange: (value: string) => void;
  onTimeOfDayChange: (value: HabitTimeOfDay) => void;
}

const segBtn = (on: boolean) =>
  `flex-1 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
    on ? 'bg-purple-600 text-white' : 'bg-white/10 text-white/70 hover:bg-white/20'
  }`;

const input =
  'w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500';

export function HabitKindToggle({
  value,
  onChange,
}: {
  value: HabitKind;
  onChange: (kind: HabitKind) => void;
}) {
  return (
    <div>
      <div className="flex gap-2" role="group" aria-label="Tipo de hábito">
        <button type="button" className={segBtn(value === 'BUILD')} onClick={() => onChange('BUILD')}>
          Construir
        </button>
        <button type="button" className={segBtn(value === 'QUIT')} onClick={() => onChange('QUIT')}>
          Largar
        </button>
      </div>
      {value === 'QUIT' && (
        <p className="mt-2 text-xs text-white/50">
          Todo dia conta como limpo, a não ser que você registre uma recaída. Só precisa abrir o app
          quando algo der errado.
        </p>
      )}
    </div>
  );
}

/** Daily amount + time of day, shared by the create and edit habit modals. */
export default function HabitOptionsFields({
  habitKind,
  dailyTarget,
  targetUnit,
  timeOfDay,
  onDailyTargetChange,
  onTargetUnitChange,
  onTimeOfDayChange,
}: HabitOptionsFieldsProps) {
  return (
    <>
      {habitKind === 'BUILD' && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="dailyTarget" className="block text-sm font-medium text-white/70 mb-1">
              Quantidade diária (opcional)
            </label>
            <input
              id="dailyTarget"
              type="number"
              min="0"
              inputMode="decimal"
              value={dailyTarget}
              onChange={(e) => onDailyTargetChange(e.target.value)}
              placeholder="Ex: 20"
              className={input}
            />
          </div>
          <div>
            <label htmlFor="targetUnit" className="block text-sm font-medium text-white/70 mb-1">
              Unidade
            </label>
            <input
              id="targetUnit"
              type="text"
              value={targetUnit}
              onChange={(e) => onTargetUnitChange(e.target.value)}
              placeholder="Ex: páginas"
              className={input}
            />
          </div>
        </div>
      )}
      <div>
        <span className="block text-sm font-medium text-white/70 mb-2">Período do dia</span>
        <div className="flex gap-2" role="group" aria-label="Período do dia">
          {TIME_OF_DAY.map(({ key, label }) => (
            <button
              key={key}
              type="button"
              className={segBtn(timeOfDay === key)}
              onClick={() => onTimeOfDayChange(key)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>
    </>
  );
}

/** Maps the form strings to the API payload. */
export function habitOptionsPayload(
  habitKind: HabitKind,
  dailyTarget: string,
  targetUnit: string,
  timeOfDay: HabitTimeOfDay,
) {
  const target = Number(dailyTarget);
  const hasTarget = habitKind === 'BUILD' && dailyTarget.trim() !== '' && target > 0;
  return {
    habitKind,
    dailyTarget: hasTarget ? target : null,
    targetUnit: hasTarget ? targetUnit.trim() || null : null,
    timeOfDay,
  };
}
