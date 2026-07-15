import { Injectable, Optional } from "@nestjs/common";
import { PrismaClient } from "@prisma/client";
import { Event } from "../../domain/entities/event.entity";

export interface FindAllFilters {
  calendarId?: string;
  categoryId?: string;
  search?: string;
  eventType?: string;
}

export interface ReminderInput {
  minutesBefore: number;
  method?: string;
}

export interface UnlinkedMatch {
  id: string;
  createdAt: Date;
  /**
   * True when the row carries anything the Google sync cannot reconstruct, making it unsafe to
   * delete. Two kinds of thing qualify:
   *
   *  - relations that cascade on delete (completions, weekly goals, recurrence overrides and
   *    exceptions, derived instances, reminders);
   *  - scalar curation the sync never writes (category, category type, label).
   *
   * The sync only ever writes a dozen columns; everything else on the row was put there by the
   * user. Which of those the persistence layer cascades is its own knowledge, so the repository
   * answers the question rather than handing out counts.
   */
  hasUserData: boolean;
}

@Injectable()
export class EventRepository {
  private prisma: PrismaClient;
  private readonly ownsPrisma: boolean;

  // Test seam. @Optional() is required: without it Nest reads PrismaClient from the constructor
  // metadata, fails to find a provider for it, and the whole app refuses to boot.
  constructor(@Optional() prisma?: PrismaClient) {
    this.ownsPrisma = !prisma;
    this.prisma = prisma ?? new PrismaClient();
  }

  async create(event: Event, reminders?: ReminderInput[]): Promise<any> {
    const created = await this.prisma.event.create({
      data: {
        calendarId: event.calendarId,
        categoryId: event.categoryId,
        categoryTypeId: event.categoryTypeId,
        labelId: event.labelId,
        title: event.title,
        description: event.description,
        startTime: event.startTime,
        endTime: event.endTime,
        startDate: event.startDate,
        endDate: event.endDate,
        recurrenceRule: event.recurrenceRule,
        recurrenceMasterId: event.recurrenceMasterId,
        status: event.status,
        eventType: event.eventType || "EVENT",
        priority: event.priority,
        dueDate: event.dueDate,
        reminderDaysBefore: event.reminderDaysBefore || [],
        recurrenceType: event.recurrenceType,
        weeklyTargetCount: event.weeklyTargetCount,
        weeklyPreferredDays: event.weeklyPreferredDays || [],
        googleEventId: event.googleEventId ?? null,
        isActive: event.isActive,
      },
      include: {
        reminders: true,
      },
    });

    if (reminders && reminders.length > 0) {
      await this.prisma.eventReminder.createMany({
        data: reminders.map((r) => ({
          eventId: created.id,
          minutesBefore: r.minutesBefore,
          method: r.method || "notification",
        })),
      });

      const withReminders = await this.prisma.event.findUnique({
        where: { id: created.id },
        include: { reminders: true },
      });

      return { ...new Event(withReminders), reminders: withReminders!.reminders };
    }

    return { ...new Event(created), reminders: [] };
  }

  async findById(id: string): Promise<any | null> {
    const event = await this.prisma.event.findUnique({
      where: { id },
      include: { reminders: true },
    });

    if (!event) return null;
    return { ...new Event(event), reminders: event.reminders };
  }

  async findAll(filters?: FindAllFilters): Promise<any[]> {
    const where: any = { isActive: true };

    if (filters?.calendarId) {
      where.calendarId = filters.calendarId;
    }

    if (filters?.categoryId) {
      where.categoryId = filters.categoryId;
    }

    if (filters?.search) {
      where.OR = [
        { title: { contains: filters.search, mode: "insensitive" } },
        { description: { contains: filters.search, mode: "insensitive" } },
      ];
    }

    if (filters?.eventType) {
      where.eventType = filters.eventType;
    }

    const events = await this.prisma.event.findMany({
      where,
      orderBy: [{ displayOrder: "asc" }, { startDate: "asc" }, { startTime: "asc" }],
      include: {
        category: true,
        categoryType: true,
        label: true,
        exceptions: true,
        overrides: {
          include: {
            overrideEvent: true,
          },
        },
        completions: true,
        reminders: true,
      },
    });

    return events.map((event) => {
      return {
        ...new Event(event),
        category: event.category,
        categoryType: event.categoryType,
        label: event.label,
        exceptions: event.exceptions,
        overrides: event.overrides,
        reminders: event.reminders,
        executions: event.completions.map((completion) => ({
          id: completion.id,
          eventId: completion.eventId,
          executionDate: completion.occurrenceDate,
          completed: completion.completed,
          notes: completion.notes,
        })),
      };
    });
  }

  async findByCalendarId(calendarId: string): Promise<Event[]> {
    const events = await this.prisma.event.findMany({
      where: { calendarId, isActive: true },
      orderBy: [{ displayOrder: "asc" }, { startDate: "asc" }],
    });

    return events.map((event) => new Event(event));
  }

  async update(id: string, event: Partial<Event>, reminders?: ReminderInput[]): Promise<any> {
    const updated = await this.prisma.event.update({
      where: { id },
      data: {
        calendarId: event.calendarId,
        title: event.title,
        description: event.description,
        startTime: event.startTime,
        endTime: event.endTime,
        startDate: event.startDate,
        endDate: event.endDate,
        categoryId: event.categoryId,
        categoryTypeId: event.categoryTypeId,
        labelId: event.labelId,
        recurrenceRule: event.recurrenceRule,
        status: event.status,
        eventType: event.eventType,
        priority: event.priority,
        dueDate: event.dueDate,
        reminderDaysBefore: event.reminderDaysBefore ?? undefined,
        recurrenceType: event.recurrenceType,
        weeklyTargetCount: event.weeklyTargetCount,
        weeklyPreferredDays: event.weeklyPreferredDays,
        googleEventId: event.googleEventId,
        isActive: event.isActive,
        updatedAt: new Date(),
      },
    });

    if (reminders !== undefined) {
      // Replace strategy: delete all old, create new
      await this.prisma.eventReminder.deleteMany({ where: { eventId: id } });

      if (reminders.length > 0) {
        await this.prisma.eventReminder.createMany({
          data: reminders.map((r) => ({
            eventId: id,
            minutesBefore: r.minutesBefore,
            method: r.method || "notification",
          })),
        });
      }
    }

    const withReminders = await this.prisma.event.findUnique({
      where: { id },
      include: { reminders: true },
    });

    return { ...new Event(withReminders), reminders: withReminders!.reminders };
  }

  async findUpcomingReminders(windowMinutes: number): Promise<any[]> {
    const now = new Date();
    const events = await this.prisma.event.findMany({
      where: {
        isActive: true,
        reminders: { some: {} },
      },
      include: { reminders: true },
    });

    const results: any[] = [];

    for (const event of events) {
      // Combine startDate + startTime into a full datetime
      const eventDate = new Date(event.startDate);
      if (event.startTime) {
        const [h, m] = event.startTime.split(":").map(Number);
        eventDate.setHours(h, m, 0, 0);
      }

      for (const reminder of event.reminders) {
        const triggerAt = new Date(eventDate.getTime() - reminder.minutesBefore * 60 * 1000);
        const diffMs = triggerAt.getTime() - now.getTime();
        const diffMinutes = diffMs / 60000;

        // Trigger if within window and not more than 1 minute past
        if (diffMinutes >= -1 && diffMinutes <= windowMinutes) {
          results.push({
            eventId: event.id,
            eventTitle: event.title,
            startTime: event.startTime,
            startDate: event.startDate,
            minutesBefore: reminder.minutesBefore,
            triggerAt: triggerAt.toISOString(),
          });
        }
      }
    }

    return results;
  }

  async delete(id: string): Promise<void> {
    await this.prisma.event.delete({
      where: { id },
    });
  }

  async updateGoogleEventId(id: string, googleEventId: string): Promise<void> {
    await this.prisma.event.update({
      where: { id },
      data: { googleEventId },
    });
  }

  /**
   * Rows the broken sync may have orphaned: no Google event linked yet, matching one Google event
   * by its natural key.
   *
   * The WHERE clause is the only gate in front of a hard delete, so it is deliberately narrow:
   * the sync can only ever have produced a top-level, active EVENT. HABITs, TODOs, REMINDERs and
   * derived recurrence instances are local constructs and must never be candidates, even when
   * their title and time happen to collide with a Google event.
   */
  async findUnlinkedMatches(
    calendarId: string,
    title: string,
    startDate: Date,
    startTime: string,
  ): Promise<UnlinkedMatch[]> {
    const events = await this.prisma.event.findMany({
      where: {
        calendarId,
        googleEventId: null,
        title,
        startDate,
        startTime,
        eventType: 'EVENT',
        recurrenceMasterId: null,
        isActive: true,
      },
      select: {
        id: true,
        createdAt: true,
        categoryId: true,
        categoryTypeId: true,
        labelId: true,
        _count: {
          select: {
            // Every relation the schema cascades on delete.
            completions: true,
            weeklyGoalCompletions: true,
            overrides: true,
            overriddenBy: true,
            exceptions: true,
            derivedEvents: true,
            reminders: true,
          },
        },
      },
    });

    return events.map((event) => {
      const cascading = Object.values(event._count).reduce((sum, n) => sum + n, 0);
      const curated =
        event.categoryId !== null || event.categoryTypeId !== null || event.labelId !== null;

      return {
        id: event.id,
        createdAt: event.createdAt,
        hasUserData: cascading > 0 || curated,
      };
    });
  }

  /**
   * Active top-level EVENTs carrying no googleEventId. This is a raw denominator, not a defect
   * count: it lumps together orphans the broken sync lost and events the user simply created by
   * hand and that Google never knew about. The two cannot be told apart from the row alone.
   */
  async countUnlinkedEvents(calendarId: string): Promise<number> {
    return this.prisma.event.count({
      where: {
        calendarId,
        googleEventId: null,
        eventType: 'EVENT',
        recurrenceMasterId: null,
        isActive: true,
      },
    });
  }

  async findByGoogleEventId(
    googleEventId: string,
    calendarId: string,
  ): Promise<Event | null> {
    const event = await this.prisma.event.findUnique({
      where: { calendarId_googleEventId: { calendarId, googleEventId } },
    });
    return event ? new Event(event) : null;
  }

  /**
   * Write a Google-sourced event by its (calendar, Google event) identity, so a concurrent cron
   * tick and manual sync converge on one row instead of two.
   *
   * Only the fields Google owns are written. Category, label and reminders are user curation and
   * are deliberately left untouched — which is also why a row created here is *not* interchangeable
   * with the user's own copy of the same event.
   */
  async upsertByGoogleEventId(event: Event): Promise<void> {
    const googleEventId = event.googleEventId;
    if (!googleEventId) {
      throw new Error('upsertByGoogleEventId requires an event carrying a googleEventId');
    }

    const writable = {
      title: event.title,
      description: event.description,
      startTime: event.startTime,
      endTime: event.endTime,
      startDate: event.startDate,
      endDate: event.endDate,
      recurrenceRule: event.recurrenceRule,
      status: event.status,
    };

    await this.prisma.event.upsert({
      where: {
        calendarId_googleEventId: { calendarId: event.calendarId, googleEventId },
      },
      create: {
        ...writable,
        calendarId: event.calendarId,
        googleEventId,
        eventType: event.eventType || 'EVENT',
        isActive: event.isActive,
      },
      update: writable,
    });
  }

  async updateDisplayOrder(updates: { id: string; displayOrder: number }[]): Promise<void> {
    if (updates.length === 0) return;

    await this.prisma.$transaction(
      updates.map(({ id, displayOrder }) =>
        this.prisma.event.update({
          where: { id },
          data: { displayOrder },
        })
      )
    );
  }

  async onModuleDestroy() {
    // Only disconnect a client we created. A borrowed one belongs to whoever passed it in.
    if (this.ownsPrisma) {
      await this.prisma.$disconnect();
    }
  }
}
