import { Controller, Post, Get, Delete, Body, Query, Param, HttpCode, HttpStatus } from '@nestjs/common';
import { CreateCategoryUseCase } from '../../application/use-cases/create-category.use-case';
import { ListCategoriesByCalendarUseCase } from '../../application/use-cases/list-categories-by-calendar.use-case';
import { DeleteCategoryUseCase } from '../../application/use-cases/delete-category.use-case';
import { CreateCategoryDto } from '../dtos/create-category.dto';

@Controller('categories')
export class CategoriesController {
  constructor(
    private readonly createCategoryUseCase: CreateCategoryUseCase,
    private readonly listCategoriesByCalendarUseCase: ListCategoriesByCalendarUseCase,
    private readonly deleteCategoryUseCase: DeleteCategoryUseCase,
  ) {}

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() createCategoryDto: CreateCategoryDto) {
    const category = await this.createCategoryUseCase.execute(createCategoryDto);

    return {
      id: category.id,
      calendarId: category.calendarId,
      name: category.name,
      icon: category.icon,
      color: category.color,
      type: category.type,
      isActive: category.isActive,
      createdAt: category.createdAt,
      updatedAt: category.updatedAt,
    };
  }

  @Get()
  @HttpCode(HttpStatus.OK)
  async listByCalendar(@Query('calendarId') calendarId: string) {
    const categories = await this.listCategoriesByCalendarUseCase.execute(calendarId);

    return categories.map((category) => ({
      id: category.id,
      calendarId: category.calendarId,
      name: category.name,
      icon: category.icon,
      color: category.color,
      type: category.type,
      isActive: category.isActive,
      createdAt: category.createdAt,
      updatedAt: category.updatedAt,
    }));
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param('id') id: string) {
    await this.deleteCategoryUseCase.execute(id);
  }
}
