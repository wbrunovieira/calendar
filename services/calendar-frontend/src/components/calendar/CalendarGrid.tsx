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
  onPreviousPeriod?: () => void;
  onNextPeriod?: () => void;
}

// Navigation arrow button component
function NavigationArrow({
  direction,
  onClick,
}: {
  direction: 'left' | 'right';
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="flex items-center justify-center w-10 h-10 md:w-12 md:h-12 rounded-full bg-white/10 hover:bg-white/20 border border-white/20 hover:border-white/40 transition-all duration-200 text-white/70 hover:text-white hover:scale-110 active:scale-95 shadow-lg"
      aria-label={direction === 'left' ? 'Período anterior' : 'Próximo período'}
    >
      <svg
        className="w-5 h-5 md:w-6 md:h-6"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d={direction === 'left' ? 'M15 19l-7-7 7-7' : 'M9 5l7 7-7 7'}
        />
      </svg>
    </button>
  );
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
  onPreviousPeriod,
  onNextPeriod,
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
        <div className="flex items-start gap-2 md:gap-4">
          {/* Left navigation arrow */}
          <div className="flex-shrink-0 pt-24 md:pt-28">
            <NavigationArrow direction="left" onClick={onPreviousPeriod} />
          </div>

          {/* TimeSlot View */}
          <div className="flex-1 min-w-0">
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
              onPreviousPeriod={onPreviousPeriod}
              onNextPeriod={onNextPeriod}
              daysOfWeek={viewMode === 'week' ? [...DAYS_OF_WEEK_SHORT] : undefined}
              daysOfWeekFull={[...DAYS_OF_WEEK_FULL]}
              monthNames={[...MONTH_NAMES]}
            />
          </div>

          {/* Right navigation arrow */}
          <div className="flex-shrink-0 pt-24 md:pt-28">
            <NavigationArrow direction="right" onClick={onNextPeriod} />
          </div>
        </div>
      )}
    </div>
  );
}
