import { Injectable, NotFoundException } from '@nestjs/common';
import { CategoryRepository } from '../../infrastructure/repositories/category.repository';

@Injectable()
export class DeleteCategoryUseCase {
  constructor(private readonly categoryRepository: CategoryRepository) {}

  async execute(id: string): Promise<void> {
    const category = await this.categoryRepository.findById(id);

    if (!category) {
      throw new NotFoundException(`Category with ID ${id} not found`);
    }

    await this.categoryRepository.delete(id);
  }
}
