/**
 * Calendar Grid Component
 * Renders the appropriate calendar view based on view mode
 */

import { Event, Category, CategoryType } from '@/types/calendar';
import { TimeSlotView } from '../views/TimeSlotView';
import MonthView from '../views/MonthView';
import { MONTH_NAMES, DAYS_OF_WEEK_SHORT, DAYS_OF_WEEK_FULL } from '@/constants/calendar';
import { getDaysForView } from '@/utils/calendar';

type ViewMode = 'month' | 'week' | '3days' | 'day';

interface CalendarGridProps {
  viewMode: ViewMode;
  currentDate: Date;
  events: Event[];
  categories: Category[];
  categoryTypes: CategoryType[];
  selectedCalendars: string[];
  onTimeSlotClick: (date: string, time: string) => void;
  onEditClick: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
  onEventUpdate: () => void;
}

export default function CalendarGrid({
  viewMode,
  currentDate,
  events,
  categories,
  categoryTypes,
  selectedCalendars,
  onTimeSlotClick,
  onEditClick,
  onDeleteClick,
  onEventUpdate,
}: CalendarGridProps) {
  return (
    <div className="p-2 md:p-3" style={{ backgroundColor: '#350545' }}>
      {viewMode === 'month' ? (
        <MonthView
          currentDate={currentDate}
          events={events}
          categories={categories}
          selectedCalendars={selectedCalendars}
          onTimeSlotClick={onTimeSlotClick}
          onEditClick={onEditClick}
          onDeleteClick={onDeleteClick}
        />
      ) : (
        <TimeSlotView
          days={getDaysForView(viewMode, currentDate)}
          events={events}
          categories={categories}
          categoryTypes={categoryTypes}
          selectedCalendars={selectedCalendars}
          onEditClick={onEditClick}
          onDeleteClick={onDeleteClick}
          onEventUpdate={onEventUpdate}
          onTimeSlotClick={onTimeSlotClick}
          daysOfWeek={viewMode === 'week' ? [...DAYS_OF_WEEK_SHORT] : undefined}
          daysOfWeekFull={[...DAYS_OF_WEEK_FULL]}
          monthNames={[...MONTH_NAMES]}
        />
      )}
    </div>
  );
}
