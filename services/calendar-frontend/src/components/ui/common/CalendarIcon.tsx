'use client';

import { useEffect, useState } from 'react';

interface CalendarIconProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export default function CalendarIcon({ size = 'md', className = '' }: CalendarIconProps) {
  const [mounted, setMounted] = useState(false);
  const [currentDay, setCurrentDay] = useState<number>(1);
  const [dayOfWeek, setDayOfWeek] = useState<string>('');

  useEffect(() => {
    // Mark as mounted to prevent hydration mismatch
    setMounted(true);

    const updateDate = () => {
      const now = new Date();
      setCurrentDay(now.getDate());

      // Get day of week in Portuguese (short form)
      const days = ['DOM', 'SEG', 'TER', 'QUA', 'QUI', 'SEX', 'SÁB'];
      setDayOfWeek(days[now.getDay()]);
    };

    // Set current day and day of week on mount
    updateDate();

    // Update at midnight every day
    const now = new Date();
    const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
    const msUntilMidnight = tomorrow.getTime() - now.getTime();

    const timeoutId = setTimeout(() => {
      updateDate();

      // Set up daily interval after first midnight
      const intervalId = setInterval(() => {
        updateDate();
      }, 24 * 60 * 60 * 1000); // 24 hours

      return () => clearInterval(intervalId);
    }, msUntilMidnight);

    return () => clearTimeout(timeoutId);
  }, []);

  const sizeClasses = {
    sm: 'w-8 h-10',
    md: 'w-10 h-12',
    lg: 'w-12 h-14',
  };

  const daySizeClasses = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base',
  };

  const weekDaySizeClasses = {
    sm: 'text-[8px]',
    md: 'text-[9px]',
    lg: 'text-[10px]',
  };

  // Prevent hydration mismatch by rendering placeholder until mounted
  if (!mounted) {
    return (
      <div
        className={`${sizeClasses[size]} bg-white/20 rounded-xl flex flex-col items-center justify-center backdrop-blur-sm border border-white/10 shadow-lg overflow-hidden ${className}`}
      >
        {/* Calendar header (purple bar at top) */}
        <div className="w-full h-2 bg-gradient-to-r from-purple-500 to-purple-700" />

        {/* Placeholder */}
        <div className="flex-1" />
      </div>
    );
  }

  return (
    <div
      className={`${sizeClasses[size]} bg-white/20 rounded-xl flex flex-col items-center justify-center backdrop-blur-sm border border-white/10 shadow-lg overflow-hidden ${className}`}
    >
      {/* Calendar header (purple bar at top) */}
      <div className="w-full h-2 bg-gradient-to-r from-purple-500 to-purple-700" />

      {/* Day number */}
      <div className={`flex-1 flex items-center justify-center ${daySizeClasses[size]} font-bold text-white`}>
        {currentDay}
      </div>

      {/* Day of week */}
      <div className={`${weekDaySizeClasses[size]} font-semibold text-white/70 pb-0.5`}>
        {dayOfWeek}
      </div>
    </div>
  );
}
