/**
 * Month Event Card Component
 * Displays a single event in month view with edit/delete actions
 */

import { Event, Category } from '@/types/calendar';

interface MonthEventCardProps {
  event: Event;
  category: Category | undefined;
  calendarColor: string;
  calendarName: string;
  calendarIcon: string;
  onEditClick: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
}

export default function MonthEventCard({
  event,
  category,
  calendarColor,
  calendarName,
  calendarIcon,
  onEditClick,
  onDeleteClick,
}: MonthEventCardProps) {
  return (
    <div
      className="text-[8px] md:text-[9px] rounded flex items-center overflow-hidden group relative"
      style={{
        backgroundColor: category?.color + '80',
      }}
      title={`${calendarName} - ${event.title} - ${event.startTime}`}
    >
      <div className="px-1 py-0.5 flex items-center justify-center text-[10px]" style={{ backgroundColor: calendarColor }}>
        {calendarIcon}
      </div>
      <div className="px-1 py-0.5 truncate flex-1">
        {category?.icon} {event.title}
      </div>
      <div className="opacity-0 group-hover:opacity-100 transition-opacity absolute top-0 right-0 flex">
        <button
          onClick={e => onEditClick(event, e)}
          className="p-0.5 hover:bg-blue-600 rounded"
          title="Editar"
        >
          <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
            />
          </svg>
        </button>
        <button
          onClick={e => onDeleteClick(event, e)}
          className="p-0.5 hover:bg-red-600 rounded"
          title="Deletar"
        >
          <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        </button>
      </div>
    </div>
  );
}
