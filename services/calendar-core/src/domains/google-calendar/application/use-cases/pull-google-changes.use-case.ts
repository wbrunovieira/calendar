import { Injectable, Logger } from '@nestjs/common';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { EventRepository } from '../../../events/infrastructure/repositories/event.repository';
import { GoogleCalendarApiClient } from '../../infrastructure/services/google-calendar-api.client';
import { GoogleEventMapper } from '../../domain/services/google-event-mapper';
import { Event } from '../../../events/domain/entities/event.entity';
import { calendar_v3 } from 'googleapis';

@Injectable()
export class PullGoogleChangesUseCase {
  private readonly logger = new Logger(PullGoogleChangesUseCase.name);

  constructor(
    private readonly calendarRepository: CalendarRepository,
    private readonly eventRepository: EventRepository,
    private readonly apiClient: GoogleCalendarApiClient,
  ) {}

  async execute(calendarId: string): Promise<{ created: number; updated: number; deleted: number }> {
    const calendar = await this.calendarRepository.findById(calendarId);

    if (!calendar || !calendar.googleCalendarId || !calendar.email) {
      this.logger.warn(`Calendar ${calendarId} has no Google integration`);
      return { created: 0, updated: 0, deleted: 0 };
    }

    let created = 0;
    let updated = 0;
    let deleted = 0;

    try {
      const { events, nextSyncToken } = await this.apiClient.fetchChanges(
        calendar.googleCalendarId,
        calendar.email,
        calendar.googleSyncToken,
      );

      for (const googleEvent of events) {
        const result = await this.processGoogleEvent(googleEvent, calendar.id);
        if (result === 'created') created++;
        else if (result === 'updated') updated++;
        else if (result === 'deleted') deleted++;
      }

      if (nextSyncToken) {
        await this.calendarRepository.updateSyncState(calendarId, nextSyncToken, new Date());
      }

      this.logger.log(
        `Sync complete for calendar ${calendarId}: +${created} ~${updated} -${deleted}`,
      );
    } catch (err) {
      // Handle invalid syncToken (Google returns 410 Gone) — reset and do full sync next time
      if ((err as any)?.code === 410) {
        this.logger.warn(`SyncToken expired for calendar ${calendarId}, resetting`);
        await this.calendarRepository.update(calendarId, { googleSyncToken: null });
      } else {
        this.logger.error(`Sync failed for calendar ${calendarId}: ${(err as Error).message}`);
      }
    }

    return { created, updated, deleted };
  }

  private async processGoogleEvent(
    googleEvent: calendar_v3.Schema$Event,
    calendarId: string,
  ): Promise<'created' | 'updated' | 'deleted' | 'skipped'> {
    if (!googleEvent.id) return 'skipped';

    const isCancelled = googleEvent.status === 'cancelled';
    const existing = await this.eventRepository.findByGoogleEventId(googleEvent.id);

    if (isCancelled) {
      if (existing) {
        await this.eventRepository.delete(existing.id);
        return 'deleted';
      }
      return 'skipped';
    }

    const mapped = GoogleEventMapper.fromGoogleEvent(googleEvent, calendarId);

    if (existing) {
      await this.eventRepository.update(existing.id, {
        title: mapped.title,
        description: mapped.description,
        startDate: mapped.startDate,
        endDate: mapped.endDate,
        startTime: mapped.startTime,
        endTime: mapped.endTime,
        recurrenceRule: mapped.recurrenceRule,
        status: mapped.status,
      });
      return 'updated';
    }

    const newEvent = new Event({
      ...mapped,
      id: crypto.randomUUID(),
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
    });

    await this.eventRepository.create(newEvent);
    return 'created';
  }
}
