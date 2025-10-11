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
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-center gap-3">
          <h2 className="text-lg md:text-xl font-bold">{titleText}</h2>
          <span className="text-sm md:text-base opacity-90">{currentDate.getFullYear()}</span>
          <button
            onClick={onGoToToday}
            className="px-2 py-1 bg-white/20 hover:bg-white/30 rounded text-xs font-medium transition-all"
          >
            Hoje
          </button>
        </div>
        <div className="flex items-center justify-center gap-2 text-xs md:text-sm opacity-80">
          <span className="bg-white/10 px-2 py-0.5 rounded">Semana iniciando dia {weekDays[0].getDate()}</span>
          <span className="text-white/40">•</span>
          <span className="bg-white/10 px-2 py-0.5 rounded">{getWeekOfMonth(weekDays[0])}ª semana do mês</span>
          <span className="text-white/40">•</span>
          <span className="bg-white/10 px-2 py-0.5 rounded">Semana {getWeekNumber(weekDays[0])} do ano</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <h2 className="text-lg md:text-xl font-bold">{titleText}</h2>
      <span className="text-sm md:text-base opacity-90">{currentDate.getFullYear()}</span>
      <button
        onClick={onGoToToday}
        className="ml-2 px-2 py-1 bg-white/20 hover:bg-white/30 rounded text-xs font-medium transition-all"
      >
        Hoje
      </button>
    </div>
  );
}
