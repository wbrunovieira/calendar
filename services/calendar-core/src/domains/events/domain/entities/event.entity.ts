export class Event {
  id: string;
  calendarId: string;
  categoryId?: string | null;
  categoryTypeId?: string | null;
  title: string;
  description?: string | null;
  startTime: string; // HH:mm format
  endTime?: string | null;
  startDate: Date;
  endDate?: Date | null;

  // Recurrence (RRULE format)
  recurrenceRule?: string | null; // RRULE string ou NULL para one-time events

  // Hierarquia (Google Calendar Pattern)
  recurrenceMasterId?: string | null; // NULL = master, UUID = filho derivado

  // Status
  status: string; // CONFIRMED, CANCELLED, TENTATIVE

  // Google Calendar
  googleEventId?: string | null;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;

  constructor(data: any) {
    Object.assign(this, data);
  }

  static create(data: {
    calendarId: string;
    categoryId?: string;
    categoryTypeId?: string;
    title: string;
    description?: string;
    startTime: string;
    endTime?: string;
    startDate: Date;
    endDate?: Date;
    recurrenceRule?: string | null;
    recurrenceMasterId?: string | null;
    status?: string;
  }): Event {
    const event = new Event({
      id: '',
      calendarId: data.calendarId,
      categoryId: data.categoryId,
      categoryTypeId: data.categoryTypeId,
      title: data.title,
      description: data.description,
      startTime: data.startTime,
      endTime: data.endTime,
      startDate: data.startDate,
      endDate: data.endDate,
      recurrenceRule: data.recurrenceRule || null,
      recurrenceMasterId: data.recurrenceMasterId || null,
      status: data.status || 'CONFIRMED',
      googleEventId: null,
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
    });
    return event;
  }
}
