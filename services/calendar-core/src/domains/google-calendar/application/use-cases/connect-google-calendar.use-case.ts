import { Injectable, NotFoundException } from '@nestjs/common';
import { CalendarRepository } from '@domains/calendars/infrastructure/persistence/calendar.repository';
import { GoogleOAuthService } from '../../infrastructure/services/google-oauth.service';

@Injectable()
export class ConnectGoogleCalendarUseCase {
  constructor(
    private readonly calendarRepository: CalendarRepository,
    private readonly googleOAuthService: GoogleOAuthService,
  ) {}

  async execute(code: string, calendarId: string): Promise<{ email: string; googleCalendarId: string }> {
    const calendar = await this.calendarRepository.findById(calendarId);

    if (!calendar) {
      throw new NotFoundException(`Calendar ${calendarId} not found`);
    }

    const tokenData = await this.googleOAuthService.exchangeCode(code);
    await this.googleOAuthService.saveToken(tokenData);

    await this.calendarRepository.update(calendarId, {
      email: tokenData.email,
      googleCalendarId: 'primary',
    });

    return { email: tokenData.email, googleCalendarId: 'primary' };
  }
}
