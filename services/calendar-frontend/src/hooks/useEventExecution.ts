/**
 * Custom hook for handling event execution toggle
 * Manages marking events as completed/not completed
 */

import { useCallback } from 'react';
import { api } from '@/lib/api';
import { formatDateToString } from '@/utils/calendar/dateHelpers';

interface UseEventExecutionProps {
  onEventUpdate?: () => void;
}

export function useEventExecution({ onEventUpdate }: UseEventExecutionProps) {
  const handleToggleExecution = useCallback(
    async (eventId: string, date: Date, currentlyCompleted: boolean) => {
      const executionDate = formatDateToString(date);

      console.log('[useEventExecution] Toggle execution:', {
        eventId,
        date,
        executionDate,
        currentlyCompleted,
        newStatus: !currentlyCompleted
      });

      try {
        const result = await api.events.toggleExecution(eventId, executionDate, !currentlyCompleted);
        console.log('[useEventExecution] API response:', result);

        // Reload events to get updated execution status
        if (onEventUpdate) {
          onEventUpdate();
        }
      } catch (error) {
        console.error('[useEventExecution] Error toggling execution:', error);
        alert('Erro ao marcar evento como concluído. Verifique o console para mais detalhes.');
      }
    },
    [onEventUpdate]
  );

  return {
    handleToggleExecution,
  };
}
