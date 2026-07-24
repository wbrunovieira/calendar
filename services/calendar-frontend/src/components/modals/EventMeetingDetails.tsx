'use client';

import { Event } from '@/types/calendar';

// Read-only meeting details synced from Google (Meet link, guests, location).
// Not editable here — the app never writes these back to Google.

const RESPONSE_MARK: Record<string, string> = {
  accepted: '✓',
  declined: '✗',
  tentative: '?',
  needsAction: '•',
};

export default function EventMeetingDetails({ event }: { event: Event | null }) {
  if (!event) return null;
  const { location, meetingUrl, attendees, organizer } = event;
  const hasAttendees = Array.isArray(attendees) && attendees.length > 0;

  if (!location && !meetingUrl && !hasAttendees) return null;

  return (
    <div className="border-t border-white/10 pt-4 space-y-3 text-sm">
      <p className="text-white/60 font-medium">Detalhes da reunião (Google)</p>

      {location && (
        <div className="flex items-center gap-2 text-white/80">
          <span aria-hidden>📍</span>
          <span>{location}</span>
        </div>
      )}

      {meetingUrl && (
        <a
          href={meetingUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 bg-emerald-600/80 hover:bg-emerald-600 text-white px-3 py-2 rounded-lg transition-colors"
        >
          <span aria-hidden>🎥</span> Entrar no Google Meet
        </a>
      )}

      {hasAttendees && (
        <div className="space-y-1">
          <p className="text-white/60">Convidados ({attendees!.length})</p>
          <ul className="space-y-0.5">
            {attendees!.map((a) => (
              <li key={a.email} className="flex items-center gap-2 text-white/80">
                <span className="w-4 text-center text-white/50">
                  {RESPONSE_MARK[a.responseStatus ?? ''] ?? '•'}
                </span>
                <span className="truncate">{a.displayName || a.email}</span>
                {organizer?.email === a.email && (
                  <span className="text-white/40 text-xs">organizador</span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
