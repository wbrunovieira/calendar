import { Injectable } from '@nestjs/common';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { EventExecution } from '../../domain/entities/event-execution.entity';

@Injectable()
export class GetEventExecutionsUseCase {
  constructor(
    private readonly eventExecutionRepository: EventExecutionRepository,
  ) {}

  async execute(eventId: string): Promise<EventExecution[]> {
    return await this.eventExecutionRepository.findByEventId(eventId);
  }

  async getByDateRange(
    eventId: string,
    startDate: Date,
    endDate: Date,
  ): Promise<EventExecution[]> {
    return await this.eventExecutionRepository.findByDateRange(
      eventId,
      startDate,
      endDate,
    );
  }
}
