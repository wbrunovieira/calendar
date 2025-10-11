'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Category } from '@/types/calendar';
import { calculateEndTime, incrementTime } from '@/utils/calendar';
import ModalHeader from './ModalHeader';
import FormSelect from './FormSelect';
import FormInput from './FormInput';
import FormTextarea from './FormTextarea';
import RecurrenceFields from './RecurrenceFields';

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
      if (preservedFormData) {
        setFormData(preservedFormData as typeof formData);
      } else if (initialDate || initialTime) {
        setFormData(prev => ({
          ...prev,
          startDate: initialDate || prev.startDate,
          startTime: initialTime || prev.startTime,
          endTime: initialTime ? calculateEndTime(initialTime) : prev.endTime,
        }));
      }
    }
  }, [isOpen, initialDate, initialTime, preservedFormData]);

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

      // Keep form data for next similar event, but clear title and description
      const updatedFormData = {
        ...formData,
        title: '',
        description: '',
        startTime: createAnother ? incrementTime(formData.startTime) : formData.startTime,
        endTime: createAnother && formData.endTime ? incrementTime(formData.endTime) : formData.endTime,
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
    setFormData(prev => ({
      ...prev,
      recurrenceDaysOfWeek: prev.recurrenceDaysOfWeek.includes(day)
        ? prev.recurrenceDaysOfWeek.filter(d => d !== day)
        : [...prev.recurrenceDaysOfWeek, day].sort(),
    }));
  };

  const handleStartTimeChange = (time: string) => {
    setFormData({ ...formData, startTime: time, endTime: calculateEndTime(time) });
  };

  const filteredCategories = formData.calendarId
    ? categories.filter(c => c.calendarId === formData.calendarId)
    : [];

  const calendarOptions = calendars.map(cal => ({ value: cal.id, label: cal.name }));
  const categoryOptions = filteredCategories.map(cat => ({ value: cat.id, label: `${cat.icon} ${cat.name}` }));

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div className="bg-[#350545] rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <ModalHeader title="Criar Novo Evento" onClose={onClose} />

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="bg-red-900/50 text-red-200 px-4 py-3 rounded-lg text-sm border border-red-500/50">
              {error}
            </div>
          )}

          {/* Calendário */}
          <FormSelect
            label="Calendário"
            value={formData.calendarId}
            onChange={value => setFormData({ ...formData, calendarId: value, categoryId: '' })}
            options={calendarOptions}
            placeholder="Selecione um calendário"
            required
          />

          {/* Categoria */}
          <FormSelect
            label="Categoria"
            value={formData.categoryId}
            onChange={value => setFormData({ ...formData, categoryId: value })}
            options={categoryOptions}
            placeholder="Sem categoria"
            disabled={!formData.calendarId}
          />

          {/* Título */}
          <FormInput
            label="Título"
            type="text"
            value={formData.title}
            onChange={value => setFormData({ ...formData, title: value })}
            placeholder="Ex: Reunião com cliente"
            required
          />

          {/* Descrição */}
          <FormTextarea
            label="Descrição"
            value={formData.description}
            onChange={value => setFormData({ ...formData, description: value })}
            placeholder="Detalhes do evento..."
          />

          {/* Data e Hora */}
          <div className="grid grid-cols-2 gap-4">
            <FormInput
              label="Data Início"
              type="date"
              value={formData.startDate}
              onChange={value => setFormData({ ...formData, startDate: value })}
              required
            />
            <FormInput
              label="Data Fim"
              type="date"
              value={formData.endDate}
              onChange={value => setFormData({ ...formData, endDate: value })}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <FormInput
              label="Hora Início"
              type="time"
              value={formData.startTime}
              onChange={handleStartTimeChange}
              required
            />
            <FormInput
              label="Hora Fim"
              type="time"
              value={formData.endTime}
              onChange={value => setFormData({ ...formData, endTime: value })}
            />
          </div>

          {/* Evento Recorrente */}
          <div className="border-t border-white/10 pt-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.isRecurring}
                onChange={e => setFormData({ ...formData, isRecurring: e.target.checked })}
                className="w-4 h-4 text-[#792990] rounded focus:ring-[#792990]"
              />
              <span className="text-sm font-semibold text-white">Evento Recorrente</span>
            </label>
          </div>

          {formData.isRecurring && (
            <RecurrenceFields
              frequency={formData.recurrenceFrequency}
              daysOfWeek={formData.recurrenceDaysOfWeek}
              endDate={formData.recurrenceEndDate}
              onFrequencyChange={freq =>
                setFormData({ ...formData, recurrenceFrequency: freq as typeof formData.recurrenceFrequency })
              }
              onToggleDayOfWeek={toggleDayOfWeek}
              onEndDateChange={date => setFormData({ ...formData, recurrenceEndDate: date })}
            />
          )}

          {/* Checkbox Criar Outro Similar */}
          <div className="border-t border-white/10 pt-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={createAnother}
                onChange={e => setCreateAnother(e.target.checked)}
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
