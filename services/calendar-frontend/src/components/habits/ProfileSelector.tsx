'use client';

import type { Calendar } from '@/types/calendar';

interface ProfileSelectorProps {
  calendars: Calendar[];
  selectedId: string | null;
  onSelect: (calendarId: string | null) => void;
  loading?: boolean;
}

export default function ProfileSelector({
  calendars,
  selectedId,
  onSelect,
  loading = false,
}: ProfileSelectorProps) {
  if (loading) {
    return (
      <div
        data-testid="profile-selector-loading"
        className="flex gap-2 animate-pulse"
      >
        <div className="h-10 w-20 bg-white/10 rounded-lg" />
        <div className="h-10 w-24 bg-white/10 rounded-lg" />
        <div className="h-10 w-24 bg-white/10 rounded-lg" />
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-2">
      {/* "Todos" option */}
      <button
        onClick={() => onSelect(null)}
        className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
          selectedId === null
            ? 'bg-purple-600 text-white'
            : 'bg-white/10 text-white/70 hover:bg-white/20'
        }`}
      >
        <svg
          className="w-4 h-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4 6h16M4 10h16M4 14h16M4 18h16"
          />
        </svg>
        Todos
      </button>

      {/* Calendar options */}
      {calendars.map((calendar) => (
        <button
          key={calendar.id}
          onClick={() => onSelect(calendar.id)}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
            selectedId === calendar.id
              ? 'bg-purple-600 text-white'
              : 'bg-white/10 text-white/70 hover:bg-white/20'
          }`}
        >
          {/* Calendar type icon */}
          {calendar.type === 'personal' ? (
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
              />
            </svg>
          ) : (
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          )}

          {/* Color indicator */}
          <span
            className="w-2 h-2 rounded-full"
            style={{ backgroundColor: calendar.color }}
          />

          {calendar.name}
        </button>
      ))}
    </div>
  );
}
