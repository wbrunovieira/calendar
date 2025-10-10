import { Injectable, NotFoundException } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { UpdateEventDto } from '../../infrastructure/dtos/update-event.dto';

@Injectable()
export class UpdateEventUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(id: string, dto: UpdateEventDto) {
    const existingEvent = await this.eventRepository.findById(id);

    if (!existingEvent) {
      throw new NotFoundException(`Event with ID ${id} not found`);
    }

    const updateData: any = {};

    if (dto.calendarId !== undefined) updateData.calendarId = dto.calendarId;
    if (dto.categoryId !== undefined) updateData.categoryId = dto.categoryId;
    if (dto.title !== undefined) updateData.title = dto.title;
    if (dto.description !== undefined) updateData.description = dto.description;
    if (dto.startTime !== undefined) updateData.startTime = dto.startTime;
    if (dto.endTime !== undefined) updateData.endTime = dto.endTime;
    if (dto.startDate !== undefined) updateData.startDate = new Date(dto.startDate);
    if (dto.endDate !== undefined) updateData.endDate = dto.endDate ? new Date(dto.endDate) : null;
    if (dto.isRecurring !== undefined) updateData.isRecurring = dto.isRecurring;
    if (dto.recurrenceFrequency !== undefined) updateData.recurrenceFrequency = dto.recurrenceFrequency;
    if (dto.recurrenceInterval !== undefined) updateData.recurrenceInterval = dto.recurrenceInterval;
    if (dto.recurrenceDaysOfWeek !== undefined) updateData.recurrenceDaysOfWeek = dto.recurrenceDaysOfWeek;
    if (dto.recurrenceEndDate !== undefined) updateData.recurrenceEndDate = dto.recurrenceEndDate ? new Date(dto.recurrenceEndDate) : null;

    const updatedEvent = await this.eventRepository.update(id, updateData);
    return updatedEvent;
  }
}
