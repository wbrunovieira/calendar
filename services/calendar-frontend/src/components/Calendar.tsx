'use client';

import { useState, useEffect } from 'react';
import { calendars } from '@/data/calendars';
import { api } from '@/lib/api';
import { Event, Category } from '@/types/calendar';
import CreateEventModal from './CreateEventModal';
import ConfirmDeleteModal from './ConfirmDeleteModal';
import { TimeSlotView } from './TimeSlotView';

type ViewMode = 'month' | 'week' | '3days' | 'day';

export default function Calendar() {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [viewMode, setViewMode] = useState<ViewMode>('month');
  const [selectedCalendars, setSelectedCalendars] = useState<string[]>([
    'wb-digital-calendar',
    'bruno-personal-calendar',
  ]);
  const [events, setEvents] = useState<Event[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalInitialDate, setModalInitialDate] = useState<string>('');
  const [modalInitialTime, setModalInitialTime] = useState<string>('');
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [eventToDelete, setEventToDelete] = useState<Event | null>(null);
  const [preservedFormData, setPreservedFormData] = useState<any>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Event[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showSearchResults, setShowSearchResults] = useState(false);

  // Buscar eventos e categorias do backend
  const fetchData = async () => {
    try {
      setLoading(true);
      const [fetchedEvents, fetchedCategories] = await Promise.all([
        api.events.list(),
        api.categories.list(),
      ]);
      setEvents(fetchedEvents);
      setCategories(fetchedCategories);
    } catch (error) {
      console.error('Erro ao buscar dados:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  // Close search results when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('.search-container')) {
        setShowSearchResults(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Expand recurring events into individual occurrences
  const expandRecurringEvents = (events: Event[]): Event[] => {
    const expandedEvents: Event[] = [];
    const today = new Date();
    const futureLimit = new Date(today.getTime() + 365 * 24 * 60 * 60 * 1000); // 1 year ahead

    events.forEach(event => {
      if (!event.isRecurring) {
        expandedEvents.push(event);
        return;
      }

      // For recurring events, generate occurrences
      const startDate = new Date(event.startDate);
      let currentDate = new Date(startDate);

      // Limit to 50 occurrences per event to avoid performance issues
      let occurrenceCount = 0;
      const maxOccurrences = 50;

      while (currentDate <= futureLimit && occurrenceCount < maxOccurrences) {
        let shouldInclude = false;

        if (event.recurrenceFrequency === 'daily') {
          shouldInclude = true;
        } else if (event.recurrenceFrequency === 'weekly' && event.recurrenceDaysOfWeek) {
          const dayOfWeek = currentDate.getDay();
          shouldInclude = event.recurrenceDaysOfWeek.includes(dayOfWeek);
        } else if (event.recurrenceFrequency === 'monthly') {
          shouldInclude = currentDate.getDate() === startDate.getDate();
        } else if (event.recurrenceFrequency === 'yearly') {
          shouldInclude = currentDate.getMonth() === startDate.getMonth() &&
                         currentDate.getDate() === startDate.getDate();
        }

        if (shouldInclude) {
          expandedEvents.push({
            ...event,
            startDate: currentDate.toISOString(),
            // Add a unique identifier for this occurrence
            id: `${event.id}-${currentDate.toISOString().split('T')[0]}`,
          });
          occurrenceCount++;
        }

        // Move to next day
        currentDate.setDate(currentDate.getDate() + 1);

        // Check recurrence end date
        if (event.recurrenceEndDate && currentDate > new Date(event.recurrenceEndDate)) {
          break;
        }
      }
    });

    return expandedEvents;
  };

  // Search handler with debounce
  useEffect(() => {
    const searchEvents = async () => {
      if (searchQuery.trim().length === 0) {
        setSearchResults([]);
        setShowSearchResults(false);
        return;
      }

      if (searchQuery.trim().length < 2) {
        return; // Don't search for single character
      }

      setIsSearching(true);
      try {
        const results = await api.events.list({ search: searchQuery });
        // Expand recurring events into individual occurrences
        const expandedResults = expandRecurringEvents(results);
        // Sort by date (closest first)
        expandedResults.sort((a, b) => new Date(a.startDate).getTime() - new Date(b.startDate).getTime());
        setSearchResults(expandedResults);
        setShowSearchResults(true);
      } catch (error) {
        console.error('Erro ao buscar eventos:', error);
      } finally {
        setIsSearching(false);
      }
    };

    const debounceTimer = setTimeout(searchEvents, 300);
    return () => clearTimeout(debounceTimer);
  }, [searchQuery]);

  const handleEventCreated = (preservedData?: any) => {
    if (preservedData) {
      setPreservedFormData(preservedData);
    } else {
      setPreservedFormData(null);
    }
    fetchData();
  };

  const handleDeleteClick = (event: Event, e: React.MouseEvent) => {
    e.stopPropagation();
    setEventToDelete(event);
    setIsDeleteModalOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!eventToDelete) return;

    try {
      await api.events.delete(eventToDelete.id);
      setIsDeleteModalOpen(false);
      setEventToDelete(null);
      fetchData();
    } catch (error) {
      console.error('Erro ao deletar evento:', error);
      alert('Erro ao deletar evento. Tente novamente.');
    }
  };

  const handleCancelDelete = () => {
    setIsDeleteModalOpen(false);
    setEventToDelete(null);
  };

  const handleTimeSlotClick = (date: string, time: string) => {
    setModalInitialDate(date);
    setModalInitialTime(time);
    setIsModalOpen(true);
  };

  const handleSearchResultClick = (event: Event) => {
    // Parse the event date
    const eventDate = new Date(event.startDate);

    // Set the current date to the event date
    setCurrentDate(eventDate);

    // Change to day view to focus on the event
    setViewMode('day');

    // Close search results
    setShowSearchResults(false);
    setSearchQuery('');
  };

  const monthNames = [
    'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
    'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
  ];

  const daysOfWeek = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];
  const daysOfWeekFull = ['Domingo', 'Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado'];

  const getDaysInMonth = (date: Date) => {
    const year = date.getFullYear();
    const month = date.getMonth();
    return new Date(year, month + 1, 0).getDate();
  };

  const getFirstDayOfMonth = (date: Date) => {
    const year = date.getFullYear();
    const month = date.getMonth();
    return new Date(year, month, 1).getDay();
  };

  const previousPeriod = () => {
    if (viewMode === 'month') {
      setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() - 1));
    } else if (viewMode === 'week') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() - 7);
      setCurrentDate(newDate);
    } else if (viewMode === '3days') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() - 3);
      setCurrentDate(newDate);
    } else {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() - 1);
      setCurrentDate(newDate);
    }
  };

  const nextPeriod = () => {
    if (viewMode === 'month') {
      setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() + 1));
    } else if (viewMode === 'week') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() + 7);
      setCurrentDate(newDate);
    } else if (viewMode === '3days') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() + 3);
      setCurrentDate(newDate);
    } else {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() + 1);
      setCurrentDate(newDate);
    }
  };

  const goToToday = () => {
    setCurrentDate(new Date());
  };

  const getWeekDays = () => {
    const startOfWeek = new Date(currentDate);
    const day = startOfWeek.getDay();
    startOfWeek.setDate(startOfWeek.getDate() - day);

    const days = [];
    for (let i = 0; i < 7; i++) {
      const date = new Date(startOfWeek);
      date.setDate(date.getDate() + i);
      days.push(date);
    }
    return days;
  };

  const get3Days = () => {
    const days = [];
    for (let i = 0; i < 3; i++) {
      const date = new Date(currentDate);
      date.setDate(date.getDate() + i);
      days.push(date);
    }
    return days;
  };

  // Função para obter eventos de uma data específica
  const getEventsForDate = (date: Date): Event[] => {
    const dateString = date.toISOString().split('T')[0];

    console.log('=== getEventsForDate ===');
    console.log('Data procurada:', dateString);
    console.log('Total de eventos:', events.length);
    console.log('Calendários selecionados:', selectedCalendars);

    const filtered = events.filter(event => {
      console.log(`Verificando evento: ${event.title} (${event.startDate})`);

      // Filtrar por calendários selecionados
      if (!selectedCalendars.includes(event.calendarId)) {
        console.log(`  ❌ Calendário ${event.calendarId} não selecionado`);
        return false;
      }

      // Evento não recorrente
      if (!event.isRecurring) {
        // Extrair apenas a parte da data (YYYY-MM-DD) do timestamp
        const eventDate = event.startDate.split('T')[0];
        const match = eventDate === dateString;
        console.log(`  ${match ? '✅' : '❌'} Comparando: ${eventDate} === ${dateString} = ${match}`);
        return match;
      }

      // Evento recorrente
      if (event.recurrenceFrequency === 'daily') {
        console.log('  ✅ Evento recorrente diário');
        return true;
      }

      if (event.recurrenceFrequency === 'weekly' && event.recurrenceDaysOfWeek) {
        const dayOfWeek = date.getDay();
        const match = event.recurrenceDaysOfWeek.includes(dayOfWeek);
        console.log(`  ${match ? '✅' : '❌'} Evento recorrente semanal - dia da semana: ${dayOfWeek}`);
        return match;
      }

      console.log('  ❌ Não passou em nenhum filtro');
      return false;
    });

    console.log(`Eventos filtrados para ${dateString}:`, filtered.length);
    return filtered;
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
      currentDate.getMonth() === today.getMonth() &&
      currentDate.getFullYear() === today.getFullYear();

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
          onClick={(e) => {
            // Only open modal if clicking on empty space (not on an event)
            const target = e.target as HTMLElement;
            if (target === e.currentTarget || target.tagName === 'SPAN' || target.classList.contains('flex-col') || target.classList.contains('time-grid')) {
              const dateString = dayDate.toISOString().split('T')[0];
              handleTimeSlotClick(dateString, '09:00'); // Default to 9am for month view
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
            {dayEvents.slice(0, 2).map((event) => {
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
                  <button
                    onClick={(e) => handleDeleteClick(event, e)}
                    className="opacity-0 group-hover:opacity-100 transition-opacity absolute top-0 right-0 p-0.5 hover:bg-red-600 rounded"
                    title="Deletar"
                  >
                    <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              );
            })}
            {dayEvents.length > 2 && (
              <div className="text-[8px] opacity-70">+{dayEvents.length - 2} mais</div>
            )}
          </div>
        </div>
      );
    }

    return days;
  };

  const renderWeekView = () => {
    const weekDays = getWeekDays();
    return (
      <TimeSlotView
        days={weekDays}
        events={events}
        categories={categories}
        selectedCalendars={selectedCalendars}
        onDeleteClick={handleDeleteClick}
        onEventUpdate={fetchData}
        onTimeSlotClick={handleTimeSlotClick}
        daysOfWeek={daysOfWeek}
        daysOfWeekFull={daysOfWeekFull}
        monthNames={monthNames}
      />
    );
  };

  const render3DaysView = () => {
    const days = get3Days();
    return (
      <TimeSlotView
        days={days}
        events={events}
        categories={categories}
        selectedCalendars={selectedCalendars}
        onDeleteClick={handleDeleteClick}
        onEventUpdate={fetchData}
        onTimeSlotClick={handleTimeSlotClick}
        daysOfWeekFull={daysOfWeekFull}
        monthNames={monthNames}
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
        onDeleteClick={handleDeleteClick}
        onEventUpdate={fetchData}
        onTimeSlotClick={handleTimeSlotClick}
        daysOfWeekFull={daysOfWeekFull}
        monthNames={monthNames}
      />
    );
  };

  if (loading) {
    return (
      <div className="w-full max-w-6xl mx-auto p-2 md:p-4 h-full flex items-center justify-center">
        <div className="text-white text-xl">Carregando...</div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-6xl mx-auto p-2 md:p-4 min-h-screen flex items-start justify-center py-4 relative">
      {/* Floating Add Button */}
      <button
        onClick={() => setIsModalOpen(true)}
        className="fixed top-6 right-6 w-14 h-14 bg-gradient-to-br from-[#792990] to-[#350545] hover:from-[#8b2fa0] hover:to-[#461556] text-white rounded-full shadow-2xl hover:shadow-[#792990]/50 transition-all duration-300 flex items-center justify-center z-50 hover:scale-110"
        title="Criar novo evento"
      >
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
        </svg>
      </button>

      {/* Search Bar */}
      <div className="fixed top-6 left-1/2 transform -translate-x-1/2 z-50 w-full max-w-md px-4 search-container">
        <div className="relative">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onFocus={() => searchQuery && setShowSearchResults(true)}
            placeholder="Buscar eventos..."
            className="w-full px-4 py-3 pl-12 bg-white/10 backdrop-blur-md border border-white/20 text-white placeholder-white/50 rounded-xl focus:ring-2 focus:ring-[#792990] focus:border-transparent shadow-2xl"
          />
          <svg
            className="w-5 h-5 absolute left-4 top-1/2 transform -translate-y-1/2 text-white/50"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          {searchQuery && (
            <button
              onClick={() => {
                setSearchQuery('');
                setShowSearchResults(false);
              }}
              className="absolute right-4 top-1/2 transform -translate-y-1/2 text-white/50 hover:text-white"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>

        {/* Search Results Dropdown */}
        {showSearchResults && (
          <div className="absolute top-full mt-2 w-full bg-[#350545] border border-white/20 rounded-xl shadow-2xl max-h-96 overflow-y-auto">
            {isSearching ? (
              <div className="p-4 text-center text-white/70">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white mx-auto"></div>
              </div>
            ) : searchResults.length === 0 ? (
              <div className="p-4 text-center text-white/70">
                Nenhum evento encontrado
              </div>
            ) : (
              <div className="divide-y divide-white/10">
                {searchResults.map((event) => {
                  const category = categories.find(c => c.id === event.categoryId);
                  const calendar = calendars.find(c => c.id === event.calendarId);
                  const eventDate = new Date(event.startDate);
                  const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

                  // Format date to show day of week
                  const dayOfWeek = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'][eventDate.getDay()];
                  const isToday = eventDate.toDateString() === new Date().toDateString();
                  const isTomorrow = eventDate.toDateString() === new Date(Date.now() + 86400000).toDateString();

                  return (
                    <button
                      key={event.id}
                      onClick={() => handleSearchResultClick(event)}
                      className="w-full px-4 py-3 hover:bg-white/10 transition-colors text-left flex items-center gap-3"
                    >
                      <div
                        className="w-10 h-10 rounded-lg flex items-center justify-center text-lg flex-shrink-0"
                        style={{ backgroundColor: category?.color + '80' }}
                      >
                        {category?.icon || '📅'}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-white font-semibold truncate">{event.title}</span>
                          {event.isRecurring && (
                            <span className="text-xs bg-white/20 px-1.5 py-0.5 rounded" title="Evento recorrente">
                              🔄
                            </span>
                          )}
                          <span className="text-xs" style={{ color: calendar?.color }}>
                            {calendarIcon}
                          </span>
                        </div>
                        <div className="text-xs text-white/70 flex items-center gap-2">
                          {isToday && <span className="text-green-400 font-semibold">Hoje</span>}
                          {isTomorrow && <span className="text-blue-400 font-semibold">Amanhã</span>}
                          {!isToday && !isTomorrow && (
                            <>
                              <span>{dayOfWeek}</span>
                              <span>•</span>
                            </>
                          )}
                          <span>{eventDate.toLocaleDateString('pt-BR')}</span>
                          <span>•</span>
                          <span>{event.startTime}</span>
                          {event.description && (
                            <>
                              <span>•</span>
                              <span className="truncate">{event.description}</span>
                            </>
                          )}
                        </div>
                      </div>
                      <svg className="w-5 h-5 text-white/50 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Calendar Card */}
      <div className="rounded-2xl shadow-2xl w-full mt-20" style={{ backgroundColor: '#350545' }}>

        {/* Header */}
        <div className="bg-primary px-3 md:px-4 py-2 md:py-3 text-white">
          <div className="flex items-center justify-between mb-2">
            <button
              onClick={previousPeriod}
              className="p-1 hover:bg-white/20 rounded transition-all duration-200"
              aria-label="Período anterior"
            >
              <svg className="w-4 h-4 md:w-5 md:h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>

            <div className="text-center flex items-center gap-2">
              <h2 className="text-lg md:text-xl font-bold">
                {viewMode === 'month' && monthNames[currentDate.getMonth()]}
                {viewMode === 'week' && `Semana de ${currentDate.getDate()}`}
                {viewMode === '3days' && `${currentDate.getDate()} ${monthNames[currentDate.getMonth()]}`}
                {viewMode === 'day' && `${currentDate.getDate()} ${monthNames[currentDate.getMonth()]}`}
              </h2>
              <span className="text-sm md:text-base opacity-90">
                {currentDate.getFullYear()}
              </span>
              <button
                onClick={goToToday}
                className="ml-2 px-2 py-1 bg-white/20 hover:bg-white/30 rounded text-xs font-medium"
              >
                Hoje
              </button>
            </div>

            <button
              onClick={nextPeriod}
              className="p-1 hover:bg-white/20 rounded transition-all duration-200"
              aria-label="Próximo período"
            >
              <svg className="w-4 h-4 md:w-5 md:h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>

          {/* View Mode Selector */}
          <div className="flex gap-1 justify-center">
            <button
              onClick={() => setViewMode('month')}
              className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                viewMode === 'month' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
              }`}
            >
              Mês
            </button>
            <button
              onClick={() => setViewMode('week')}
              className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                viewMode === 'week' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
              }`}
            >
              Semana
            </button>
            <button
              onClick={() => setViewMode('3days')}
              className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                viewMode === '3days' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
              }`}
            >
              3 Dias
            </button>
            <button
              onClick={() => setViewMode('day')}
              className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                viewMode === 'day' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
              }`}
            >
              Dia
            </button>
          </div>
        </div>

        {/* Calendar Grid */}
        <div className="p-2 md:p-3" style={{ backgroundColor: '#350545' }}>
          {viewMode === 'month' && (
            <>
              <div className="grid grid-cols-7 gap-1 mb-3">
                {daysOfWeek.map((day, index) => (
                  <div
                    key={day}
                    className={`
                      text-center font-bold text-sm md:text-base py-1.5 rounded-lg
                      transition-all duration-200
                      ${index === 0 || index === 6
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
            {calendars.map((calendar) => (
              <button
                key={calendar.id}
                onClick={() => toggleCalendar(calendar.id)}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  selectedCalendars.includes(calendar.id)
                    ? 'bg-white/20 text-white'
                    : 'bg-white/5 text-white/50 hover:bg-white/10'
                }`}
              >
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: calendar.color }}
                />
                <span>{calendar.name}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Footer Info */}
        <div className="px-3 py-2 border-t" style={{ backgroundColor: '#350545', borderColor: '#792990' }}>
          <div className="flex items-center justify-center gap-4 text-xs text-white flex-wrap">
            <div className="flex items-center gap-1">
              <div className="w-3 h-3 rounded" style={{ background: 'linear-gradient(135deg, #792990 0%, #ffffff 100%)' }}></div>
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
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setPreservedFormData(null);
        }}
        onEventCreated={handleEventCreated}
        calendars={calendars}
        categories={categories}
        initialDate={modalInitialDate}
        initialTime={modalInitialTime}
        preservedFormData={preservedFormData}
      />

      {/* Modal de Confirmação de Exclusão */}
      <ConfirmDeleteModal
        isOpen={isDeleteModalOpen}
        event={eventToDelete}
        onConfirm={handleConfirmDelete}
        onCancel={handleCancelDelete}
      />
    </div>
  );
}
