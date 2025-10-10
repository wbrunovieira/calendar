import { Injectable, NotFoundException } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';

@Injectable()
export class DeleteEventUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(id: string): Promise<void> {
    const event = await this.eventRepository.findById(id);

    if (!event) {
      throw new NotFoundException(`Event with ID ${id} not found`);
    }

    await this.eventRepository.delete(id);
  }
}
