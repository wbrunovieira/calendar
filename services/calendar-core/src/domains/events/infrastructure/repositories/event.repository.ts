import { Injectable } from "@nestjs/common";
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

@Injectable()
export class EventRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
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

  async findByGoogleEventId(googleEventId: string): Promise<Event | null> {
    const event = await this.prisma.event.findFirst({
      where: { googleEventId },
    });
    return event ? new Event(event) : null;
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
    await this.prisma.$disconnect();
  }
}
