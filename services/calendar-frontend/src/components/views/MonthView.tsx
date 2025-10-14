/**
 * Month View Component
 * Displays the calendar in month view with day cells and events
 */

import { Event, Category } from '@/types/calendar';
import { useMonthCalendar } from '@/hooks/useMonthCalendar';
import { useEventsStats } from '@/hooks/useEventsStats';
import DaysOfWeekHeader from '../month/DaysOfWeekHeader';
import MonthDayCell from '../month/MonthDayCell';

interface MonthViewProps {
  currentDate: Date;
  events: Event[];
  categories: Category[];
  selectedCalendars: string[];
  onTimeSlotClick: (date: string, time: string) => void;
  onEditClick: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
}

export default function MonthView({
  currentDate,
  events,
  categories,
  selectedCalendars,
  onTimeSlotClick,
  onEditClick,
  onDeleteClick,
}: MonthViewProps) {
  const { daysInMonth, firstDay, today, isCurrentMonth, getEventsForDate } = useMonthCalendar({
    currentDate,
    events,
    selectedCalendars,
  });

  // Fetch stats for the entire month
  const firstDayOfMonth = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1);
  const lastDayOfMonth = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 0);
  const startDate = firstDayOfMonth.toISOString().split('T')[0];
  const endDate = lastDayOfMonth.toISOString().split('T')[0];

  const { stats } = useEventsStats({
    startDate,
    endDate,
    groupBy: 'day',
    enabled: true,
  });

  // Create a map of stats by date for quick lookup
  const statsMap = new Map(stats.map(stat => [stat.key, stat]));

  const renderMonthDays = () => {
    const days = [];

    // Empty cells before first day of month
    for (let i = 0; i < firstDay; i++) {
      days.push(<div key={`empty-${i}`} className="aspect-square p-1" />);
    }

    // Days of the month
    for (let day = 1; day <= daysInMonth; day++) {
      const isToday = isCurrentMonth && day === today.getDate();
      const dayDate = new Date(currentDate.getFullYear(), currentDate.getMonth(), day);
      const dayEvents = getEventsForDate(dayDate);
      const dateString = dayDate.toISOString().split('T')[0];
      const dayStat = statsMap.get(dateString);

      days.push(
        <MonthDayCell
          key={day}
          day={day}
          date={dayDate}
          isToday={isToday}
          events={dayEvents}
          categories={categories}
          stats={dayStat ? {
            total: dayStat.total,
            completed: dayStat.completed,
            percentage: dayStat.percentage
          } : undefined}
          onTimeSlotClick={onTimeSlotClick}
          onEditClick={onEditClick}
          onDeleteClick={onDeleteClick}
        />
      );
    }

    return days;
  };

  return (
    <>
      <DaysOfWeekHeader />
      <div className="grid grid-cols-7 gap-1">{renderMonthDays()}</div>
    </>
  );
}
