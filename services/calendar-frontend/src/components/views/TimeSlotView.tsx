'use client';

import { useState } from 'react';
import { Event, Category, CategoryType } from '@/types/calendar';
import { calendars } from '@/data/calendars';
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
import DayViewHabitsSection from '../timeslot/DayViewHabitsSection';
import DayViewTasksSection from '../timeslot/DayViewTasksSection';
import DayViewRemindersSection from '../timeslot/DayViewRemindersSection';
import EditTodoModal from '../habits/EditTodoModal';

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
  onPreviousPeriod?: () => void;
  onNextPeriod?: () => void;
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
  onPreviousPeriod,
  onNextPeriod,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
}: TimeSlotViewProps) {
  const hours = HOURS_ARRAY;
  const today = new Date();
  const { handleDragStart, handleDragEnd, handleDragOver, handleDrop } = useDragAndDrop();

  // State for editing tasks from day view
  const [editingTodo, setEditingTodo] = useState<Event | null>(null);

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
              />
            );
          })}
        </div>

        {/* Today's Habits, Reminders and Tasks Sections - only show in single day view */}
        {isSingleDay && (
          <>
            {/* Day navigation for bottom sections */}
            {(onPreviousPeriod || onNextPeriod) && (
              <div className="mt-6 flex items-center justify-between px-2">
                <button
                  onClick={onPreviousPeriod}
                  className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-white/40 transition-all duration-200 text-white/70 hover:text-white"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                  </svg>
                  <span className="text-sm">Dia anterior</span>
                </button>
                <button
                  onClick={onNextPeriod}
                  className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-white/10 hover:bg-white/20 border border-white/20 hover:border-white/40 transition-all duration-200 text-white/70 hover:text-white"
                >
                  <span className="text-sm">Proximo dia</span>
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </button>
              </div>
            )}
            <DayViewHabitsSection date={days[0]} />
            <DayViewRemindersSection date={days[0]} onReminderToggled={onEventUpdate} />
            <DayViewTasksSection
              date={days[0]}
              onTaskToggled={onEventUpdate}
              onEditTask={setEditingTodo}
            />
          </>
        )}
      </div>

      {/* Edit Todo Modal */}
      <EditTodoModal
        isOpen={editingTodo !== null}
        onClose={() => setEditingTodo(null)}
        onUpdated={() => {
          setEditingTodo(null);
          onEventUpdate?.();
        }}
        todo={editingTodo}
        calendars={calendars}
      />
    </>
  );
}
