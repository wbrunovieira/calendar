/**
 * Time Slot Event Card Component
 * Displays an event card in the TimeSlotView with drag, resize, and action buttons
 */

import { Event, Category } from '@/types/calendar';
import { PIXELS_PER_HOUR } from '@/constants/timeSlotView';
import { timeToPixels, calculateEventHeight } from '@/utils/timeCalculations';

interface Calendar {
  id: string;
  name: string;
  color: string;
  type: string;
}

interface TimeSlotEventCardProps {
  event: Event;
  date: Date;
  category?: Category;
  calendar?: Calendar;
  daysCount: number;
  positionInfo: { column: number; totalColumns: number };
  resizingEventId?: string;
  onDragStart: (e: React.DragEvent) => void;
  onDragEnd: (e: React.DragEvent) => void;
  onResizeStart: (edge: 'top' | 'bottom', e: React.MouseEvent, height: number, top: number) => void;
  onToggleExecution: (e: React.MouseEvent) => void;
  onEditClick?: (e: React.MouseEvent) => void;
  onDeleteClick: (e: React.MouseEvent) => void;
}

export default function TimeSlotEventCard({
  event,
  date,
  category,
  calendar,
  daysCount,
  positionInfo,
  resizingEventId,
  onDragStart,
  onDragEnd,
  onResizeStart,
  onToggleExecution,
  onEditClick,
  onDeleteClick,
}: TimeSlotEventCardProps) {
  // Check execution status from event executions
  const executionDate = date.toISOString().split('T')[0];
  const execution = event.executions?.find(exec =>
    exec.executionDate.split('T')[0] === executionDate
  );
  const isCompleted = execution?.completed || false;

  // Calculate position and height based on time
  let topPosition = 0;
  let eventHeight = PIXELS_PER_HOUR; // Default 1 hour

  try {
    topPosition = timeToPixels(event.startTime);

    if (event.endTime) {
      eventHeight = calculateEventHeight(event.startTime, event.endTime);
    }
  } catch {
    // Error parsing time for event
  }

  const textSize = daysCount === 7 ? 'text-sm' : daysCount <= 3 ? 'text-base' : 'text-lg';
  const padding = daysCount === 7 ? 'px-2 py-2' : daysCount <= 3 ? 'px-3 py-2' : 'px-4 py-3';

  const widthPercentage = 100 / positionInfo.totalColumns;
  const leftPercentage = positionInfo.column * widthPercentage;

  const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

  return (
    <div
      key={event.id}
      data-event-id={event.id}
      draggable
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      className={`absolute rounded-lg text-white flex flex-col overflow-visible group ${
        isCompleted
          ? 'ring-2 ring-green-500 shadow-lg shadow-green-500/50'
          : ''
      } transition-all duration-300`}
      style={{
        backgroundColor: isCompleted
          ? category?.color + '60'
          : category?.color + '90',
        top: `${topPosition}px`,
        height: `${eventHeight}px`,
        minHeight: daysCount === 7 ? '96px' : daysCount <= 3 ? '110px' : '120px',
        left: `${leftPercentage}%`,
        width: `${widthPercentage}%`,
        zIndex: 10 + positionInfo.column,
        cursor: resizingEventId === event.id ? 'ns-resize' : 'move',
      }}
    >
      {/* Calendar icon badge - positioned at top left */}
      <div
        className="absolute -top-2 -left-2 rounded-full flex items-center justify-center shadow-lg z-30"
        style={{
          backgroundColor: calendar?.color,
          width: daysCount === 7 ? '24px' : daysCount <= 3 ? '28px' : '32px',
          height: daysCount === 7 ? '24px' : daysCount <= 3 ? '28px' : '32px',
          fontSize: daysCount === 7 ? '12px' : daysCount <= 3 ? '14px' : '16px'
        }}
      >
        {calendarIcon}
      </div>

      {/* Top resize handle */}
      <div
        className="absolute top-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
        onMouseDown={(e) => onResizeStart('top', e, eventHeight, topPosition)}
        title="Arrastar para ajustar início"
      >
        <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-t-lg" />
      </div>

      {/* Event content */}
      <div className="flex flex-1 overflow-visible cursor-move relative">
        <div className={`flex-1 ${padding} flex flex-col ${isCompleted ? 'relative' : ''}`}>
          {/* Category Icon - centered, prominent */}
          <div className="text-center pb-1">
            <span className={`${daysCount === 7 ? 'text-2xl' : daysCount <= 3 ? 'text-3xl' : 'text-4xl'}`}>
              {category?.icon}
            </span>
          </div>

          {/* Time - single line, bold */}
          <div className={`font-bold ${daysCount === 7 ? 'text-xs' : textSize} text-center leading-none whitespace-nowrap pb-2 ${isCompleted ? 'line-through opacity-70' : ''}`}>
            {event.startTime}{event.endTime && ` - ${event.endTime}`}
          </div>

          {/* Subtle divider */}
          <div className="w-full h-px bg-white/20 mb-2"></div>

          {/* Event Title - more space */}
          <div className={`${textSize} leading-relaxed ${isCompleted ? 'line-through opacity-70' : ''}`}>
            {event.title}
          </div>

          {isCompleted && (
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
              <div className="bg-green-500 rounded-full p-2 shadow-lg animate-pulse">
                <svg className={daysCount === 7 ? "w-6 h-6" : "w-8 h-8"} fill="white" viewBox="0 0 24 24">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                </svg>
              </div>
            </div>
          )}
        </div>

        <div className="flex flex-col gap-1 pr-1 pt-1">
          <button
            onClick={onToggleExecution}
            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-green-600 rounded"
            title={isCompleted ? "Marcar como não realizado" : "Marcar como realizado"}
          >
            {isCompleted ? (
              <svg className={daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="currentColor" viewBox="0 0 24 24">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
              </svg>
            ) : (
              <svg className={daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <rect x="3" y="3" width="18" height="18" rx="2" strokeWidth={2} />
              </svg>
            )}
          </button>
          {onEditClick && (
            <button
              onClick={onEditClick}
              className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-blue-600 rounded"
              title="Editar"
            >
              <svg className={daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
          )}
          <button
            onClick={onDeleteClick}
            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-600 rounded"
            title="Deletar"
          >
            <svg className={daysCount === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      {/* Bottom resize handle */}
      <div
        className="absolute bottom-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
        onMouseDown={(e) => onResizeStart('bottom', e, eventHeight, topPosition)}
        title="Arrastar para ajustar fim"
      >
        <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-b-lg mt-2" />
      </div>
    </div>
  );
}
