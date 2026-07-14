import { describe, it, expect, beforeEach, vi } from 'vitest';
import { MockProxy } from 'vitest-mock-extended';
import { BackfillGoogleEventIdsUseCase } from './backfill-google-event-ids.use-case';
import { Calendar } from '../../../calendars/domain/entities/calendar.entity';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { EventRepository } from '../../../events/infrastructure/repositories/event.repository';
import { GoogleCalendarApiClient } from '../../infrastructure/services/google-calendar-api.client';
import { createMockRepository } from '../../../../test/helpers/mock-builders';

function makeCalendar(): Calendar {
  return Calendar.create({
    id: 'cal-1',
    userId: 'user-1',
    name: 'Personal',
    color: '#350545',
    type: 'personal',
    email: 'personal@example.com',
    googleCalendarId: 'primary',
  });
}

function makeGoogleEvent(overrides = {}) {
  return {
    id: 'g-evt-1',
    summary: 'Design review',
    start: { dateTime: '2026-07-13T11:30:00-03:00' },
    end: { dateTime: '2026-07-13T11:55:00-03:00' },
    status: 'confirmed',
    ...overrides,
  };
}

/** A local row the broken sync inserted with google_event_id = NULL. */
function orphan(id: string, createdAt: string, hasUserData = false) {
  return { id, createdAt: new Date(createdAt), hasUserData };
}

describe('BackfillGoogleEventIdsUseCase', () => {
  let useCase: BackfillGoogleEventIdsUseCase;
  let calendarRepository: MockProxy<CalendarRepository>;
  let eventRepository: MockProxy<EventRepository>;
  let apiClient: MockProxy<GoogleCalendarApiClient>;

  beforeEach(() => {
    vi.clearAllMocks();
    calendarRepository = createMockRepository<CalendarRepository>();
    eventRepository = createMockRepository<EventRepository>();
    apiClient = createMockRepository<GoogleCalendarApiClient>();

    useCase = new BackfillGoogleEventIdsUseCase(calendarRepository, eventRepository, apiClient);

    calendarRepository.findAllGoogleConnected.mockResolvedValue([makeCalendar()]);
    apiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent()] as never,
      nextSyncToken: 'tok',
    });
    eventRepository.findByGoogleEventId.mockResolvedValue(null);
    eventRepository.countUnlinkedEvents.mockResolvedValue(0);
  });

  describe('dry run (the default)', () => {
    it('reports the plan without touching a single row', async () => {
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-a', '2026-07-11T13:10:00Z'),
        orphan('evt-b', '2026-07-12T17:50:00Z'),
      ]);

      const plan = await useCase.execute({ dryRun: true });

      expect(plan.linked).toEqual([expect.objectContaining({ eventId: 'evt-a' })]);
      expect(plan.deleted).toEqual([expect.objectContaining({ eventId: 'evt-b' })]);
      expect(eventRepository.updateGoogleEventId).not.toHaveBeenCalled();
      expect(eventRepository.delete).not.toHaveBeenCalled();
    });

    it('produces the same plan as apply, so the reviewed plan is the one that executes', async () => {
      // The whole safety story is "read the dry run, then --apply". If apply took a different path
      // — because its own writes change what later queries see — the reviewed plan would be a lie.
      const arrange = () => {
        calendarRepository.findAllGoogleConnected.mockResolvedValue([makeCalendar()]);
        apiClient.fetchChanges.mockResolvedValue({
          events: [
            makeGoogleEvent({ id: 'g-evt-1', summary: 'Design review' }),
            makeGoogleEvent({ id: 'g-evt-2', summary: 'Retro' }),
          ] as never,
          nextSyncToken: 'tok',
        });
        eventRepository.findByGoogleEventId.mockResolvedValue(null);
        eventRepository.countUnlinkedEvents.mockResolvedValue(7);
        eventRepository.findUnlinkedMatches.mockResolvedValue([
          orphan('evt-a', '2026-07-11T13:10:00Z'),
          orphan('evt-b', '2026-07-12T17:50:00Z'),
        ]);
      };

      arrange();
      const dry = await useCase.execute({ dryRun: true });

      vi.clearAllMocks();
      arrange();
      const applied = await useCase.execute({ dryRun: false });

      expect(applied).toEqual(dry);
    });
  });

  describe('apply', () => {
    it('links the survivor and deletes the duplicate', async () => {
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-a', '2026-07-11T13:10:00Z'),
        orphan('evt-b', '2026-07-12T17:50:00Z'),
      ]);

      await useCase.execute({ dryRun: false });

      expect(eventRepository.updateGoogleEventId).toHaveBeenCalledWith('evt-a', 'g-evt-1');
      expect(eventRepository.delete).toHaveBeenCalledWith('evt-b');
      expect(eventRepository.delete).toHaveBeenCalledOnce();
    });

    it('links without deleting when there is exactly one local copy', async () => {
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-only', '2026-07-11T13:10:00Z'),
      ]);

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.linked).toHaveLength(1);
      expect(plan.deleted).toEqual([]);
      expect(eventRepository.delete).not.toHaveBeenCalled();
    });
  });

  describe('safety rails', () => {
    it('reconciles a bare stale copy against the row the fixed sync already re-imported', async () => {
      // Between deploy and backfill the cron keeps running. For an orphan whose Google event still
      // exists, the fixed sync inserts a correctly linked twin. The UNIQUE index means that twin
      // owns the identity — trying to link the stale copy would abort the run mid-write.
      eventRepository.findByGoogleEventId.mockResolvedValue({ id: 'evt-twin' } as never);
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-stale', '2026-07-11T13:10:00Z'),
      ]);

      const plan = await useCase.execute({ dryRun: false });

      expect(eventRepository.updateGoogleEventId).not.toHaveBeenCalled();
      expect(plan.deleted).toEqual([expect.objectContaining({ eventId: 'evt-stale' })]);
      expect(eventRepository.delete).toHaveBeenCalledWith('evt-stale');
    });

    it('refuses to cascade away a stale copy carrying user data, even next to a linked twin', async () => {
      // The twin the sync re-imported is BARE: the sync writes no category, label or reminders. So
      // the stale copy next to it may be the only one holding the user's curation. Deleting it
      // would keep the empty row and destroy the meaningful one.
      eventRepository.findByGoogleEventId.mockResolvedValue({ id: 'evt-twin' } as never);
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-curated', '2026-07-11T13:10:00Z', true),
      ]);

      const plan = await useCase.execute({ dryRun: false });

      expect(eventRepository.delete).not.toHaveBeenCalled();
      expect(plan.deleted).toEqual([]);
      expect(plan.conflicts).toEqual([
        expect.objectContaining({ reason: expect.stringContaining('carries user data') }),
      ]);
    });

    it('never deletes a row that carries user data — reports a conflict instead', async () => {
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-a', '2026-07-11T13:10:00Z', true),
        orphan('evt-b', '2026-07-12T17:50:00Z', true),
      ]);

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.deleted).toEqual([]);
      expect(plan.conflicts).toHaveLength(1);
      expect(eventRepository.delete).not.toHaveBeenCalled();
    });

    it('refuses to touch anything when two Google events share one natural key', async () => {
      // Matching is by (title, date, time). When two Google events collide on it, there is no way
      // to know which local row belongs to which — guessing would delete the wrong one.
      apiClient.fetchChanges.mockResolvedValue({
        events: [makeGoogleEvent({ id: 'g-evt-1' }), makeGoogleEvent({ id: 'g-evt-2' })] as never,
        nextSyncToken: 'tok',
      });
      eventRepository.findUnlinkedMatches.mockResolvedValue([
        orphan('evt-a', '2026-07-11T13:10:00Z'),
        orphan('evt-b', '2026-07-12T17:50:00Z'),
      ]);

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.linked).toEqual([]);
      expect(plan.deleted).toEqual([]);
      expect(plan.conflicts).toHaveLength(2);
      expect(eventRepository.delete).not.toHaveBeenCalled();
      expect(eventRepository.findUnlinkedMatches).not.toHaveBeenCalled();
    });

    it('records a conflict and carries on when a whole calendar cannot be read from Google', async () => {
      // An expired refresh token on calendar 2 must not throw away the plan already applied for
      // calendar 1 — the script would exit without ever printing what it deleted.
      apiClient.fetchChanges.mockRejectedValue(new Error('invalid_grant'));

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.conflicts).toEqual([
        expect.objectContaining({ reason: expect.stringContaining('invalid_grant') }),
      ]);
    });

    it('leaves local-only events completely alone', async () => {
      eventRepository.findUnlinkedMatches.mockResolvedValue([]);

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.linked).toEqual([]);
      expect(plan.deleted).toEqual([]);
      expect(plan.unmatchedGoogleEvents).toBe(1);
      expect(eventRepository.delete).not.toHaveBeenCalled();
    });

    it('keeps going and records a conflict when one Google event blows up', async () => {
      apiClient.fetchChanges.mockResolvedValue({
        events: [
          makeGoogleEvent({ id: 'g-evt-1', summary: 'Design review' }),
          makeGoogleEvent({ id: 'g-evt-2', summary: 'Retro' }),
        ] as never,
        nextSyncToken: 'tok',
      });
      eventRepository.findUnlinkedMatches
        .mockRejectedValueOnce(new Error('connection reset'))
        .mockResolvedValueOnce([orphan('evt-ok', '2026-07-11T13:10:00Z')]);

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.conflicts).toEqual([
        expect.objectContaining({ googleEventId: 'g-evt-1', reason: expect.stringContaining('connection reset') }),
      ]);
      expect(plan.linked).toEqual([expect.objectContaining({ eventId: 'evt-ok' })]);
    });

    it('matches candidates scoped to the calendar being processed', async () => {
      eventRepository.findUnlinkedMatches.mockResolvedValue([]);

      await useCase.execute({ dryRun: true });

      expect(eventRepository.findUnlinkedMatches).toHaveBeenCalledWith(
        'cal-1',
        'Design review',
        new Date('2026-07-13'),
        '11:30',
      );
    });

    it('skips a Google event that is cancelled', async () => {
      apiClient.fetchChanges.mockResolvedValue({
        events: [makeGoogleEvent({ status: 'cancelled' })] as never,
        nextSyncToken: 'tok',
      });

      const plan = await useCase.execute({ dryRun: false });

      expect(plan.linked).toEqual([]);
      expect(eventRepository.findUnlinkedMatches).not.toHaveBeenCalled();
    });

    it('reports the orphans it could not repair, rather than implying the job is done', async () => {
      // Orphans whose title/time changed in Google, or that fall outside the API lookback window,
      // can never be matched by natural key. They are the residual duplication hazard.
      eventRepository.findUnlinkedMatches.mockResolvedValue([]);
      eventRepository.countUnlinkedEvents.mockResolvedValue(42);

      const plan = await useCase.execute({ dryRun: true });

      expect(plan.unlinkedEventsRemaining).toBe(42);
    });
  });
});
