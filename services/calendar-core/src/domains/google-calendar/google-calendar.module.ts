import { Module } from '@nestjs/common';
import { ScheduleModule } from '@nestjs/schedule';
import { CalendarsModule } from '@domains/calendars/calendars.module';
import { GoogleCalendarController } from './infrastructure/controllers/google-calendar.controller';
import { PrismaGoogleOAuthTokenRepository } from './infrastructure/persistence/prisma-google-oauth-token.repository';
import { GoogleOAuthService } from './infrastructure/services/google-oauth.service';
import { GoogleCalendarApiClient } from './infrastructure/services/google-calendar-api.client';
import { GoogleCalendarPollingService } from './infrastructure/services/google-calendar-polling.service';
import { ConnectGoogleCalendarUseCase } from './application/use-cases/connect-google-calendar.use-case';
import { DisconnectGoogleCalendarUseCase } from './application/use-cases/disconnect-google-calendar.use-case';
import { SyncEventToGoogleUseCase } from './application/use-cases/sync-event-to-google.use-case';
import { PullGoogleChangesUseCase } from './application/use-cases/pull-google-changes.use-case';
import { GOOGLE_OAUTH_TOKEN_REPOSITORY } from './domain/repositories/google-oauth-token.repository.interface';
import { EventRepository } from '@domains/events/infrastructure/repositories/event.repository';

@Module({
  imports: [ScheduleModule.forRoot(), CalendarsModule],
  controllers: [GoogleCalendarController],
  providers: [
    {
      provide: GOOGLE_OAUTH_TOKEN_REPOSITORY,
      useClass: PrismaGoogleOAuthTokenRepository,
    },
    EventRepository,
    GoogleOAuthService,
    GoogleCalendarApiClient,
    GoogleCalendarPollingService,
    ConnectGoogleCalendarUseCase,
    DisconnectGoogleCalendarUseCase,
    SyncEventToGoogleUseCase,
    PullGoogleChangesUseCase,
  ],
  exports: [GoogleOAuthService, GoogleCalendarApiClient, SyncEventToGoogleUseCase],
})
export class GoogleCalendarModule {}
