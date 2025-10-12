import { Injectable, NotFoundException } from '@nestjs/common';
import { CategoryTypeRepository } from '../../infrastructure/persistence/category-type.repository';

@Injectable()
export class DeleteCategoryTypeUseCase {
  constructor(private readonly repository: CategoryTypeRepository) {}

  async execute(id: string): Promise<void> {
    const categoryType = await this.repository.findById(id);
    if (!categoryType) {
      throw new NotFoundException(`Category type with id '${id}' not found`);
    }

    await this.repository.delete(id);
  }
}
