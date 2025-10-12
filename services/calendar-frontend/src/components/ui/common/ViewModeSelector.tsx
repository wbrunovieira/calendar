/**
 * View Mode Selector Component
 * Buttons to switch between calendar view modes
 */

type ViewMode = 'month' | 'week' | '3days' | 'day';

interface ViewModeOption {
  value: ViewMode;
  label: string;
}

interface ViewModeSelectorProps {
  currentMode: ViewMode;
  onModeChange: (mode: ViewMode) => void;
}

const VIEW_MODES: ViewModeOption[] = [
  { value: 'month', label: 'Mês' },
  { value: 'week', label: 'Semana' },
  { value: '3days', label: '3 Dias' },
  { value: 'day', label: 'Dia' },
];

export default function ViewModeSelector({ currentMode, onModeChange }: ViewModeSelectorProps) {
  return (
    <div className="flex gap-2 justify-center">
      {VIEW_MODES.map(mode => (
        <button
          key={mode.value}
          onClick={() => onModeChange(mode.value)}
          className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all duration-300 ${
            currentMode === mode.value
              ? 'bg-white/30 shadow-lg border border-white/20 scale-105'
              : 'bg-white/10 hover:bg-white/20 border border-white/5 hover:scale-105 hover:shadow-md'
          } backdrop-blur-sm`}
        >
          {mode.label}
        </button>
      ))}
    </div>
  );
}
