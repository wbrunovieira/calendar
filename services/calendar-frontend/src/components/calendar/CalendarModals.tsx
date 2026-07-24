/**
 * Calendar Modals Component
 * Groups all calendar-related modals in one place
 */

import { calendars } from '@/data/calendars';
import { Event, Category, CategoryType } from '@/types/calendar';
import CreateEventModal from '../modals/CreateEventModal';
import EditEventModal from '../modals/EditEventModal';
import EventDetailModal from '../modals/EventDetailModal';
import ConfirmDeleteModal from '../modals/ConfirmDeleteModal';
import DeleteRecurringEventModal from '../modals/DeleteRecurringEventModal';

interface CalendarModalsProps {
  // Create modal
  isCreateModalOpen: boolean;
  onCloseCreateModal: () => void;
  onEventCreated: (preservedData?: Record<string, unknown>) => void;
  categories: Category[];
  categoryTypes: CategoryType[];
  modalInitialDate: string;
  modalInitialTime: string;
  preservedFormData?: Record<string, unknown>;

  // Detail (read-only) modal
  isDetailModalOpen: boolean;
  eventToView: Event | null;
  onCloseDetailModal: () => void;
  onEditFromDetail: () => void;
  onDeleteFromDetail: (e: React.MouseEvent) => void;

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
  categoryTypes,
  modalInitialDate,
  modalInitialTime,
  preservedFormData,
  isDetailModalOpen,
  eventToView,
  onCloseDetailModal,
  onEditFromDetail,
  onDeleteFromDetail,
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
        categoryTypes={categoryTypes}
        initialDate={modalInitialDate}
        initialTime={modalInitialTime}
        preservedFormData={preservedFormData}
      />

      {/* Detalhe (read-only) do evento — abre ao clicar */}
      <EventDetailModal
        isOpen={isDetailModalOpen}
        event={eventToView}
        onClose={onCloseDetailModal}
        onEdit={onEditFromDetail}
        onDelete={onDeleteFromDetail}
      />

      {/* Modal de Editar Evento */}
      <EditEventModal
        isOpen={isEditModalOpen}
        onClose={onCloseEditModal}
        onEventUpdated={onEventUpdated}
        event={eventToEdit}
        calendars={calendars}
        categories={categories}
        categoryTypes={categoryTypes}
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
