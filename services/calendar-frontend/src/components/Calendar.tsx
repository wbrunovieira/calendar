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
import {
  MONTH_NAMES,
  DAYS_OF_WEEK_SHORT,
  DAYS_OF_WEEK_FULL,
  DEFAULT_EVENT_TIME,
} from '@/constants/calendar';
import { getDaysInMonth, getFirstDayOfMonth, getWeekDays, getNextNDays } from '@/utils/calendar';
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

  // Função para obter eventos de uma data específica
  // Backend já retorna eventos expandidos, então apenas filtramos por data e calendários selecionados
  const getEventsForDate = (date: Date): Event[] => {
    const dateString = date.toISOString().split('T')[0];

    return events.filter(event => {
      // Filtrar por calendários selecionados
      if (!selectedCalendars.includes(event.calendarId)) {
        return false;
      }

      // Comparar apenas a data (YYYY-MM-DD)
      const eventDate = event.startDate.split('T')[0];
      return eventDate === dateString;
    });
  };

  const toggleCalendar = (calendarId: string) => {
    setSelectedCalendars(prev => {
      if (prev.includes(calendarId)) {
        return prev.filter(id => id !== calendarId);
      } else {
        return [...prev, calendarId];
      }
    });
  };

  const renderMonthView = () => {
    const daysInMonth = getDaysInMonth(currentDate);
    const firstDay = getFirstDayOfMonth(currentDate);
    const days = [];
    const today = new Date();
    const isCurrentMonth =
      currentDate.getMonth() === today.getMonth() && currentDate.getFullYear() === today.getFullYear();

    for (let i = 0; i < firstDay; i++) {
      days.push(<div key={`empty-${i}`} className="aspect-square p-1" />);
    }

    for (let day = 1; day <= daysInMonth; day++) {
      const isToday = isCurrentMonth && day === today.getDate();
      const dayDate = new Date(currentDate.getFullYear(), currentDate.getMonth(), day);
      const dayEvents = getEventsForDate(dayDate);

      days.push(
        <div
          key={day}
          className={`
            aspect-square p-2 flex flex-col items-start justify-start
            cursor-pointer transition-all duration-200
            border border-white/10
            hover:border-[#792990]/50 hover:shadow-lg hover:shadow-[#792990]/20
            relative overflow-hidden
            ${isToday ? 'text-white font-bold border-[#792990] shadow-xl shadow-[#792990]/30' : 'text-white'}
          `}
          style={
            isToday
              ? { background: 'linear-gradient(135deg, #350545 0%, #792990 100%)' }
              : { backgroundColor: 'rgba(255, 255, 255, 0.02)' }
          }
          onClick={e => {
            // Only open modal if clicking on empty space (not on an event)
            const target = e.target as HTMLElement;
            if (
              target === e.currentTarget ||
              target.tagName === 'SPAN' ||
              target.classList.contains('flex-col') ||
              target.classList.contains('time-grid')
            ) {
              const dateString = dayDate.toISOString().split('T')[0];
              modals.handleTimeSlotClick(dateString, DEFAULT_EVENT_TIME);
            }
          }}
        >
          {/* Subtle time grid background with hour labels */}
          <div className="time-grid absolute inset-0 pointer-events-none">
            {[6, 10, 14, 18, 22].map((hour, i) => (
              <div
                key={i}
                className="absolute left-0 right-0 border-t border-white/10 flex items-center"
                style={{ top: `${(i + 1) * 16.66}%` }}
              >
                <span className="text-[8px] text-white/30 ml-0.5">{hour.toString().padStart(2, '0')}h</span>
              </div>
            ))}
          </div>

          <span className="text-xs md:text-sm mb-1 relative z-10">{day}</span>
          <div className="flex flex-col gap-0.5 w-full">
            {dayEvents.slice(0, 2).map(event => {
              const category = categories.find(c => c.id === event.categoryId);
              const calendar = calendars.find(c => c.id === event.calendarId);
              const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
              return (
                <div
                  key={event.id}
                  className="text-[8px] md:text-[9px] rounded flex items-center overflow-hidden group relative"
                  style={{
                    backgroundColor: category?.color + '80',
                  }}
                  title={`${calendar?.name} - ${event.title} - ${event.startTime}`}
                >
                  <div
                    className="px-1 py-0.5 flex items-center justify-center text-[10px]"
                    style={{ backgroundColor: calendar?.color }}
                  >
                    {calendarIcon}
                  </div>
                  <div className="px-1 py-0.5 truncate flex-1">
                    {category?.icon} {event.title}
                  </div>
                  <div className="opacity-0 group-hover:opacity-100 transition-opacity absolute top-0 right-0 flex">
                    <button
                      onClick={e => modals.handleEditClick(event, e)}
                      className="p-0.5 hover:bg-blue-600 rounded"
                      title="Editar"
                    >
                      <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                        />
                      </svg>
                    </button>
                    <button
                      onClick={e => modals.handleDeleteClick(event, e)}
                      className="p-0.5 hover:bg-red-600 rounded"
                      title="Deletar"
                    >
                      <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  </div>
                </div>
              );
            })}
            {dayEvents.length > 2 && <div className="text-[8px] opacity-70">+{dayEvents.length - 2} mais</div>}
          </div>
        </div>
      );
    }

    return days;
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
            <>
              <div className="grid grid-cols-7 gap-1 mb-3">
                {DAYS_OF_WEEK_SHORT.map((day, index) => (
                  <div
                    key={day}
                    className={`
                      text-center font-bold text-sm md:text-base py-1.5 rounded-lg
                      transition-all duration-200
                      ${
                        index === 0 || index === 6
                          ? 'bg-gradient-to-r from-[#792990]/30 to-[#350545]/30 text-white/90 border border-[#792990]/40'
                          : 'bg-white/5 text-white/80 border border-white/10'
                      }
                      hover:bg-[#792990]/40 hover:border-[#792990]/60 hover:text-white
                    `}
                  >
                    {day}
                  </div>
                ))}
              </div>
              <div className="grid grid-cols-7 gap-1">{renderMonthView()}</div>
            </>
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
