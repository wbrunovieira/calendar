import { Injectable, NotFoundException } from '@nestjs/common';
import { Category } from '../../domain/entities/category.entity';
import { CategoryRepository } from '../../infrastructure/repositories/category.repository';
import { UpdateCategoryDto } from '../../infrastructure/dtos/update-category.dto';

@Injectable()
export class UpdateCategoryUseCase {
  constructor(private readonly categoryRepository: CategoryRepository) {}

  async execute(id: string, dto: UpdateCategoryDto): Promise<Category> {
    // Buscar categoria existente
    const category = await this.categoryRepository.findById(id);

    if (!category) {
      throw new NotFoundException(`Category with id ${id} not found`);
    }

    // Atualizar a entidade de domínio
    category.update({
      name: dto.name,
      icon: dto.icon,
      color: dto.color,
      type: dto.type,
    });

    // Persistir no banco
    return await this.categoryRepository.update(id, category);
  }
}
