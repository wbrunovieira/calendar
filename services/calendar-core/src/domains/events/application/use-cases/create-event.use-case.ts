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
      const freq = dto.recurrenceFrequency.toUpperCase() as 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY';
      recurrenceRule = RRuleHelper.toString({
        freq,
        interval: dto.recurrenceInterval || 1,
        byweekday: dto.recurrenceDaysOfWeek,
        bymonthday: dto.recurrenceDayOfMonth,
        until: dto.recurrenceEndDate ? new Date(dto.recurrenceEndDate) : undefined,
      });
    }

    const event = Event.create({
      calendarId: dto.calendarId,
      categoryId: dto.categoryId,
      categoryTypeId: dto.categoryTypeId,
      title: dto.title,
      description: dto.description,
      startTime: dto.startTime,
      endTime: dto.endTime,
      startDate: dto.startDate ? new Date(dto.startDate) : new Date(),
      endDate: dto.endDate ? new Date(dto.endDate) : undefined,
      recurrenceRule,
      recurrenceMasterId: null,
      status: 'CONFIRMED',
    });

    return await this.eventRepository.create(event);
  }
}
