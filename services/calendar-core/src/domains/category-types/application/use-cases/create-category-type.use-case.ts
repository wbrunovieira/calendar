import { Injectable, ConflictException } from '@nestjs/common';
import { CategoryTypeRepository } from '../../infrastructure/persistence/category-type.repository';
import { CategoryType } from '../../domain/entities/category-type.entity';
import { CreateCategoryTypeDto } from '../dto/create-category-type.dto';

@Injectable()
export class CreateCategoryTypeUseCase {
  constructor(private readonly repository: CategoryTypeRepository) {}

  async execute(dto: CreateCategoryTypeDto): Promise<CategoryType> {
    // Check if value already exists for this calendar
    const existing = await this.repository.findByValue(dto.value, dto.calendarId);
    if (existing) {
      throw new ConflictException(`Category type with value '${dto.value}' already exists for this calendar`);
    }

    const categoryType = CategoryType.create({
      calendarId: dto.calendarId,
      name: dto.name,
      value: dto.value,
      color: dto.color,
      icon: dto.icon,
    });

    return this.repository.create(categoryType);
  }
}
