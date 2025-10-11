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
import { MONTH_NAMES, DAYS_OF_WEEK_SHORT, DAYS_OF_WEEK_FULL } from '@/constants/calendar';
import { getWeekDays, getNextNDays } from '@/utils/calendar';
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

  const renderWeekView = () => {
    const weekDays = getWeekDays(currentDate);
    return (
      <TimeSlotView
        days={weekDays}
        events={events}
        categories={categories}
        selectedCalendars={selectedCalendars}
        onEditClick={modals.handleEditClick}
        onDeleteClick={modals.handleDeleteClick}
        onEventUpdate={refetch}
        onTimeSlotClick={modals.handleTimeSlotClick}
        daysOfWeek={[...DAYS_OF_WEEK_SHORT]}
        daysOfWeekFull={[...DAYS_OF_WEEK_FULL]}
        monthNames={[...MONTH_NAMES]}
      />
    );
  };

  const render3DaysView = () => {
    const days = getNextNDays(currentDate, 3);
    return (
      <TimeSlotView
        days={days}
        events={events}
        categories={categories}
        selectedCalendars={selectedCalendars}
        onEditClick={modals.handleEditClick}
        onDeleteClick={modals.handleDeleteClick}
        onEventUpdate={refetch}
        onTimeSlotClick={modals.handleTimeSlotClick}
        daysOfWeekFull={[...DAYS_OF_WEEK_FULL]}
        monthNames={[...MONTH_NAMES]}
      />
    );
  };

  const renderDayView = () => {
    return (
      <TimeSlotView
        days={[currentDate]}
        events={events}
        categories={categories}
        selectedCalendars={selectedCalendars}
        onEditClick={modals.handleEditClick}
        onDeleteClick={modals.handleDeleteClick}
        onEventUpdate={refetch}
        onTimeSlotClick={modals.handleTimeSlotClick}
        daysOfWeekFull={[...DAYS_OF_WEEK_FULL]}
        monthNames={[...MONTH_NAMES]}
      />
    );
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
      <button
        onClick={modals.openCreateModal}
        className="fixed top-6 right-6 w-14 h-14 bg-gradient-to-br from-[#792990] to-[#350545] hover:from-[#8b2fa0] hover:to-[#461556] text-white rounded-full shadow-2xl hover:shadow-[#792990]/50 transition-all duration-300 flex items-center justify-center z-50 hover:scale-110"
        title="Criar novo evento"
      >
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
        </svg>
      </button>

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
          {viewMode === 'month' && (
            <MonthView
              currentDate={currentDate}
              events={events}
              categories={categories}
              selectedCalendars={selectedCalendars}
              onTimeSlotClick={modals.handleTimeSlotClick}
              onEditClick={modals.handleEditClick}
              onDeleteClick={modals.handleDeleteClick}
            />
          )}

          {viewMode === 'week' && renderWeekView()}

          {viewMode === '3days' && render3DaysView()}

          {viewMode === 'day' && renderDayView()}
        </div>

        {/* Calendar Selector */}
        <div className="px-3 py-2 border-t" style={{ backgroundColor: '#350545', borderColor: '#792990' }}>
          <div className="flex items-center justify-center gap-3 flex-wrap">
            {calendars.map(calendar => (
              <button
                key={calendar.id}
                onClick={() => toggleCalendar(calendar.id)}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  selectedCalendars.includes(calendar.id)
                    ? 'bg-white/20 text-white'
                    : 'bg-white/5 text-white/50 hover:bg-white/10'
                }`}
              >
                <div className="w-3 h-3 rounded-full" style={{ backgroundColor: calendar.color }} />
                <span>{calendar.name}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Footer Info */}
        <div className="px-3 py-2 border-t" style={{ backgroundColor: '#350545', borderColor: '#792990' }}>
          <div className="flex items-center justify-center gap-4 text-xs text-white flex-wrap">
            <div className="flex items-center gap-1">
              <div
                className="w-3 h-3 rounded"
                style={{ background: 'linear-gradient(135deg, #792990 0%, #ffffff 100%)' }}
              ></div>
              <span>Hoje</span>
            </div>
            <div className="h-3 w-px bg-white/20"></div>
            <div className="flex items-center gap-1.5">
              <span>💼</span>
              <div className="w-3 h-3 rounded" style={{ backgroundColor: '#350545' }}></div>
              <span>WB Digital</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span>👤</span>
              <div className="w-3 h-3 rounded" style={{ backgroundColor: '#792990' }}></div>
              <span>Pessoal</span>
            </div>
          </div>
        </div>
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
