import Calendar from '@/components/calendar/Calendar';

export default function Home() {
  return (
    <div
      className="min-h-screen flex flex-col py-8 px-4 relative overflow-hidden"
      style={{
        background: 'linear-gradient(135deg, #350545 0%, #792990 50%, #350545 100%)',
      }}
    >
      {/* Animated gradient overlay */}
      <div
        className="absolute inset-0 opacity-30"
        style={{
          background: 'radial-gradient(circle at 20% 50%, #792990 0%, transparent 50%), radial-gradient(circle at 80% 80%, #350545 0%, transparent 50%)',
        }}
      />

      <div className="max-w-7xl mx-auto w-full flex flex-col flex-1 relative z-10">
        {/* Calendar Component */}
        <div className="flex-1 flex items-center justify-center">
          <Calendar />
        </div>
      </div>
    </div>
  );
}
