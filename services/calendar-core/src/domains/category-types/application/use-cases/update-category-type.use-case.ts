import {
  Injectable,
  NotFoundException,
  ConflictException,
} from '@nestjs/common';
import { CategoryTypeRepository } from '../../infrastructure/persistence/category-type.repository';
import { CategoryType } from '../../domain/entities/category-type.entity';
import { UpdateCategoryTypeDto } from '../dto/update-category-type.dto';

@Injectable()
export class UpdateCategoryTypeUseCase {
  constructor(private readonly repository: CategoryTypeRepository) {}

  async execute(id: string, dto: UpdateCategoryTypeDto): Promise<CategoryType> {
    const categoryType = await this.repository.findById(id);
    if (!categoryType) {
      throw new NotFoundException(`Category type with id '${id}' not found`);
    }

    // Check if new value conflicts with existing in the same calendar
    if (dto.value && dto.value !== categoryType.value) {
      const existing = await this.repository.findByValue(
        dto.value,
        categoryType.calendarId,
      );
      if (existing) {
        throw new ConflictException(
          `Category type with value '${dto.value}' already exists for this calendar`,
        );
      }
    }

    categoryType.update(dto);

    return this.repository.update(id, categoryType);
  }
}
