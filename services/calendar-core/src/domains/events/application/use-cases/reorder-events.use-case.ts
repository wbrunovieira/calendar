import { Injectable } from "@nestjs/common";
import { EventRepository } from "../../infrastructure/repositories/event.repository";

interface ReorderEventsInput {
  orderedIds: string[];
}

@Injectable()
export class ReorderEventsUseCase {
  constructor(private readonly eventRepository: EventRepository) {}

  async execute(input: ReorderEventsInput): Promise<void> {
    const updates = input.orderedIds.map((id, index) => ({
      id,
      displayOrder: index,
    }));

    await this.eventRepository.updateDisplayOrder(updates);
  }
}
