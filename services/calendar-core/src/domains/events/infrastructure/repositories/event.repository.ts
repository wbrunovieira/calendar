import { Injectable } from "@nestjs/common";
import { PrismaClient } from "@prisma/client";
import { Event } from "../../domain/entities/event.entity";

export interface FindAllFilters {
  calendarId?: string;
  categoryId?: string;
  search?: string;
  eventType?: string;
}

@Injectable()
export class EventRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async create(event: Event): Promise<Event> {
    const created = await this.prisma.event.create({
      data: {
        calendarId: event.calendarId,
        categoryId: event.categoryId,
        categoryTypeId: event.categoryTypeId,
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
        isActive: event.isActive,
      },
    });

    return new Event(created);
  }

  async findById(id: string): Promise<Event | null> {
    const event = await this.prisma.event.findUnique({
      where: { id },
    });

    return event ? new Event(event) : null;
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
        exceptions: true,
        overrides: {
          include: {
            overrideEvent: true,
          },
        },
        completions: true,
      },
    });

    return events.map((event) => ({
      ...new Event(event),
      category: event.category,
      categoryType: event.categoryType,
      exceptions: event.exceptions,
      overrides: event.overrides,
      executions: event.completions.map((completion) => ({
        id: completion.id,
        eventId: completion.eventId,
        executionDate: completion.occurrenceDate,
        completed: completion.completed,
        notes: completion.notes,
      })),
    }));
  }

  async findByCalendarId(calendarId: string): Promise<Event[]> {
    const events = await this.prisma.event.findMany({
      where: { calendarId, isActive: true },
      orderBy: [{ displayOrder: "asc" }, { startDate: "asc" }],
    });

    return events.map((event) => new Event(event));
  }

  async update(id: string, event: Partial<Event>): Promise<Event> {
    const updated = await this.prisma.event.update({
      where: { id },
      data: {
        title: event.title,
        description: event.description,
        startTime: event.startTime,
        endTime: event.endTime,
        startDate: event.startDate,
        endDate: event.endDate,
        categoryId: event.categoryId,
        categoryTypeId: event.categoryTypeId,
        recurrenceRule: event.recurrenceRule,
        status: event.status,
        eventType: event.eventType,
        priority: event.priority,
        dueDate: event.dueDate,
        isActive: event.isActive,
        updatedAt: new Date(),
      },
    });

    return new Event(updated);
  }

  async delete(id: string): Promise<void> {
    await this.prisma.event.delete({
      where: { id },
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
    await this.prisma.$disconnect();
  }
}
