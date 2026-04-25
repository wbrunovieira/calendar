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
    const connectedCalendars = await this.calendarRepository.findAllGoogleConnected();

    if (connectedCalendars.length === 0) return;

    this.logger.log(`Polling ${connectedCalendars.length} connected Google Calendar(s)`);

    for (const calendar of connectedCalendars) {
      await this.pullGoogleChanges.execute(calendar.id);
    }
  }
}
