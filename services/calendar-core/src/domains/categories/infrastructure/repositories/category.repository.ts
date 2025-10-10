import { Injectable } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { Category } from '../../domain/entities/category.entity';

@Injectable()
export class CategoryRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async create(category: Category): Promise<Category> {
    const created = await this.prisma.category.create({
      data: {
        calendarId: category.calendarId,
        name: category.name,
        icon: category.icon,
        color: category.color,
        type: category.type,
        isActive: category.isActive,
      },
    });

    return new Category(created);
  }

  async findById(id: string): Promise<Category | null> {
    const category = await this.prisma.category.findUnique({
      where: { id },
    });

    return category ? new Category(category) : null;
  }

  async findByCalendarId(calendarId: string): Promise<Category[]> {
    const categories = await this.prisma.category.findMany({
      where: { calendarId, isActive: true },
      orderBy: { name: 'asc' },
    });

    return categories.map((cat) => new Category(cat));
  }

  async update(id: string, category: Partial<Category>): Promise<Category> {
    const updated = await this.prisma.category.update({
      where: { id },
      data: {
        name: category.name,
        icon: category.icon,
        color: category.color,
        type: category.type,
        isActive: category.isActive,
        updatedAt: new Date(),
      },
    });

    return new Category(updated);
  }

  async delete(id: string): Promise<void> {
    await this.prisma.category.delete({
      where: { id },
    });
  }

  async onModuleDestroy() {
    await this.prisma.$disconnect();
  }
}
