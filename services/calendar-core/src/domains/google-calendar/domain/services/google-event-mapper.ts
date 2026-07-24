import { calendar_v3 } from 'googleapis';
import { Event } from '../../../events/domain/entities/event.entity';
import { GoogleEventPayload } from '../../infrastructure/services/google-calendar-api.client';

const TIMEZONE = 'America/Sao_Paulo';

/**
 * A Google event projected onto our Event shape. Every field here is always produced by
 * fromGoogleEvent — unlike Partial<Event>, which would make callers guard fields that cannot
 * actually be missing.
 */
export interface EventAttendee {
  email: string;
  displayName: string | null;
  responseStatus: string | null;
}

export interface EventOrganizer {
  email: string;
  displayName: string | null;
}

export interface MappedGoogleEvent {
  googleEventId: string;
  calendarId: string;
  title: string;
  description: string | null;
  startDate: Date;
  endDate: Date | null;
  startTime: string;
  endTime: string | null;
  recurrenceRule: string | null;
  status: string;
  eventType: string;
  location: string | null;
  meetingUrl: string | null;
  attendees: EventAttendee[] | null;
  organizer: EventOrganizer | null;
}

export class GoogleEventMapper {
  static toGoogleEvent(event: Event): GoogleEventPayload {
    const isAllDay = !event.endTime;

    const startDateStr = event.startDate.toISOString().split('T')[0];
    const endDateStr = (event.endDate ?? event.startDate).toISOString().split('T')[0];

    const start = isAllDay
      ? { date: startDateStr }
      : { dateTime: `${startDateStr}T${event.startTime}:00`, timeZone: TIMEZONE };

    const end = isAllDay
      ? { date: endDateStr }
      : {
          dateTime: `${endDateStr}T${(event.endTime ?? event.startTime)}:00`,
          timeZone: TIMEZONE,
        };

    const payload: GoogleEventPayload = {
      summary: event.title,
      start,
      end,
    };

    if (event.description) {
      payload.description = event.description;
    }

    if (event.recurrenceRule) {
      payload.recurrence = [`RRULE:${event.recurrenceRule}`];
    }

    if (event.reminders && event.reminders.length > 0) {
      payload.reminders = {
        useDefault: false,
        overrides: event.reminders.map((r) => ({
          method: r.method === 'notification' ? 'popup' : r.method,
          minutes: r.minutesBefore,
        })),
      };
    } else {
      payload.reminders = { useDefault: true };
    }

    if (event.status === 'CANCELLED') {
      payload.status = 'cancelled';
    }

    return payload;
  }

  static fromGoogleEvent(
    googleEvent: calendar_v3.Schema$Event,
    calendarId: string,
  ): MappedGoogleEvent {
    const isAllDay = Boolean(googleEvent.start?.date && !googleEvent.start?.dateTime);

    const startDateStr = isAllDay
      ? googleEvent.start!.date!
      : googleEvent.start!.dateTime!.split('T')[0];

    const endDateStr = isAllDay
      ? googleEvent.end!.date!
      : googleEvent.end!.dateTime!.split('T')[0];

    const startTime = isAllDay
      ? '00:00'
      : googleEvent.start!.dateTime!.split('T')[1].substring(0, 5);

    const endTime = isAllDay ? null : googleEvent.end!.dateTime!.split('T')[1].substring(0, 5);

    const rrule = googleEvent.recurrence
      ?.find((r) => r.startsWith('RRULE:'))
      ?.replace('RRULE:', '');

    const status = googleEvent.status === 'cancelled' ? 'CANCELLED' : 'CONFIRMED';

    return {
      googleEventId: googleEvent.id!,
      calendarId,
      title: googleEvent.summary ?? '(sem título)',
      description: googleEvent.description ?? null,
      startDate: new Date(startDateStr),
      endDate: endDateStr !== startDateStr ? new Date(endDateStr) : null,
      startTime,
      endTime,
      recurrenceRule: rrule ?? null,
      status,
      eventType: 'EVENT',
      location: googleEvent.location ?? null,
      meetingUrl: this.extractMeetingUrl(googleEvent),
      attendees: this.extractAttendees(googleEvent),
      organizer: googleEvent.organizer?.email
        ? { email: googleEvent.organizer.email, displayName: googleEvent.organizer.displayName ?? null }
        : null,
    };
  }

  /** The video call link — Google's `hangoutLink`, or the video entry point of `conferenceData`. */
  private static extractMeetingUrl(googleEvent: calendar_v3.Schema$Event): string | null {
    if (googleEvent.hangoutLink) return googleEvent.hangoutLink;
    const video = googleEvent.conferenceData?.entryPoints?.find(
      (e) => e.entryPointType === 'video' && e.uri,
    );
    return video?.uri ?? null;
  }

  private static extractAttendees(
    googleEvent: calendar_v3.Schema$Event,
  ): EventAttendee[] | null {
    if (!googleEvent.attendees?.length) return null;
    return googleEvent.attendees
      .filter((a) => a.email)
      .map((a) => ({
        email: a.email!,
        displayName: a.displayName ?? null,
        responseStatus: a.responseStatus ?? null,
      }));
  }
}
