import { Controller, Post, Get, Delete, Body, Param, Query, HttpCode, HttpStatus } from '@nestjs/common';
import { CreateEventUseCase } from '../../application/use-cases/create-event.use-case';
import { DeleteEventUseCase } from '../../application/use-cases/delete-event.use-case';
import { ListEventsUseCase } from '../../application/use-cases/list-events.use-case';
import { CreateEventDto } from '../dtos/create-event.dto';

@Controller('events')
export class EventsController {
  constructor(
    private readonly createEventUseCase: CreateEventUseCase,
    private readonly deleteEventUseCase: DeleteEventUseCase,
    private readonly listEventsUseCase: ListEventsUseCase,
  ) {}

  @Get()
  @HttpCode(HttpStatus.OK)
  async list(
    @Query('calendarId') calendarId?: string,
    @Query('categoryId') categoryId?: string,
    @Query('search') search?: string,
  ) {
    const events = await this.listEventsUseCase.execute({
      calendarId,
      categoryId,
      search,
    });

    return events.map((event) => ({
      id: event.id,
      calendarId: event.calendarId,
      categoryId: event.categoryId,
      title: event.title,
      description: event.description,
      startTime: event.startTime,
      endTime: event.endTime,
      startDate: event.startDate,
      endDate: event.endDate,
      isRecurring: event.isRecurring,
      recurrenceFrequency: event.recurrenceFrequency,
      recurrenceInterval: event.recurrenceInterval,
      recurrenceDaysOfWeek: event.recurrenceDaysOfWeek,
      recurrenceDayOfMonth: event.recurrenceDayOfMonth,
      recurrenceWeekOfMonth: event.recurrenceWeekOfMonth,
      recurrenceEndDate: event.recurrenceEndDate,
      googleEventId: event.googleEventId,
      isActive: event.isActive,
      createdAt: event.createdAt,
      updatedAt: event.updatedAt,
    }));
  }

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() createEventDto: CreateEventDto) {
    const event = await this.createEventUseCase.execute(createEventDto);

    return {
      id: event.id,
      calendarId: event.calendarId,
      categoryId: event.categoryId,
      title: event.title,
      description: event.description,
      startTime: event.startTime,
      endTime: event.endTime,
      startDate: event.startDate,
      endDate: event.endDate,
      isRecurring: event.isRecurring,
      recurrenceFrequency: event.recurrenceFrequency,
      recurrenceInterval: event.recurrenceInterval,
      recurrenceDaysOfWeek: event.recurrenceDaysOfWeek,
      recurrenceDayOfMonth: event.recurrenceDayOfMonth,
      recurrenceWeekOfMonth: event.recurrenceWeekOfMonth,
      recurrenceEndDate: event.recurrenceEndDate,
      googleEventId: event.googleEventId,
      isActive: event.isActive,
      createdAt: event.createdAt,
      updatedAt: event.updatedAt,
    };
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param('id') id: string) {
    await this.deleteEventUseCase.execute(id);
  }
}
