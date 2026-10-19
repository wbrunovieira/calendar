import { Injectable, OnModuleDestroy, Optional } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';

export interface HabitSlip {
  id: string;
  eventId: string;
  slipDate: Date;
  trigger?: string | null;
  note?: string | null;
}

@Injectable()
export class HabitSlipRepository implements OnModuleDestroy {
  private prisma: PrismaClient;
  private readonly ownsPrisma: boolean;

  // Test seam; see EventRepository for why @Optional() is required.
  constructor(@Optional() prisma?: PrismaClient) {
    this.ownsPrisma = !prisma;
    this.prisma = prisma ?? new PrismaClient();
  }

  async upsert(input: Omit<HabitSlip, 'id'>): Promise<HabitSlip> {
    const data = { trigger: input.trigger ?? null, note: input.note ?? null };
    return this.prisma.habitSlip.upsert({
      where: {
        eventId_slipDate: { eventId: input.eventId, slipDate: input.slipDate },
      },
      create: { eventId: input.eventId, slipDate: input.slipDate, ...data },
      update: data,
    });
  }

  async delete(eventId: string, slipDate: Date): Promise<void> {
    await this.prisma.habitSlip.deleteMany({ where: { eventId, slipDate } });
  }

  async findByEventIds(eventIds: string[]): Promise<HabitSlip[]> {
    if (!eventIds.length) return [];
    return this.prisma.habitSlip.findMany({
      where: { eventId: { in: eventIds } },
      orderBy: { slipDate: 'asc' },
    });
  }

  async onModuleDestroy() {
    if (this.ownsPrisma) await this.prisma.$disconnect();
  }
}
