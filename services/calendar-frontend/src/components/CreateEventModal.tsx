'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Category } from '@/types/calendar';

interface CreateEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  onEventCreated: (preservedData?: Record<string, unknown>) => void;
  calendars: Array<{ id: string; name: string; color: string; type: string }>;
  categories: Category[];
  initialDate?: string;
  initialTime?: string;
  preservedFormData?: Record<string, unknown>;
}

export default function CreateEventModal({
  isOpen,
  onClose,
  onEventCreated,
  calendars,
  categories,
  initialDate,
  initialTime,
  preservedFormData,
}: CreateEventModalProps) {
  const [formData, setFormData] = useState({
    calendarId: '',
    categoryId: '',
    title: '',
    description: '',
    startTime: initialTime || '',
    endTime: '',
    startDate: initialDate || '',
    endDate: '',
    isRecurring: false,
    recurrenceFrequency: 'weekly' as 'daily' | 'weekly' | 'monthly' | 'yearly',
    recurrenceInterval: 1,
    recurrenceDaysOfWeek: [] as number[],
    recurrenceEndDate: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [createAnother, setCreateAnother] = useState(false);

  // Update form data when modal opens with initial values or preserved data
  useEffect(() => {
    if (isOpen) {
      // If we have preserved data from "create another", use it
      if (preservedFormData) {
        setFormData(preservedFormData as typeof formData);
      } else if (initialDate || initialTime) {
        // Calculate end time as start time + 1 hour
        let endTime = '';
        if (initialTime) {
          const [hours, minutes] = initialTime.split(':').map(Number);
          const endHour = (hours + 1) % 24;
          endTime = `${endHour.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
        }

        setFormData(prev => {
          const newData = {
            ...prev,
            startDate: initialDate || prev.startDate,
            startTime: initialTime || prev.startTime,
            endTime: endTime || prev.endTime,
          };
          return newData;
        });
      }
    }
  }, [isOpen, initialDate, initialTime, preservedFormData]);

  const daysOfWeek = [
    { value: 0, label: 'Dom' },
    { value: 1, label: 'Seg' },
    { value: 2, label: 'Ter' },
    { value: 3, label: 'Qua' },
    { value: 4, label: 'Qui' },
    { value: 5, label: 'Sex' },
    { value: 6, label: 'Sáb' },
  ];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!formData.calendarId || !formData.title || !formData.startTime || !formData.startDate) {
      setError('Por favor, preencha os campos obrigatórios');
      return;
    }

    const payload: Record<string, unknown> = {
      calendarId: formData.calendarId,
      title: formData.title,
      startTime: formData.startTime,
      startDate: formData.startDate,
      isRecurring: formData.isRecurring,
    };

    // Only add optional fields if they have values
    if (formData.categoryId) payload.categoryId = formData.categoryId;
    if (formData.description) payload.description = formData.description;
    if (formData.endTime) payload.endTime = formData.endTime;
    if (formData.endDate) payload.endDate = formData.endDate;

    if (formData.isRecurring) {
      payload.recurrenceFrequency = formData.recurrenceFrequency;
      payload.recurrenceInterval = formData.recurrenceInterval;
      if (formData.recurrenceFrequency === 'weekly' && formData.recurrenceDaysOfWeek.length > 0) {
        payload.recurrenceDaysOfWeek = formData.recurrenceDaysOfWeek;
      }
      if (formData.recurrenceEndDate) {
        payload.recurrenceEndDate = formData.recurrenceEndDate;
      }
    }

    try {
      setLoading(true);
      await api.events.create(payload);

      // Calculate new start time (1 hour later)
      let newStartTime = formData.startTime;
      let newEndTime = formData.endTime;

      if (createAnother && formData.startTime) {
        const [hours, minutes] = formData.startTime.split(':').map(Number);
        const newStartHour = (hours + 1) % 24;
        newStartTime = `${newStartHour.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;

        if (formData.endTime) {
          const [endHours, endMinutes] = formData.endTime.split(':').map(Number);
          const newEndHour = (endHours + 1) % 24;
          newEndTime = `${newEndHour.toString().padStart(2, '0')}:${endMinutes.toString().padStart(2, '0')}`;
        }
      }

      // Keep form data for next similar event, but clear title and description
      const updatedFormData = {
        ...formData,
        title: '',
        description: '',
        startDate: formData.startDate,
        startTime: newStartTime,
        endTime: newEndTime,
      };

      // Pass preserved data to parent if "create another" is checked
      if (createAnother) {
        onEventCreated(updatedFormData);
      } else {
        onEventCreated();
        onClose();
      }
    } catch {
      setError('Erro ao criar evento. Tente novamente.');
    } finally {
      setLoading(false);
    }
  };

  const toggleDayOfWeek = (day: number) => {
    setFormData((prev) => ({
      ...prev,
      recurrenceDaysOfWeek: prev.recurrenceDaysOfWeek.includes(day)
        ? prev.recurrenceDaysOfWeek.filter((d) => d !== day)
        : [...prev.recurrenceDaysOfWeek, day].sort(),
    }));
  };

  const filteredCategories = formData.calendarId
    ? categories.filter((c) => c.calendarId === formData.calendarId)
    : [];

  if (!isOpen) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
    >
      <div className="bg-[#350545] rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-gradient-to-r from-[#350545] to-[#792990] text-white px-6 py-4 flex items-center justify-between border-b border-white/10">
          <h2 className="text-2xl font-bold">Criar Novo Evento</h2>
          <button
            onClick={onClose}
            className="text-white hover:bg-white/20 rounded-full p-2 transition-colors"
            aria-label="Fechar"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="bg-red-900/50 text-red-200 px-4 py-3 rounded-lg text-sm border border-red-500/50">{error}</div>
          )}

          {/* Calendário */}
          <div>
            <label className="block text-sm font-semibold text-white mb-2">
              Calendário <span className="text-red-400">*</span>
            </label>
            <select
              value={formData.calendarId}
              onChange={(e) => setFormData({ ...formData, calendarId: e.target.value, categoryId: '' })}
              className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
              required
            >
              <option value="" className="bg-[#350545] text-white">Selecione um calendário</option>
              {calendars.map((calendar) => (
                <option key={calendar.id} value={calendar.id} className="bg-[#350545] text-white">
                  {calendar.name}
                </option>
              ))}
            </select>
          </div>

          {/* Categoria */}
          <div>
            <label className="block text-sm font-semibold text-white mb-2">Categoria</label>
            <select
              value={formData.categoryId}
              onChange={(e) => setFormData({ ...formData, categoryId: e.target.value })}
              className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent disabled:opacity-50"
              disabled={!formData.calendarId}
            >
              <option value="" className="bg-[#350545] text-white">Sem categoria</option>
              {filteredCategories.map((category) => (
                <option key={category.id} value={category.id} className="bg-[#350545] text-white">
                  {category.icon} {category.name}
                </option>
              ))}
            </select>
          </div>

          {/* Título */}
          <div>
            <label className="block text-sm font-semibold text-white mb-2">
              Título <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white placeholder-white/50 rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
              placeholder="Ex: Reunião com cliente"
              required
            />
          </div>

          {/* Descrição */}
          <div>
            <label className="block text-sm font-semibold text-white mb-2">Descrição</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white placeholder-white/50 rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent resize-none"
              rows={3}
              placeholder="Detalhes do evento..."
            />
          </div>

          {/* Data e Hora */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-semibold text-white mb-2">
                Data Início <span className="text-red-400">*</span>
              </label>
              <input
                type="date"
                value={formData.startDate}
                onChange={(e) => setFormData({ ...formData, startDate: e.target.value })}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-semibold text-white mb-2">Data Fim</label>
              <input
                type="date"
                value={formData.endDate}
                onChange={(e) => setFormData({ ...formData, endDate: e.target.value })}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-semibold text-white mb-2">
                Hora Início <span className="text-red-400">*</span>
              </label>
              <input
                type="time"
                value={formData.startTime}
                onChange={(e) => {
                  const newStartTime = e.target.value;
                  // Calculate end time as start time + 1 hour
                  let newEndTime = '';
                  if (newStartTime) {
                    const [hours, minutes] = newStartTime.split(':').map(Number);
                    const endHour = (hours + 1) % 24;
                    newEndTime = `${endHour.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
                  }
                  setFormData({ ...formData, startTime: newStartTime, endTime: newEndTime });
                }}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-semibold text-white mb-2">Hora Fim</label>
              <input
                type="time"
                value={formData.endTime}
                onChange={(e) => setFormData({ ...formData, endTime: e.target.value })}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
              />
            </div>
          </div>

          {/* Evento Recorrente */}
          <div className="border-t border-white/10 pt-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.isRecurring}
                onChange={(e) => setFormData({ ...formData, isRecurring: e.target.checked })}
                className="w-4 h-4 text-[#792990] rounded focus:ring-[#792990]"
              />
              <span className="text-sm font-semibold text-white">Evento Recorrente</span>
            </label>
          </div>

          {formData.isRecurring && (
            <div className="space-y-4 bg-white/5 p-4 rounded-lg border border-white/10">
              {/* Frequência */}
              <div>
                <label className="block text-sm font-semibold text-white mb-2">Frequência</label>
                <select
                  value={formData.recurrenceFrequency}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      recurrenceFrequency: e.target.value as 'daily' | 'weekly' | 'monthly' | 'yearly',
                    })
                  }
                  className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
                >
                  <option value="daily" className="bg-[#350545] text-white">Diário</option>
                  <option value="weekly" className="bg-[#350545] text-white">Semanal</option>
                  <option value="monthly" className="bg-[#350545] text-white">Mensal</option>
                  <option value="yearly" className="bg-[#350545] text-white">Anual</option>
                </select>
              </div>

              {/* Dias da Semana (apenas para frequência semanal) */}
              {formData.recurrenceFrequency === 'weekly' && (
                <div>
                  <label className="block text-sm font-semibold text-white mb-2">
                    Dias da Semana
                  </label>
                  <div className="flex gap-2 flex-wrap">
                    {daysOfWeek.map((day) => (
                      <button
                        key={day.value}
                        type="button"
                        onClick={() => toggleDayOfWeek(day.value)}
                        className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                          formData.recurrenceDaysOfWeek.includes(day.value)
                            ? 'bg-[#792990] text-white'
                            : 'bg-white/10 text-white border border-white/20'
                        }`}
                      >
                        {day.label}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Data de Término da Recorrência */}
              <div>
                <label className="block text-sm font-semibold text-white mb-2">
                  Repetir até
                </label>
                <input
                  type="date"
                  value={formData.recurrenceEndDate}
                  onChange={(e) => setFormData({ ...formData, recurrenceEndDate: e.target.value })}
                  className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
                />
              </div>
            </div>
          )}

          {/* Checkbox Criar Outro Similar */}
          <div className="border-t border-white/10 pt-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={createAnother}
                onChange={(e) => setCreateAnother(e.target.checked)}
                className="w-4 h-4 text-green-600 rounded focus:ring-green-500 focus:ring-2"
              />
              <span className="text-sm font-medium text-white">
                Criar outro evento similar (mantém calendário, categoria e horários)
              </span>
            </label>
          </div>

          {/* Botões */}
          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-6 py-3 bg-white/10 text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/20"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-6 py-3 bg-gradient-to-r from-[#792990] to-[#350545] text-white rounded-lg font-semibold hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              {loading ? 'Criando...' : 'Criar Evento'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
