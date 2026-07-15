import { beforeEach, describe, expect, it } from 'vitest';
import { DeepMockProxy } from 'vitest-mock-extended';
import { PrismaClient } from '@prisma/client';
import { EventRepository } from './event.repository';
import { Event } from '../../domain/entities/event.entity';
import { createMockPrisma } from '../../../../test/helpers/mock-builders';

/**
 * Regression suite for the Google Calendar duplication bug.
 *
 * Events pulled from Google were persisted with google_event_id = NULL, so the idempotency lookup
 * in PullGoogleChangesUseCase never matched and every re-delivery inserted a new row.
 */
describe('EventRepository', () => {
  let repository: EventRepository;
  let prisma: DeepMockProxy<PrismaClient>;

  const buildEvent = (overrides: Partial<Event> = {}): Event =>
    new Event({
      id: 'evt-1',
      calendarId: 'cal-1',
      title: 'Design review',
      description: null,
      startTime: '11:30',
      endTime: '11:55',
      startDate: new Date('2026-07-13'),
      endDate: null,
      recurrenceRule: null,
      recurrenceMasterId: null,
      status: 'CONFIRMED',
      eventType: 'EVENT',
      isActive: true,
      createdAt: new Date('2026-07-11T13:10:00Z'),
      updatedAt: new Date('2026-07-11T13:10:00Z'),
      ...overrides,
    });

  beforeEach(() => {
    prisma = createMockPrisma();
    repository = new EventRepository(prisma);
  });

  describe('create', () => {
    it('persists googleEventId so the event can be matched on the next sync', async () => {
      const event = buildEvent({ googleEventId: 'google-abc-123' });
      prisma.event.create.mockResolvedValue({ ...event, reminders: [] } as never);

      await repository.create(event);

      expect(prisma.event.create).toHaveBeenCalledWith(
        expect.objectContaining({
          data: expect.objectContaining({ googleEventId: 'google-abc-123' }),
        }),
      );
    });

    it('persists null googleEventId for events created locally', async () => {
      const event = buildEvent({ googleEventId: null });
      prisma.event.create.mockResolvedValue({ ...event, reminders: [] } as never);

      await repository.create(event);

      expect(prisma.event.create).toHaveBeenCalledWith(
        expect.objectContaining({
          data: expect.objectContaining({ googleEventId: null }),
        }),
      );
    });
  });

  describe('update', () => {
    beforeEach(() => {
      // update() always reloads the row with its reminders before returning.
      prisma.event.findUnique.mockResolvedValue({ ...buildEvent(), reminders: [] } as never);
    });

    it('persists googleEventId when provided', async () => {
      prisma.event.update.mockResolvedValue(buildEvent() as never);

      await repository.update('evt-1', { googleEventId: 'google-abc-123' });

      expect(prisma.event.update).toHaveBeenCalledWith(
        expect.objectContaining({
          data: expect.objectContaining({ googleEventId: 'google-abc-123' }),
        }),
      );
    });

    it('leaves googleEventId untouched when the field is absent from the patch', async () => {
      prisma.event.update.mockResolvedValue(buildEvent() as never);

      await repository.update('evt-1', { title: 'New title' });

      const [[arg]] = prisma.event.update.mock.calls as unknown as [
        [{ data: Record<string, unknown> }],
      ];
      expect(arg.data.googleEventId).toBeUndefined();
    });
  });

  describe('upsertByGoogleEventId', () => {
    it('writes by the (calendar, Google event) identity so a concurrent sync cannot duplicate', async () => {
      prisma.event.upsert.mockResolvedValue(buildEvent() as never);

      await repository.upsertByGoogleEventId(buildEvent({ googleEventId: 'google-abc-123' }));

      expect(prisma.event.upsert).toHaveBeenCalledWith(
        expect.objectContaining({
          where: {
            calendarId_googleEventId: {
              calendarId: 'cal-1',
              googleEventId: 'google-abc-123',
            },
          },
        }),
      );
    });

    it('refuses an event with no googleEventId rather than writing an unidentifiable row', async () => {
      await expect(
        repository.upsertByGoogleEventId(buildEvent({ googleEventId: null })),
      ).rejects.toThrow(/requires an event carrying a googleEventId/);

      expect(prisma.event.upsert).not.toHaveBeenCalled();
    });
  });

  describe('findByGoogleEventId', () => {
    it('looks the event up on the composite unique key, scoped to one calendar', async () => {
      prisma.event.findUnique.mockResolvedValue(null as never);

      await repository.findByGoogleEventId('google-abc-123', 'cal-1');

      expect(prisma.event.findUnique).toHaveBeenCalledWith({
        where: {
          calendarId_googleEventId: {
            calendarId: 'cal-1',
            googleEventId: 'google-abc-123',
          },
        },
      });
    });

    it('hydrates an Event entity when a match exists', async () => {
      prisma.event.findUnique.mockResolvedValue(
        buildEvent({ googleEventId: 'google-abc-123' }) as never,
      );

      const found = await repository.findByGoogleEventId('google-abc-123', 'cal-1');

      expect(found).toBeInstanceOf(Event);
      expect(found?.googleEventId).toBe('google-abc-123');
    });
  });

  /**
   * This WHERE clause is the only gate in front of a hard delete during the backfill, and delete
   * cascades to completions, overrides and derived instances. Every constraint below exists to keep
   * a local row that merely *looks* like a Google event from being destroyed.
   */
  describe('findUnlinkedMatches', () => {
    beforeEach(() => {
      prisma.event.findMany.mockResolvedValue([] as never);
    });

    it('only considers unlinked, active, top-level EVENTs in the given calendar', async () => {
      await repository.findUnlinkedMatches('cal-1', 'Design review', new Date('2026-07-13'), '11:30');

      expect(prisma.event.findMany).toHaveBeenCalledWith(
        expect.objectContaining({
          where: {
            calendarId: 'cal-1',
            googleEventId: null,
            title: 'Design review',
            startDate: new Date('2026-07-13'),
            startTime: '11:30',
            // The sync can only ever have produced these. A HABIT, TODO or REMINDER sharing the
            // title and time is a local construct and must never be a deletion candidate.
            eventType: 'EVENT',
            recurrenceMasterId: null,
            isActive: true,
          },
        }),
      );
    });

    it('flags a row as carrying history when any cascading relation is present', async () => {
      prisma.event.findMany.mockResolvedValue([
        {
          id: 'evt-a',
          createdAt: new Date('2026-07-11T13:10:00Z'),
          categoryId: null,
          categoryTypeId: null,
          labelId: null,
          _count: {
            completions: 0,
            weeklyGoalCompletions: 0,
            overrides: 0,
            overriddenBy: 1, // this row IS a recurrence override — deleting it cascades it away
            exceptions: 0,
            derivedEvents: 0,
            reminders: 0,
          },
        },
      ] as never);

      const [match] = await repository.findUnlinkedMatches(
        'cal-1',
        'Design review',
        new Date('2026-07-13'),
        '11:30',
      );

      expect(match.hasUserData).toBe(true);
    });

    it('flags a row as safe to discard only when every cascading relation is empty', async () => {
      prisma.event.findMany.mockResolvedValue([
        {
          id: 'evt-a',
          createdAt: new Date('2026-07-11T13:10:00Z'),
          categoryId: null,
          categoryTypeId: null,
          labelId: null,
          _count: {
            completions: 0,
            weeklyGoalCompletions: 0,
            overrides: 0,
            overriddenBy: 0,
            exceptions: 0,
            derivedEvents: 0,
            reminders: 0,
          },
        },
      ] as never);

      const [match] = await repository.findUnlinkedMatches(
        'cal-1',
        'Design review',
        new Date('2026-07-13'),
        '11:30',
      );

      expect(match.hasUserData).toBe(false);
    });

    it('flags a row the user categorised, even with no relations at all', async () => {
      // The sync writes no categoryId/categoryTypeId/labelId. A row carrying one was curated by the
      // user, and the sync cannot put it back — so it is not interchangeable with a fresh import.
      prisma.event.findMany.mockResolvedValue([
        {
          id: 'evt-a',
          createdAt: new Date('2026-07-11T13:10:00Z'),
          categoryId: 'cat-work',
          categoryTypeId: null,
          labelId: null,
          _count: {
            completions: 0,
            weeklyGoalCompletions: 0,
            overrides: 0,
            overriddenBy: 0,
            exceptions: 0,
            derivedEvents: 0,
            reminders: 0,
          },
        },
      ] as never);

      const [match] = await repository.findUnlinkedMatches(
        'cal-1',
        'Design review',
        new Date('2026-07-13'),
        '11:30',
      );

      expect(match.hasUserData).toBe(true);
    });

    it('flags a row carrying reminders, which the sync never recreates', async () => {
      // fromGoogleEvent does not map reminders and upsertByGoogleEventId does not write them:
      // every EventReminder was typed in by the user, and delete cascades it away for good.
      prisma.event.findMany.mockResolvedValue([
        {
          id: 'evt-a',
          createdAt: new Date('2026-07-11T13:10:00Z'),
          categoryId: null,
          categoryTypeId: null,
          labelId: null,
          _count: {
            completions: 0,
            weeklyGoalCompletions: 0,
            overrides: 0,
            overriddenBy: 0,
            exceptions: 0,
            derivedEvents: 0,
            reminders: 2,
          },
        },
      ] as never);

      const [match] = await repository.findUnlinkedMatches(
        'cal-1',
        'Design review',
        new Date('2026-07-13'),
        '11:30',
      );

      expect(match.hasUserData).toBe(true);
    });

    it('counts habit goal history, which a delete would also cascade away', async () => {
      prisma.event.findMany.mockResolvedValue([
        {
          id: 'evt-a',
          createdAt: new Date('2026-07-11T13:10:00Z'),
          categoryId: null,
          categoryTypeId: null,
          labelId: null,
          _count: {
            completions: 0,
            weeklyGoalCompletions: 4,
            overrides: 0,
            overriddenBy: 0,
            exceptions: 0,
            derivedEvents: 0,
            reminders: 0,
          },
        },
      ] as never);

      const [match] = await repository.findUnlinkedMatches(
        'cal-1',
        'Design review',
        new Date('2026-07-13'),
        '11:30',
      );

      expect(match.hasUserData).toBe(true);
    });
  });
});
