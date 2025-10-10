import { Injectable } from '@nestjs/common';
import { Category } from '../../domain/entities/category.entity';
import { CategoryRepository } from '../../infrastructure/repositories/category.repository';

@Injectable()
export class ListCategoriesByCalendarUseCase {
  constructor(private readonly categoryRepository: CategoryRepository) {}

  async execute(calendarId: string): Promise<Category[]> {
    return await this.categoryRepository.findByCalendarId(calendarId);
  }
}
