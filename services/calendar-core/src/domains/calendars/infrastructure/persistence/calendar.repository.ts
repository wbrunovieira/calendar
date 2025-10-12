import { Injectable } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { Calendar } from '../../domain/entities/calendar.entity';

@Injectable()
export class CalendarRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async create(calendar: Calendar): Promise<Calendar> {
    const created = await this.prisma.calendar.create({
      data: {
        id: calendar.id,
        userId: calendar.userId,
        name: calendar.name,
        email: calendar.email,
        color: calendar.color,
        type: calendar.type,
        googleCalendarId: calendar.googleCalendarId,
        financeProfileId: calendar.financeProfileId,
        isActive: calendar.isActive,
      },
    });

    return Calendar.create(created);
  }

  async findAll(userId?: string): Promise<Calendar[]> {
    const calendars = await this.prisma.calendar.findMany({
      where: userId ? { userId } : undefined,
      orderBy: { createdAt: 'desc' },
    });

    return calendars.map((calendar) => Calendar.create(calendar));
  }

  async findById(id: string): Promise<Calendar | null> {
    const calendar = await this.prisma.calendar.findUnique({
      where: { id },
    });

    return calendar ? Calendar.create(calendar) : null;
  }

  async update(id: string, data: Partial<Calendar>): Promise<Calendar> {
    const updated = await this.prisma.calendar.update({
      where: { id },
      data: {
        name: data.name,
        email: data.email,
        color: data.color,
        type: data.type,
        googleCalendarId: data.googleCalendarId,
        financeProfileId: data.financeProfileId,
        isActive: data.isActive,
      },
    });

    return Calendar.create(updated);
  }

  async delete(id: string): Promise<void> {
    await this.prisma.calendar.delete({
      where: { id },
    });
  }

  async findByFinanceProfileId(financeProfileId: string): Promise<Calendar[]> {
    const calendars = await this.prisma.calendar.findMany({
      where: { financeProfileId },
    });

    return calendars.map((calendar) => Calendar.create(calendar));
  }

  async onModuleDestroy() {
    await this.prisma.$disconnect();
  }
}
