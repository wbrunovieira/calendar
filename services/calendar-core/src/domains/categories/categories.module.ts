import { Module } from '@nestjs/common';
import { CategoriesController } from './infrastructure/controllers/categories.controller';
import { CreateCategoryUseCase } from './application/use-cases/create-category.use-case';
import { ListCategoriesByCalendarUseCase } from './application/use-cases/list-categories-by-calendar.use-case';
import { DeleteCategoryUseCase } from './application/use-cases/delete-category.use-case';
import { UpdateCategoryUseCase } from './application/use-cases/update-category.use-case';
import { CategoryRepository } from './infrastructure/repositories/category.repository';

@Module({
  controllers: [CategoriesController],
  providers: [
    CreateCategoryUseCase,
    ListCategoriesByCalendarUseCase,
    DeleteCategoryUseCase,
    UpdateCategoryUseCase,
    CategoryRepository,
  ],
  exports: [CategoryRepository],
})
export class CategoriesModule {}
