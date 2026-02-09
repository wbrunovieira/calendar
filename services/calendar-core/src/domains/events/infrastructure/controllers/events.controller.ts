import {
  Controller,
  Post,
  Get,
  Delete,
  Put,
  Body,
  Param,
  Query,
  HttpCode,
  HttpStatus,
} from "@nestjs/common";
import { EventRepository } from "../repositories/event.repository";
import { CreateEventUseCase } from "../../application/use-cases/create-event.use-case";
import { DeleteEventUseCase } from "../../application/use-cases/delete-event.use-case";
import { ListEventsUseCase } from "../../application/use-cases/list-events.use-case";
import { UpdateEventUseCase } from "../../application/use-cases/update-event.use-case";
import { ToggleEventExecutionUseCase } from "../../application/use-cases/toggle-event-execution.use-case";
import { GetEventExecutionsUseCase } from "../../application/use-cases/get-event-executions.use-case";
import { GetEventsStatsUseCase } from "../../application/use-cases/get-events-stats.use-case";
import { GetHabitsStatsUseCase } from "../../application/use-cases/get-habits-stats.use-case";
import { GetWeeklyProgressUseCase } from "../../application/use-cases/get-weekly-progress.use-case";
import { ReorderEventsUseCase } from "../../application/use-cases/reorder-events.use-case";
import { CreateEventDto } from "../dtos/create-event.dto";
import { UpdateEventDto } from "../dtos/update-event.dto";
import { ToggleEventExecutionDto } from "../dtos/toggle-event-execution.dto";
import { RRuleHelper } from "../../domain/utils/rrule-helper";

@Controller("events")
export class EventsController {
  constructor(
    private readonly eventRepository: EventRepository,
    private readonly createEventUseCase: CreateEventUseCase,
    private readonly deleteEventUseCase: DeleteEventUseCase,
    private readonly listEventsUseCase: ListEventsUseCase,
    private readonly updateEventUseCase: UpdateEventUseCase,
    private readonly toggleEventExecutionUseCase: ToggleEventExecutionUseCase,
    private readonly getEventExecutionsUseCase: GetEventExecutionsUseCase,
    private readonly getEventsStatsUseCase: GetEventsStatsUseCase,
    private readonly getHabitsStatsUseCase: GetHabitsStatsUseCase,
    private readonly getWeeklyProgressUseCase: GetWeeklyProgressUseCase,
    private readonly reorderEventsUseCase: ReorderEventsUseCase,
  ) {}

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
    @Query("calendarId") calendarId?: string,
    @Query("categoryId") categoryId?: string,
    @Query("search") search?: string,
    @Query("startDate") startDate?: string,
    @Query("endDate") endDate?: string,
    @Query("eventType") eventType?: string,
  ) {
    const events = await this.listEventsUseCase.execute({
      calendarId,
      categoryId,
      search,
      startDate,
      endDate,
      eventType,
    });
    return events;
  }

  @Get("upcoming-reminders")
  @HttpCode(HttpStatus.OK)
  async getUpcomingReminders(
    @Query("windowMinutes") windowMinutes?: string,
  ) {
    const window = parseInt(windowMinutes || "5", 10);
    return await this.eventRepository.findUpcomingReminders(window);
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
      categoryTypeId: event.categoryTypeId,
      title: event.title,
      description: event.description,
      startTime: event.startTime,
      endTime: event.endTime,
      startDate: event.startDate,
      endDate: event.endDate,
      ...legacy,
      eventType: event.eventType,
      priority: event.priority,
      dueDate: event.dueDate,
      reminders: event.reminders || [],
      googleEventId: event.googleEventId,
      isActive: event.isActive,
      createdAt: event.createdAt,
      updatedAt: event.updatedAt,
    };
  }

  @Put(":id")
  @HttpCode(HttpStatus.OK)
  async update(
    @Param("id") id: string,
    @Body() updateEventDto: UpdateEventDto,
  ) {
    const event = await this.updateEventUseCase.execute(id, updateEventDto);
    const legacy = this.convertRRuleToLegacy(event.recurrenceRule ?? null);

    return {
      id: event.id,
      calendarId: event.calendarId,
      categoryId: event.categoryId,
      categoryTypeId: event.categoryTypeId,
      title: event.title,
      description: event.description,
      startTime: event.startTime,
      endTime: event.endTime,
      startDate: event.startDate,
      endDate: event.endDate,
      ...legacy,
      eventType: event.eventType,
      priority: event.priority,
      dueDate: event.dueDate,
      reminders: event.reminders || [],
      googleEventId: event.googleEventId,
      isActive: event.isActive,
      createdAt: event.createdAt,
      updatedAt: event.updatedAt,
    };
  }

  @Delete(":id")
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param("id") id: string) {
    await this.deleteEventUseCase.execute(id);
  }

  @Delete(":id/recurring")
  @HttpCode(HttpStatus.NO_CONTENT)
  async deleteRecurring(
    @Param("id") id: string,
    @Query("scope") scope: "this" | "future" | "all",
    @Query("occurrenceDate") occurrenceDate: string,
  ) {
    await this.deleteEventUseCase.executeRecurring(id, scope, occurrenceDate);
  }

  @Post("executions/toggle")
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

  @Post("reorder")
  @HttpCode(HttpStatus.OK)
  async reorder(@Body() body: { orderedIds: string[] }) {
    await this.reorderEventsUseCase.execute({ orderedIds: body.orderedIds });
    return { success: true };
  }

  @Get("habits/stats")
  @HttpCode(HttpStatus.OK)
  async getHabitsStats(
    @Query("calendarId") calendarId?: string,
    @Query("categoryId") categoryId?: string,
  ) {
    return await this.getHabitsStatsUseCase.execute({
      calendarId,
      categoryId,
    });
  }

  @Get("habits/weekly-progress")
  @HttpCode(HttpStatus.OK)
  async getWeeklyProgress(
    @Query("calendarId") calendarId?: string,
    @Query("categoryId") categoryId?: string,
  ) {
    return await this.getWeeklyProgressUseCase.execute({
      calendarId,
      categoryId,
    });
  }

  @Get("stats")
  @HttpCode(HttpStatus.OK)
  async getStats(
    @Query("startDate") startDate: string,
    @Query("endDate") endDate: string,
    @Query("calendarId") calendarId?: string,
    @Query("categoryId") categoryId?: string,
    @Query("categoryTypeId") categoryTypeId?: string,
    @Query("groupBy")
    groupBy?: "day" | "week" | "month" | "category" | "categoryType" | "total",
    @Query("includeBreakdown") includeBreakdown?: string,
  ) {
    return await this.getEventsStatsUseCase.execute({
      startDate,
      endDate,
      calendarId,
      categoryId,
      categoryTypeId,
      groupBy,
      includeBreakdown: includeBreakdown === "true",
    });
  }

  @Get(":id/executions")
  @HttpCode(HttpStatus.OK)
  async getExecutions(@Param("id") eventId: string) {
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
