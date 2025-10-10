import { Controller, Post, Get, Body, Query, HttpCode, HttpStatus } from '@nestjs/common';
import { CreateCategoryUseCase } from '../../application/use-cases/create-category.use-case';
import { ListCategoriesByCalendarUseCase } from '../../application/use-cases/list-categories-by-calendar.use-case';
import { CreateCategoryDto } from '../dtos/create-category.dto';

@Controller('categories')
export class CategoriesController {
  constructor(
    private readonly createCategoryUseCase: CreateCategoryUseCase,
    private readonly listCategoriesByCalendarUseCase: ListCategoriesByCalendarUseCase,
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
}
