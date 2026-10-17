'use client';

import { useState } from 'react';
import type { Event, HabitStats } from '@/types/calendar';
import { SLIP_TRIGGERS, triggerLabel } from '@/lib/habits';
import HabitDetailPanel from './HabitDetailPanel';

interface QuitHabitCardProps {
  habit: Event;
  stats?: HabitStats;
  expanded: boolean;
  onToggleExpanded: () => void;
  onEdit: () => void;
  onSlip: (trigger?: string, note?: string) => Promise<void>;
  onUndoSlip: () => Promise<void>;
  onTogglePause: () => void;
}

/** A habit being quit: big clean-day count, slip logging with a trigger, and undo. */
export default function QuitHabitCard({
  habit,
  stats,
  expanded,
  onToggleExpanded,
  onEdit,
  onSlip,
  onUndoSlip,
  onTogglePause,
}: QuitHabitCardProps) {
  const [logging, setLogging] = useState(false);
  const [trigger, setTrigger] = useState<string | undefined>();
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);

  const clean = stats?.currentStreak ?? 0;
  const best = stats?.longestStreak ?? 0;
  const slips = stats?.slipsLast30Days ?? 0;
  const todaySlip = stats?.todaySlip;

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    try {
      await fn();
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-5 border border-white/20">
      <div className="flex items-start justify-between gap-2">
        <button type="button" onClick={onToggleExpanded} className="text-left font-semibold text-white hover:underline">
          {habit.title}
        </button>
        <button
          type="button"
          onClick={onEdit}
          className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg"
          aria-label={`Editar ${habit.title}`}
        >
          ✎
        </button>
      </div>

      <p className="mt-2 text-white/60 text-sm">
        <b className={`text-4xl font-extrabold mr-1 ${todaySlip ? 'text-red-300' : 'text-emerald-300'}`}>{clean}</b>
        {clean === 1 ? 'dia limpo' : 'dias limpos'}
      </p>
      <p className="text-xs text-white/50 mt-1">
        Melhor: {best} {best === 1 ? 'dia' : 'dias'}. {slips} {slips === 1 ? 'recaída' : 'recaídas'} nos últimos 30
        dias{stats?.topTrigger ? `, mais por ${triggerLabel(stats.topTrigger).toLowerCase()}` : ''}.
      </p>

      {logging ? (
        <div className="mt-4 rounded-xl bg-black/20 p-3">
          <p className="text-sm font-semibold text-white">O que levou a isso?</p>
          <div className="flex flex-wrap gap-1.5 mt-2">
            {SLIP_TRIGGERS.map((t) => (
              <button
                key={t.value}
                type="button"
                onClick={() => setTrigger(trigger === t.value ? undefined : t.value)}
                className={`px-3 py-1 rounded-full text-xs border ${
                  trigger === t.value
                    ? 'bg-white text-[#350545] border-white'
                    : 'border-white/30 text-white/70 hover:border-white/60'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
          <input
            type="text"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Nota (opcional)"
            className="mt-2 w-full bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-sm text-white placeholder-white/40"
          />
          <div className="flex gap-2 mt-2">
            <button
              type="button"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await onSlip(trigger, note.trim() || undefined);
                  setLogging(false);
                  setTrigger(undefined);
                  setNote('');
                })
              }
              className="px-3 py-1.5 rounded-lg text-sm bg-red-500/80 text-white hover:bg-red-500 disabled:opacity-50"
            >
              Registrar recaída
            </button>
            <button
              type="button"
              onClick={() => setLogging(false)}
              className="px-3 py-1.5 rounded-lg text-sm bg-white/10 text-white/70 hover:bg-white/20"
            >
              Cancelar
            </button>
          </div>
          <p className="text-xs text-white/40 mt-2">
            A recaída zera a contagem. Sua melhor sequência e o histórico continuam.
          </p>
        </div>
      ) : todaySlip ? (
        <div className="mt-4 flex items-center gap-3 text-sm text-white/70">
          <span>
            Recaída hoje{todaySlip.trigger ? ` (${triggerLabel(todaySlip.trigger).toLowerCase()})` : ''}.
          </span>
          <button type="button" disabled={busy} onClick={() => run(onUndoSlip)} className="text-purple-300 hover:text-white">
            Desfazer
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setLogging(true)}
          className="mt-4 px-3 py-1.5 rounded-lg text-sm bg-white/10 text-white/80 hover:bg-white/20"
        >
          Recaí hoje
        </button>
      )}

      {expanded && stats && <HabitDetailPanel stats={stats} onTogglePause={onTogglePause} />}
    </div>
  );
}
