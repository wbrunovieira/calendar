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
    <div className="flex gap-1 justify-center">
      {VIEW_MODES.map(mode => (
        <button
          key={mode.value}
          onClick={() => onModeChange(mode.value)}
          className={`px-2 py-1 rounded text-xs font-medium transition-all ${
            currentMode === mode.value ? 'bg-white/30' : 'bg-white/10 hover:bg-white/20'
          }`}
        >
          {mode.label}
        </button>
      ))}
    </div>
  );
}
