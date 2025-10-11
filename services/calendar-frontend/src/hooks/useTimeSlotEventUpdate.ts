/**
 * Custom hook for handling event updates in TimeSlotView
 * Manages drag & drop, recurring event actions, and API updates
 */

import { useState, useCallback } from 'react';
import { Event } from '@/types/calendar';
import { api } from '@/lib/api';
import { extractBaseEventId } from '@/utils/timeCalculations';
import { RecurringEventAction } from '@/components/RecurringEventActionModal';

interface UseTimeSlotEventUpdateProps {
  events: Event[];
  onEventUpdate?: () => void;
}

interface PendingUpdate {
  eventId: string;
  eventTitle: string;
  originalDate: string;
  newDate: string;
  newTime?: string;
  newEndTime?: string;
}

interface DragContext {
  eventId: string;
  originalDate: string;
}

export function useTimeSlotEventUpdate({ events, onEventUpdate }: UseTimeSlotEventUpdateProps) {
  const [showRecurringActionModal, setShowRecurringActionModal] = useState(false);
  const [dragContext, setDragContext] = useState<DragContext | null>(null);
  const [pendingUpdate, setPendingUpdate] = useState<PendingUpdate | null>(null);

  const handleEventUpdate = useCallback(
    async (
      eventId: string,
      newDate: string,
      newTime?: string,
      newEndTime?: string,
      recurringAction?: RecurringEventAction,
      originalDate?: string
    ) => {
      try {
        const event = events.find(e => e.id === eventId);
        if (!event) return;

        // Determine the original date - use dragContext, passed originalDate, or event's startDate
        const eventOriginalDate = originalDate || dragContext?.originalDate || event.startDate.split('T')[0];

        // If event is recurring and no action specified, show the modal
        if (event.isRecurring && !recurringAction) {
          setPendingUpdate({
            eventId,
            eventTitle: event.title,
            originalDate: eventOriginalDate,
            newDate,
            newTime,
            newEndTime,
          });
          setShowRecurringActionModal(true);
          return;
        }

        // Extract the base event ID if it's an expanded recurring event
        const baseEventId = extractBaseEventId(eventId);

        const updatePayload: Record<string, unknown> = {
          startDate: newDate,
        };

        if (newTime) updatePayload.startTime = newTime;
        if (newEndTime) updatePayload.endTime = newEndTime;

        // Add recurring action scope if provided
        if (recurringAction) {
          updatePayload.recurringEditScope = recurringAction;
          // For all scopes, we need to pass the original occurrence date
          updatePayload.occurrenceDate = eventOriginalDate;
        }

        await api.events.update(baseEventId, updatePayload);

        // Clear drag context after update
        setDragContext(null);

        if (onEventUpdate) {
          onEventUpdate();
        }
      } catch {
        // Clear drag context even on error
        setDragContext(null);
      }
    },
    [events, dragContext, onEventUpdate]
  );

  const handleRecurringActionSelect = async (action: RecurringEventAction) => {
    setShowRecurringActionModal(false);

    if (pendingUpdate) {
      await handleEventUpdate(
        pendingUpdate.eventId,
        pendingUpdate.newDate,
        pendingUpdate.newTime,
        pendingUpdate.newEndTime,
        action,
        pendingUpdate.originalDate
      );
      setPendingUpdate(null);
    }
  };

  const handleRecurringActionClose = () => {
    setShowRecurringActionModal(false);
    setPendingUpdate(null);
  };

  return {
    handleEventUpdate,
    handleRecurringActionSelect,
    handleRecurringActionClose,
    showRecurringActionModal,
    dragContext,
    setDragContext,
    pendingUpdate,
  };
}
