'use client';

import { useState, useEffect } from 'react';
import { calendars } from '@/data/calendars';
import { api } from '@/lib/api';
import { Event, Category } from '@/types/calendar';

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

  // Buscar eventos e categorias do backend
  useEffect(() => {
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

    fetchData();
  }, []);

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

    return events.filter(event => {
      // Filtrar por calendários selecionados
      if (!selectedCalendars.includes(event.calendarId)) return false;

      // Evento não recorrente
      if (!event.isRecurring) {
        return event.startDate === dateString;
      }

      // Evento recorrente
      if (event.recurrenceFrequency === 'daily') {
        return true;
      }

      if (event.recurrenceFrequency === 'weekly' && event.recurrenceDaysOfWeek) {
        const dayOfWeek = date.getDay();
        return event.recurrenceDaysOfWeek.includes(dayOfWeek);
      }

      return false;
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
            aspect-square p-1 flex flex-col items-start justify-start
            rounded cursor-pointer transition-all duration-200
            hover:scale-105
            ${isToday ? 'text-white font-bold shadow-lg' : 'text-white'}
          `}
          style={
            isToday
              ? { background: 'linear-gradient(135deg, #350545 0%, #792990 100%)' }
              : {}
          }
          onMouseEnter={(e) => {
            if (!isToday) e.currentTarget.style.backgroundColor = '#79299015';
          }}
          onMouseLeave={(e) => {
            if (!isToday) e.currentTarget.style.backgroundColor = '';
          }}
        >
          <span className="text-xs md:text-sm mb-1">{day}</span>
          <div className="flex flex-col gap-0.5 w-full">
            {dayEvents.slice(0, 2).map((event) => {
              const category = categories.find(c => c.id === event.categoryId);
              const calendar = calendars.find(c => c.id === event.calendarId);
              const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
              return (
                <div
                  key={event.id}
                  className="text-[8px] md:text-[9px] rounded flex items-center overflow-hidden"
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
    const today = new Date();

    return weekDays.map((date, index) => {
      const isToday = date.toDateString() === today.toDateString();
      const dayEvents = getEventsForDate(date);

      return (
        <div
          key={index}
          className={`p-3 rounded cursor-pointer transition-all duration-200 hover:bg-white/5 ${
            isToday ? 'bg-gradient-to-br from-[#350545] to-[#792990]' : ''
          } flex flex-col`}
        >
          <div className="text-center mb-2">
            <div className="text-xs opacity-70 text-white">{daysOfWeek[date.getDay()]}</div>
            <div className={`text-2xl font-bold ${isToday ? 'text-white' : 'text-white'}`}>
              {date.getDate()}
            </div>
          </div>
          <div className="flex flex-col gap-1 flex-1">
            {dayEvents.map((event) => {
              const category = categories.find(c => c.id === event.categoryId);
              const calendar = calendars.find(c => c.id === event.calendarId);
              const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
              return (
                <div
                  key={event.id}
                  className="text-[10px] rounded flex overflow-hidden"
                  style={{
                    backgroundColor: category?.color + '80',
                  }}
                  title={`${calendar?.name} - ${event.description}`}
                >
                  <div
                    className="px-1.5 py-1 flex items-center justify-center text-xs"
                    style={{ backgroundColor: calendar?.color }}
                  >
                    {calendarIcon}
                  </div>
                  <div className="px-2 py-1 flex-1">
                    <div className="font-semibold flex items-center gap-1">
                      <span>{category?.icon}</span>
                      <span>{event.startTime}</span>
                    </div>
                    <div className="truncate">{event.title}</div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      );
    });
  };

  const render3DaysView = () => {
    const days = get3Days();
    const today = new Date();

    return days.map((date, index) => {
      const isToday = date.toDateString() === today.toDateString();
      const dayEvents = getEventsForDate(date);

      return (
        <div
          key={index}
          className={`p-4 rounded cursor-pointer transition-all duration-200 hover:bg-white/5 ${
            isToday ? 'bg-gradient-to-br from-[#350545] to-[#792990]' : ''
          } flex flex-col`}
        >
          <div className="text-center mb-3">
            <div className="text-sm opacity-70 text-white">{daysOfWeekFull[date.getDay()]}</div>
            <div className={`text-3xl font-bold ${isToday ? 'text-white' : 'text-white'}`}>
              {date.getDate()}
            </div>
            <div className="text-xs opacity-70 text-white mt-1">
              {monthNames[date.getMonth()]}
            </div>
          </div>
          <div className="flex flex-col gap-2 flex-1">
            {dayEvents.map((event) => {
              const category = categories.find(c => c.id === event.categoryId);
              const calendar = calendars.find(c => c.id === event.calendarId);
              const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
              return (
                <div
                  key={event.id}
                  className="text-xs rounded flex overflow-hidden"
                  style={{
                    backgroundColor: category?.color + '80',
                  }}
                  title={`${calendar?.name} - ${event.description}`}
                >
                  <div
                    className="px-2 py-2 flex items-center justify-center text-sm"
                    style={{ backgroundColor: calendar?.color }}
                  >
                    {calendarIcon}
                  </div>
                  <div className="px-3 py-2 flex-1">
                    <div className="font-semibold flex items-center gap-1">
                      <span>{category?.icon}</span>
                      <span>{event.startTime}</span>
                      {event.endTime && <span>- {event.endTime}</span>}
                    </div>
                    <div className="mt-1">{event.title}</div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      );
    });
  };

  const renderDayView = () => {
    const today = new Date();
    const isToday = currentDate.toDateString() === today.toDateString();
    const dayEvents = getEventsForDate(currentDate);

    return (
      <div className="p-4 md:p-8">
        <div className="text-center mb-6">
          <div className="text-sm opacity-70 text-white mb-2">
            {daysOfWeekFull[currentDate.getDay()]}
          </div>
          <div
            className={`text-6xl font-bold mb-2 ${isToday ? 'text-white' : 'text-white'}`}
          >
            {currentDate.getDate()}
          </div>
          <div className="text-xl opacity-90 text-white">
            {monthNames[currentDate.getMonth()]} {currentDate.getFullYear()}
          </div>
        </div>

        {/* Lista de eventos do dia */}
        <div className="max-w-2xl mx-auto">
          <div className="flex flex-col gap-3">
            {dayEvents.length === 0 ? (
              <div className="text-center text-white/50 py-8">
                Nenhum evento para este dia
              </div>
            ) : (
              dayEvents.map((event) => {
                const category = categories.find(c => c.id === event.categoryId);
                const calendar = calendars.find(c => c.id === event.calendarId);
                const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
                return (
                  <div
                    key={event.id}
                    className="rounded-lg text-white flex overflow-hidden"
                    style={{ backgroundColor: category?.color + '40' }}
                  >
                    <div
                      className="px-3 py-4 flex items-center justify-center text-3xl"
                      style={{ backgroundColor: calendar?.color }}
                    >
                      {calendarIcon}
                    </div>
                    <div className="flex-1 p-4">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2 mb-2">
                          <span className="text-2xl">{category?.icon}</span>
                          <div>
                            <div className="font-bold text-lg">{event.title}</div>
                            <div className="text-sm opacity-80 flex items-center gap-1">
                              <span>{calendar?.name}</span>
                              <span className="text-xs opacity-50">•</span>
                              <span>{category?.name}</span>
                            </div>
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="font-semibold">{event.startTime}</div>
                          {event.endTime && <div className="text-sm opacity-80">até {event.endTime}</div>}
                        </div>
                      </div>
                      {event.description && (
                        <div className="text-sm opacity-90 mt-2">{event.description}</div>
                      )}
                      {event.isRecurring && (
                        <div className="text-xs opacity-70 mt-2 flex items-center gap-1">
                          <span>🔁</span>
                          <span>Evento recorrente</span>
                        </div>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>
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
    <div className="w-full max-w-6xl mx-auto p-2 md:p-4 h-full flex items-center justify-center">
      {/* Calendar Card */}
      <div className="rounded-2xl shadow-2xl overflow-hidden w-full" style={{ backgroundColor: '#350545', maxHeight: '90vh' }}>

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
              <div className="grid grid-cols-7 gap-1 mb-2">
                {daysOfWeek.map((day) => (
                  <div key={day} className="text-center font-semibold text-xs md:text-sm py-1 text-white">
                    {day}
                  </div>
                ))}
              </div>
              <div className="grid grid-cols-7 gap-1">{renderMonthView()}</div>
            </>
          )}

          {viewMode === 'week' && (
            <div className="grid grid-cols-7 gap-2">{renderWeekView()}</div>
          )}

          {viewMode === '3days' && (
            <div className="grid grid-cols-3 gap-4">{render3DaysView()}</div>
          )}

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
    </div>
  );
}
