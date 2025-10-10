import { Injectable } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { EventExecution } from '../../domain/entities/event-execution.entity';

@Injectable()
export class EventExecutionRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async findByEventAndDate(
    eventId: string,
    executionDate: Date,
  ): Promise<EventExecution | null> {
    const execution = await this.prisma.eventExecution.findUnique({
      where: {
        eventId_executionDate: {
          eventId,
          executionDate,
        },
      },
    });

    return execution ? new EventExecution(execution) : null;
  }

  async findByEventId(eventId: string): Promise<EventExecution[]> {
    const executions = await this.prisma.eventExecution.findMany({
      where: { eventId },
      orderBy: { executionDate: 'desc' },
    });

    return executions.map((exec) => new EventExecution(exec));
  }

  async findByDateRange(
    eventId: string,
    startDate: Date,
    endDate: Date,
  ): Promise<EventExecution[]> {
    const executions = await this.prisma.eventExecution.findMany({
      where: {
        eventId,
        executionDate: {
          gte: startDate,
          lte: endDate,
        },
      },
      orderBy: { executionDate: 'asc' },
    });

    return executions.map((exec) => new EventExecution(exec));
  }

  async create(execution: Partial<EventExecution>): Promise<EventExecution> {
    const created = await this.prisma.eventExecution.create({
      data: {
        eventId: execution.eventId!,
        executionDate: execution.executionDate!,
        completed: execution.completed ?? false,
        completedAt: execution.completedAt,
        notes: execution.notes,
      },
    });

    return new EventExecution(created);
  }

  async update(id: string, data: Partial<EventExecution>): Promise<EventExecution> {
    const updated = await this.prisma.eventExecution.update({
      where: { id },
      data: {
        completed: data.completed,
        completedAt: data.completedAt,
        notes: data.notes,
        updatedAt: new Date(),
      },
    });

    return new EventExecution(updated);
  }

  async upsert(
    eventId: string,
    executionDate: Date,
    completed: boolean,
    notes?: string,
  ): Promise<EventExecution> {
    const upserted = await this.prisma.eventExecution.upsert({
      where: {
        eventId_executionDate: {
          eventId,
          executionDate,
        },
      },
      create: {
        eventId,
        executionDate,
        completed,
        completedAt: completed ? new Date() : null,
        notes,
      },
      update: {
        completed,
        completedAt: completed ? new Date() : null,
        notes,
        updatedAt: new Date(),
      },
    });

    return new EventExecution(upserted);
  }

  async onModuleDestroy() {
    await this.prisma.$disconnect();
  }
}
