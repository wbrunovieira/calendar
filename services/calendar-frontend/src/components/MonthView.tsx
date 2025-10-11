/**
 * Month View Component
 * Displays the calendar in month view with day cells and events
 */

import { Event, Category } from '@/types/calendar';
import { useMonthCalendar } from '@/hooks/useMonthCalendar';
import DaysOfWeekHeader from './DaysOfWeekHeader';
import MonthDayCell from './MonthDayCell';

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

      days.push(
        <MonthDayCell
          key={day}
          day={day}
          date={dayDate}
          isToday={isToday}
          events={dayEvents}
          categories={categories}
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
