/**
 * Calendar Card Component
 * Groups the calendar header, grid, and footer into a single card
 */

import { Event, Category, CategoryType } from '@/types/calendar';
import CalendarHeader from '../calendar/CalendarHeader';
import CalendarGrid from '../calendar/CalendarGrid';
import CalendarFooter from '../calendar/CalendarFooter';

type ViewMode = 'month' | 'week' | '3days' | 'day';

interface CalendarCardProps {
  // Navigation props
  currentDate: Date;
  viewMode: ViewMode;
  onPreviousPeriod: () => void;
  onNextPeriod: () => void;
  onGoToToday: () => void;
  onViewModeChange: (mode: ViewMode) => void;

  // Data props
  events: Event[];
  categories: Category[];
  categoryTypes: CategoryType[];
  selectedCalendars: string[];

  // Event handlers
  onTimeSlotClick: (date: string, time: string) => void;
  onEditClick: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
  onEventUpdate: () => void;
  onToggleCalendar: (calendarId: string) => void;
}

export default function CalendarCard({
  currentDate,
  viewMode,
  onPreviousPeriod,
  onNextPeriod,
  onGoToToday,
  onViewModeChange,
  events,
  categories,
  categoryTypes,
  selectedCalendars,
  onTimeSlotClick,
  onEditClick,
  onDeleteClick,
  onEventUpdate,
  onToggleCalendar,
}: CalendarCardProps) {
  return (
    <div className="rounded-2xl shadow-2xl w-full mt-20" style={{ backgroundColor: '#350545' }}>
      {/* Header */}
      <CalendarHeader
        currentDate={currentDate}
        viewMode={viewMode}
        onPreviousPeriod={onPreviousPeriod}
        onNextPeriod={onNextPeriod}
        onGoToToday={onGoToToday}
        onViewModeChange={onViewModeChange}
      />

      {/* Calendar Grid */}
      <CalendarGrid
        viewMode={viewMode}
        currentDate={currentDate}
        events={events}
        categories={categories}
        categoryTypes={categoryTypes}
        selectedCalendars={selectedCalendars}
        onTimeSlotClick={onTimeSlotClick}
        onEditClick={onEditClick}
        onDeleteClick={onDeleteClick}
        onEventUpdate={onEventUpdate}
        onPreviousPeriod={onPreviousPeriod}
        onNextPeriod={onNextPeriod}
      />

      {/* Calendar Footer */}
      <CalendarFooter selectedCalendars={selectedCalendars} onToggleCalendar={onToggleCalendar} />
    </div>
  );
}
