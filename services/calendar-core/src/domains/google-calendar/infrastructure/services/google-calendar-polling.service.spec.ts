import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { MockProxy } from 'vitest-mock-extended';
import { GoogleCalendarPollingService } from './google-calendar-polling.service';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { PullGoogleChangesUseCase } from '../../application/use-cases/pull-google-changes.use-case';
import { Calendar } from '../../../calendars/domain/entities/calendar.entity';
import { createMockRepository } from '../../../../test/helpers/mock-builders';

function makeCalendar(id: string): Calendar {
  return Calendar.create({
    id,
    userId: 'user-1',
    name: 'Work',
    color: '#0077B5',
    type: 'professional',
    email: 'work@example.com',
    googleCalendarId: 'primary',
  });
}

describe('GoogleCalendarPollingService', () => {
  let service: GoogleCalendarPollingService;
  let calendarRepository: MockProxy<CalendarRepository>;
  let pullGoogleChanges: MockProxy<PullGoogleChangesUseCase>;

  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GOOGLE_SYNC_ENABLED;

    calendarRepository = createMockRepository<CalendarRepository>();
    pullGoogleChanges = createMockRepository<PullGoogleChangesUseCase>();
    service = new GoogleCalendarPollingService(calendarRepository, pullGoogleChanges);

    calendarRepository.findAllGoogleConnected.mockResolvedValue([makeCalendar('cal-1')]);
  });

  afterEach(() => {
    delete process.env.GOOGLE_SYNC_ENABLED;
  });

  it('polls every connected calendar by default', async () => {
    calendarRepository.findAllGoogleConnected.mockResolvedValue([
      makeCalendar('cal-1'),
      makeCalendar('cal-2'),
    ]);

    await service.pollAllConnectedCalendars();

    expect(pullGoogleChanges.execute).toHaveBeenCalledWith('cal-1');
    expect(pullGoogleChanges.execute).toHaveBeenCalledWith('cal-2');
  });

  it('does not poll when the sync is switched off', async () => {
    // The kill switch for the backfill window: while stale rows are being linked to their Google
    // event, a tick that re-imports one of them creates a duplicate the backfill has to reconcile.
    process.env.GOOGLE_SYNC_ENABLED = 'false';

    await service.pollAllConnectedCalendars();

    expect(pullGoogleChanges.execute).not.toHaveBeenCalled();
    expect(calendarRepository.findAllGoogleConnected).not.toHaveBeenCalled();
  });

  it('keeps polling the remaining calendars when one of them fails', async () => {
    calendarRepository.findAllGoogleConnected.mockResolvedValue([
      makeCalendar('cal-1'),
      makeCalendar('cal-2'),
    ]);
    pullGoogleChanges.execute.mockRejectedValueOnce(new Error('token expired'));

    await service.pollAllConnectedCalendars();

    expect(pullGoogleChanges.execute).toHaveBeenCalledWith('cal-2');
  });
});
