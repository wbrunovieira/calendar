import { describe, it, expect, beforeAll, afterAll, beforeEach } from 'vitest';
import { PrismaClient } from '@prisma/client';
import { EventRepository } from '../src/domains/events/infrastructure/repositories/event.repository';
import { Event } from '../src/domains/events/domain/entities/event.entity';

/**
 * The duplication bug was "a column did not survive the round-trip to Postgres". No amount of
 * mocking PrismaClient can catch that class of defect — a mock happily accepts a data object with a
 * missing field, and knows nothing about the UNIQUE index. These tests talk to a real database.
 */
describe('Google event identity (e2e)', () => {
  let prisma: PrismaClient;
  let repository: EventRepository;

  const CALENDAR_ID = 'e2e-cal-1';
  const OTHER_CALENDAR_ID = 'e2e-cal-2';
  const USER_ID = 'e2e-user-1';

  const buildEvent = (overrides: Partial<Event> = {}): Event =>
    new Event({
      calendarId: CALENDAR_ID,
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
      ...overrides,
    });

  beforeAll(async () => {
    prisma = new PrismaClient();
    repository = new EventRepository(prisma);

    await prisma.user.create({
      data: {
        id: USER_ID,
        email: 'e2e@example.com',
        passwordHash: 'x',
        name: 'E2E',
      },
    });

    for (const id of [CALENDAR_ID, OTHER_CALENDAR_ID]) {
      await prisma.calendar.create({
        data: { id, userId: USER_ID, name: id, color: '#000000', type: 'personal' },
      });
    }
  });

  afterAll(async () => {
    await prisma.event.deleteMany({ where: { calendarId: { in: [CALENDAR_ID, OTHER_CALENDAR_ID] } } });
    await prisma.calendar.deleteMany({ where: { id: { in: [CALENDAR_ID, OTHER_CALENDAR_ID] } } });
    await prisma.user.deleteMany({ where: { id: USER_ID } });
    await prisma.$disconnect();
  });

  beforeEach(async () => {
    await prisma.event.deleteMany({
      where: { calendarId: { in: [CALENDAR_ID, OTHER_CALENDAR_ID] } },
    });
  });

  it('round-trips googleEventId through the database', async () => {
    // The original bug in one assertion: create() dropped the column, so the row came back NULL.
    await repository.create(buildEvent({ googleEventId: 'g-evt-round-trip' }));

    const found = await repository.findByGoogleEventId('g-evt-round-trip', CALENDAR_ID);

    expect(found).not.toBeNull();
    expect(found?.googleEventId).toBe('g-evt-round-trip');
  });

  it('rejects a second row for the same Google event in the same calendar', async () => {
    // Proves the migration actually applied. Without the UNIQUE index this silently inserts.
    await repository.create(buildEvent({ googleEventId: 'g-evt-dup' }));

    await expect(
      repository.create(buildEvent({ googleEventId: 'g-evt-dup' })),
    ).rejects.toMatchObject({ code: 'P2002' });

    const rows = await prisma.event.count({ where: { googleEventId: 'g-evt-dup' } });
    expect(rows).toBe(1);
  });

  it('allows the same Google event to be mirrored into a different calendar', async () => {
    await repository.create(buildEvent({ googleEventId: 'g-evt-mirrored' }));
    await repository.create(
      buildEvent({ googleEventId: 'g-evt-mirrored', calendarId: OTHER_CALENDAR_ID }),
    );

    const rows = await prisma.event.count({ where: { googleEventId: 'g-evt-mirrored' } });
    expect(rows).toBe(2);
  });

  it('does not treat locally-created events as duplicates of each other', async () => {
    // Postgres treats NULLs as distinct in a unique index. If it did not, the constraint would
    // reject the second habit/todo the user creates and break the whole app.
    await repository.create(buildEvent({ googleEventId: null, title: 'Local A' }));
    await repository.create(buildEvent({ googleEventId: null, title: 'Local B' }));

    const rows = await prisma.event.count({ where: { calendarId: CALENDAR_ID } });
    expect(rows).toBe(2);
  });

  it('upserts by Google identity instead of inserting a duplicate', async () => {
    await repository.upsertByGoogleEventId(
      buildEvent({ googleEventId: 'g-evt-upsert', title: 'First title' }),
    );
    await repository.upsertByGoogleEventId(
      buildEvent({ googleEventId: 'g-evt-upsert', title: 'Renamed in Google' }),
    );

    const rows = await prisma.event.findMany({ where: { googleEventId: 'g-evt-upsert' } });

    expect(rows).toHaveLength(1);
    expect(rows[0].title).toBe('Renamed in Google');
  });

  it('round-trips the meeting details (JSONB attendees/organizer) through Postgres', async () => {
    const attendees = [
      { email: 'natalia@zubale.com', displayName: 'Natalia', responseStatus: 'accepted' },
      { email: 'bruno@wbdigitalsolutions.com', displayName: 'Bruno', responseStatus: 'needsAction' },
    ];
    const organizer = { email: 'natalia@zubale.com', displayName: 'Natalia' };

    await repository.upsertByGoogleEventId(
      buildEvent({
        googleEventId: 'g-evt-meet',
        title: 'Technical Interview',
        location: 'Rio de Janeiro',
        meetingUrl: 'https://meet.google.com/vrt-wviz-kwr',
        attendees,
        organizer,
      }),
    );

    const row = await prisma.event.findUnique({
      where: { calendarId_googleEventId: { calendarId: CALENDAR_ID, googleEventId: 'g-evt-meet' } },
    });

    expect(row?.location).toBe('Rio de Janeiro');
    expect(row?.meetingUrl).toBe('https://meet.google.com/vrt-wviz-kwr');
    expect(row?.attendees).toEqual(attendees);
    expect(row?.organizer).toEqual(organizer);
  });

  it('stores null meeting details as SQL NULL, not the JSON string "null"', async () => {
    await repository.upsertByGoogleEventId(
      buildEvent({ googleEventId: 'g-evt-nomeet', attendees: null, organizer: null }),
    );

    const row = await prisma.event.findUnique({
      where: { calendarId_googleEventId: { calendarId: CALENDAR_ID, googleEventId: 'g-evt-nomeet' } },
    });

    expect(row?.meetingUrl).toBeNull();

    // Prisma reads both SQL NULL and JSON `null` as JS null, so the checks above can't tell them
    // apart. `IS NULL` only matches SQL NULL — this proves upsert wrote Prisma.DbNull, not JsonNull.
    const [raw] = await prisma.$queryRaw<Array<{ attendees_null: boolean; organizer_null: boolean }>>`
      SELECT attendees IS NULL AS attendees_null, organizer IS NULL AS organizer_null
      FROM events
      WHERE calendar_id = ${CALENDAR_ID} AND google_event_id = 'g-evt-nomeet'
    `;
    expect(raw.attendees_null).toBe(true);
    expect(raw.organizer_null).toBe(true);
  });

  it('does not wipe the user categorisation when Google re-delivers the event', async () => {
    // upsert's `update` payload deliberately omits categoryId/labelId. If a future refactor spread
    // the create payload into update, every sync tick would silently strip the user's curation.
    const category = await prisma.category.create({
      data: { calendarId: CALENDAR_ID, name: 'Work', color: '#111111' },
    });

    try {
      await repository.upsertByGoogleEventId(buildEvent({ googleEventId: 'g-evt-cat' }));
      await prisma.event.update({
        where: { calendarId_googleEventId: { calendarId: CALENDAR_ID, googleEventId: 'g-evt-cat' } },
        data: { categoryId: category.id },
      });

      await repository.upsertByGoogleEventId(
        buildEvent({ googleEventId: 'g-evt-cat', title: 'Renamed in Google' }),
      );

      const row = await prisma.event.findUnique({
        where: { calendarId_googleEventId: { calendarId: CALENDAR_ID, googleEventId: 'g-evt-cat' } },
      });

      expect(row?.title).toBe('Renamed in Google');
      expect(row?.categoryId).toBe(category.id);
    } finally {
      await prisma.event.deleteMany({ where: { googleEventId: 'g-evt-cat' } });
      await prisma.category.delete({ where: { id: category.id } });
    }
  });

  it('marks a row the user categorised as unsafe to delete', async () => {
    const category = await prisma.category.create({
      data: { calendarId: CALENDAR_ID, name: 'Work', color: '#111111' },
    });

    try {
      const created = await repository.create(
        buildEvent({ googleEventId: null, title: 'Curated' }),
      );
      await prisma.event.update({
        where: { id: created.id },
        data: { categoryId: category.id },
      });

      const [match] = await repository.findUnlinkedMatches(
        CALENDAR_ID,
        'Curated',
        new Date('2026-07-13'),
        '11:30',
      );

      expect(match.hasUserData).toBe(true);
    } finally {
      await prisma.category.delete({ where: { id: category.id } });
    }
  });

  it('never offers a HABIT as a deletion candidate to the backfill', async () => {
    // findUnlinkedMatches gates a hard delete. A habit sharing a Google event's title and time
    // must be invisible to it.
    await repository.create(
      buildEvent({ googleEventId: null, eventType: 'HABIT', title: 'Gym' }),
    );
    await repository.create(
      buildEvent({ googleEventId: null, eventType: 'EVENT', title: 'Gym' }),
    );

    const matches = await repository.findUnlinkedMatches(
      CALENDAR_ID,
      'Gym',
      new Date('2026-07-13'),
      '11:30',
    );

    expect(matches).toHaveLength(1);
  });

  it('flags an event that carries completions as unsafe to delete', async () => {
    const created = await repository.create(
      buildEvent({ googleEventId: null, title: 'With history' }),
    );
    await prisma.eventCompletion.create({
      data: { eventId: created.id, occurrenceDate: new Date('2026-07-13'), completed: true },
    });

    const [match] = await repository.findUnlinkedMatches(
      CALENDAR_ID,
      'With history',
      new Date('2026-07-13'),
      '11:30',
    );

    expect(match.hasUserData).toBe(true);
  });
});
