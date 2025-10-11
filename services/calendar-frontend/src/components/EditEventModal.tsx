'use client';

import { Event, Category } from '@/types/calendar';
import { useEditEventForm } from '@/hooks/useEditEventForm';
import ModalContainer from './ModalContainer';
import ModalHeader from './ModalHeader';
import ModalFooter from './ModalFooter';
import EventFormFields from './EventFormFields';
import FormCheckbox from './FormCheckbox';
import RecurrenceFields from './RecurrenceFields';
import RecurringEventActionModal from './RecurringEventActionModal';

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
  const {
    formData,
    setFormData,
    loading,
    error,
    showRecurringActionModal,
    calendarOptions,
    categoryOptions,
    handleCalendarChange,
    handleStartTimeChange,
    toggleDayOfWeek,
    handleSubmit,
    handleRecurringActionSelect,
    handleRecurringActionClose,
  } = useEditEventForm({
    isOpen,
    event,
    calendars,
    categories,
    onEventUpdated,
    onClose,
  });

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
    <ModalContainer isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Editar Evento" onClose={onClose} />

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

        {/* Botões */}
        <ModalFooter
          onCancel={onClose}
          submitText="Salvar Alterações"
          submitType="submit"
          loading={loading}
          loadingText="Salvando..."
        />
      </form>
    </ModalContainer>
  );
}
