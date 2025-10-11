/**
 * Calendar Modals Component
 * Groups all calendar-related modals in one place
 */

import { calendars } from '@/data/calendars';
import { Event, Category } from '@/types/calendar';
import CreateEventModal from './CreateEventModal';
import EditEventModal from './EditEventModal';
import ConfirmDeleteModal from './ConfirmDeleteModal';
import DeleteRecurringEventModal from './DeleteRecurringEventModal';

interface CalendarModalsProps {
  // Create modal
  isCreateModalOpen: boolean;
  onCloseCreateModal: () => void;
  onEventCreated: (preservedData?: Record<string, unknown>) => void;
  categories: Category[];
  modalInitialDate: string;
  modalInitialTime: string;
  preservedFormData?: Record<string, unknown>;

  // Edit modal
  isEditModalOpen: boolean;
  onCloseEditModal: () => void;
  onEventUpdated: () => void;
  eventToEdit: Event | null;

  // Delete modal
  isDeleteModalOpen: boolean;
  eventToDelete: Event | null;
  onConfirmDelete: () => void;
  onCancelDelete: () => void;

  // Recurring delete modal
  showDeleteRecurringModal: boolean;
  onRecurringDeleteClose: () => void;
  onRecurringDeleteSelect: (action: 'this' | 'all' | 'future') => void;
}

export default function CalendarModals({
  isCreateModalOpen,
  onCloseCreateModal,
  onEventCreated,
  categories,
  modalInitialDate,
  modalInitialTime,
  preservedFormData,
  isEditModalOpen,
  onCloseEditModal,
  onEventUpdated,
  eventToEdit,
  isDeleteModalOpen,
  eventToDelete,
  onConfirmDelete,
  onCancelDelete,
  showDeleteRecurringModal,
  onRecurringDeleteClose,
  onRecurringDeleteSelect,
}: CalendarModalsProps) {
  return (
    <>
      {/* Modal de Criar Evento */}
      <CreateEventModal
        isOpen={isCreateModalOpen}
        onClose={onCloseCreateModal}
        onEventCreated={onEventCreated}
        calendars={calendars}
        categories={categories}
        initialDate={modalInitialDate}
        initialTime={modalInitialTime}
        preservedFormData={preservedFormData}
      />

      {/* Modal de Editar Evento */}
      <EditEventModal
        isOpen={isEditModalOpen}
        onClose={onCloseEditModal}
        onEventUpdated={onEventUpdated}
        event={eventToEdit}
        calendars={calendars}
        categories={categories}
      />

      {/* Modal de Confirmação de Exclusão */}
      <ConfirmDeleteModal
        isOpen={isDeleteModalOpen}
        event={eventToDelete}
        onConfirm={onConfirmDelete}
        onCancel={onCancelDelete}
      />

      {/* Modal de Deletar Evento Recorrente */}
      <DeleteRecurringEventModal
        isOpen={showDeleteRecurringModal}
        onClose={onRecurringDeleteClose}
        onSelect={onRecurringDeleteSelect}
        eventTitle={eventToEdit?.title || ''}
      />
    </>
  );
}
