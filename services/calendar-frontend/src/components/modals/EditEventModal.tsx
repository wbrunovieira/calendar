'use client';

import { Event, Category, CategoryType } from '@/types/calendar';
import { useEditEventForm } from '@/hooks/useEditEventForm';
import ModalContainer from '../ui/modal/ModalContainer';
import ModalHeader from '../ui/modal/ModalHeader';
import ModalFooter from '../ui/modal/ModalFooter';
import EventFormFields from '../forms/EventFormFields';
import FormCheckbox from '../forms/FormCheckbox';
import RecurrenceFields from '../forms/RecurrenceFields';
import EventRemindersField from '../forms/EventRemindersField';
import RecurringEventActionModal from '../modals/RecurringEventActionModal';
import EventMeetingDetails from './EventMeetingDetails';

interface EditEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  onEventUpdated: () => void;
  event: Event | null;
  calendars: Array<{ id: string; name: string; color: string; type: string }>;
  categories: Category[];
  categoryTypes: CategoryType[];
}

export default function EditEventModal({
  isOpen,
  onClose,
  onEventUpdated,
  event,
  calendars,
  categories,
  categoryTypes,
}: EditEventModalProps) {
  const {
    formData,
    setFormData,
    loading,
    error,
    showRecurringActionModal,
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
    handleRecurringActionSelect,
    handleRecurringActionClose,
  } = useEditEventForm({
    isOpen,
    event,
    calendars,
    categories,
    categoryTypes,
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

        <EventMeetingDetails event={event} />

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
