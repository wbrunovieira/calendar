import {
  Controller,
  Post,
  Get,
  Delete,
  Put,
  Body,
  Param,
  HttpCode,
  HttpStatus,
  Query,
} from '@nestjs/common';
import { CreateCategoryTypeUseCase } from '../../application/use-cases/create-category-type.use-case';
import { ListCategoryTypesUseCase } from '../../application/use-cases/list-category-types.use-case';
import { UpdateCategoryTypeUseCase } from '../../application/use-cases/update-category-type.use-case';
import { DeleteCategoryTypeUseCase } from '../../application/use-cases/delete-category-type.use-case';
import { CreateCategoryTypeDto } from '../../application/dto/create-category-type.dto';
import { UpdateCategoryTypeDto } from '../../application/dto/update-category-type.dto';

@Controller('category-types')
export class CategoryTypesController {
  constructor(
    private readonly createCategoryTypeUseCase: CreateCategoryTypeUseCase,
    private readonly listCategoryTypesUseCase: ListCategoryTypesUseCase,
    private readonly updateCategoryTypeUseCase: UpdateCategoryTypeUseCase,
    private readonly deleteCategoryTypeUseCase: DeleteCategoryTypeUseCase,
  ) {}

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() createCategoryTypeDto: CreateCategoryTypeDto) {
    const categoryType = await this.createCategoryTypeUseCase.execute(
      createCategoryTypeDto,
    );

    return {
      id: categoryType.id,
      calendarId: categoryType.calendarId,
      name: categoryType.name,
      value: categoryType.value,
      icon: categoryType.icon,
      color: categoryType.color,
      isActive: categoryType.isActive,
      createdAt: categoryType.createdAt,
      updatedAt: categoryType.updatedAt,
    };
  }

  @Get()
  @HttpCode(HttpStatus.OK)
  async list(@Query('calendarId') calendarId?: string) {
    const types = await this.listCategoryTypesUseCase.execute(calendarId);

    return types.map((type) => ({
      id: type.id,
      calendarId: type.calendarId,
      name: type.name,
      value: type.value,
      icon: type.icon,
      color: type.color,
      isActive: type.isActive,
      createdAt: type.createdAt,
      updatedAt: type.updatedAt,
    }));
  }

  @Put(':id')
  @HttpCode(HttpStatus.OK)
  async update(
    @Param('id') id: string,
    @Body() updateCategoryTypeDto: UpdateCategoryTypeDto,
  ) {
    const categoryType = await this.updateCategoryTypeUseCase.execute(
      id,
      updateCategoryTypeDto,
    );

    return {
      id: categoryType.id,
      calendarId: categoryType.calendarId,
      name: categoryType.name,
      value: categoryType.value,
      icon: categoryType.icon,
      color: categoryType.color,
      isActive: categoryType.isActive,
      createdAt: categoryType.createdAt,
      updatedAt: categoryType.updatedAt,
    };
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param('id') id: string) {
    await this.deleteCategoryTypeUseCase.execute(id);
  }
}
