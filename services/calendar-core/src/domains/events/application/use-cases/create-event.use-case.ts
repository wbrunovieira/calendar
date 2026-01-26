import { Injectable } from '@nestjs/common';
import { Event } from '../../domain/entities/event.entity';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { CreateEventDto } from '../../infrastructure/dtos/create-event.dto';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

@Injectable()
export class CreateEventUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(dto: CreateEventDto): Promise<Event> {
    // Converter formato antigo para RRULE se for recorrente
    let recurrenceRule: string | null = null;
    if (dto.isRecurring && dto.recurrenceFrequency) {
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
    });

    return await this.eventRepository.create(event);
  }
}
