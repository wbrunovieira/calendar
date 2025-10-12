import {
  Controller,
  Get,
  Post,
  Put,
  Delete,
  Body,
  Param,
  Query,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { CreateCalendarDto } from '../../application/dto/create-calendar.dto';
import { UpdateCalendarDto } from '../../application/dto/update-calendar.dto';
import { ListCalendarsUseCase } from '../../application/use-cases/list-calendars.use-case';
import { CreateCalendarUseCase } from '../../application/use-cases/create-calendar.use-case';
import { UpdateCalendarUseCase } from '../../application/use-cases/update-calendar.use-case';
import { DeleteCalendarUseCase } from '../../application/use-cases/delete-calendar.use-case';

@Controller('calendars')
export class CalendarsController {
  constructor(
    private readonly listCalendars: ListCalendarsUseCase,
    private readonly createCalendar: CreateCalendarUseCase,
    private readonly updateCalendar: UpdateCalendarUseCase,
    private readonly deleteCalendar: DeleteCalendarUseCase,
  ) {}

  @Get()
  async list(@Query('userId') userId?: string) {
    const calendars = await this.listCalendars.execute(userId);
    return {
      data: calendars,
      total: calendars.length,
    };
  }

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() dto: CreateCalendarDto) {
    const calendar = await this.createCalendar.execute(dto);
    return { data: calendar };
  }

  @Put(':id')
  async update(@Param('id') id: string, @Body() dto: UpdateCalendarDto) {
    const calendar = await this.updateCalendar.execute(id, dto);
    return { data: calendar };
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param('id') id: string) {
    await this.deleteCalendar.execute(id);
  }
}
