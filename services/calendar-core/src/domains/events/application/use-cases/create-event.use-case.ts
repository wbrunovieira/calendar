import { Injectable, Optional } from '@nestjs/common';
import { Event } from '../../domain/entities/event.entity';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { CreateEventDto } from '../../infrastructure/dtos/create-event.dto';
import { RRuleHelper } from '../../domain/utils/rrule-helper';
import { CalendarRepository } from '@domains/calendars/infrastructure/persistence/calendar.repository';
import { SyncEventToGoogleUseCase } from '@domains/google-calendar/application/use-cases/sync-event-to-google.use-case';

@Injectable()
export class CreateEventUseCase {
  constructor(
    private readonly eventRepository: EventRepository,
    @Optional() private readonly calendarRepository: CalendarRepository,
    @Optional() private readonly syncToGoogle: SyncEventToGoogleUseCase,
  ) {}

  async execute(dto: CreateEventDto): Promise<Event> {
    // Use direct recurrenceRule if provided, otherwise convert legacy format
    let recurrenceRule: string | null = null;

    if (dto.recurrenceRule) {
      // Direct RRULE string from frontend (preferred)
      recurrenceRule = dto.recurrenceRule;
    } else if (dto.isRecurring && dto.recurrenceFrequency) {
      const freq = dto.recurrenceFrequency.toUpperCase() as
        | 'DAILY'
        | 'WEEKLY'
        | 'MONTHLY'
        | 'YEARLY';

      // Parse recurrence end date as local date
      let untilDate: Date | undefined;
      if (dto.recurrenceEndDate) {
        const [y, m, d] = dto.recurrenceEndDate.split('-').map(Number);
        untilDate = new Date(y, m - 1, d);
      }

      const rruleOptions = {
        freq,
        interval: dto.recurrenceInterval || 1,
        byweekday: dto.recurrenceDaysOfWeek,
        bymonthday: dto.recurrenceDayOfMonth,
        until: untilDate,
      };

      recurrenceRule = RRuleHelper.toString(rruleOptions);
    }

    // Parse dates as local dates, not UTC
    let startDate: Date;
    if (dto.startDate) {
      const [y, m, d] = dto.startDate.split('-').map(Number);
      startDate = new Date(y, m - 1, d);
    } else {
      startDate = new Date();
      startDate.setHours(0, 0, 0, 0);
    }

    let endDate: Date | undefined;
    if (dto.endDate) {
      const [y, m, d] = dto.endDate.split('-').map(Number);
      endDate = new Date(y, m - 1, d);
    }

    // Parse dueDate for TODOs
    let dueDate: Date | null = null;
    if (dto.dueDate) {
      const [y, m, d] = dto.dueDate.split('-').map(Number);
      dueDate = new Date(y, m - 1, d);
    }

    // Determine recurrence type
    let recurrenceType: string | null = null;
    let weeklyTargetCount: number | null = null;
    let weeklyPreferredDays: string[] = [];

    if (dto.recurrenceType === 'FLEXIBLE' && dto.weeklyTargetCount) {
      // Flexible weekly habit
      recurrenceType = 'FLEXIBLE';
      weeklyTargetCount = dto.weeklyTargetCount;
      weeklyPreferredDays = dto.weeklyPreferredDays || [];
      // For flexible habits, we use a simple FREQ=WEEKLY without BYDAY
      // The actual days are flexible based on weeklyTargetCount
      if (!recurrenceRule) {
        recurrenceRule = 'FREQ=WEEKLY';
      }
    } else if (recurrenceRule) {
      // Fixed recurrence (traditional)
      recurrenceType = 'FIXED';
    }

    const event = Event.create({
      calendarId: dto.calendarId,
      categoryId: dto.categoryId,
      categoryTypeId: dto.categoryTypeId,
      labelId: dto.labelId || null,
      title: dto.title,
      description: dto.description,
      startTime: dto.startTime,
      endTime: dto.endTime,
      startDate,
      endDate,
      recurrenceRule,
      recurrenceMasterId: null,
      status: 'CONFIRMED',
      eventType: dto.eventType || 'EVENT',
      priority: dto.priority || null,
      dueDate,
      reminderDaysBefore: dto.reminderDaysBefore || null,
      recurrenceType,
      weeklyTargetCount,
      weeklyPreferredDays,
    });

    const created = await this.eventRepository.create(event, dto.reminders);

    if (this.syncToGoogle && this.calendarRepository && !dto.syncSource) {
      const calendar = await this.calendarRepository.findById(dto.calendarId);
      if (calendar) {
        const googleEventId = await this.syncToGoogle.onCreate(created, calendar);
        if (googleEventId) {
          await this.eventRepository.updateGoogleEventId(created.id, googleEventId);
          created.googleEventId = googleEventId;
        }
      }
    }

    return created;
  }
}
