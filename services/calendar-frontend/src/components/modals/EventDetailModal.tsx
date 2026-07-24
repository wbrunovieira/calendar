'use client';

import { Event } from '@/types/calendar';
import ModalContainer from '../ui/modal/ModalContainer';
import ModalHeader from '../ui/modal/ModalHeader';
import EventMeetingDetails from './EventMeetingDetails';

interface EventDetailModalProps {
  isOpen: boolean;
  event: Event | null;
  onClose: () => void;
  onEdit: () => void;
  onDelete: (e: React.MouseEvent) => void;
}

function formatDate(iso?: string): string {
  if (!iso) return '';
  const [y, m, d] = iso.split('T')[0].split('-');
  return `${d}/${m}/${y}`;
}

export default function EventDetailModal({
  isOpen,
  event,
  onClose,
  onEdit,
  onDelete,
}: EventDetailModalProps) {
  if (!event) return null;

  const timeRange = event.endTime
    ? `${event.startTime}–${event.endTime}`
    : event.startTime;

  return (
    <ModalContainer isOpen={isOpen} onClose={onClose}>
      <ModalHeader title={event.title} onClose={onClose} />

      <div className="p-6 space-y-4">
        <div className="flex items-center gap-2 text-white/80 text-sm">
          <span aria-hidden>🗓️</span>
          <span>
            {formatDate(event.occurrenceDate || event.startDate)} · {timeRange}
          </span>
        </div>

        {event.description && (
          <p className="text-sm text-white/70 whitespace-pre-line">{event.description}</p>
        )}

        <EventMeetingDetails event={event} />

        <div className="flex gap-3 pt-2 border-t border-white/10">
          <button
            type="button"
            onClick={onDelete}
            className="px-4 py-3 text-red-300 hover:text-red-200 hover:bg-red-900/30 rounded-lg font-semibold transition-colors"
          >
            Excluir
          </button>
          <button
            type="button"
            onClick={onClose}
            className="flex-1 px-6 py-3 bg-white/10 text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/20"
          >
            Fechar
          </button>
          <button
            type="button"
            onClick={onEdit}
            className="flex-1 px-6 py-3 bg-purple-600 text-white rounded-lg font-semibold hover:bg-purple-700 transition-colors"
          >
            Editar
          </button>
        </div>
      </div>
    </ModalContainer>
  );
}
