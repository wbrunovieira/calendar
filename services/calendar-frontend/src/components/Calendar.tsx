'use client';

import { useState } from 'react';

type ViewMode = 'month' | 'week' | '3days' | 'day';

export default function Calendar() {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [viewMode, setViewMode] = useState<ViewMode>('month');

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

      days.push(
        <div
          key={day}
          className={`
            aspect-square p-1 flex flex-col items-center justify-center
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
          <span className="text-xs md:text-sm">{day}</span>
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

      return (
        <div
          key={index}
          className={`p-3 rounded cursor-pointer transition-all duration-200 hover:bg-white/5 ${
            isToday ? 'bg-gradient-to-br from-[#350545] to-[#792990]' : ''
          }`}
        >
          <div className="text-center">
            <div className="text-xs opacity-70 text-white">{daysOfWeek[date.getDay()]}</div>
            <div className={`text-2xl font-bold ${isToday ? 'text-white' : 'text-white'}`}>
              {date.getDate()}
            </div>
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

      return (
        <div
          key={index}
          className={`p-4 rounded cursor-pointer transition-all duration-200 hover:bg-white/5 ${
            isToday ? 'bg-gradient-to-br from-[#350545] to-[#792990]' : ''
          }`}
        >
          <div className="text-center">
            <div className="text-sm opacity-70 text-white">{daysOfWeekFull[date.getDay()]}</div>
            <div className={`text-3xl font-bold ${isToday ? 'text-white' : 'text-white'}`}>
              {date.getDate()}
            </div>
            <div className="text-xs opacity-70 text-white mt-1">
              {monthNames[date.getMonth()]}
            </div>
          </div>
        </div>
      );
    });
  };

  const renderDayView = () => {
    const today = new Date();
    const isToday = currentDate.toDateString() === today.toDateString();

    return (
      <div className="p-8 text-center">
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
    );
  };

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

        {/* Footer Info */}
        <div className="px-3 py-2 border-t" style={{ backgroundColor: '#350545', borderColor: '#792990' }}>
          <div className="flex items-center justify-center gap-2 text-xs text-white">
            <div className="flex items-center gap-1">
              <div className="w-3 h-3 rounded" style={{ background: 'linear-gradient(135deg, #792990 0%, #ffffff 100%)' }}></div>
              <span>Hoje</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
