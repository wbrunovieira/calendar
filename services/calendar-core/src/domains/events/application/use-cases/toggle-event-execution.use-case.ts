import { Injectable } from '@nestjs/common';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { EventExecution } from '../../domain/entities/event-execution.entity';

interface ToggleEventExecutionInput {
  eventId: string;
  executionDate: Date;
  completed: boolean;
  notes?: string;
}

@Injectable()
export class ToggleEventExecutionUseCase {
  constructor(
    private readonly eventExecutionRepository: EventExecutionRepository,
  ) {}

  async execute(input: ToggleEventExecutionInput): Promise<EventExecution> {
    const execution = await this.eventExecutionRepository.upsert(
      input.eventId,
      input.executionDate,
      input.completed,
      input.notes,
    );

    return execution;
  }
}
