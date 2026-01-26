export class CalendarEvent {
  constructor(
    public readonly id: string,
    public readonly googleEventId: string,
    public readonly calendarId: string,
    public readonly title: string,
    public readonly startDateTime: Date,
    public readonly endDateTime: Date,
    public readonly isAllDay: boolean,
    public readonly attendees: string[],
    public readonly status: EventStatus,
    public readonly visibility: EventVisibility,
    public readonly createdAt: Date,
    public readonly updatedAt: Date,
    public readonly description?: string,
    public readonly location?: string,
    public readonly recurrence?: string,
    public readonly reminders?: Reminder[],
  ) {}
}

export interface Reminder {
  method: 'email' | 'popup';
  minutes: number;
}

export enum EventStatus {
  CONFIRMED = 'confirmed',
  TENTATIVE = 'tentative',
  CANCELLED = 'cancelled',
}

export enum EventVisibility {
  DEFAULT = 'default',
  PUBLIC = 'public',
  PRIVATE = 'private',
  CONFIDENTIAL = 'confidential',
}
