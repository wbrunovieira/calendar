'use client';

import { Event } from '@/types/calendar';
import CalendarSearch from './CalendarSearch';
import CalendarCard from './CalendarCard';
import CalendarModals from './CalendarModals';
import CalendarLoading from './CalendarLoading';
import FloatingAddButton from './FloatingAddButton';
import { useCalendarData } from '@/hooks/useCalendarData';
import { useCalendarNavigation } from '@/hooks/useCalendarNavigation';
import { useCalendarSearch } from '@/hooks/useCalendarSearch';
import { useEventModals } from '@/hooks/useEventModals';
import { useCalendarSelection } from '@/hooks/useCalendarSelection';

export default function Calendar() {
  // Custom hooks
  const { events, categories, loading, refetch } = useCalendarData();
  const { currentDate, viewMode, setViewMode, previousPeriod, nextPeriod, goToToday, navigateToDate } =
    useCalendarNavigation();
  const { selectedCalendars, toggleCalendar } = useCalendarSelection();
  const modals = useEventModals({ onEventChange: refetch });

  const handleSearchResultNavigate = (event: Event) => {
    const eventDate = new Date(event.startDate);
    navigateToDate(eventDate);
    setViewMode('day');
  };

  const search = useCalendarSearch({
    categories,
    onResultClick: handleSearchResultNavigate,
  });

  if (loading) {
    return <CalendarLoading />;
  }

  return (
    <div className="w-full max-w-[1800px] mx-auto p-2 md:p-4 min-h-screen flex items-start justify-center py-4 relative">
      {/* Floating Add Button */}
      <FloatingAddButton onClick={modals.openCreateModal} />

      {/* Search Bar */}
      <CalendarSearch search={search} categories={categories} />

      {/* Calendar Card */}
      <CalendarCard
        currentDate={currentDate}
        viewMode={viewMode}
        onPreviousPeriod={previousPeriod}
        onNextPeriod={nextPeriod}
        onGoToToday={goToToday}
        onViewModeChange={setViewMode}
        events={events}
        categories={categories}
        selectedCalendars={selectedCalendars}
        onTimeSlotClick={modals.handleTimeSlotClick}
        onEditClick={modals.handleEditClick}
        onDeleteClick={modals.handleDeleteClick}
        onEventUpdate={refetch}
        onToggleCalendar={toggleCalendar}
      />

      {/* Modals */}
      <CalendarModals
        isCreateModalOpen={modals.isModalOpen}
        onCloseCreateModal={modals.closeCreateModal}
        onEventCreated={modals.handleEventCreated}
        categories={categories}
        modalInitialDate={modals.modalInitialDate}
        modalInitialTime={modals.modalInitialTime}
        preservedFormData={modals.preservedFormData || undefined}
        isEditModalOpen={modals.isEditModalOpen}
        onCloseEditModal={modals.closeEditModal}
        onEventUpdated={modals.handleEventUpdated}
        eventToEdit={modals.eventToEdit}
        isDeleteModalOpen={modals.isDeleteModalOpen}
        eventToDelete={modals.eventToDelete}
        onConfirmDelete={modals.handleConfirmDelete}
        onCancelDelete={modals.handleCancelDelete}
        showDeleteRecurringModal={modals.showDeleteRecurringModal}
        onRecurringDeleteClose={modals.handleRecurringDeleteClose}
        onRecurringDeleteSelect={modals.handleRecurringDeleteSelect}
      />
    </div>
  );
}
