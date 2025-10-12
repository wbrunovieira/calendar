import Calendar from '@/components/calendar/Calendar';
import AppLayout from '@/components/navigation/AppLayout';

export default function Home() {
  return (
    <AppLayout>
      {/* Calendar Component */}
      <div className="flex-1 flex items-center justify-center">
        <Calendar />
      </div>
    </AppLayout>
  );
}
