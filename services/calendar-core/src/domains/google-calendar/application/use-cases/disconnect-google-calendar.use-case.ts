import { Injectable, NotFoundException } from '@nestjs/common';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { GoogleOAuthService } from '../../infrastructure/services/google-oauth.service';

@Injectable()
export class DisconnectGoogleCalendarUseCase {
  constructor(
    private readonly calendarRepository: CalendarRepository,
    private readonly googleOAuthService: GoogleOAuthService,
  ) {}

  async execute(calendarId: string): Promise<void> {
    const calendar = await this.calendarRepository.findById(calendarId);

    if (!calendar) {
      throw new NotFoundException(`Calendar ${calendarId} not found`);
    }

    if (calendar.email) {
      await this.googleOAuthService.revokeToken(calendar.email);
    }

    await this.calendarRepository.update(calendarId, {
      googleCalendarId: null,
      googleSyncToken: null,
      lastSyncAt: null,
    });
  }
}
