import { describe, it, expect, beforeEach, vi } from 'vitest';
import { BadRequestException, NotFoundException } from '@nestjs/common';
import { LogHabitAmountUseCase } from './log-habit-amount.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';

describe('LogHabitAmountUseCase', () => {
  let useCase: LogHabitAmountUseCase;
  let eventRepository: Partial<EventRepository>;
  let executionRepository: Partial<EventExecutionRepository>;
  const date = new Date('2026-10-12');

  const habit = (overrides: Record<string, unknown> = {}) => ({
    id: 'habit-1',
    eventType: 'HABIT',
    habitKind: 'BUILD',
    dailyTarget: 20,
    ...overrides,
  });

  beforeEach(() => {
    eventRepository = { findById: vi.fn().mockResolvedValue(habit()) };
    executionRepository = {
      upsert: vi
        .fn()
        .mockImplementation(
          async (eventId, executionDate, completed, notes, value) => ({
            eventId,
            executionDate,
            completed,
            value,
          }),
        ),
    };
    useCase = new LogHabitAmountUseCase(
      eventRepository as EventRepository,
      executionRepository as EventExecutionRepository,
    );
  });

  it('marks the day done when the amount reaches the daily target', async () => {
    const result = await useCase.execute({
      eventId: 'habit-1',
      executionDate: date,
      value: 20,
    });

    expect(executionRepository.upsert).toHaveBeenCalledWith(
      'habit-1',
      date,
      true,
      undefined,
      20,
    );
    expect(result.completed).toBe(true);
  });

  it('keeps the day open while the amount is below the target', async () => {
    const result = await useCase.execute({
      eventId: 'habit-1',
      executionDate: date,
      value: 15,
    });

    expect(executionRepository.upsert).toHaveBeenCalledWith(
      'habit-1',
      date,
      false,
      undefined,
      15,
    );
    expect(result.completed).toBe(false);
  });

  it('rejects a negative amount', async () => {
    await expect(
      useCase.execute({ eventId: 'habit-1', executionDate: date, value: -1 }),
    ).rejects.toBeInstanceOf(BadRequestException);
    expect(executionRepository.upsert).not.toHaveBeenCalled();
  });

  it('rejects a habit without a daily target', async () => {
    vi.mocked(eventRepository.findById!).mockResolvedValue(
      habit({ dailyTarget: null }) as never,
    );

    await expect(
      useCase.execute({ eventId: 'habit-1', executionDate: date, value: 3 }),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('rejects events that are not habits', async () => {
    vi.mocked(eventRepository.findById!).mockResolvedValue(
      habit({ eventType: 'TODO' }) as never,
    );

    await expect(
      useCase.execute({ eventId: 'habit-1', executionDate: date, value: 3 }),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('throws NotFound for an unknown event', async () => {
    vi.mocked(eventRepository.findById!).mockResolvedValue(null);

    await expect(
      useCase.execute({ eventId: 'nope', executionDate: date, value: 3 }),
    ).rejects.toBeInstanceOf(NotFoundException);
  });
});
