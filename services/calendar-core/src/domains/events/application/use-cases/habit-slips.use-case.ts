import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import {
  HabitSlip,
  HabitSlipRepository,
} from '../../infrastructure/repositories/habit-slip.repository';

interface LogHabitSlipInput {
  eventId: string;
  date: string; // YYYY-MM-DD
  trigger?: string;
  note?: string;
}

const localDateKey = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

/** A slip on a QUIT habit breaks the clean-day count for that date. */
@Injectable()
export class LogHabitSlipUseCase {
  constructor(
    private readonly eventRepository: EventRepository,
    private readonly habitSlipRepository: HabitSlipRepository,
  ) {}

  async execute(input: LogHabitSlipInput): Promise<HabitSlip> {
    if (input.date > localDateKey(new Date())) {
      throw new BadRequestException('A slip cannot be logged in the future');
    }

    const event = await this.eventRepository.findById(input.eventId);
    if (!event) {
      throw new NotFoundException(`Event with ID ${input.eventId} not found`);
    }
    if (event.eventType !== 'HABIT' || event.habitKind !== 'QUIT') {
      throw new BadRequestException(
        'Slips can only be logged on habits you are quitting',
      );
    }

    return this.habitSlipRepository.upsert({
      eventId: input.eventId,
      slipDate: new Date(input.date),
      trigger: input.trigger,
      note: input.note,
    });
  }
}

@Injectable()
export class RemoveHabitSlipUseCase {
  constructor(private readonly habitSlipRepository: HabitSlipRepository) {}

  async execute(eventId: string, date: string): Promise<void> {
    await this.habitSlipRepository.delete(eventId, new Date(date));
  }
}
