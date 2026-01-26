'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import type { Calendar, Category } from '@/types/calendar';

interface CreateHabitTodoModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated: () => void;
  eventType: 'HABIT' | 'TODO';
  calendars: Calendar[];
  selectedCalendarId: string | null;
}

export default function CreateHabitTodoModal({
  isOpen,
  onClose,
  onCreated,
  eventType,
  calendars,
  selectedCalendarId,
}: CreateHabitTodoModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [calendarId, setCalendarId] = useState(selectedCalendarId || '');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly'>('daily');
  const [startTime, setStartTime] = useState('09:00');
  const [priority, setPriority] = useState<number | undefined>(undefined);
  const [dueDate, setDueDate] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);

  // Reset form when modal opens/closes or eventType changes
  useEffect(() => {
    if (isOpen) {
      setTitle('');
      setDescription('');
      setCalendarId(selectedCalendarId || '');
      setFrequency('daily');
      setStartTime('09:00');
      setPriority(undefined);
      setDueDate('');
      setCategoryId('');
    }
  }, [isOpen, selectedCalendarId]);

  // Fetch categories when calendar changes (for TODOs)
  useEffect(() => {
    if (eventType === 'TODO' && calendarId) {
      api.categories.list(calendarId).then(setCategories).catch(console.error);
    } else {
      setCategories([]);
    }
  }, [eventType, calendarId]);

  const effectiveCalendarId = selectedCalendarId || calendarId;

  const canCreate = title.trim() !== '' && effectiveCalendarId !== '';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canCreate) return;

    setLoading(true);
    try {
      const today = new Date();
      const startDate = today.toISOString().split('T')[0];

      const baseData = {
        calendarId: effectiveCalendarId,
        title: title.trim(),
        description: description.trim() || undefined,
        eventType,
        startDate,
        startTime: eventType === 'HABIT' ? startTime : '00:00',
      };

      if (eventType === 'HABIT') {
        // Build recurrence rule
        const freqMap = {
          daily: 'FREQ=DAILY',
          weekly: 'FREQ=WEEKLY',
          monthly: 'FREQ=MONTHLY',
        };

        await api.events.create({
          ...baseData,
          recurrenceRule: freqMap[frequency],
        });
      } else {
        // TODO
        await api.events.create({
          ...baseData,
          priority: priority || undefined,
          dueDate: dueDate || undefined,
          categoryId: categoryId || undefined,
        });
      }

      onCreated();
      onClose();
    } catch (error) {
      console.error('Error creating event:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-[#2a0a3a] rounded-2xl p-6 w-full max-w-md border border-white/20 shadow-xl">
        <h2 className="text-xl font-bold text-white mb-6">
          {eventType === 'HABIT' ? 'Novo Habito' : 'Nova Tarefa'}
        </h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Calendar Selector (only when no calendar is pre-selected) */}
          {selectedCalendarId === null && (
            <div>
              <label htmlFor="calendar" className="block text-sm font-medium text-white/70 mb-1">
                Perfil
              </label>
              <select
                id="calendar"
                value={calendarId}
                onChange={(e) => setCalendarId(e.target.value)}
                className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
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
              placeholder={eventType === 'HABIT' ? 'Ex: Meditacao' : 'Ex: Criar post LinkedIn'}
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500"
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
              className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500 resize-none"
            />
          </div>

          {/* Habit-specific fields */}
          {eventType === 'HABIT' && (
            <>
              <div>
                <label htmlFor="frequency" className="block text-sm font-medium text-white/70 mb-1">
                  Frequencia
                </label>
                <select
                  id="frequency"
                  value={frequency}
                  onChange={(e) => setFrequency(e.target.value as 'daily' | 'weekly' | 'monthly')}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                >
                  <option value="daily">Diario</option>
                  <option value="weekly">Semanal</option>
                  <option value="monthly">Mensal</option>
                </select>
              </div>

              <div>
                <label htmlFor="startTime" className="block text-sm font-medium text-white/70 mb-1">
                  Horario
                </label>
                <input
                  type="time"
                  id="startTime"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
              </div>
            </>
          )}

          {/* Todo-specific fields */}
          {eventType === 'TODO' && (
            <>
              <div>
                <label htmlFor="priority" className="block text-sm font-medium text-white/70 mb-1">
                  Prioridade
                </label>
                <select
                  id="priority"
                  value={priority || ''}
                  onChange={(e) => setPriority(e.target.value ? Number(e.target.value) : undefined)}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                >
                  <option value="">Sem prioridade</option>
                  <option value="1">Alta</option>
                  <option value="2">Media</option>
                  <option value="3">Baixa</option>
                </select>
              </div>

              <div>
                <label htmlFor="dueDate" className="block text-sm font-medium text-white/70 mb-1">
                  Data de Vencimento
                </label>
                <input
                  type="date"
                  id="dueDate"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
              </div>

              {categories.length > 0 && (
                <div>
                  <label htmlFor="category" className="block text-sm font-medium text-white/70 mb-1">
                    Categoria
                  </label>
                  <select
                    id="category"
                    value={categoryId}
                    onChange={(e) => setCategoryId(e.target.value)}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <option value="">Sem categoria</option>
                    {categories.map((cat) => (
                      <option key={cat.id} value={cat.id}>
                        {cat.icon} {cat.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </>
          )}

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
              className="flex-1 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Criando...' : 'Criar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
