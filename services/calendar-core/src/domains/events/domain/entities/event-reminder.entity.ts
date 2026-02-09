export class EventReminder {
  id: string;
  eventId: string;
  minutesBefore: number;
  method: string;
  createdAt: Date;

  constructor(data: any) {
    Object.assign(this, data);
  }

  static create(data: {
    eventId: string;
    minutesBefore: number;
    method?: string;
  }): EventReminder {
    return new EventReminder({
      id: '',
      eventId: data.eventId,
      minutesBefore: data.minutesBefore,
      method: data.method || 'notification',
      createdAt: new Date(),
    });
  }
}
