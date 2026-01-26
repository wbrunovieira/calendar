'use client';

import { useState, useEffect, useMemo } from 'react';
import { api } from '@/lib/api';
import type { Calendar, Category, CategoryType, Event } from '@/types/calendar';
import LabelSelector from '@/components/labels/LabelSelector';

interface EditHabitModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdated: () => void;
  habit: Event | null;
  calendars: Calendar[];
}

export default function EditHabitModal({
  isOpen,
  onClose,
  onUpdated,
  habit,
  calendars,
}: EditHabitModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [calendarId, setCalendarId] = useState('');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly'>('daily');
  const [startTime, setStartTime] = useState('09:00');
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

  // Load habit data when modal opens
  useEffect(() => {
    if (isOpen && habit) {
      setTitle(habit.title || '');
      setDescription(habit.description || '');
      setCalendarId(habit.calendarId || '');
      setStartTime(habit.startTime || '09:00');
      setCategoryId(habit.categoryId || '');
      setCategoryTypeId(habit.categoryTypeId || '');
      setLabelId(habit.labelId || undefined);

      // Set frequency from recurrenceFrequency
      if (habit.recurrenceFrequency) {
        setFrequency(habit.recurrenceFrequency as 'daily' | 'weekly' | 'monthly');
      } else {
        setFrequency('daily');
      }

      // Set flexible habit fields
      setRecurrenceType(habit.recurrenceType || 'FIXED');
      setWeeklyTargetCount(habit.weeklyTargetCount || 2);
      setWeeklyPreferredDays(habit.weeklyPreferredDays || []);
    }
  }, [isOpen, habit]);

  // Fetch categories and category types when calendar changes
  useEffect(() => {
    if (calendarId) {
      Promise.all([
        api.categories.list(calendarId),
        api.categoryTypes.list(calendarId),
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
  }, [calendarId]);

  // Filter categories based on selected categoryType
  const filteredCategories = useMemo(() => {
    if (!categoryTypeId) return categories;
    return categories.filter(cat =>
      cat.categoryTypes?.some(ct => ct.id === categoryTypeId)
    );
  }, [categories, categoryTypeId]);

  const canSave = title.trim() !== '' && calendarId !== '';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSave || !habit) return;

    setLoading(true);
    try {
      const freqMap = {
        daily: 'FREQ=DAILY',
        weekly: 'FREQ=WEEKLY',
        monthly: 'FREQ=MONTHLY',
      };

      const updateData: Record<string, unknown> = {
        calendarId,
        title: title.trim(),
        description: description.trim() || null,
        startTime,
        categoryId: categoryId || null,
        categoryTypeId: categoryTypeId || null,
        labelId: labelId || null,
        recurrenceRule: freqMap[frequency],
      };

      // Add flexible weekly habit fields
      if (frequency === 'weekly' && recurrenceType === 'FLEXIBLE') {
        updateData.recurrenceType = 'FLEXIBLE';
        updateData.weeklyTargetCount = weeklyTargetCount;
        updateData.weeklyPreferredDays = weeklyPreferredDays.length > 0 ? weeklyPreferredDays : null;
      } else {
        updateData.recurrenceType = 'FIXED';
        updateData.weeklyTargetCount = null;
        updateData.weeklyPreferredDays = null;
      }

      await api.events.update(habit.originalEventId || habit.id, updateData);

      onUpdated();
      onClose();
    } catch (error) {
      console.error('Error updating habit:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!habit) return;

    const confirmed = window.confirm('Tem certeza que deseja excluir este habito?');
    if (!confirmed) return;

    setLoading(true);
    try {
      await api.events.delete(habit.originalEventId || habit.id);
      onUpdated();
      onClose();
    } catch (error) {
      console.error('Error deleting habit:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen || !habit) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-[#2a0a3a] rounded-2xl p-6 w-full max-w-md border border-white/20 shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold text-white mb-6">Editar Habito</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Calendar Selector */}
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
              placeholder="Ex: Meditacao"
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

          {/* Frequency */}
          <div>
            <label htmlFor="frequency" className="block text-sm font-medium text-white/70 mb-1">
              Frequencia
            </label>
            <select
              id="frequency"
              value={frequency}
              onChange={(e) => {
                setFrequency(e.target.value as 'daily' | 'weekly' | 'monthly');
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

          {/* Start Time */}
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
              <label htmlFor="categoryType" className="block text-sm font-medium text-white/70 mb-1">
                Tipo
              </label>
              <select
                id="categoryType"
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
              <label htmlFor="category" className="block text-sm font-medium text-white/70 mb-1">
                Categoria
              </label>
              <select
                id="category"
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
          {calendarId && (
            <LabelSelector
              calendarId={calendarId}
              selectedLabelId={labelId}
              onSelect={setLabelId}
            />
          )}

          {/* Buttons */}
          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={handleDelete}
              disabled={loading}
              className="px-4 py-2 bg-red-600/20 text-red-400 rounded-lg hover:bg-red-600/30 transition-colors disabled:opacity-50"
            >
              Excluir
            </button>
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={!canSave || loading}
              className="flex-1 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Salvando...' : 'Salvar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
