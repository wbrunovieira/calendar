'use client';

import { Category } from '@/types/calendar';
import { useEventForm } from '@/hooks/useEventForm';
import ModalContainer from './ModalContainer';
import ModalHeader from './ModalHeader';
import EventFormFields from './EventFormFields';
import FormCheckbox from './FormCheckbox';
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
  const {
    formData,
    setFormData,
    loading,
    error,
    createAnother,
    setCreateAnother,
    calendarOptions,
    categoryOptions,
    handleCalendarChange,
    handleStartTimeChange,
    toggleDayOfWeek,
    handleSubmit,
  } = useEventForm({
    isOpen,
    initialDate,
    initialTime,
    preservedFormData,
    calendars,
    categories,
    onEventCreated,
    onClose,
  });

  return (
    <ModalContainer isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Criar Novo Evento" onClose={onClose} />

      <form onSubmit={handleSubmit} className="p-6 space-y-4">
        {error && (
          <div className="bg-red-900/50 text-red-200 px-4 py-3 rounded-lg text-sm border border-red-500/50">
            {error}
          </div>
        )}

        <EventFormFields
          formData={formData}
          calendarOptions={calendarOptions}
          categoryOptions={categoryOptions}
          onCalendarChange={handleCalendarChange}
          onCategoryChange={value => setFormData({ ...formData, categoryId: value })}
          onTitleChange={value => setFormData({ ...formData, title: value })}
          onDescriptionChange={value => setFormData({ ...formData, description: value })}
          onStartDateChange={value => setFormData({ ...formData, startDate: value })}
          onEndDateChange={value => setFormData({ ...formData, endDate: value })}
          onStartTimeChange={handleStartTimeChange}
          onEndTimeChange={value => setFormData({ ...formData, endTime: value })}
        />

        {/* Evento Recorrente */}
        <div className="border-t border-white/10 pt-4">
          <FormCheckbox
            label="Evento Recorrente"
            checked={formData.isRecurring}
            onChange={checked => setFormData({ ...formData, isRecurring: checked })}
          />
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
          <FormCheckbox
            label="Criar outro evento similar (mantém calendário, categoria e horários)"
            checked={createAnother}
            onChange={setCreateAnother}
            colorClass="text-green-600"
          />
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
    </ModalContainer>
  );
}
