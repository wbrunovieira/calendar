/**
 * Empty Day Message Component
 * Displays message when a day has no events
 */

interface EmptyDayMessageProps {
  daysCount: number;
}

export default function EmptyDayMessage({ daysCount }: EmptyDayMessageProps) {
  return (
    <div className="absolute inset-0 flex items-center justify-center">
      <div className={`text-white/30 ${daysCount === 7 ? 'text-sm' : 'text-base'}`}>
        {daysCount === 7 ? 'Vazio' : 'Sem eventos'}
      </div>
    </div>
  );
}
