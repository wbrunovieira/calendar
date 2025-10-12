import { Injectable, NotFoundException } from '@nestjs/common';
import { CalendarRepository } from '../../infrastructure/persistence/calendar.repository';
import { Calendar } from '../../domain/entities/calendar.entity';
import { UpdateCalendarDto } from '../dto/update-calendar.dto';

@Injectable()
export class UpdateCalendarUseCase {
  constructor(private readonly repository: CalendarRepository) {}

  async execute(id: string, dto: UpdateCalendarDto): Promise<Calendar> {
    const calendar = await this.repository.findById(id);
    if (!calendar) {
      throw new NotFoundException(`Calendar with id ${id} not found`);
    }

    return this.repository.update(id, dto);
  }
}
