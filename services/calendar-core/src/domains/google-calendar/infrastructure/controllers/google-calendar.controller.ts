import {
  Controller,
  Get,
  Delete,
  Param,
  Query,
  Redirect,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { GoogleOAuthService } from '../services/google-oauth.service';
import { ConnectGoogleCalendarUseCase } from '../../application/use-cases/connect-google-calendar.use-case';
import { DisconnectGoogleCalendarUseCase } from '../../application/use-cases/disconnect-google-calendar.use-case';

@Controller('google-calendar')
export class GoogleCalendarController {
  constructor(
    private readonly googleOAuthService: GoogleOAuthService,
    private readonly connectGoogleCalendar: ConnectGoogleCalendarUseCase,
    private readonly disconnectGoogleCalendar: DisconnectGoogleCalendarUseCase,
  ) {}

  @Get('auth/connect')
  @Redirect()
  connect(@Query('calendarId') calendarId: string) {
    const url = this.googleOAuthService.getAuthorizationUrl(calendarId);
    return { url };
  }

  @Get('auth/callback')
  @Redirect()
  async callback(
    @Query('code') code: string,
    @Query('state') calendarId: string,
  ) {
    await this.connectGoogleCalendar.execute(code, calendarId);
    const frontendUrl = process.env.GOOGLE_FRONTEND_REDIRECT_URI || 'http://localhost:3000/settings/integrations';
    return { url: `${frontendUrl}?connected=true` };
  }

  @Delete('auth/disconnect/:calendarId')
  @HttpCode(HttpStatus.NO_CONTENT)
  async disconnect(@Param('calendarId') calendarId: string) {
    await this.disconnectGoogleCalendar.execute(calendarId);
  }
}
