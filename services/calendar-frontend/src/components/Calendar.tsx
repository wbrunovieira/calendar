'use client';

import { useState } from 'react';
import { calendars } from '@/data/calendars';
import { Event } from '@/types/calendar';
import CreateEventModal from './CreateEventModal';
import EditEventModal from './EditEventModal';
import ConfirmDeleteModal from './ConfirmDeleteModal';
import DeleteRecurringEventModal from './DeleteRecurringEventModal';
import { TimeSlotView } from './TimeSlotView';
import CalendarSearch from './CalendarSearch';
import CalendarHeader from './CalendarHeader';
import MonthView from './MonthView';
import CalendarFooter from './CalendarFooter';
import FloatingAddButton from './FloatingAddButton';
import { MONTH_NAMES, DAYS_OF_WEEK_SHORT, DAYS_OF_WEEK_FULL } from '@/constants/calendar';
import { getDaysForView } from '@/utils/calendar';
import { useCalendarData } from '@/hooks/useCalendarData';
import { useCalendarNavigation } from '@/hooks/useCalendarNavigation';
import { useCalendarSearch } from '@/hooks/useCalendarSearch';
import { useEventModals } from '@/hooks/useEventModals';

export default function Calendar() {
  const [selectedCalendars, setSelectedCalendars] = useState<string[]>([
    'wb-digital-calendar',
    'bruno-personal-calendar',
  ]);

  // Custom hooks
  const { events, categories, loading, refetch } = useCalendarData();

  const { currentDate, viewMode, setViewMode, previousPeriod, nextPeriod, goToToday, navigateToDate } =
    useCalendarNavigation();

  const handleSearchResultNavigate = (event: Event) => {
    const eventDate = new Date(event.startDate);
    navigateToDate(eventDate);
    setViewMode('day');
  };

  const search = useCalendarSearch({
    categories,
    onResultClick: handleSearchResultNavigate,
  });

  const modals = useEventModals({
    onEventChange: refetch,
  });

  const toggleCalendar = (calendarId: string) => {
    setSelectedCalendars(prev => {
      if (prev.includes(calendarId)) {
        return prev.filter(id => id !== calendarId);
      } else {
        return [...prev, calendarId];
      }
    });
  };

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
        <div className="p-2 md:p-3" style={{ backgroundColor: '#350545' }}>
          {viewMode === 'month' ? (
            <MonthView
              currentDate={currentDate}
              events={events}
              categories={categories}
              selectedCalendars={selectedCalendars}
              onTimeSlotClick={modals.handleTimeSlotClick}
              onEditClick={modals.handleEditClick}
              onDeleteClick={modals.handleDeleteClick}
            />
          ) : (
            <TimeSlotView
              days={getDaysForView(viewMode, currentDate)}
              events={events}
              categories={categories}
              selectedCalendars={selectedCalendars}
              onEditClick={modals.handleEditClick}
              onDeleteClick={modals.handleDeleteClick}
              onEventUpdate={refetch}
              onTimeSlotClick={modals.handleTimeSlotClick}
              daysOfWeek={viewMode === 'week' ? [...DAYS_OF_WEEK_SHORT] : undefined}
              daysOfWeekFull={[...DAYS_OF_WEEK_FULL]}
              monthNames={[...MONTH_NAMES]}
            />
          )}
        </div>

        {/* Calendar Footer */}
        <CalendarFooter selectedCalendars={selectedCalendars} onToggleCalendar={toggleCalendar} />
      </div>

      {/* Modal de Criar Evento */}
      <CreateEventModal
        isOpen={modals.isModalOpen}
        onClose={modals.closeCreateModal}
        onEventCreated={modals.handleEventCreated}
        calendars={calendars}
        categories={categories}
        initialDate={modals.modalInitialDate}
        initialTime={modals.modalInitialTime}
        preservedFormData={modals.preservedFormData || undefined}
      />

      {/* Modal de Editar Evento */}
      <EditEventModal
        isOpen={modals.isEditModalOpen}
        onClose={modals.closeEditModal}
        onEventUpdated={modals.handleEventUpdated}
        event={modals.eventToEdit}
        calendars={calendars}
        categories={categories}
      />

      {/* Modal de Confirmação de Exclusão */}
      <ConfirmDeleteModal
        isOpen={modals.isDeleteModalOpen}
        event={modals.eventToDelete}
        onConfirm={modals.handleConfirmDelete}
        onCancel={modals.handleCancelDelete}
      />

      {/* Modal de Deletar Evento Recorrente */}
      <DeleteRecurringEventModal
        isOpen={modals.showDeleteRecurringModal}
        onClose={modals.handleRecurringDeleteClose}
        onSelect={modals.handleRecurringDeleteSelect}
        eventTitle={modals.eventToEdit?.title || ''}
      />
    </div>
  );
}
