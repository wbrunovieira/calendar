export class Event {
  id: string;
  calendarId: string;
  categoryId?: string | null;
  categoryTypeId?: string | null;
  labelId?: string | null;
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

  // Tipo de evento: EVENT (compromisso), HABIT (habito), TODO (tarefa)
  eventType: string;

  // Para TODOs: prioridade (1=alta, 2=media, 3=baixa)
  priority?: number | null;

  // Para TODOs: data de vencimento
  dueDate?: Date | null;

  // Para REMINDERs: dias de antecedência para lembrar (ex: [7, 3, 1, 0] = 7 dias antes, 3 dias antes, etc)
  reminderDaysBefore?: number[] | null;

  // Hábitos semanais flexíveis
  recurrenceType?: string | null; // "FIXED" | "FLEXIBLE"
  weeklyTargetCount?: number | null; // Meta semanal (ex: 2 vezes por semana)
  weeklyPreferredDays?: string[]; // Dias preferidos ["MO", "TU", "WE", "TH", "FR", "SA", "SU"]

  // Reminders/Alerts
  reminders?: Array<{ id: string; minutesBefore: number; method: string }>;

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
    labelId?: string | null;
    title: string;
    description?: string;
    startTime: string;
    endTime?: string;
    startDate: Date;
    endDate?: Date;
    recurrenceRule?: string | null;
    recurrenceMasterId?: string | null;
    status?: string;
    eventType?: string;
    priority?: number | null;
    dueDate?: Date | null;
    reminderDaysBefore?: number[] | null;
    recurrenceType?: string | null;
    weeklyTargetCount?: number | null;
    weeklyPreferredDays?: string[];
  }): Event {
    const event = new Event({
      id: '',
      calendarId: data.calendarId,
      categoryId: data.categoryId,
      categoryTypeId: data.categoryTypeId,
      labelId: data.labelId ?? null,
      title: data.title,
      description: data.description,
      startTime: data.startTime,
      endTime: data.endTime,
      startDate: data.startDate,
      endDate: data.endDate,
      recurrenceRule: data.recurrenceRule || null,
      recurrenceMasterId: data.recurrenceMasterId || null,
      status: data.status || 'CONFIRMED',
      eventType: data.eventType || 'EVENT',
      priority: data.priority || null,
      dueDate: data.dueDate || null,
      reminderDaysBefore: data.reminderDaysBefore || null,
      recurrenceType: data.recurrenceType || null,
      weeklyTargetCount: data.weeklyTargetCount || null,
      weeklyPreferredDays: data.weeklyPreferredDays || [],
      googleEventId: null,
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
    });
    return event;
  }
}
