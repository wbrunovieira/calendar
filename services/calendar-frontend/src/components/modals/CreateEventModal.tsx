'use client';

import { Category, CategoryType } from '@/types/calendar';
import { useEventForm } from '@/hooks/useEventForm';
import ModalContainer from '../ui/modal/ModalContainer';
import ModalHeader from '../ui/modal/ModalHeader';
import ModalFooter from '../ui/modal/ModalFooter';
import EventFormFields from '../forms/EventFormFields';
import FormCheckbox from '../forms/FormCheckbox';
import RecurrenceFields from '../forms/RecurrenceFields';
import EventRemindersField from '../forms/EventRemindersField';

interface CreateEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  onEventCreated: (preservedData?: Record<string, unknown>) => void;
  calendars: Array<{ id: string; name: string; color: string; type: string }>;
  categories: Category[];
  categoryTypes: CategoryType[];
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
  categoryTypes,
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
    categoryTypeOptions,
    availableTypes,
    handleCalendarChange,
    handleCategoryChange,
    handleCategoryTypeChange,
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
    categoryTypes,
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
          categoryTypeOptions={categoryTypeOptions}
          availableTypes={availableTypes}
          onCalendarChange={handleCalendarChange}
          onCategoryChange={handleCategoryChange}
          onCategoryTypeChange={handleCategoryTypeChange}
          onTitleChange={value => setFormData({ ...formData, title: value })}
          onDescriptionChange={value => setFormData({ ...formData, description: value })}
          onStartDateChange={value => setFormData({ ...formData, startDate: value })}
          onEndDateChange={value => setFormData({ ...formData, endDate: value })}
          onStartTimeChange={handleStartTimeChange}
          onEndTimeChange={value => setFormData({ ...formData, endTime: value })}
        />

        {/* Alertas */}
        <EventRemindersField
          reminders={formData.reminders || []}
          onChange={reminders => setFormData({ ...formData, reminders })}
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
        <ModalFooter
          onCancel={onClose}
          submitText="Criar Evento"
          submitType="submit"
          loading={loading}
          loadingText="Criando..."
        />
      </form>
    </ModalContainer>
  );
}
