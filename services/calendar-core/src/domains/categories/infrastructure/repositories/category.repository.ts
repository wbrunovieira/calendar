import { Injectable } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { Category } from '../../domain/entities/category.entity';

@Injectable()
export class CategoryRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async create(category: Category, typeIds?: string[]): Promise<any> {
    const created = await this.prisma.category.create({
      data: {
        calendarId: category.calendarId,
        name: category.name,
        icon: category.icon,
        color: category.color,
        type: category.type, // mantido para backward compatibility
        isActive: category.isActive,
      },
      include: {
        categoryTypes: {
          include: {
            categoryType: true,
          },
        },
      },
    });

    // Se foram fornecidos typeIds, criar relacionamentos
    if (typeIds && typeIds.length > 0) {
      await this.prisma.categoryToType.createMany({
        data: typeIds.map(typeId => ({
          categoryId: created.id,
          categoryTypeId: typeId,
        })),
        skipDuplicates: true,
      });
    }

    // Buscar categoria com types incluídos
    return this.findByIdWithTypes(created.id);
  }

  async findById(id: string): Promise<Category | null> {
    const category = await this.prisma.category.findUnique({
      where: { id },
    });

    return category ? new Category(category) : null;
  }

  async findByIdWithTypes(id: string): Promise<any | null> {
    const category = await this.prisma.category.findUnique({
      where: { id },
      include: {
        categoryTypes: {
          include: {
            categoryType: true,
          },
        },
      },
    });

    if (!category) return null;

    return {
      ...new Category(category),
      categoryTypes: category.categoryTypes.map(ct => ct.categoryType),
    };
  }

  async findByCalendarId(calendarId: string): Promise<any[]> {
    const categories = await this.prisma.category.findMany({
      where: { calendarId, isActive: true },
      orderBy: { name: 'asc' },
      include: {
        categoryTypes: {
          include: {
            categoryType: true,
          },
        },
      },
    });

    return categories.map((cat) => ({
      ...new Category(cat),
      categoryTypes: cat.categoryTypes.map(ct => ct.categoryType),
    }));
  }

  async update(id: string, category: Partial<Category>, typeIds?: string[]): Promise<any> {
    // Atualizar campos básicos da categoria
    const updated = await this.prisma.category.update({
      where: { id },
      data: {
        name: category.name,
        icon: category.icon,
        color: category.color,
        type: category.type, // mantido para backward compatibility
        isActive: category.isActive,
        updatedAt: new Date(),
      },
    });

    // Se typeIds foram fornecidos, atualizar relacionamentos
    if (typeIds !== undefined) {
      // Remover todos os relacionamentos existentes
      await this.prisma.categoryToType.deleteMany({
        where: { categoryId: id },
      });

      // Criar novos relacionamentos
      if (typeIds.length > 0) {
        await this.prisma.categoryToType.createMany({
          data: typeIds.map(typeId => ({
            categoryId: id,
            categoryTypeId: typeId,
          })),
          skipDuplicates: true,
        });
      }
    }

    // Retornar categoria com types incluídos
    return this.findByIdWithTypes(id);
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
