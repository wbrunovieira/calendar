'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import type { Calendar } from '@/types/calendar';

interface CreateReminderModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated: () => void;
  calendars: Calendar[];
  selectedCalendarId?: string | null;
}

const REMINDER_DAYS_OPTIONS = [
  { value: 0, label: 'No dia' },
  { value: 1, label: '1 dia antes' },
  { value: 3, label: '3 dias antes' },
  { value: 7, label: '1 semana antes' },
  { value: 14, label: '2 semanas antes' },
  { value: 30, label: '1 mes antes' },
];

export default function CreateReminderModal({
  isOpen,
  onClose,
  onCreated,
  calendars,
  selectedCalendarId,
}: CreateReminderModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [calendarId, setCalendarId] = useState(selectedCalendarId || '');
  const [reminderDate, setReminderDate] = useState('');
  const [startTime, setStartTime] = useState('09:00');
  const [priority, setPriority] = useState<number | undefined>(undefined);
  const [reminderDaysBefore, setReminderDaysBefore] = useState<number[]>([0]);
  const [loading, setLoading] = useState(false);

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setTitle('');
      setDescription('');
      setCalendarId(selectedCalendarId || '');
      // Default to tomorrow
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      setReminderDate(tomorrow.toISOString().split('T')[0]);
      setStartTime('09:00');
      setPriority(undefined);
      setReminderDaysBefore([0]);
    }
  }, [isOpen, selectedCalendarId]);

  const effectiveCalendarId = selectedCalendarId || calendarId;
  const canCreate = title.trim() !== '' && effectiveCalendarId !== '' && reminderDate !== '';

  const toggleReminderDay = (day: number) => {
    setReminderDaysBefore(prev =>
      prev.includes(day)
        ? prev.filter(d => d !== day)
        : [...prev, day].sort((a, b) => b - a)
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canCreate) return;

    setLoading(true);
    try {
      await api.events.create({
        calendarId: effectiveCalendarId,
        title: title.trim(),
        description: description.trim() || undefined,
        eventType: 'REMINDER',
        startDate: reminderDate,
        startTime,
        priority: priority || undefined,
        reminderDaysBefore: reminderDaysBefore.length > 0 ? reminderDaysBefore : [0],
      });

      onCreated();
      onClose();
    } catch (error) {
      console.error('Error creating reminder:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-[#2a0a3a] rounded-2xl p-6 w-full max-w-md border border-amber-500/30 shadow-xl">
        <h2 className="text-xl font-bold text-white mb-6 flex items-center gap-2">
          <span>🔔</span> Novo Lembrete
        </h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Calendar Selector */}
          {!selectedCalendarId && (
            <div>
              <label htmlFor="calendar" className="block text-sm font-medium text-white/70 mb-1">
                Perfil
              </label>
              <select
                id="calendar"
                value={calendarId}
                onChange={(e) => setCalendarId(e.target.value)}
                className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-amber-500"
              >
                <option value="">Selecione um perfil</option>
                {calendars.map((cal) => (
                  <option key={cal.id} value={cal.id}>
                    {cal.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Title */}
          <div>
            <label htmlFor="title" className="block text-sm font-medium text-white/70 mb-1">
              Titulo
            </label>
            <input
              type="text"
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Ex: Cobrar resposta do cliente"
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-amber-500"
            />
          </div>

          {/* Description */}
          <div>
            <label htmlFor="description" className="block text-sm font-medium text-white/70 mb-1">
              Descricao (opcional)
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-amber-500 resize-none"
            />
          </div>

          {/* Reminder Date */}
          <div>
            <label htmlFor="reminderDate" className="block text-sm font-medium text-white/70 mb-1">
              Data do Lembrete
            </label>
            <input
              type="date"
              id="reminderDate"
              value={reminderDate}
              onChange={(e) => setReminderDate(e.target.value)}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-amber-500"
            />
          </div>

          {/* Time */}
          <div>
            <label htmlFor="startTime" className="block text-sm font-medium text-white/70 mb-1">
              Horario
            </label>
            <input
              type="time"
              id="startTime"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-amber-500"
            />
          </div>

          {/* Priority */}
          <div>
            <label htmlFor="priority" className="block text-sm font-medium text-white/70 mb-1">
              Prioridade
            </label>
            <select
              id="priority"
              value={priority || ''}
              onChange={(e) => setPriority(e.target.value ? Number(e.target.value) : undefined)}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-amber-500"
            >
              <option value="">Sem prioridade</option>
              <option value="1">Alta</option>
              <option value="2">Media</option>
              <option value="3">Baixa</option>
            </select>
          </div>

          {/* Reminder Days Before */}
          <div>
            <label className="block text-sm font-medium text-white/70 mb-2">
              Quando lembrar?
            </label>
            <div className="flex flex-wrap gap-2">
              {REMINDER_DAYS_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => toggleReminderDay(option.value)}
                  className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                    reminderDaysBefore.includes(option.value)
                      ? 'bg-amber-600 text-white'
                      : 'bg-white/10 text-white/70 hover:bg-white/20'
                  }`}
                >
                  {option.label}
                </button>
              ))}
            </div>
            <p className="text-xs text-white/50 mt-2">
              Selecione quando deseja ser lembrado
            </p>
          </div>

          {/* Buttons */}
          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={!canCreate || loading}
              className="flex-1 px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Criando...' : 'Criar Lembrete'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
