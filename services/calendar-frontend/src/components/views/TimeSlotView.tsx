'use client';

import { Event, Category, CategoryType } from '@/types/calendar';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { useTimeSlotEventUpdate } from '@/hooks/useTimeSlotEventUpdate';
import { useEventResize } from '@/hooks/useEventResize';
import { useEventExecution } from '@/hooks/useEventExecution';
import { useEventsStats } from '@/hooks/useEventsStats';
import { HOURS_ARRAY } from '@/constants/timeSlotView';
import { calculateTimeFromOffset } from '@/utils/timeCalculations';
import RecurringEventActionModal from '../modals/RecurringEventActionModal';
import TimeColumn from '../timeslot/TimeColumn';
import TimeSlotDayColumn from '../timeslot/TimeSlotDayColumn';
import TimeSlotSummaryStats from '../timeslot/TimeSlotSummaryStats';

interface TimeSlotViewProps {
  days: Date[];
  events: Event[];
  categories: Category[];
  categoryTypes: CategoryType[];
  selectedCalendars: string[];
  onEditClick?: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
  onEventUpdate?: () => void;
  onTimeSlotClick?: (date: string, time: string) => void;
  daysOfWeek?: string[];
  daysOfWeekFull: string[];
  monthNames: string[];
}

export function TimeSlotView({
  days,
  events,
  categories,
  categoryTypes,
  selectedCalendars,
  onEditClick,
  onDeleteClick,
  onEventUpdate,
  onTimeSlotClick,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
}: TimeSlotViewProps) {
  const hours = HOURS_ARRAY;
  const today = new Date();
  const { handleDragStart, handleDragEnd, handleDragOver, handleDrop } = useDragAndDrop();

  // Custom hooks for business logic
  const {
    handleEventUpdate,
    handleRecurringActionSelect,
    handleRecurringActionClose,
    showRecurringActionModal,
    setDragContext,
    pendingUpdate,
  } = useTimeSlotEventUpdate({ events, onEventUpdate });

  const { handleResizeStart, isResizing, justResized, resizingEvent } = useEventResize({
    events,
    onEventUpdate: handleEventUpdate,
  });

  const { handleToggleExecution } = useEventExecution({ onEventUpdate });

  // Fetch stats for the visible days
  const startDate = days[0]?.toISOString().split('T')[0];
  const endDate = days[days.length - 1]?.toISOString().split('T')[0];

  const { stats } = useEventsStats({
    startDate: startDate || '',
    endDate: endDate || '',
    groupBy: 'day',
    enabled: !!startDate && !!endDate,
  });

  // Create a map of stats by date for quick lookup
  const statsMap = new Map(stats.map(stat => [stat.key, stat]));

  // Calculate summary stats for visible days
  const summaryStats = {
    totalEvents: stats.reduce((sum, stat) => sum + stat.total, 0),
    completedEvents: stats.reduce((sum, stat) => sum + stat.completed, 0),
    percentage: stats.length > 0
      ? Math.round((stats.reduce((sum, stat) => sum + stat.completed, 0) / stats.reduce((sum, stat) => sum + stat.total, 0)) * 100) || 0
      : 0,
    daysCount: days.length,
    perfectDays: stats.filter(stat => stat.percentage === 100 && stat.total > 0).length,
  };

  const getEventsForDate = (date: Date): Event[] => {
    const dateString = date.toISOString().split('T')[0];

    return events.filter(event => {
      if (!selectedCalendars.includes(event.calendarId)) {
        return false;
      }

      // Backend já expande eventos recorrentes, então apenas comparamos a data
      // Se o evento tem occurrenceDate, significa que já foi expandido pelo backend
      const eventDate = event.startDate.split('T')[0];
      return eventDate === dateString;
    });
  };

  const isSingleDay = days.length === 1;

  return (
    <>
      <RecurringEventActionModal
        isOpen={showRecurringActionModal}
        onClose={handleRecurringActionClose}
        onSelect={handleRecurringActionSelect}
        eventTitle={pendingUpdate?.eventTitle || ''}
      />
      <div className={`p-4 w-full ${isSingleDay ? 'max-w-5xl mx-auto' : ''}`}>
        {/* Summary Stats - only show for multiple days */}
        {days.length > 1 && summaryStats.totalEvents > 0 && (
          <TimeSlotSummaryStats stats={summaryStats} />
        )}

        <div className={`flex ${isSingleDay ? 'gap-4' : 'gap-3'} max-h-[900px] overflow-y-auto w-full ${isSingleDay ? 'bg-gradient-to-br from-white/5 to-white/0 rounded-2xl p-4 backdrop-blur-sm shadow-2xl' : ''}`}>
          {/* Time column */}
          <TimeColumn hours={hours} days={days} daysOfWeek={daysOfWeek} />

          {/* Day columns */}
          {days.map((date, dayIndex) => {
            const isToday = date.toDateString() === today.toDateString();
            const dayEvents = getEventsForDate(date);
            const dateString = date.toISOString().split('T')[0];
            const dayStat = statsMap.get(dateString);

            return (
              <TimeSlotDayColumn
                key={dayIndex}
                date={date}
                dayIndex={dayIndex}
                isToday={isToday}
                dayEvents={dayEvents}
                categories={categories}
                categoryTypes={categoryTypes}
                hours={hours}
                daysCount={days.length}
                daysOfWeek={daysOfWeek}
                daysOfWeekFull={daysOfWeekFull}
                monthNames={monthNames}
                resizingEventId={resizingEvent?.id}
                isResizing={isResizing}
                justResized={justResized}
                dayStats={dayStat ? {
                  total: dayStat.total,
                  completed: dayStat.completed,
                  percentage: dayStat.percentage
                } : undefined}
                onDragOver={handleDragOver}
                onDrop={e => {
                  const container = e.currentTarget;
                  const rect = container.getBoundingClientRect();
                  const offsetY = e.clientY - rect.top;
                  const time = calculateTimeFromOffset(offsetY);
                  handleDrop(date, time)(e, handleEventUpdate);
                }}
                onTimeSlotClick={onTimeSlotClick}
                onEventDragStart={(event, executionDate, e) => {
                  setDragContext({
                    eventId: event.id,
                    originalDate: executionDate,
                  });
                  handleDragStart(event)(e);
                }}
                onEventDragEnd={handleDragEnd}
                onEventResizeStart={handleResizeStart}
                onEventToggleExecution={(eventId, date, isCompleted, e) => {
                  e.stopPropagation();
                  handleToggleExecution(eventId, date, isCompleted);
                }}
                onEventEdit={onEditClick}
                onEventDelete={onDeleteClick}
              />
            );
          })}
        </div>
      </div>
    </>
  );
}
