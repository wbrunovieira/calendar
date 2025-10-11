/**
 * Recurrence Fields Component
 * Form fields for recurring event configuration
 */

import FormSelect from '../forms/FormSelect';
import FormInput from '../forms/FormInput';
import DaysOfWeekSelector from '../ui/common/DaysOfWeekSelector';

type RecurrenceFrequency = 'daily' | 'weekly' | 'monthly' | 'yearly';

interface RecurrenceFieldsProps {
  frequency: RecurrenceFrequency;
  daysOfWeek: number[];
  endDate: string;
  onFrequencyChange: (frequency: RecurrenceFrequency) => void;
  onToggleDayOfWeek: (day: number) => void;
  onEndDateChange: (date: string) => void;
}

const FREQUENCY_OPTIONS = [
  { value: 'daily', label: 'Diário' },
  { value: 'weekly', label: 'Semanal' },
  { value: 'monthly', label: 'Mensal' },
  { value: 'yearly', label: 'Anual' },
];

export default function RecurrenceFields({
  frequency,
  daysOfWeek,
  endDate,
  onFrequencyChange,
  onToggleDayOfWeek,
  onEndDateChange,
}: RecurrenceFieldsProps) {
  return (
    <div className="space-y-4 bg-white/5 p-4 rounded-lg border border-white/10">
      {/* Frequência */}
      <FormSelect
        label="Frequência"
        value={frequency}
        onChange={value => onFrequencyChange(value as RecurrenceFrequency)}
        options={FREQUENCY_OPTIONS}
      />

      {/* Dias da Semana (apenas para frequência semanal) */}
      {frequency === 'weekly' && (
        <DaysOfWeekSelector selectedDays={daysOfWeek} onToggleDay={onToggleDayOfWeek} />
      )}

      {/* Data de Término da Recorrência */}
      <FormInput label="Repetir até" type="date" value={endDate} onChange={onEndDateChange} />
    </div>
  );
}
