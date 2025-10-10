import { Injectable } from '@nestjs/common';
import { Event } from '../../domain/entities/event.entity';
import { EventRepository } from '../../infrastructure/repositories/event.repository';

export interface ListEventsFilters {
  calendarId?: string;
  categoryId?: string;
  search?: string;
}

@Injectable()
export class ListEventsUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(filters?: ListEventsFilters): Promise<Event[]> {
    return await this.eventRepository.findAll(filters);
  }
}
