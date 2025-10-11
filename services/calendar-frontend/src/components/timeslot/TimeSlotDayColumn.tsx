/**
 * Time Slot Day Column Component
 * Renders a single day column with event positioning logic and event cards
 */

import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { calculateTimeFromOffset } from '@/utils/timeCalculations';
import TimeSlotDayHeader from '../timeslot/TimeSlotDayHeader';
import TimeSlotGrid from '../timeslot/TimeSlotGrid';
import TimeSlotEventCard from '../timeslot/TimeSlotEventCard';
import EmptyDayMessage from '../ui/common/EmptyDayMessage';

interface TimeSlotDayColumnProps {
  date: Date;
  dayIndex: number;
  isToday: boolean;
  dayEvents: Event[];
  categories: Category[];
  hours: number[];
  daysCount: number;
  daysOfWeek?: string[];
  daysOfWeekFull: string[];
  monthNames: string[];
  resizingEventId?: string;
  isResizing: boolean;
  justResized: boolean;
  onDragOver: (e: React.DragEvent) => void;
  onDrop: (e: React.DragEvent) => void;
  onTimeSlotClick?: (dateString: string, time: string) => void;
  onEventDragStart: (event: Event, executionDate: string, e: React.DragEvent) => void;
  onEventDragEnd: (e: React.DragEvent) => void;
  onEventResizeStart: (eventId: string, edge: 'top' | 'bottom', e: React.MouseEvent, height: number, top: number) => void;
  onEventToggleExecution: (eventId: string, date: Date, isCompleted: boolean, e: React.MouseEvent) => void;
  onEventEdit?: (event: Event, e: React.MouseEvent) => void;
  onEventDelete: (event: Event, e: React.MouseEvent) => void;
}

export default function TimeSlotDayColumn({
  date,
  dayIndex,
  isToday,
  dayEvents,
  categories,
  hours,
  daysCount,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
  resizingEventId,
  isResizing,
  justResized,
  onDragOver,
  onDrop,
  onTimeSlotClick,
  onEventDragStart,
  onEventDragEnd,
  onEventResizeStart,
  onEventToggleExecution,
  onEventEdit,
  onEventDelete,
}: TimeSlotDayColumnProps) {
  // Calculate event positions for this day
  const eventsWithTimes = dayEvents.map(event => {
    const startTotalMinutes = event.startTime.split(':').map(Number).reduce((h, m) => h * 60 + m);
    const endTotalMinutes = event.endTime ? event.endTime.split(':').map(Number).reduce((h, m) => h * 60 + m) : startTotalMinutes + 60;
    return {
      event,
      start: startTotalMinutes,
      end: endTotalMinutes,
      column: 0,
      totalColumns: 1,
    };
  });

  eventsWithTimes.sort((a, b) => a.start !== b.start ? a.start - b.start : (b.end - b.start) - (a.end - a.start));

  const columns: Array<Array<typeof eventsWithTimes[0]>> = [];
  eventsWithTimes.forEach(eventTime => {
    let placedInColumn = false;
    for (let i = 0; i < columns.length; i++) {
      const column = columns[i];
      const hasOverlap = column.some(existing => !(eventTime.end <= existing.start || eventTime.start >= existing.end));
      if (!hasOverlap) {
        column.push(eventTime);
        eventTime.column = i;
        placedInColumn = true;
        break;
      }
    }
    if (!placedInColumn) {
      columns.push([eventTime]);
      eventTime.column = columns.length - 1;
    }
  });

  eventsWithTimes.forEach(eventTime => {
    const overlappingEvents = eventsWithTimes.filter(other => !(eventTime.end <= other.start || eventTime.start >= other.end));
    const maxColumn = Math.max(...overlappingEvents.map(e => e.column));
    eventTime.totalColumns = maxColumn + 1;
  });

  const eventPositions = new Map();
  eventsWithTimes.forEach(eventTime => {
    eventPositions.set(eventTime.event.id, {
      column: eventTime.column,
      totalColumns: eventTime.totalColumns,
    });
  });

  const handleTimeSlotClick = (e: React.MouseEvent) => {
    // Don't open modal if currently resizing or just finished resizing
    if (isResizing || justResized) {
      return;
    }

    // Only open modal if clicking on empty space (not on an event or button)
    const target = e.target as HTMLElement;
    const isEvent = target.closest('.cursor-move'); // Event cards have cursor-move
    const isButton = target.closest('button'); // Delete buttons
    const isResizeHandle = target.closest('.group\\/handle'); // Resize handles

    if (!isEvent && !isButton && !isResizeHandle) {
      const container = e.currentTarget;
      const rect = container.getBoundingClientRect();
      const offsetY = e.clientY - rect.top;

      const time = calculateTimeFromOffset(offsetY);
      const dateString = date.toISOString().split('T')[0];

      if (onTimeSlotClick) {
        onTimeSlotClick(dateString, time);
      }
    }
  };

  const isSingleDay = daysCount === 1;

  return (
    <div key={dayIndex} className="flex-1 min-w-0 flex flex-col">
      {/* Day header */}
      <TimeSlotDayHeader
        date={date}
        isToday={isToday}
        daysOfWeek={daysOfWeek}
        daysOfWeekFull={daysOfWeekFull}
        monthNames={monthNames}
        daysCount={daysCount}
      />

      {/* Time grid for this day */}
      <div
        className={`flex-1 relative border-l min-h-[1632px] transition-all duration-300 cursor-pointer ${
          isSingleDay
            ? 'border-white/20 hover:bg-white/10 rounded-r-xl shadow-inner'
            : 'border-white/10 hover:bg-white/5'
        }`}
        onDragOver={onDragOver}
        onDrop={onDrop}
        onClick={handleTimeSlotClick}
      >
        {/* Time slot grid */}
        <TimeSlotGrid hours={hours} />

        {/* Events positioned absolutely */}
        {dayEvents.map((event) => {
          const category = categories.find(c => c.id === event.categoryId);
          const calendar = calendars.find(c => c.id === event.calendarId);
          const executionDate = date.toISOString().split('T')[0];
          const positionInfo = eventPositions.get(event.id) || { column: 0, totalColumns: 1 };
          const execution = event.executions?.find(exec => exec.executionDate.split('T')[0] === executionDate);
          const isCompleted = execution?.completed || false;

          return (
            <TimeSlotEventCard
              key={event.id}
              event={event}
              date={date}
              category={category}
              calendar={calendar}
              daysCount={daysCount}
              positionInfo={positionInfo}
              resizingEventId={resizingEventId}
              onDragStart={(e) => onEventDragStart(event, executionDate, e)}
              onDragEnd={onEventDragEnd}
              onResizeStart={(edge, e, height, top) => onEventResizeStart(event.id, edge, e, height, top)}
              onToggleExecution={(e) => onEventToggleExecution(event.id, date, isCompleted, e)}
              onEditClick={onEventEdit ? (e) => onEventEdit(event, e) : undefined}
              onDeleteClick={(e) => onEventDelete(event, e)}
            />
          );
        })}

        {/* Show message if no events for this day */}
        {dayEvents.length === 0 && (
          <EmptyDayMessage daysCount={daysCount} />
        )}
      </div>
    </div>
  );
}
