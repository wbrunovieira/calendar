import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ToggleEventExecutionUseCase } from './toggle-event-execution.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';

describe('ToggleEventExecutionUseCase', () => {
  let eventRepository: Partial<EventRepository>;
  let executionRepository: Partial<EventExecutionRepository>;
  let useCase: ToggleEventExecutionUseCase;
  const date = new Date('2026-10-12');

  beforeEach(() => {
    eventRepository = {
      findById: vi.fn().mockResolvedValue({ id: 'h', eventType: 'HABIT', dailyTarget: 20 }),
    };
    executionRepository = { upsert: vi.fn().mockResolvedValue({}) };
    useCase = new ToggleEventExecutionUseCase(
      executionRepository as EventExecutionRepository,
      eventRepository as EventRepository,
    );
  });

  it('fills an amount habit to its target when ticked', async () => {
    await useCase.execute({ eventId: 'h', executionDate: date, completed: true });

    expect(executionRepository.upsert).toHaveBeenCalledWith('h', date, true, undefined, 20);
  });

  it('clears the amount when an amount habit is unticked', async () => {
    await useCase.execute({ eventId: 'h', executionDate: date, completed: false });

    expect(executionRepository.upsert).toHaveBeenCalledWith('h', date, false, undefined, 0);
  });

  it('leaves the value alone for events without a daily target', async () => {
    vi.mocked(eventRepository.findById!).mockResolvedValue({ id: 'h', eventType: 'TODO' } as never);

    await useCase.execute({ eventId: 'h', executionDate: date, completed: true, notes: 'n' });

    expect(executionRepository.upsert).toHaveBeenCalledWith('h', date, true, 'n', undefined);
  });
});
