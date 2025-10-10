import { Injectable } from '@nestjs/common';
import { Event } from '../../domain/entities/event.entity';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { CreateEventDto } from '../../infrastructure/dtos/create-event.dto';

@Injectable()
export class CreateEventUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(dto: CreateEventDto): Promise<Event> {
    const event = Event.create({
      calendarId: dto.calendarId,
      categoryId: dto.categoryId,
      title: dto.title,
      description: dto.description,
      startTime: dto.startTime,
      endTime: dto.endTime,
      startDate: dto.startDate ? new Date(dto.startDate) : undefined,
      endDate: dto.endDate ? new Date(dto.endDate) : undefined,
      isRecurring: dto.isRecurring,
      recurrenceFrequency: dto.recurrenceFrequency,
      recurrenceInterval: dto.recurrenceInterval,
      recurrenceDaysOfWeek: dto.recurrenceDaysOfWeek,
      recurrenceDayOfMonth: dto.recurrenceDayOfMonth,
      recurrenceWeekOfMonth: dto.recurrenceWeekOfMonth,
      recurrenceEndDate: dto.recurrenceEndDate ? new Date(dto.recurrenceEndDate) : undefined,
    });

    return await this.eventRepository.create(event);
  }
}
