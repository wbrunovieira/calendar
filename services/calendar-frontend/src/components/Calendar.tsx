'use client';

import { Event } from '@/types/calendar';
import CalendarSearch from './CalendarSearch';
import CalendarHeader from './CalendarHeader';
import CalendarGrid from './CalendarGrid';
import CalendarFooter from './CalendarFooter';
import CalendarModals from './CalendarModals';
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
    return (
      <div className="w-full max-w-[1800px] mx-auto p-2 md:p-4 h-full flex items-center justify-center">
        <div className="text-white text-xl">Carregando...</div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-[1800px] mx-auto p-2 md:p-4 min-h-screen flex items-start justify-center py-4 relative">
      {/* Floating Add Button */}
      <FloatingAddButton onClick={modals.openCreateModal} />

      {/* Search Bar */}
      <CalendarSearch
        searchQuery={search.searchQuery}
        setSearchQuery={search.setSearchQuery}
        searchResults={search.searchResults}
        isSearching={search.isSearching}
        showSearchResults={search.showSearchResults}
        setShowSearchResults={search.setShowSearchResults}
        onSearchResultClick={search.handleSearchResultClick}
        onClearSearch={search.clearSearch}
        categories={categories}
      />

      {/* Calendar Card */}
      <div className="rounded-2xl shadow-2xl w-full mt-20" style={{ backgroundColor: '#350545' }}>
        {/* Header */}
        <CalendarHeader
          currentDate={currentDate}
          viewMode={viewMode}
          onPreviousPeriod={previousPeriod}
          onNextPeriod={nextPeriod}
          onGoToToday={goToToday}
          onViewModeChange={setViewMode}
        />

        {/* Calendar Grid */}
        <CalendarGrid
          viewMode={viewMode}
          currentDate={currentDate}
          events={events}
          categories={categories}
          selectedCalendars={selectedCalendars}
          onTimeSlotClick={modals.handleTimeSlotClick}
          onEditClick={modals.handleEditClick}
          onDeleteClick={modals.handleDeleteClick}
          onEventUpdate={refetch}
        />

        {/* Calendar Footer */}
        <CalendarFooter selectedCalendars={selectedCalendars} onToggleCalendar={toggleCalendar} />
      </div>

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
