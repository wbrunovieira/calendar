/**
 * Time Slot Grid Component
 * Displays the time slot grid background for a day
 */

interface TimeSlotGridProps {
  hours: number[];
}

export default function TimeSlotGrid({ hours }: TimeSlotGridProps) {
  return (
    <>
      {hours.map(hour => (
        <div key={hour} className="h-24 border-b border-white/5 relative">
          <div className="h-12 border-b border-dashed border-white/5" />
          <div className="h-12" />
        </div>
      ))}
    </>
  );
}
