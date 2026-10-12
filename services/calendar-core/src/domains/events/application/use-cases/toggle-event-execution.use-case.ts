import { Injectable, Optional } from '@nestjs/common';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
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
    @Optional() private readonly eventRepository?: EventRepository,
  ) {}

  async execute(input: ToggleEventExecutionInput): Promise<EventExecution> {
    // Ticking a habit with a daily amount from a plain checkbox (e.g. the day view)
    // must keep the amount consistent: done means the full target, undone means zero.
    const event = await this.eventRepository?.findById(input.eventId);
    const target = event?.dailyTarget as number | null | undefined;
    const value = target ? (input.completed ? target : 0) : undefined;

    const execution = await this.eventExecutionRepository.upsert(
      input.eventId,
      input.executionDate,
      input.completed,
      input.notes,
      value,
    );

    return execution;
  }
}
