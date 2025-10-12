/**
 * Calendar Header Component
 * Displays navigation controls, current date/period, and view mode selector
 */

import NavigationButton from '../ui/common/NavigationButton';
import HeaderTitle from '../ui/common/HeaderTitle';
import ViewModeSelector from '../ui/common/ViewModeSelector';

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
    <div className="bg-gradient-to-br from-[#350545] via-[#4a0860] to-[#792990] px-4 md:px-6 py-4 md:py-5 text-white shadow-2xl shadow-purple-900/50 backdrop-blur-sm border-b border-purple-600/20">
      {/* Navigation Row */}
      <div className="flex items-center justify-between mb-4">
        <NavigationButton direction="previous" onClick={onPreviousPeriod} label="Período anterior" />

        <HeaderTitle currentDate={currentDate} viewMode={viewMode} onGoToToday={onGoToToday} />

        <NavigationButton direction="next" onClick={onNextPeriod} label="Próximo período" />
      </div>

      {/* View Mode Selector */}
      <ViewModeSelector currentMode={viewMode} onModeChange={onViewModeChange} />
    </div>
  );
}
