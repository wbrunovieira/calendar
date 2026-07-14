import { Module } from '@nestjs/common';
import { CalendarRepository } from '../calendars/infrastructure/persistence/calendar.repository';
import { EventRepository } from '../events/infrastructure/repositories/event.repository';
import { PrismaGoogleOAuthTokenRepository } from './infrastructure/persistence/prisma-google-oauth-token.repository';
import { GoogleOAuthService } from './infrastructure/services/google-oauth.service';
import { GoogleCalendarApiClient } from './infrastructure/services/google-calendar-api.client';
import { BackfillGoogleEventIdsUseCase } from './application/use-cases/backfill-google-event-ids.use-case';
import { GOOGLE_OAUTH_TOKEN_REPOSITORY } from './domain/repositories/google-oauth-token.repository.interface';

/**
 * Wiring for the one-shot backfill script only.
 *
 * Deliberately does NOT import GoogleCalendarModule: that module pulls in ScheduleModule and the
 * 5-minute polling cron, which would fire inside the script's own process and mutate the very rows
 * the backfill is linking and deleting.
 */
@Module({
  providers: [
    { provide: GOOGLE_OAUTH_TOKEN_REPOSITORY, useClass: PrismaGoogleOAuthTokenRepository },
    CalendarRepository,
    EventRepository,
    GoogleOAuthService,
    GoogleCalendarApiClient,
    BackfillGoogleEventIdsUseCase,
  ],
})
export class BackfillModule {}
