import { Injectable } from '@nestjs/common';
import { google, calendar_v3 } from 'googleapis';
import { GoogleOAuthService } from './google-oauth.service';

export interface GoogleEventPayload {
  summary: string;
  description?: string;
  start: { dateTime?: string; date?: string; timeZone?: string };
  end: { dateTime?: string; date?: string; timeZone?: string };
  recurrence?: string[];
  reminders?: {
    useDefault: boolean;
    overrides?: Array<{ method: string; minutes: number }>;
  };
  status?: string;
}

export interface GoogleChangesResult {
  events: calendar_v3.Schema$Event[];
  nextSyncToken: string;
}

const TIMEZONE = 'America/Sao_Paulo';

@Injectable()
export class GoogleCalendarApiClient {
  constructor(private readonly googleOAuthService: GoogleOAuthService) {}

  private async getCalendarClient(accountEmail: string) {
    const accessToken = await this.googleOAuthService.getValidAccessToken(accountEmail);
    const auth = new google.auth.OAuth2();
    auth.setCredentials({ access_token: accessToken });
    return google.calendar({ version: 'v3', auth });
  }

  async createEvent(
    googleCalendarId: string,
    accountEmail: string,
    payload: GoogleEventPayload,
  ): Promise<string> {
    const calendar = await this.getCalendarClient(accountEmail);
    const response = await calendar.events.insert({
      calendarId: googleCalendarId,
      requestBody: payload,
    });
    return response.data.id!;
  }

  async updateEvent(
    googleCalendarId: string,
    googleEventId: string,
    accountEmail: string,
    payload: Partial<GoogleEventPayload>,
  ): Promise<void> {
    const calendar = await this.getCalendarClient(accountEmail);
    await calendar.events.patch({
      calendarId: googleCalendarId,
      eventId: googleEventId,
      requestBody: payload,
    });
  }

  async deleteEvent(
    googleCalendarId: string,
    googleEventId: string,
    accountEmail: string,
  ): Promise<void> {
    const calendar = await this.getCalendarClient(accountEmail);
    await calendar.events.delete({
      calendarId: googleCalendarId,
      eventId: googleEventId,
    });
  }

  async fetchChanges(
    googleCalendarId: string,
    accountEmail: string,
    syncToken?: string | null,
  ): Promise<GoogleChangesResult> {
    const calendar = await this.getCalendarClient(accountEmail);

    const params: calendar_v3.Params$Resource$Events$List = {
      calendarId: googleCalendarId,
      singleEvents: false,
      timeZone: TIMEZONE,
    };

    if (syncToken) {
      params.syncToken = syncToken;
    } else {
      // First sync: fetch events from 6 months ago onwards
      const sixMonthsAgo = new Date();
      sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6);
      params.timeMin = sixMonthsAgo.toISOString();
    }

    const events: calendar_v3.Schema$Event[] = [];
    let pageToken: string | undefined;

    do {
      const response = await calendar.events.list({ ...params, pageToken });
      events.push(...(response.data.items ?? []));
      pageToken = response.data.nextPageToken ?? undefined;

      if (response.data.nextSyncToken) {
        return { events, nextSyncToken: response.data.nextSyncToken };
      }
    } while (pageToken);

    return { events, nextSyncToken: '' };
  }
}
