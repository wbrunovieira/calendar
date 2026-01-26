import { Injectable } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { CategoryType } from '../../domain/entities/category-type.entity';

@Injectable()
export class CategoryTypeRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async create(categoryType: CategoryType): Promise<CategoryType> {
    const created = await this.prisma.categoryType.create({
      data: {
        calendarId: categoryType.calendarId,
        name: categoryType.name,
        value: categoryType.value,
        icon: categoryType.icon,
        color: categoryType.color,
        isActive: categoryType.isActive,
      },
    });

    return new CategoryType(created);
  }

  async findAll(calendarId?: string): Promise<CategoryType[]> {
    const types = await this.prisma.categoryType.findMany({
      where: {
        isActive: true,
        ...(calendarId && { calendarId }),
      },
      orderBy: { name: 'asc' },
    });

    return types.map((type) => new CategoryType(type));
  }

  async findById(id: string): Promise<CategoryType | null> {
    const type = await this.prisma.categoryType.findUnique({
      where: { id },
    });

    return type ? new CategoryType(type) : null;
  }

  async findByValue(
    value: string,
    calendarId: string,
  ): Promise<CategoryType | null> {
    const type = await this.prisma.categoryType.findFirst({
      where: { value, calendarId, isActive: true },
    });

    return type ? new CategoryType(type) : null;
  }

  async update(
    id: string,
    categoryType: Partial<CategoryType>,
  ): Promise<CategoryType> {
    const updated = await this.prisma.categoryType.update({
      where: { id },
      data: {
        name: categoryType.name,
        value: categoryType.value,
        icon: categoryType.icon,
        color: categoryType.color,
        isActive: categoryType.isActive,
        updatedAt: new Date(),
      },
    });

    return new CategoryType(updated);
  }

  async delete(id: string): Promise<void> {
    await this.prisma.categoryType.delete({
      where: { id },
    });
  }

  async onModuleDestroy() {
    await this.prisma.$disconnect();
  }
}
