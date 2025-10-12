import { Injectable } from '@nestjs/common';
import { CalendarRepository } from '../../infrastructure/persistence/calendar.repository';
import { Calendar } from '../../domain/entities/calendar.entity';
import { CreateCalendarDto } from '../dto/create-calendar.dto';

@Injectable()
export class CreateCalendarUseCase {
  constructor(private readonly repository: CalendarRepository) {}

  async execute(dto: CreateCalendarDto): Promise<Calendar> {
    const calendar = Calendar.create({
      userId: dto.userId,
      name: dto.name,
      email: dto.email,
      color: dto.color,
      type: dto.type,
      googleCalendarId: dto.googleCalendarId,
      financeProfileId: dto.financeProfileId,
      isActive: dto.isActive ?? true,
    });

    return this.repository.create(calendar);
  }
}
