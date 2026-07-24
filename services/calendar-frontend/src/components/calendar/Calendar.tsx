'use client';

import { Event } from '@/types/calendar';
import CalendarSearch from '../calendar/CalendarSearch';
import CalendarCard from '../calendar/CalendarCard';
import CalendarModals from '../calendar/CalendarModals';
import CalendarLoading from '../calendar/CalendarLoading';
import FloatingAddButton from '../ui/common/FloatingAddButton';
import { useCalendarData } from '@/hooks/useCalendarData';
import { useCalendarNavigation } from '@/hooks/useCalendarNavigation';
import { useCalendarSearch } from '@/hooks/useCalendarSearch';
import { useEventModals } from '@/hooks/useEventModals';
import { useCalendarSelection } from '@/hooks/useCalendarSelection';

export default function Calendar() {
  // Custom hooks
  const { events, categories, categoryTypes, loading, refetch } = useCalendarData();
  const { currentDate, viewMode, setViewMode, previousPeriod, nextPeriod, goToToday, navigateToDate } =
    useCalendarNavigation();
  const { selectedCalendars, toggleCalendar } = useCalendarSelection();
  const modals = useEventModals({ onEventChange: refetch });

  const handleSearchResultNavigate = (event: Event) => {
    // Parse date string as local date to avoid timezone issues
    // "2025-10-20" should be Oct 20, not Oct 19 (which happens with UTC interpretation)
    const [year, month, day] = event.startDate.split('-').map(Number);
    const eventDate = new Date(year, month - 1, day);
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
        categoryTypes={categoryTypes}
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
        categoryTypes={categoryTypes}
        modalInitialDate={modals.modalInitialDate}
        modalInitialTime={modals.modalInitialTime}
        preservedFormData={modals.preservedFormData || undefined}
        isDetailModalOpen={modals.isDetailModalOpen}
        eventToView={modals.eventToView}
        onCloseDetailModal={modals.closeDetailModal}
        onEditFromDetail={modals.handleEditFromDetail}
        onDeleteFromDetail={modals.handleDeleteFromDetail}
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
