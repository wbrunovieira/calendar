import { describe, it, expect, beforeEach, vi } from 'vitest';
import { PullGoogleChangesUseCase } from './pull-google-changes.use-case';
import { Calendar } from '../../../calendars/domain/entities/calendar.entity';
import { Event } from '../../../events/domain/entities/event.entity';

const mockCalendarRepository = {
  findById: vi.fn(),
  update: vi.fn(),
  updateSyncState: vi.fn(),
};

const mockEventRepository = {
  findByGoogleEventId: vi.fn(),
  upsertByGoogleEventId: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
};

const mockApiClient = {
  fetchChanges: vi.fn(),
};

function makeCalendar(overrides: Partial<Calendar> = {}): Calendar {
  return Calendar.create({
    id: 'cal-1',
    userId: 'user-1',
    name: 'Work',
    color: '#0077B5',
    type: 'professional',
    email: 'work@example.com',
    googleCalendarId: 'primary',
    ...overrides,
  });
}

function makeGoogleEvent(overrides = {}) {
  return {
    id: 'g-evt-1',
    summary: 'Google Meeting',
    start: { dateTime: '2024-01-15T10:00:00-03:00' },
    end: { dateTime: '2024-01-15T11:00:00-03:00' },
    status: 'confirmed',
    ...overrides,
  };
}

function makeLocalEvent(overrides = {}): Event {
  return Object.assign(new Event({}), {
    id: 'local-evt-1',
    googleEventId: 'g-evt-1',
    calendarId: 'cal-1',
    ...overrides,
  });
}

describe('PullGoogleChangesUseCase', () => {
  let useCase: PullGoogleChangesUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new PullGoogleChangesUseCase(
      mockCalendarRepository as any,
      mockEventRepository as any,
      mockApiClient as any,
    );
  });

  it('should create local event for new Google event', async () => {
    mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent()],
      nextSyncToken: 'new-sync-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(null);

    const result = await useCase.execute('cal-1');

    expect(result.created).toBe(1);
    expect(result.updated).toBe(0);
    expect(result.deleted).toBe(0);
    expect(mockEventRepository.upsertByGoogleEventId).toHaveBeenCalledOnce();
  });

  it('should update local event when Google event already exists', async () => {
    mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent({ summary: 'Updated Meeting' })],
      nextSyncToken: 'new-sync-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(makeLocalEvent());

    const result = await useCase.execute('cal-1');

    expect(result.updated).toBe(1);
    expect(result.created).toBe(0);
    expect(mockEventRepository.upsertByGoogleEventId).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'Updated Meeting', googleEventId: 'g-evt-1' }),
    );
  });

  it('should delete local event when Google event is cancelled', async () => {
    mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent({ status: 'cancelled' })],
      nextSyncToken: 'new-sync-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(makeLocalEvent());

    const result = await useCase.execute('cal-1');

    expect(result.deleted).toBe(1);
    expect(mockEventRepository.delete).toHaveBeenCalledWith('local-evt-1');
  });

  it('should persist nextSyncToken after successful sync', async () => {
    mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [],
      nextSyncToken: 'sync-token-xyz',
    });

    await useCase.execute('cal-1');

    expect(mockCalendarRepository.updateSyncState).toHaveBeenCalledWith(
      'cal-1',
      'sync-token-xyz',
      expect.any(Date),
    );
  });

  it('should return zeros and skip when calendar has no Google integration', async () => {
    mockCalendarRepository.findById.mockResolvedValue(makeCalendar({ googleCalendarId: null }));

    const result = await useCase.execute('cal-1');

    expect(result).toEqual({ created: 0, updated: 0, deleted: 0, failed: 0 });
    expect(mockApiClient.fetchChanges).not.toHaveBeenCalled();
  });

  it('should reset syncToken when Google returns 410 Gone (token expired)', async () => {
    mockCalendarRepository.findById.mockResolvedValue(
      makeCalendar({ googleSyncToken: 'expired-token' }),
    );
    mockApiClient.fetchChanges.mockRejectedValue(Object.assign(new Error('Gone'), { code: 410 }));

    const result = await useCase.execute('cal-1');

    expect(mockCalendarRepository.update).toHaveBeenCalledWith('cal-1', { googleSyncToken: null });
    expect(result).toEqual({ created: 0, updated: 0, deleted: 0, failed: 0 });
  });

  it('should skip cancelled event that does not exist locally', async () => {
    mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent({ status: 'cancelled' })],
      nextSyncToken: 'new-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(null);

    const result = await useCase.execute('cal-1');

    expect(result.deleted).toBe(0);
    expect(mockEventRepository.delete).not.toHaveBeenCalled();
  });

  describe('idempotency (duplication regression)', () => {
    it('writes the event carrying its googleEventId', async () => {
      mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [makeGoogleEvent({ id: 'g-evt-42' })],
        nextSyncToken: 'new-token',
      });
      mockEventRepository.findByGoogleEventId.mockResolvedValue(null);

      await useCase.execute('cal-1');

      expect(mockEventRepository.upsertByGoogleEventId).toHaveBeenCalledWith(
        expect.objectContaining({ googleEventId: 'g-evt-42' }),
      );
    });

    it('looks the event up scoped to the calendar being synced', async () => {
      mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [makeGoogleEvent({ id: 'g-evt-42' })],
        nextSyncToken: 'new-token',
      });
      mockEventRepository.findByGoogleEventId.mockResolvedValue(null);

      await useCase.execute('cal-1');

      expect(mockEventRepository.findByGoogleEventId).toHaveBeenCalledWith('g-evt-42', 'cal-1');
    });

    it('does not insert a second row when the same Google event is delivered again', async () => {
      mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [makeGoogleEvent({ id: 'g-evt-42' })],
        nextSyncToken: 'new-token',
      });

      // 1st sync: nothing local yet -> created, persisting the googleEventId.
      mockEventRepository.findByGoogleEventId.mockResolvedValueOnce(null);
      const first = await useCase.execute('cal-1');

      // 2nd sync (syncToken reset, or the event was edited in Google): the row now exists and is
      // found by its googleEventId -> updated in place, never inserted again.
      mockEventRepository.findByGoogleEventId.mockResolvedValueOnce(
        makeLocalEvent({ googleEventId: 'g-evt-42' }),
      );
      const second = await useCase.execute('cal-1');

      expect(first.created).toBe(1);
      expect(second.created).toBe(0);
      expect(second.updated).toBe(1);
      // Both ticks write by identity, so Postgres can only ever hold one row for this Google event.
      expect(mockEventRepository.upsertByGoogleEventId).toHaveBeenCalledTimes(2);
    });
  });

  describe('per-event fault isolation', () => {
    // A malformed event (no start/end) makes GoogleEventMapper throw. When the try/catch wrapped
    // the whole loop, one bad event aborted the run before the syncToken was saved — and the next
    // tick full-synced from scratch, hit the same event, and looped forever.
    const malformedEvent = { id: 'g-evt-broken', summary: 'Broken', status: 'confirmed' };

    beforeEach(() => {
      mockCalendarRepository.findById.mockResolvedValue(makeCalendar());
      mockEventRepository.findByGoogleEventId.mockResolvedValue(null);
    });

    it('keeps processing the remaining events when one event fails to map', async () => {
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [malformedEvent, makeGoogleEvent({ id: 'g-evt-ok' })],
        nextSyncToken: 'new-token',
      });

      const result = await useCase.execute('cal-1');

      expect(result.created).toBe(1);
      expect(mockEventRepository.upsertByGoogleEventId).toHaveBeenCalledWith(
        expect.objectContaining({ googleEventId: 'g-evt-ok' }),
      );
    });

    it('reports the events it could not process', async () => {
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [malformedEvent, makeGoogleEvent({ id: 'g-evt-ok' })],
        nextSyncToken: 'new-token',
      });

      const result = await useCase.execute('cal-1');

      expect(result.failed).toBe(1);
    });

    it('still persists the syncToken when an event fails, so the next tick is a delta not a full re-sync', async () => {
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [malformedEvent],
        nextSyncToken: 'token-after-failure',
      });

      await useCase.execute('cal-1');

      expect(mockCalendarRepository.updateSyncState).toHaveBeenCalledWith(
        'cal-1',
        'token-after-failure',
        expect.any(Date),
      );
    });

    it('counts a write failure without swallowing the rest of the batch', async () => {
      mockApiClient.fetchChanges.mockResolvedValue({
        events: [makeGoogleEvent({ id: 'g-evt-a' }), makeGoogleEvent({ id: 'g-evt-b' })],
        nextSyncToken: 'new-token',
      });
      mockEventRepository.upsertByGoogleEventId
        .mockRejectedValueOnce(new Error('deadlock detected'))
        .mockResolvedValueOnce(undefined);

      const result = await useCase.execute('cal-1');

      expect(result.failed).toBe(1);
      expect(result.created).toBe(1);
    });

    it('still aborts the whole run when the Google API itself fails', async () => {
      mockApiClient.fetchChanges.mockRejectedValue(new Error('network down'));

      const result = await useCase.execute('cal-1');

      expect(result).toEqual({ created: 0, updated: 0, deleted: 0, failed: 0 });
      expect(mockCalendarRepository.updateSyncState).not.toHaveBeenCalled();
    });
  });
});
