import { useState } from 'react';
import { Event } from '@/types/calendar';

export function useDragAndDrop() {
  const [draggedEvent, setDraggedEvent] = useState<Event | null>(null);

  const handleDragStart = (event: Event) => (e: React.DragEvent) => {
    setDraggedEvent(event);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/html', e.currentTarget.innerHTML);

    // Add visual feedback
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '0.5';
    }
  };

  const handleDragEnd = (e: React.DragEvent) => {
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '1';
    }
    setDraggedEvent(null);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = (newDate: Date, newTime?: string) => async (
    e: React.DragEvent,
    onUpdate: (eventId: string, newDate: string, newTime?: string) => Promise<void>
  ) => {
    e.preventDefault();

    if (!draggedEvent) return;

    const dateString = newDate.toISOString().split('T')[0];

    try {
      await onUpdate(draggedEvent.id, dateString, newTime);
    } catch (error) {
      console.error('Error updating event:', error);
    }

    setDraggedEvent(null);
  };

  return {
    draggedEvent,
    handleDragStart,
    handleDragEnd,
    handleDragOver,
    handleDrop,
  };
}
