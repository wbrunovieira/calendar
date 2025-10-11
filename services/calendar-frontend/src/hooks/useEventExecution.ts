/**
 * Custom hook for handling event execution toggle
 * Manages marking events as completed/not completed
 */

import { useCallback } from 'react';
import { api } from '@/lib/api';

interface UseEventExecutionProps {
  onEventUpdate?: () => void;
}

export function useEventExecution({ onEventUpdate }: UseEventExecutionProps) {
  const handleToggleExecution = useCallback(
    async (eventId: string, date: Date, currentlyCompleted: boolean) => {
      const executionDate = date.toISOString().split('T')[0];

      try {
        await api.events.toggleExecution(eventId, executionDate, !currentlyCompleted);

        // Reload events to get updated execution status
        if (onEventUpdate) {
          onEventUpdate();
        }
      } catch {
        // Error handling for execution toggle
      }
    },
    [onEventUpdate]
  );

  return {
    handleToggleExecution,
  };
}
