'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Event, Category } from '@/types/calendar';
import RecurringEventActionModal, { RecurringEventAction } from './RecurringEventActionModal';

interface EditEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  onEventUpdated: () => void;
  event: Event | null;
  calendars: Array<{ id: string; name: string; color: string; type: string }>;
  categories: Category[];
}

export default function EditEventModal({
  isOpen,
  onClose,
  onEventUpdated,
  event,
  calendars,
  categories,
}: EditEventModalProps) {
  const [formData, setFormData] = useState({
    calendarId: '',
    categoryId: '',
    title: '',
    description: '',
    startTime: '',
    endTime: '',
    startDate: '',
    endDate: '',
    isRecurring: false,
    recurrenceFrequency: 'weekly' as 'daily' | 'weekly' | 'monthly' | 'yearly',
    recurrenceInterval: 1,
    recurrenceDaysOfWeek: [] as number[],
    recurrenceEndDate: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showRecurringActionModal, setShowRecurringActionModal] = useState(false);
  const [recurringAction, setRecurringAction] = useState<RecurringEventAction | null>(null);

  // Load event data when modal opens
  useEffect(() => {
    if (isOpen && event) {
      setFormData({
        calendarId: event.calendarId || '',
        categoryId: event.categoryId || '',
        title: event.title || '',
        description: event.description || '',
        startTime: event.startTime || '',
        endTime: event.endTime || '',
        startDate: event.startDate?.split('T')[0] || '',
        endDate: event.endDate?.split('T')[0] || '',
        isRecurring: event.isRecurring || false,
        recurrenceFrequency: event.recurrenceFrequency || 'weekly',
        recurrenceInterval: event.recurrenceInterval || 1,
        recurrenceDaysOfWeek: event.recurrenceDaysOfWeek || [],
        recurrenceEndDate: event.recurrenceEndDate?.split('T')[0] || '',
      });
      setError('');

      // If event is recurring, show the action modal first
      if (event.isRecurring) {
        setShowRecurringActionModal(true);
        setRecurringAction(null);
      } else {
        setShowRecurringActionModal(false);
      }
    }
  }, [isOpen, event]);

  const daysOfWeek = [
    { value: 0, label: 'Dom' },
    { value: 1, label: 'Seg' },
    { value: 2, label: 'Ter' },
    { value: 3, label: 'Qua' },
    { value: 4, label: 'Qui' },
    { value: 5, label: 'Sex' },
    { value: 6, label: 'Sáb' },
  ];

  const handleRecurringActionSelect = (action: RecurringEventAction) => {
    setRecurringAction(action);
    setShowRecurringActionModal(false);
  };

  const handleRecurringActionClose = () => {
    setShowRecurringActionModal(false);
    onClose(); // Close the entire edit modal if user cancels recurring action
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!event) return;

    if (!formData.calendarId || !formData.title || !formData.startTime || !formData.startDate) {
      setError('Por favor, preencha os campos obrigatórios');
      return;
    }

    const payload: any = {
      calendarId: formData.calendarId,
      categoryId: formData.categoryId || undefined,
      title: formData.title,
      description: formData.description || undefined,
      startTime: formData.startTime,
      endTime: formData.endTime || undefined,
      startDate: formData.startDate,
      endDate: formData.endDate || undefined,
      isRecurring: formData.isRecurring,
      recurrenceFrequency: formData.isRecurring ? formData.recurrenceFrequency : undefined,
      recurrenceInterval: formData.isRecurring ? formData.recurrenceInterval : undefined,
      recurrenceDaysOfWeek:
        formData.isRecurring && formData.recurrenceFrequency === 'weekly'
          ? formData.recurrenceDaysOfWeek
          : undefined,
      recurrenceEndDate: formData.isRecurring && formData.recurrenceEndDate ? formData.recurrenceEndDate : undefined,
    };

    // Add recurring action scope if this is a recurring event
    if (event.isRecurring && recurringAction) {
      payload.recurringEditScope = recurringAction;
    }

    try {
      setLoading(true);
      // For regular events, use the ID as is
      // For expanded recurring events from search (format: "uuid-YYYY-MM-DD"), extract the UUID part
      let eventId = event.id;

      // Check if this looks like an expanded recurring event ID (has date pattern at the end)
      const datePattern = /-\d{4}-\d{2}-\d{2}$/;
      if (datePattern.test(event.id)) {
        // Extract the UUID part before the date
        eventId = event.id.replace(datePattern, '');
      }

      console.log('Original event.id:', event.id);
      console.log('Extracted eventId for API:', eventId);

      await api.events.update(eventId, payload);
      onEventUpdated();
      onClose();
    } catch (err) {
      console.error('Erro ao atualizar evento:', err);
      setError('Erro ao atualizar evento. Tente novamente.');
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

  if (!isOpen || !event) {
    return null;
  }

  // If this is a recurring event and we haven't selected an action yet, show the action modal
  if (event.isRecurring && showRecurringActionModal) {
    return (
      <RecurringEventActionModal
        isOpen={showRecurringActionModal}
        onClose={handleRecurringActionClose}
        onSelect={handleRecurringActionSelect}
        eventTitle={event.title}
      />
    );
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
          <h2 className="text-2xl font-bold">Editar Evento</h2>
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
              {loading ? 'Salvando...' : 'Salvar Alterações'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
