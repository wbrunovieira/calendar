/**
 * Header Title Component
 * Displays the calendar title based on view mode with week info for week view
 */

import { useMemo } from 'react';
import { MONTH_NAMES } from '@/constants/calendar';
import { getWeekDays, getWeekOfMonth, getWeekNumber, getHeaderTitleText } from '@/utils/calendar';

type ViewMode = 'month' | 'week' | '3days' | 'day';

interface HeaderTitleProps {
  currentDate: Date;
  viewMode: ViewMode;
  onGoToToday: () => void;
}

export default function HeaderTitle({ currentDate, viewMode, onGoToToday }: HeaderTitleProps) {
  // Memoize week days calculation to avoid repeated computation
  const weekDays = useMemo(() => {
    return viewMode === 'week' ? getWeekDays(currentDate) : null;
  }, [currentDate, viewMode]);

  const titleText = getHeaderTitleText(viewMode, currentDate, MONTH_NAMES);

  if (viewMode === 'week' && weekDays) {
    return (
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-center gap-3">
          <h2 className="text-2xl md:text-3xl font-extrabold drop-shadow-lg">{titleText}</h2>
          <span className="text-base md:text-lg opacity-90 font-semibold">{currentDate.getFullYear()}</span>
          <button
            onClick={onGoToToday}
            className="px-3 py-1.5 bg-white/20 hover:bg-white/30 rounded-lg text-sm font-semibold transition-all shadow-md hover:shadow-lg backdrop-blur-sm border border-white/10 hover:scale-105"
          >
            Hoje
          </button>
        </div>
        <div className="flex items-center justify-center gap-2 text-xs md:text-sm opacity-90">
          <span className="bg-white/15 px-3 py-1 rounded-lg font-medium backdrop-blur-sm">Semana iniciando dia {weekDays[0].getDate()}</span>
          <span className="text-white/50">•</span>
          <span className="bg-white/15 px-3 py-1 rounded-lg font-medium backdrop-blur-sm">{getWeekOfMonth(weekDays[0])}ª semana do mês</span>
          <span className="text-white/50">•</span>
          <span className="bg-white/15 px-3 py-1 rounded-lg font-medium backdrop-blur-sm">Semana {getWeekNumber(weekDays[0])} do ano</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3">
      <h2 className="text-2xl md:text-3xl font-extrabold drop-shadow-lg">{titleText}</h2>
      <span className="text-base md:text-lg opacity-90 font-semibold">{currentDate.getFullYear()}</span>
      <button
        onClick={onGoToToday}
        className="ml-2 px-3 py-1.5 bg-white/20 hover:bg-white/30 rounded-lg text-sm font-semibold transition-all shadow-md hover:shadow-lg backdrop-blur-sm border border-white/10 hover:scale-105"
      >
        Hoje
      </button>
    </div>
  );
}
