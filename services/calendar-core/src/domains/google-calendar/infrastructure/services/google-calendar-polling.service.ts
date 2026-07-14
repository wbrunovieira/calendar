import { Injectable, Logger } from '@nestjs/common';
import { Cron } from '@nestjs/schedule';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { PullGoogleChangesUseCase } from '../../application/use-cases/pull-google-changes.use-case';

@Injectable()
export class GoogleCalendarPollingService {
  private readonly logger = new Logger(GoogleCalendarPollingService.name);

  constructor(
    private readonly calendarRepository: CalendarRepository,
    private readonly pullGoogleChanges: PullGoogleChangesUseCase,
  ) {}

  @Cron('*/5 * * * *')
  async pollAllConnectedCalendars(): Promise<void> {
    // Kill switch. Set GOOGLE_SYNC_ENABLED=false to run the one-shot backfill: a tick landing
    // mid-run would re-import a row the backfill is busy linking, and create a duplicate of it.
    if (process.env.GOOGLE_SYNC_ENABLED === 'false') {
      this.logger.warn('Google Calendar sync is disabled (GOOGLE_SYNC_ENABLED=false), skipping');
      return;
    }

    const connectedCalendars = await this.calendarRepository.findAllGoogleConnected();

    if (connectedCalendars.length === 0) return;

    this.logger.log(`Polling ${connectedCalendars.length} connected Google Calendar(s)`);

    for (const calendar of connectedCalendars) {
      // One calendar failing (expired refresh token, Google down) must not stop the others.
      try {
        await this.pullGoogleChanges.execute(calendar.id);
      } catch (err) {
        this.logger.error(`Polling failed for calendar ${calendar.id}: ${(err as Error).message}`);
      }
    }
  }
}
