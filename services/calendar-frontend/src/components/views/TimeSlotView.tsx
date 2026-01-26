'use client';

import { useEffect } from 'react';
import { Event, Category, CategoryType } from '@/types/calendar';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { useTimeSlotEventUpdate } from '@/hooks/useTimeSlotEventUpdate';
import { useEventResize } from '@/hooks/useEventResize';
import { useEventExecution } from '@/hooks/useEventExecution';
import { useEventsStats } from '@/hooks/useEventsStats';
import { HOURS_ARRAY } from '@/constants/timeSlotView';
import { calculateTimeFromOffset } from '@/utils/timeCalculations';
import { formatDateToString } from '@/utils/calendar/dateHelpers';
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
  const startDate = days[0] ? formatDateToString(days[0]) : '';
  const endDate = days[days.length - 1] ? formatDateToString(days[days.length - 1]) : '';

  const { stats } = useEventsStats({
    startDate: startDate || '',
    endDate: endDate || '',
    groupBy: 'day',
    enabled: !!startDate && !!endDate,
  });

  // Create a map of stats by date for quick lookup
  const statsMap = new Map(stats.map(stat => [stat.key, stat]));

  // Calculate stats directly from visible events (more reliable)
  const visibleDatesSet = new Set(days.map(day => formatDateToString(day)));

  console.log('[TimeSlotView] Debug visible events:', {
    visibleDates: Array.from(visibleDatesSet),
    totalEvents: events.length,
    eventsWithDates: events.slice(0, 5).map(e => ({
      title: e.title,
      calendarId: e.calendarId,
      startDate: e.startDate,
      extracted: e.startDate.split('T')[0]
    }))
  });

  const visibleEvents = events.filter(event => {
    if (!selectedCalendars.includes(event.calendarId)) return false;
    // Backend retorna startDate como string. Se tem "T", pega só a parte da data
    const eventDate = typeof event.startDate === 'string'
      ? event.startDate.split('T')[0]
      : formatDateToString(new Date(event.startDate));
    return visibleDatesSet.has(eventDate);
  });

  console.log('[TimeSlotView] Filtered visible events:', {
    count: visibleEvents.length,
    byCalendar: selectedCalendars.map(calId => ({
      calendarId: calId,
      count: visibleEvents.filter(e => e.calendarId === calId).length,
      events: visibleEvents.filter(e => e.calendarId === calId).map(e => ({
        title: e.title,
        startTime: e.startTime,
        endTime: e.endTime
      }))
    }))
  });

  // Log detalhado de cada calendário
  selectedCalendars.forEach(calId => {
    const calEvents = visibleEvents.filter(e => e.calendarId === calId);
    console.log(`\n📅 Calendar ${calId} - Total: ${calEvents.length}`);
    calEvents.forEach((evt, idx) => {
      console.log(`  ${idx + 1}. "${evt.title}" (${evt.startTime} - ${evt.endTime})`);
    });
  });

  // Count completed events
  const completedCount = visibleEvents.filter(event => {
    // Backend retorna startDate como string. Se tem "T", pega só a parte da data
    const eventDate = typeof event.startDate === 'string'
      ? event.startDate.split('T')[0]
      : formatDateToString(new Date(event.startDate));
    const execution = event.executions?.find(exec => {
      // executionDate também é string, trata da mesma forma
      const execDate = exec.executionDate
        ? (typeof exec.executionDate === 'string' ? exec.executionDate.split('T')[0] : formatDateToString(new Date(exec.executionDate)))
        : '';

      // Debug
      if (event.executions && event.executions.length > 0) {
        console.log(`[Completion Check] Event: "${event.title}"`, {
          eventDate,
          executions: event.executions.map(e => ({
            executionDate: e.executionDate,
            parsed: e.executionDate ? (typeof e.executionDate === 'string' ? e.executionDate.split('T')[0] : formatDateToString(new Date(e.executionDate))) : '',
            completed: e.completed
          })),
          match: execDate === eventDate
        });
      }

      return execDate === eventDate;
    });
    return execution?.completed || false;
  }).length;

  // Calculate stats by calendar
  const calendarStats = selectedCalendars.map(calendarId => {
    const calendarEvents = visibleEvents.filter(e => e.calendarId === calendarId);
    const calendarCompleted = calendarEvents.filter(event => {
      // Backend retorna startDate como string. Se tem "T", pega só a parte da data
      const eventDate = typeof event.startDate === 'string'
        ? event.startDate.split('T')[0]
        : formatDateToString(new Date(event.startDate));
      const execution = event.executions?.find(exec => {
        // executionDate também é string, trata da mesma forma
        const execDate = exec.executionDate
          ? (typeof exec.executionDate === 'string' ? exec.executionDate.split('T')[0] : formatDateToString(new Date(exec.executionDate)))
          : '';
        return execDate === eventDate;
      });
      return execution?.completed || false;
    }).length;

    return {
      calendarId,
      total: calendarEvents.length,
      completed: calendarCompleted,
      percentage: calendarEvents.length > 0
        ? Math.round((calendarCompleted / calendarEvents.length) * 100)
        : 0
    };
  }).filter(stat => stat.total > 0); // Only include calendars with events

  // Calculate summary stats from visible events
  const summaryStats = {
    totalEvents: visibleEvents.length,
    completedEvents: completedCount,
    percentage: visibleEvents.length > 0
      ? Math.round((completedCount / visibleEvents.length) * 100)
      : 0,
    daysCount: days.length,
    perfectDays: days.filter(day => {
      const dateStr = formatDateToString(day);
      const dayStat = statsMap.get(dateStr);
      return dayStat && dayStat.percentage === 100 && dayStat.total > 0;
    }).length,
    byCalendar: calendarStats,
  };

  const getEventsForDate = (date: Date): Event[] => {
    // Use formatDateToString to avoid UTC conversion issues
    const dateString = formatDateToString(date);

    return events.filter(event => {
      if (!selectedCalendars.includes(event.calendarId)) {
        return false;
      }

      // Backend retorna startDate como string. Se tem "T", pega só a parte da data
      const eventDate = typeof event.startDate === 'string'
        ? event.startDate.split('T')[0]
        : formatDateToString(new Date(event.startDate));
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
        {/* Summary Stats */}
        {summaryStats.totalEvents > 0 && (
          <TimeSlotSummaryStats stats={summaryStats} />
        )}

        <div className={`flex ${isSingleDay ? 'gap-4' : 'gap-3'} max-h-[900px] overflow-y-auto w-full ${isSingleDay ? 'bg-gradient-to-br from-white/5 to-white/0 rounded-2xl p-4 backdrop-blur-sm shadow-2xl' : ''}`}>
          {/* Time column */}
          <TimeColumn hours={hours} days={days} daysOfWeek={daysOfWeek} />

          {/* Day columns */}
          {days.map((date, dayIndex) => {
            const isToday = date.toDateString() === today.toDateString();
            const dayEvents = getEventsForDate(date);
            const dateString = formatDateToString(date);
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
                onHabitToggled={onEventUpdate}
              />
            );
          })}
        </div>
      </div>
    </>
  );
}
