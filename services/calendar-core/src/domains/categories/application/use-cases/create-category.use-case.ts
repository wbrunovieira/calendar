import { Injectable } from '@nestjs/common';
import { Category } from '../../domain/entities/category.entity';
import { CategoryRepository } from '../../infrastructure/repositories/category.repository';
import { CreateCategoryDto } from '../../infrastructure/dtos/create-category.dto';

@Injectable()
export class CreateCategoryUseCase {
  constructor(private readonly categoryRepository: CategoryRepository) {}

  async execute(dto: CreateCategoryDto): Promise<Category> {
    // Criar a entidade de domínio
    const category = Category.create({
      calendarId: dto.calendarId,
      name: dto.name,
      color: dto.color,
      icon: dto.icon,
      type: dto.type,
    });

    // Persistir no banco
    return await this.categoryRepository.create(category);
  }
}
