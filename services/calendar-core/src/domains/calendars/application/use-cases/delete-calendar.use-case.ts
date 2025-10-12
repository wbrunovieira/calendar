import { Injectable, NotFoundException } from '@nestjs/common';
import { CalendarRepository } from '../../infrastructure/persistence/calendar.repository';

@Injectable()
export class DeleteCalendarUseCase {
  constructor(private readonly repository: CalendarRepository) {}

  async execute(id: string): Promise<void> {
    const calendar = await this.repository.findById(id);
    if (!calendar) {
      throw new NotFoundException(`Calendar with id ${id} not found`);
    }

    await this.repository.delete(id);
  }
}
