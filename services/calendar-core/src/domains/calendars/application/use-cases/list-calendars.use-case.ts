import { Injectable } from '@nestjs/common';
import { CalendarRepository } from '../../infrastructure/persistence/calendar.repository';
import { Calendar } from '../../domain/entities/calendar.entity';

@Injectable()
export class ListCalendarsUseCase {
  constructor(private readonly repository: CalendarRepository) {}

  async execute(userId?: string): Promise<Calendar[]> {
    return this.repository.findAll(userId);
  }
}
