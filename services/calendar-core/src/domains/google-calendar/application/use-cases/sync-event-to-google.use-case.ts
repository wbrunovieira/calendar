import { Injectable, Logger } from '@nestjs/common';
import { Event } from '../../../events/domain/entities/event.entity';
import { Calendar } from '../../../calendars/domain/entities/calendar.entity';
import { GoogleCalendarApiClient } from '../../infrastructure/services/google-calendar-api.client';
import { GoogleEventMapper } from '../../domain/services/google-event-mapper';

@Injectable()
export class SyncEventToGoogleUseCase {
  private readonly logger = new Logger(SyncEventToGoogleUseCase.name);

  constructor(private readonly apiClient: GoogleCalendarApiClient) {}

  async onCreate(event: Event, calendar: Calendar): Promise<string | null> {
    if (!calendar.googleCalendarId || !calendar.email) return null;

    try {
      const payload = GoogleEventMapper.toGoogleEvent(event);
      const googleEventId = await this.apiClient.createEvent(
        calendar.googleCalendarId,
        calendar.email,
        payload,
      );
      return googleEventId;
    } catch (err) {
      this.logger.error(`Failed to create event in Google Calendar: ${(err as Error).message}`);
      return null;
    }
  }

  async onUpdate(event: Event, calendar: Calendar): Promise<void> {
    if (!event.googleEventId || !calendar.googleCalendarId || !calendar.email) return;

    try {
      const payload = GoogleEventMapper.toGoogleEvent(event);
      await this.apiClient.updateEvent(
        calendar.googleCalendarId,
        event.googleEventId,
        calendar.email,
        payload,
      );
    } catch (err) {
      this.logger.error(`Failed to update event in Google Calendar: ${(err as Error).message}`);
    }
  }

  async onDelete(googleEventId: string, calendar: Calendar): Promise<void> {
    if (!googleEventId || !calendar.googleCalendarId || !calendar.email) return;

    try {
      await this.apiClient.deleteEvent(
        calendar.googleCalendarId,
        googleEventId,
        calendar.email,
      );
    } catch (err) {
      this.logger.error(`Failed to delete event in Google Calendar: ${(err as Error).message}`);
    }
  }
}
