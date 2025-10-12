import { Module } from '@nestjs/common';
import { CategoryTypesController } from './infrastructure/http/category-types.controller';
import { CreateCategoryTypeUseCase } from './application/use-cases/create-category-type.use-case';
import { ListCategoryTypesUseCase } from './application/use-cases/list-category-types.use-case';
import { UpdateCategoryTypeUseCase } from './application/use-cases/update-category-type.use-case';
import { DeleteCategoryTypeUseCase } from './application/use-cases/delete-category-type.use-case';
import { CategoryTypeRepository } from './infrastructure/persistence/category-type.repository';

@Module({
  controllers: [CategoryTypesController],
  providers: [
    CreateCategoryTypeUseCase,
    ListCategoryTypesUseCase,
    UpdateCategoryTypeUseCase,
    DeleteCategoryTypeUseCase,
    CategoryTypeRepository,
  ],
  exports: [CategoryTypeRepository],
})
export class CategoryTypesModule {}
