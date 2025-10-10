export class Event {
  id: string;
  calendarId: string;
  categoryId?: string | null;
  title: string;
  description?: string | null;
  startTime: string; // HH:mm format
  endTime?: string | null;
  startDate?: Date | null;
  endDate?: Date | null;
  // Recurrence
  isRecurring: boolean;
  recurrenceFrequency?: string | null; // 'daily', 'weekly', 'monthly', 'yearly'
  recurrenceInterval?: number | null;
  recurrenceDaysOfWeek: number[]; // [0,1,2,3,4,5,6]
  recurrenceDayOfMonth?: number | null;
  recurrenceWeekOfMonth?: number | null;
  recurrenceEndDate?: Date | null;
  // Google Calendar
  googleEventId?: string | null;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;

  constructor(data: Event) {
    Object.assign(this, data);
  }

  static create(data: {
    calendarId: string;
    categoryId?: string;
    title: string;
    description?: string;
    startTime: string;
    endTime?: string;
    startDate?: Date;
    endDate?: Date;
    isRecurring?: boolean;
    recurrenceFrequency?: string;
    recurrenceInterval?: number;
    recurrenceDaysOfWeek?: number[];
    recurrenceDayOfMonth?: number;
    recurrenceWeekOfMonth?: number;
    recurrenceEndDate?: Date;
  }): Event {
    return new Event({
      id: '',
      calendarId: data.calendarId,
      categoryId: data.categoryId,
      title: data.title,
      description: data.description,
      startTime: data.startTime,
      endTime: data.endTime,
      startDate: data.startDate,
      endDate: data.endDate,
      isRecurring: data.isRecurring ?? false,
      recurrenceFrequency: data.recurrenceFrequency,
      recurrenceInterval: data.recurrenceInterval ?? 1,
      recurrenceDaysOfWeek: data.recurrenceDaysOfWeek ?? [],
      recurrenceDayOfMonth: data.recurrenceDayOfMonth,
      recurrenceWeekOfMonth: data.recurrenceWeekOfMonth,
      recurrenceEndDate: data.recurrenceEndDate,
      googleEventId: null,
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
    });
  }
}
