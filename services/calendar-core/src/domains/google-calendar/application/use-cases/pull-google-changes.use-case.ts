import { Injectable, Logger } from '@nestjs/common';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { EventRepository } from '../../../events/infrastructure/repositories/event.repository';
import { GoogleCalendarApiClient } from '../../infrastructure/services/google-calendar-api.client';
import { GoogleEventMapper } from '../../domain/services/google-event-mapper';
import { Event } from '../../../events/domain/entities/event.entity';
import { calendar_v3 } from 'googleapis';

export interface SyncResult {
  created: number;
  updated: number;
  deleted: number;
  failed: number;
}

@Injectable()
export class PullGoogleChangesUseCase {
  private readonly logger = new Logger(PullGoogleChangesUseCase.name);

  constructor(
    private readonly calendarRepository: CalendarRepository,
    private readonly eventRepository: EventRepository,
    private readonly apiClient: GoogleCalendarApiClient,
  ) {}

  async execute(calendarId: string): Promise<SyncResult> {
    const calendar = await this.calendarRepository.findById(calendarId);

    if (!calendar || !calendar.googleCalendarId || !calendar.email) {
      this.logger.warn(`Calendar ${calendarId} has no Google integration`);
      return { created: 0, updated: 0, deleted: 0, failed: 0 };
    }

    let created = 0;
    let updated = 0;
    let deleted = 0;
    let failed = 0;

    try {
      const { events, nextSyncToken } = await this.apiClient.fetchChanges(
        calendar.googleCalendarId,
        calendar.email,
        calendar.googleSyncToken,
      );

      for (const googleEvent of events) {
        // Isolate each event: a single malformed one must not abort the run, because that would
        // also skip the syncToken write below and force a full re-sync on every subsequent tick.
        try {
          const result = await this.processGoogleEvent(googleEvent, calendar.id);
          if (result === 'created') created++;
          else if (result === 'updated') updated++;
          else if (result === 'deleted') deleted++;
        } catch (err) {
          failed++;
          this.logger.error(
            `Failed to process Google event ${googleEvent.id} on calendar ${calendarId}: ${(err as Error).message}`,
          );
        }
      }

      if (nextSyncToken) {
        await this.calendarRepository.updateSyncState(calendarId, nextSyncToken, new Date());
      } else {
        this.logger.warn(
          `Google returned no syncToken for calendar ${calendarId}; next sync will be a full re-sync`,
        );
      }

      this.logger.log(
        `Sync complete for calendar ${calendarId}: +${created} ~${updated} -${deleted} !${failed}`,
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

    return { created, updated, deleted, failed };
  }

  private async processGoogleEvent(
    googleEvent: calendar_v3.Schema$Event,
    calendarId: string,
  ): Promise<'created' | 'updated' | 'deleted' | 'skipped'> {
    if (!googleEvent.id) return 'skipped';

    const isCancelled = googleEvent.status === 'cancelled';
    const existing = await this.eventRepository.findByGoogleEventId(googleEvent.id, calendarId);

    if (isCancelled) {
      if (existing) {
        await this.eventRepository.delete(existing.id);
        return 'deleted';
      }
      return 'skipped';
    }

    const mapped = GoogleEventMapper.fromGoogleEvent(googleEvent, calendarId);

    // Write by (calendar, Google event) identity. The UNIQUE index arbitrates a concurrent cron
    // tick and manual sync, so the read above only decides the counter, never correctness.
    await this.eventRepository.upsertByGoogleEventId(
      new Event({
        ...mapped,
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      }),
    );

    return existing ? 'updated' : 'created';
  }
}
