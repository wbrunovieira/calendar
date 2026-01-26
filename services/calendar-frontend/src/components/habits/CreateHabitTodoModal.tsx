'use client';

import { useState, useEffect, useMemo } from 'react';
import { api } from '@/lib/api';
import type { Calendar, Category, CategoryType } from '@/types/calendar';
import LabelSelector from '@/components/labels/LabelSelector';

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
  const [categoryTypeId, setCategoryTypeId] = useState('');
  const [labelId, setLabelId] = useState<string | undefined>(undefined);
  const [categories, setCategories] = useState<Category[]>([]);
  const [categoryTypes, setCategoryTypes] = useState<CategoryType[]>([]);
  const [loading, setLoading] = useState(false);
  // Flexible weekly habit states
  const [recurrenceType, setRecurrenceType] = useState<'FIXED' | 'FLEXIBLE'>('FIXED');
  const [weeklyTargetCount, setWeeklyTargetCount] = useState(2);
  const [weeklyPreferredDays, setWeeklyPreferredDays] = useState<string[]>([]);

  const DAYS_OF_WEEK = [
    { value: 'MO', label: 'Seg' },
    { value: 'TU', label: 'Ter' },
    { value: 'WE', label: 'Qua' },
    { value: 'TH', label: 'Qui' },
    { value: 'FR', label: 'Sex' },
    { value: 'SA', label: 'Sab' },
    { value: 'SU', label: 'Dom' },
  ];

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
      setCategoryTypeId('');
      setLabelId(undefined);
      setRecurrenceType('FIXED');
      setWeeklyTargetCount(2);
      setWeeklyPreferredDays([]);
    }
  }, [isOpen, selectedCalendarId]);

  // Fetch categories and category types when calendar changes
  useEffect(() => {
    const effectiveId = selectedCalendarId || calendarId;
    if (effectiveId) {
      Promise.all([
        api.categories.list(effectiveId),
        api.categoryTypes.list(effectiveId),
      ])
        .then(([cats, types]) => {
          setCategories(cats);
          setCategoryTypes(types);
        })
        .catch(console.error);
    } else {
      setCategories([]);
      setCategoryTypes([]);
    }
  }, [calendarId, selectedCalendarId]);

  const effectiveCalendarId = selectedCalendarId || calendarId;

  // Filter categories based on selected categoryType
  const filteredCategories = useMemo(() => {
    if (!categoryTypeId) return categories;
    return categories.filter(cat =>
      cat.categoryTypes?.some(ct => ct.id === categoryTypeId)
    );
  }, [categories, categoryTypeId]);

  // Reset categoryId when categoryTypeId changes
  useEffect(() => {
    if (categoryTypeId && categoryId) {
      const categoryStillValid = filteredCategories.some(cat => cat.id === categoryId);
      if (!categoryStillValid) {
        setCategoryId('');
      }
    }
  }, [categoryTypeId, categoryId, filteredCategories]);

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
        // Common habit fields
        const habitData = {
          ...baseData,
          categoryId: categoryId || undefined,
          categoryTypeId: categoryTypeId || undefined,
          labelId: labelId || undefined,
        };

        // Build recurrence rule based on frequency and type
        if (frequency === 'weekly' && recurrenceType === 'FLEXIBLE') {
          // Flexible weekly habit
          await api.events.create({
            ...habitData,
            recurrenceRule: 'FREQ=WEEKLY',
            recurrenceType: 'FLEXIBLE',
            weeklyTargetCount,
            weeklyPreferredDays: weeklyPreferredDays.length > 0 ? weeklyPreferredDays : undefined,
          });
        } else {
          // Fixed habit (daily, weekly with fixed days, monthly)
          const freqMap = {
            daily: 'FREQ=DAILY',
            weekly: 'FREQ=WEEKLY',
            monthly: 'FREQ=MONTHLY',
          };

          await api.events.create({
            ...habitData,
            recurrenceRule: freqMap[frequency],
            recurrenceType: 'FIXED',
          });
        }
      } else {
        // TODO
        await api.events.create({
          ...baseData,
          priority: priority || undefined,
          dueDate: dueDate || undefined,
          categoryId: categoryId || undefined,
          categoryTypeId: categoryTypeId || undefined,
          labelId: labelId || undefined,
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
                  onChange={(e) => {
                    setFrequency(e.target.value as 'daily' | 'weekly' | 'monthly');
                    // Reset to FIXED when changing frequency
                    if (e.target.value !== 'weekly') {
                      setRecurrenceType('FIXED');
                    }
                  }}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                >
                  <option value="daily">Diario</option>
                  <option value="weekly">Semanal</option>
                  <option value="monthly">Mensal</option>
                </select>
              </div>

              {/* Weekly habit type selection */}
              {frequency === 'weekly' && (
                <div>
                  <label className="block text-sm font-medium text-white/70 mb-2">
                    Tipo de Recorrencia
                  </label>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => setRecurrenceType('FIXED')}
                      className={`flex-1 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                        recurrenceType === 'FIXED'
                          ? 'bg-purple-600 text-white'
                          : 'bg-white/10 text-white/70 hover:bg-white/20'
                      }`}
                    >
                      Dias Fixos
                    </button>
                    <button
                      type="button"
                      onClick={() => setRecurrenceType('FLEXIBLE')}
                      className={`flex-1 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                        recurrenceType === 'FLEXIBLE'
                          ? 'bg-purple-600 text-white'
                          : 'bg-white/10 text-white/70 hover:bg-white/20'
                      }`}
                    >
                      Meta Flexivel
                    </button>
                  </div>
                  <p className="text-xs text-white/50 mt-1">
                    {recurrenceType === 'FIXED'
                      ? 'Aparece em dias especificos da semana'
                      : 'Meta de vezes por semana, qualquer dia'}
                  </p>
                </div>
              )}

              {/* Flexible weekly: target count */}
              {frequency === 'weekly' && recurrenceType === 'FLEXIBLE' && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-white/70 mb-2">
                      Quantas vezes por semana?
                    </label>
                    <div className="flex items-center gap-3">
                      <button
                        type="button"
                        onClick={() => setWeeklyTargetCount(Math.max(1, weeklyTargetCount - 1))}
                        className="w-10 h-10 rounded-lg bg-white/10 text-white hover:bg-white/20 transition-colors text-xl"
                      >
                        -
                      </button>
                      <span className="text-2xl font-bold text-white w-12 text-center">
                        {weeklyTargetCount}
                      </span>
                      <button
                        type="button"
                        onClick={() => setWeeklyTargetCount(Math.min(7, weeklyTargetCount + 1))}
                        className="w-10 h-10 rounded-lg bg-white/10 text-white hover:bg-white/20 transition-colors text-xl"
                      >
                        +
                      </button>
                      <span className="text-white/60 text-sm">vezes por semana</span>
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-white/70 mb-2">
                      Dias Preferidos (opcional)
                    </label>
                    <p className="text-xs text-white/50 mb-2">
                      Ajuda a organizar sua agenda, mas nao e obrigatorio
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {DAYS_OF_WEEK.map((day) => (
                        <button
                          key={day.value}
                          type="button"
                          onClick={() => {
                            setWeeklyPreferredDays((prev) =>
                              prev.includes(day.value)
                                ? prev.filter((d) => d !== day.value)
                                : [...prev, day.value]
                            );
                          }}
                          className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                            weeklyPreferredDays.includes(day.value)
                              ? 'bg-emerald-600 text-white'
                              : 'bg-white/10 text-white/70 hover:bg-white/20'
                          }`}
                        >
                          {day.label}
                        </button>
                      ))}
                    </div>
                  </div>
                </>
              )}

              <div>
                <label htmlFor="startTime" className="block text-sm font-medium text-white/70 mb-1">
                  Horario Preferido
                </label>
                <input
                  type="time"
                  id="startTime"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
              </div>

              {/* Category Type */}
              {categoryTypes.length > 0 && (
                <div>
                  <label htmlFor="habitCategoryType" className="block text-sm font-medium text-white/70 mb-1">
                    Tipo
                  </label>
                  <select
                    id="habitCategoryType"
                    value={categoryTypeId}
                    onChange={(e) => {
                      setCategoryTypeId(e.target.value);
                      setCategoryId(''); // Reset category when type changes
                    }}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <option value="">Selecione um tipo</option>
                    {categoryTypes.map((type) => (
                      <option key={type.id} value={type.id}>
                        {type.icon} {type.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Category - only shown when type is selected */}
              {categoryTypeId && filteredCategories.length > 0 && (
                <div>
                  <label htmlFor="habitCategory" className="block text-sm font-medium text-white/70 mb-1">
                    Categoria
                  </label>
                  <select
                    id="habitCategory"
                    value={categoryId}
                    onChange={(e) => setCategoryId(e.target.value)}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <option value="">Selecione uma categoria</option>
                    {filteredCategories.map((cat) => (
                      <option key={cat.id} value={cat.id}>
                        {cat.icon} {cat.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Label Selector */}
              {effectiveCalendarId && (
                <LabelSelector
                  calendarId={effectiveCalendarId}
                  selectedLabelId={labelId}
                  onSelect={setLabelId}
                />
              )}
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

              {/* Category Type */}
              {categoryTypes.length > 0 && (
                <div>
                  <label htmlFor="todoCategoryType" className="block text-sm font-medium text-white/70 mb-1">
                    Tipo
                  </label>
                  <select
                    id="todoCategoryType"
                    value={categoryTypeId}
                    onChange={(e) => {
                      setCategoryTypeId(e.target.value);
                      setCategoryId(''); // Reset category when type changes
                    }}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <option value="">Selecione um tipo</option>
                    {categoryTypes.map((type) => (
                      <option key={type.id} value={type.id}>
                        {type.icon} {type.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Category - only shown when type is selected */}
              {categoryTypeId && filteredCategories.length > 0 && (
                <div>
                  <label htmlFor="todoCategory" className="block text-sm font-medium text-white/70 mb-1">
                    Categoria
                  </label>
                  <select
                    id="todoCategory"
                    value={categoryId}
                    onChange={(e) => setCategoryId(e.target.value)}
                    className="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <option value="">Selecione uma categoria</option>
                    {filteredCategories.map((cat) => (
                      <option key={cat.id} value={cat.id}>
                        {cat.icon} {cat.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Label Selector */}
              {effectiveCalendarId && (
                <LabelSelector
                  calendarId={effectiveCalendarId}
                  selectedLabelId={labelId}
                  onSelect={setLabelId}
                />
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
