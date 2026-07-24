/**
 * Custom hook for managing event modals state and actions
 */

import { useState } from 'react';
import { api } from '@/lib/api';
import { Event } from '@/types/calendar';
import { DeleteRecurringEventAction } from '@/components/modals/DeleteRecurringEventModal';

interface UseEventModalsProps {
  onEventChange: () => void;
}

export function useEventModals({ onEventChange }: UseEventModalsProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalInitialDate, setModalInitialDate] = useState<string>('');
  const [modalInitialTime, setModalInitialTime] = useState<string>('');
  const [preservedFormData, setPreservedFormData] = useState<Record<string, unknown> | null>(null);

  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [eventToEdit, setEventToEdit] = useState<Event | null>(null);

  // Read-only detail view opened by clicking an event; editing is a button inside it.
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [eventToView, setEventToView] = useState<Event | null>(null);

  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [eventToDelete, setEventToDelete] = useState<Event | null>(null);

  const [showDeleteRecurringModal, setShowDeleteRecurringModal] = useState(false);
  const [deleteOccurrenceDate, setDeleteOccurrenceDate] = useState<string>('');

  const handleEventCreated = (preservedData?: Record<string, unknown>) => {
    if (preservedData) {
      setPreservedFormData(preservedData);
    } else {
      setPreservedFormData(null);
    }
    onEventChange();
  };

  // Clicking an event (its body or the pencil) opens the read-only detail view.
  const handleEditClick = (event: Event, e: React.MouseEvent) => {
    e.stopPropagation();
    setEventToView(event);
    setIsDetailModalOpen(true);
  };

  const closeDetailModal = () => setIsDetailModalOpen(false);

  // "Edit" button inside the detail view: hand off to the edit form.
  const handleEditFromDetail = () => {
    if (!eventToView) return;
    setIsDetailModalOpen(false);
    setEventToEdit(eventToView);
    setIsEditModalOpen(true);
  };

  const handleDeleteFromDetail = (e: React.MouseEvent) => {
    if (!eventToView) return;
    setIsDetailModalOpen(false);
    handleDeleteClick(eventToView, e);
  };

  const handleDeleteClick = (event: Event, e: React.MouseEvent) => {
    e.stopPropagation();

    // Extract occurrence date if it's an expanded recurring event
    let occurrenceDate = event.startDate.split('T')[0];
    if (event.occurrenceDate) {
      occurrenceDate = event.occurrenceDate;
    }

    setEventToDelete(event);
    setDeleteOccurrenceDate(occurrenceDate);

    // If event is recurring, show the recurring delete modal
    if (event.isRecurring) {
      setShowDeleteRecurringModal(true);
    } else {
      setIsDeleteModalOpen(true);
    }
  };

  const handleEventUpdated = () => {
    onEventChange();
  };

  const handleRecurringDeleteSelect = async (action: DeleteRecurringEventAction) => {
    setShowDeleteRecurringModal(false);

    if (!eventToDelete) return;

    try {
      // Extract the base event ID if it's an expanded recurring event
      let baseEventId = eventToDelete.id;
      const datePattern = /-\d{4}-\d{2}-\d{2}$/;
      if (datePattern.test(eventToDelete.id)) {
        baseEventId = eventToDelete.id.replace(datePattern, '');
      }

      // Call delete API with the scope
      await api.events.deleteRecurring(baseEventId, action, deleteOccurrenceDate);

      setEventToDelete(null);
      setDeleteOccurrenceDate('');
      onEventChange();
    } catch {
      alert('Erro ao deletar evento. Tente novamente.');
    }
  };

  const handleRecurringDeleteClose = () => {
    setShowDeleteRecurringModal(false);
    setEventToDelete(null);
    setDeleteOccurrenceDate('');
  };

  const handleConfirmDelete = async () => {
    if (!eventToDelete) return;

    try {
      await api.events.delete(eventToDelete.id);
      setIsDeleteModalOpen(false);
      setEventToDelete(null);
      onEventChange();
    } catch {
      alert('Erro ao deletar evento. Tente novamente.');
    }
  };

  const handleCancelDelete = () => {
    setIsDeleteModalOpen(false);
    setEventToDelete(null);
  };

  const handleTimeSlotClick = (date: string, time: string) => {
    setModalInitialDate(date);
    setModalInitialTime(time);
    setIsModalOpen(true);
  };

  const closeCreateModal = () => {
    setIsModalOpen(false);
    setPreservedFormData(null);
  };

  const closeEditModal = () => {
    setIsEditModalOpen(false);
  };

  const openCreateModal = () => {
    setIsModalOpen(true);
  };

  return {
    // Create modal
    isModalOpen,
    modalInitialDate,
    modalInitialTime,
    preservedFormData,
    openCreateModal,
    closeCreateModal,
    handleTimeSlotClick,
    handleEventCreated,

    // Detail (read-only) modal
    isDetailModalOpen,
    eventToView,
    closeDetailModal,
    handleEditFromDetail,
    handleDeleteFromDetail,

    // Edit modal
    isEditModalOpen,
    eventToEdit,
    closeEditModal,
    handleEditClick,
    handleEventUpdated,

    // Delete modal
    isDeleteModalOpen,
    eventToDelete,
    handleDeleteClick,
    handleConfirmDelete,
    handleCancelDelete,

    // Recurring delete modal
    showDeleteRecurringModal,
    handleRecurringDeleteSelect,
    handleRecurringDeleteClose,
  };
}
