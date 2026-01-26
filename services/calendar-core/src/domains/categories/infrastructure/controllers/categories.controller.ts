import {
  Controller,
  Post,
  Get,
  Delete,
  Put,
  Body,
  Query,
  Param,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { CreateCategoryUseCase } from '../../application/use-cases/create-category.use-case';
import { ListCategoriesByCalendarUseCase } from '../../application/use-cases/list-categories-by-calendar.use-case';
import { DeleteCategoryUseCase } from '../../application/use-cases/delete-category.use-case';
import { UpdateCategoryUseCase } from '../../application/use-cases/update-category.use-case';
import { CreateCategoryDto } from '../dtos/create-category.dto';
import { UpdateCategoryDto } from '../dtos/update-category.dto';

@Controller('categories')
export class CategoriesController {
  constructor(
    private readonly createCategoryUseCase: CreateCategoryUseCase,
    private readonly listCategoriesByCalendarUseCase: ListCategoriesByCalendarUseCase,
    private readonly deleteCategoryUseCase: DeleteCategoryUseCase,
    private readonly updateCategoryUseCase: UpdateCategoryUseCase,
  ) {}

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() createCategoryDto: CreateCategoryDto) {
    const category: any =
      await this.createCategoryUseCase.execute(createCategoryDto);

    return {
      id: category.id,
      calendarId: category.calendarId,
      name: category.name,
      icon: category.icon,
      color: category.color,
      type: category.type,
      categoryTypes: category.categoryTypes || [],
      isActive: category.isActive,
      createdAt: category.createdAt,
      updatedAt: category.updatedAt,
    };
  }

  @Get()
  @HttpCode(HttpStatus.OK)
  async listByCalendar(@Query('calendarId') calendarId: string) {
    const categories: any[] =
      await this.listCategoriesByCalendarUseCase.execute(calendarId);

    return categories.map((category) => ({
      id: category.id,
      calendarId: category.calendarId,
      name: category.name,
      icon: category.icon,
      color: category.color,
      type: category.type,
      categoryTypes: category.categoryTypes || [],
      isActive: category.isActive,
      createdAt: category.createdAt,
      updatedAt: category.updatedAt,
    }));
  }

  @Put(':id')
  @HttpCode(HttpStatus.OK)
  async update(
    @Param('id') id: string,
    @Body() updateCategoryDto: UpdateCategoryDto,
  ) {
    const category: any = await this.updateCategoryUseCase.execute(
      id,
      updateCategoryDto,
    );

    return {
      id: category.id,
      calendarId: category.calendarId,
      name: category.name,
      icon: category.icon,
      color: category.color,
      type: category.type,
      categoryTypes: category.categoryTypes || [],
      isActive: category.isActive,
      createdAt: category.createdAt,
      updatedAt: category.updatedAt,
    };
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param('id') id: string) {
    await this.deleteCategoryUseCase.execute(id);
  }
}
