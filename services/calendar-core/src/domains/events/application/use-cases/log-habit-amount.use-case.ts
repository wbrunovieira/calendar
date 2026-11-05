import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { EventExecution } from '../../domain/entities/event-execution.entity';

interface LogHabitAmountInput {
  eventId: string;
  executionDate: Date;
  value: number;
}

/**
 * Records how much of a habit's daily amount was done (e.g. 15 of 20 pages).
 * The day only counts as completed once the value reaches the daily target.
 */
@Injectable()
export class LogHabitAmountUseCase {
  constructor(
    private readonly eventRepository: EventRepository,
    private readonly eventExecutionRepository: EventExecutionRepository,
  ) {}

  async execute(input: LogHabitAmountInput): Promise<EventExecution> {
    if (input.value < 0) {
      throw new BadRequestException('Amount cannot be negative');
    }

    const event = await this.eventRepository.findById(input.eventId);
    if (!event) {
      throw new NotFoundException(`Event with ID ${input.eventId} not found`);
    }
    if (event.eventType !== 'HABIT' || !event.dailyTarget) {
      throw new BadRequestException(
        'Only habits with a daily target accept an amount',
      );
    }

    return this.eventExecutionRepository.upsert(
      input.eventId,
      input.executionDate,
      input.value >= event.dailyTarget,
      undefined,
      input.value,
    );
  }
}
