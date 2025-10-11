import { Controller, Post, Get, Delete, Put, Body, Param, Query, HttpCode, HttpStatus } from '@nestjs/common';
import { CreateEventUseCase } from '../../application/use-cases/create-event.use-case';
import { DeleteEventUseCase } from '../../application/use-cases/delete-event.use-case';
import { ListEventsUseCase } from '../../application/use-cases/list-events.use-case';
import { UpdateEventUseCase } from '../../application/use-cases/update-event.use-case';
import { ToggleEventExecutionUseCase } from '../../application/use-cases/toggle-event-execution.use-case';
import { GetEventExecutionsUseCase } from '../../application/use-cases/get-event-executions.use-case';
import { CreateEventDto } from '../dtos/create-event.dto';
import { UpdateEventDto } from '../dtos/update-event.dto';
import { ToggleEventExecutionDto } from '../dtos/toggle-event-execution.dto';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

@Controller('events')
export class EventsController {
  constructor(
    private readonly createEventUseCase: CreateEventUseCase,
    private readonly deleteEventUseCase: DeleteEventUseCase,
    private readonly listEventsUseCase: ListEventsUseCase,
    private readonly updateEventUseCase: UpdateEventUseCase,
    private readonly toggleEventExecutionUseCase: ToggleEventExecutionUseCase,
    private readonly getEventExecutionsUseCase: GetEventExecutionsUseCase,
  ) {}

  // Helper: converter RRULE para formato antigo (compatibilidade frontend)
  private convertRRuleToLegacy(recurrenceRule: string | null): any {
    if (!recurrenceRule) {
      return {
        isRecurring: false,
        recurrenceFrequency: null,
        recurrenceInterval: null,
        recurrenceDaysOfWeek: [],
        recurrenceDayOfMonth: null,
        recurrenceWeekOfMonth: null,
        recurrenceEndDate: null,
      };
    }

    const rule = RRuleHelper.parse(recurrenceRule);
    if (!rule) {
      return {
        isRecurring: false,
        recurrenceFrequency: null,
        recurrenceInterval: null,
        recurrenceDaysOfWeek: [],
        recurrenceDayOfMonth: null,
        recurrenceWeekOfMonth: null,
        recurrenceEndDate: null,
      };
    }

    return {
      isRecurring: true,
      recurrenceFrequency: rule.freq.toLowerCase(),
      recurrenceInterval: rule.interval || 1,
      recurrenceDaysOfWeek: rule.byweekday || [],
      recurrenceDayOfMonth: rule.bymonthday || null,
      recurrenceWeekOfMonth: null,
      recurrenceEndDate: rule.until || null,
    };
  }

  @Get()
  @HttpCode(HttpStatus.OK)
  async list(
    @Query('calendarId') calendarId?: string,
    @Query('categoryId') categoryId?: string,
    @Query('search') search?: string,
    @Query('startDate') startDate?: string,
    @Query('endDate') endDate?: string,
  ) {
    const events = await this.listEventsUseCase.execute({
      calendarId,
      categoryId,
      search,
      startDate,
      endDate,
    });

    // Eventos já vêm expandidos e convertidos do ListEventsUseCase
    // Não precisa converter novamente
    return events;
  }

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() createEventDto: CreateEventDto) {
    const event = await this.createEventUseCase.execute(createEventDto);
    const legacy = this.convertRRuleToLegacy(event.recurrenceRule ?? null);

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
      ...legacy,
      googleEventId: event.googleEventId,
      isActive: event.isActive,
      createdAt: event.createdAt,
      updatedAt: event.updatedAt,
    };
  }

  @Put(':id')
  @HttpCode(HttpStatus.OK)
  async update(@Param('id') id: string, @Body() updateEventDto: UpdateEventDto) {
    const event = await this.updateEventUseCase.execute(id, updateEventDto);
    const legacy = this.convertRRuleToLegacy(event.recurrenceRule ?? null);

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
      ...legacy,
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

  @Post('executions/toggle')
  @HttpCode(HttpStatus.OK)
  async toggleExecution(@Body() dto: ToggleEventExecutionDto) {
    const execution = await this.toggleEventExecutionUseCase.execute({
      eventId: dto.eventId,
      executionDate: new Date(dto.executionDate),
      completed: dto.completed,
      notes: dto.notes,
    });

    return {
      id: execution.id,
      eventId: execution.eventId,
      executionDate: execution.executionDate,
      completed: execution.completed,
      completedAt: execution.completedAt,
      notes: execution.notes,
      createdAt: execution.createdAt,
      updatedAt: execution.updatedAt,
    };
  }

  @Get(':id/executions')
  @HttpCode(HttpStatus.OK)
  async getExecutions(@Param('id') eventId: string) {
    const executions = await this.getEventExecutionsUseCase.execute(eventId);

    return executions.map((execution) => ({
      id: execution.id,
      eventId: execution.eventId,
      executionDate: execution.executionDate,
      completed: execution.completed,
      completedAt: execution.completedAt,
      notes: execution.notes,
      createdAt: execution.createdAt,
      updatedAt: execution.updatedAt,
    }));
  }
}
