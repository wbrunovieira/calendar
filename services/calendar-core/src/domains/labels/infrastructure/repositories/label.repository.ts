import { Injectable } from '@nestjs/common';
import { PrismaService } from '../../../../prisma/prisma.service';
import { Label } from '../../domain/entities/label.entity';
import { LabelRepository } from './label.repository.interface';

@Injectable()
export class PrismaLabelRepository implements LabelRepository {
  constructor(private readonly prisma: PrismaService) {}

  async create(label: Label): Promise<Label> {
    const data = label.toJSON();

    const created = await this.prisma.label.create({
      data: {
        id: data.id,
        calendarId: data.calendarId,
        name: data.name,
        color: data.color,
        isActive: data.isActive,
      },
    });

    return new Label({
      id: created.id,
      calendarId: created.calendarId,
      name: created.name,
      color: created.color,
      isActive: created.isActive,
      createdAt: created.createdAt,
      updatedAt: created.updatedAt,
    });
  }

  async findById(id: string): Promise<Label | null> {
    const label = await this.prisma.label.findUnique({
      where: { id },
    });

    if (!label) {
      return null;
    }

    return new Label({
      id: label.id,
      calendarId: label.calendarId,
      name: label.name,
      color: label.color,
      isActive: label.isActive,
      createdAt: label.createdAt,
      updatedAt: label.updatedAt,
    });
  }

  async findByCalendarId(calendarId: string): Promise<Label[]> {
    const labels = await this.prisma.label.findMany({
      where: {
        calendarId,
        isActive: true,
      },
      orderBy: {
        name: 'asc',
      },
    });

    return labels.map(
      (label) =>
        new Label({
          id: label.id,
          calendarId: label.calendarId,
          name: label.name,
          color: label.color,
          isActive: label.isActive,
          createdAt: label.createdAt,
          updatedAt: label.updatedAt,
        }),
    );
  }

  async update(label: Label): Promise<Label> {
    const data = label.toJSON();

    const updated = await this.prisma.label.update({
      where: { id: data.id },
      data: {
        name: data.name,
        color: data.color,
        isActive: data.isActive,
      },
    });

    return new Label({
      id: updated.id,
      calendarId: updated.calendarId,
      name: updated.name,
      color: updated.color,
      isActive: updated.isActive,
      createdAt: updated.createdAt,
      updatedAt: updated.updatedAt,
    });
  }

  async delete(id: string): Promise<void> {
    await this.prisma.label.delete({
      where: { id },
    });
  }
}
