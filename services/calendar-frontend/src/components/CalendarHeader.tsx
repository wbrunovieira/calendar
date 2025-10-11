/**
 * Calendar Header Component
 * Displays navigation controls, current date/period, and view mode selector
 */

import { MONTH_NAMES } from '@/constants/calendar';
import { getWeekDays, getWeekOfMonth, getWeekNumber } from '@/utils/calendar';

type ViewMode = 'month' | 'week' | '3days' | 'day';

interface CalendarHeaderProps {
  currentDate: Date;
  viewMode: ViewMode;
  onPreviousPeriod: () => void;
  onNextPeriod: () => void;
  onGoToToday: () => void;
  onViewModeChange: (mode: ViewMode) => void;
}

export default function CalendarHeader({
  currentDate,
  viewMode,
  onPreviousPeriod,
  onNextPeriod,
  onGoToToday,
  onViewModeChange,
}: CalendarHeaderProps) {
  return (
    <div className="bg-primary px-3 md:px-4 py-2 md:py-3 text-white">
      <div className="flex items-center justify-between mb-2">
        <button
          onClick={onPreviousPeriod}
          className="p-1 hover:bg-white/20 rounded transition-all duration-200"
          aria-label="Período anterior"
        >
          <svg className="w-4 h-4 md:w-5 md:h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        <div className="text-center">
          {viewMode === 'week' ? (
            <div className="flex flex-col gap-1">
              <div className="flex items-center justify-center gap-3">
                <h2 className="text-lg md:text-xl font-bold">
                  {(() => {
                    const weekDays = getWeekDays(currentDate);
                    const firstDay = weekDays[0];
                    const lastDay = weekDays[6];
                    const sameMonth = firstDay.getMonth() === lastDay.getMonth();
                    return sameMonth
                      ? `${MONTH_NAMES[firstDay.getMonth()]}`
                      : `${MONTH_NAMES[firstDay.getMonth()]} / ${MONTH_NAMES[lastDay.getMonth()]}`;
                  })()}
                </h2>
                <span className="text-sm md:text-base opacity-90">{currentDate.getFullYear()}</span>
                <button
                  onClick={onGoToToday}
                  className="px-2 py-1 bg-white/20 hover:bg-white/30 rounded text-xs font-medium"
                >
                  Hoje
                </button>
              </div>
              <div className="flex items-center justify-center gap-2 text-xs md:text-sm opacity-80">
                <span className="bg-white/10 px-2 py-0.5 rounded">
                  Semana iniciando dia {getWeekDays(currentDate)[0].getDate()}
                </span>
                <span className="text-white/40">•</span>
                <span className="bg-white/10 px-2 py-0.5 rounded">
                  {getWeekOfMonth(getWeekDays(currentDate)[0])}ª semana do mês
                </span>
                <span className="text-white/40">•</span>
                <span className="bg-white/10 px-2 py-0.5 rounded">
                  Semana {getWeekNumber(getWeekDays(currentDate)[0])} do ano
                </span>
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <h2 className="text-lg md:text-xl font-bold">
                {viewMode === 'month' && MONTH_NAMES[currentDate.getMonth()]}
                {viewMode === '3days' && `${currentDate.getDate()} ${MONTH_NAMES[currentDate.getMonth()]}`}
                {viewMode === 'day' && `${currentDate.getDate()} ${MONTH_NAMES[currentDate.getMonth()]}`}
              </h2>
              <span className="text-sm md:text-base opacity-90">{currentDate.getFullYear()}</span>
              <button
                onClick={onGoToToday}
                className="ml-2 px-2 py-1 bg-white/20 hover:bg-white/30 rounded text-xs font-medium"
              >
                Hoje
              </button>
            </div>
          )}
        </div>

        <button
          onClick={onNextPeriod}
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
          onClick={() => onViewModeChange('month')}
          className={`px-2 py-1 rounded text-xs font-medium transition-all ${
            viewMode === 'month' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
          }`}
        >
          Mês
        </button>
        <button
          onClick={() => onViewModeChange('week')}
          className={`px-2 py-1 rounded text-xs font-medium transition-all ${
            viewMode === 'week' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
          }`}
        >
          Semana
        </button>
        <button
          onClick={() => onViewModeChange('3days')}
          className={`px-2 py-1 rounded text-xs font-medium transition-all ${
            viewMode === '3days' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
          }`}
        >
          3 Dias
        </button>
        <button
          onClick={() => onViewModeChange('day')}
          className={`px-2 py-1 rounded text-xs font-medium transition-all ${
            viewMode === 'day' ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
          }`}
        >
          Dia
        </button>
      </div>
    </div>
  );
}
