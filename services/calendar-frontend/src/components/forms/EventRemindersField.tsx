'use client';

import { useState } from 'react';
import { ReminderInput } from '@/types/calendar';

const PRESETS = [
  { label: 'Na hora do evento', value: 0 },
  { label: '5 minutos antes', value: 5 },
  { label: '10 minutos antes', value: 10 },
  { label: '15 minutos antes', value: 15 },
  { label: '30 minutos antes', value: 30 },
  { label: '1 hora antes', value: 60 },
  { label: '1 dia antes', value: 1440 },
];

function formatMinutes(minutes: number): string {
  const preset = PRESETS.find(p => p.value === minutes);
  if (preset) return preset.label;
  if (minutes < 60) return `${minutes} minutos antes`;
  if (minutes < 1440) {
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    return mins > 0 ? `${hours}h${mins}min antes` : `${hours} hora${hours > 1 ? 's' : ''} antes`;
  }
  const days = Math.floor(minutes / 1440);
  return `${days} dia${days > 1 ? 's' : ''} antes`;
}

interface EventRemindersFieldProps {
  reminders: ReminderInput[];
  onChange: (reminders: ReminderInput[]) => void;
}

export default function EventRemindersField({ reminders, onChange }: EventRemindersFieldProps) {
  const [showCustom, setShowCustom] = useState(false);
  const [customValue, setCustomValue] = useState(15);
  const [customUnit, setCustomUnit] = useState<'minutes' | 'hours' | 'days'>('minutes');

  const addPreset = (minutesBefore: number) => {
    if (reminders.some(r => r.minutesBefore === minutesBefore)) return;
    onChange([...reminders, { minutesBefore, method: 'notification' }]);
  };

  const addCustom = () => {
    let minutes = customValue;
    if (customUnit === 'hours') minutes = customValue * 60;
    if (customUnit === 'days') minutes = customValue * 1440;
    if (reminders.some(r => r.minutesBefore === minutes)) {
      setShowCustom(false);
      return;
    }
    onChange([...reminders, { minutesBefore: minutes, method: 'notification' }]);
    setShowCustom(false);
  };

  const remove = (index: number) => {
    onChange(reminders.filter((_, i) => i !== index));
  };

  return (
    <div className="border-t border-white/10 pt-4">
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Alertas
      </label>

      {/* Current reminders */}
      {reminders.length > 0 && (
        <div className="space-y-1 mb-2">
          {reminders.map((r, i) => (
            <div key={i} className="flex items-center justify-between bg-white/5 rounded-lg px-3 py-2 text-sm">
              <span className="text-gray-300">
                <span className="mr-2">&#128276;</span>
                {formatMinutes(r.minutesBefore)}
              </span>
              <button
                type="button"
                onClick={() => remove(i)}
                className="text-gray-500 hover:text-red-400 transition-colors ml-2"
              >
                &#10005;
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add reminder */}
      {!showCustom ? (
        <div className="flex flex-wrap gap-1">
          <select
            className="bg-white/10 border border-white/20 rounded-lg px-2 py-1.5 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-[#792990]"
            value=""
            onChange={(e) => {
              const val = e.target.value;
              if (val === 'custom') {
                setShowCustom(true);
              } else {
                addPreset(parseInt(val, 10));
              }
            }}
          >
            <option value="" disabled>+ Adicionar alerta</option>
            {PRESETS.map(p => (
              <option key={p.value} value={p.value} disabled={reminders.some(r => r.minutesBefore === p.value)}>
                {p.label}
              </option>
            ))}
            <option value="custom">Personalizado...</option>
          </select>
        </div>
      ) : (
        <div className="flex items-center gap-2 bg-white/5 rounded-lg p-2">
          <input
            type="number"
            min={1}
            value={customValue}
            onChange={e => setCustomValue(parseInt(e.target.value, 10) || 1)}
            className="w-16 bg-white/10 border border-white/20 rounded px-2 py-1 text-sm text-white focus:outline-none focus:ring-2 focus:ring-[#792990]"
          />
          <select
            value={customUnit}
            onChange={e => setCustomUnit(e.target.value as 'minutes' | 'hours' | 'days')}
            className="bg-white/10 border border-white/20 rounded px-2 py-1 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-[#792990]"
          >
            <option value="minutes">minutos</option>
            <option value="hours">horas</option>
            <option value="days">dias</option>
          </select>
          <span className="text-sm text-gray-400">antes</span>
          <button
            type="button"
            onClick={addCustom}
            className="text-sm text-[#792990] hover:text-[#9b3db5] font-medium"
          >
            OK
          </button>
          <button
            type="button"
            onClick={() => setShowCustom(false)}
            className="text-sm text-gray-500 hover:text-gray-300"
          >
            Cancelar
          </button>
        </div>
      )}
    </div>
  );
}
