import { Injectable } from '@nestjs/common';
import { CategoryTypeRepository } from '../../infrastructure/persistence/category-type.repository';
import { CategoryType } from '../../domain/entities/category-type.entity';

@Injectable()
export class ListCategoryTypesUseCase {
  constructor(private readonly repository: CategoryTypeRepository) {}

  async execute(calendarId?: string): Promise<CategoryType[]> {
    return this.repository.findAll(calendarId);
  }
}
