import { describe, it, expect, beforeEach, vi } from 'vitest';
import { BadRequestException, NotFoundException } from '@nestjs/common';
import {
  LogHabitSlipUseCase,
  RemoveHabitSlipUseCase,
} from './habit-slips.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { HabitSlipRepository } from '../../infrastructure/repositories/habit-slip.repository';

describe('Habit slips', () => {
  let eventRepository: Partial<EventRepository>;
  let slipRepository: Partial<HabitSlipRepository>;
  const today = new Date('2026-10-12T15:00:00Z');

  const quitHabit = (overrides: Record<string, unknown> = {}) => ({
    id: 'habit-1',
    eventType: 'HABIT',
    habitKind: 'QUIT',
    ...overrides,
  });

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(today);
    eventRepository = { findById: vi.fn().mockResolvedValue(quitHabit()) };
    slipRepository = {
      upsert: vi
        .fn()
        .mockImplementation(async (input) => ({ id: 'slip-1', ...input })),
      delete: vi.fn().mockResolvedValue(undefined),
    };
  });

  describe('LogHabitSlipUseCase', () => {
    const useCase = () =>
      new LogHabitSlipUseCase(
        eventRepository as EventRepository,
        slipRepository as HabitSlipRepository,
      );

    it('records the slip with its trigger and note', async () => {
      const slip = await useCase().execute({
        eventId: 'habit-1',
        date: '2026-10-12',
        trigger: 'Stress',
        note: 'deadline',
      });

      expect(slipRepository.upsert).toHaveBeenCalledWith({
        eventId: 'habit-1',
        slipDate: new Date('2026-10-12'),
        trigger: 'Stress',
        note: 'deadline',
      });
      expect(slip.id).toBe('slip-1');
    });

    it('only accepts QUIT habits', async () => {
      vi.mocked(eventRepository.findById!).mockResolvedValue(
        quitHabit({ habitKind: 'BUILD' }) as never,
      );

      await expect(
        useCase().execute({ eventId: 'habit-1', date: '2026-10-12' }),
      ).rejects.toBeInstanceOf(BadRequestException);
      expect(slipRepository.upsert).not.toHaveBeenCalled();
    });

    it('refuses a slip in the future', async () => {
      await expect(
        useCase().execute({ eventId: 'habit-1', date: '2026-10-14' }),
      ).rejects.toBeInstanceOf(BadRequestException);
    });

    it('throws NotFound for an unknown habit', async () => {
      vi.mocked(eventRepository.findById!).mockResolvedValue(null);

      await expect(
        useCase().execute({ eventId: 'x', date: '2026-10-12' }),
      ).rejects.toBeInstanceOf(NotFoundException);
    });
  });

  describe('RemoveHabitSlipUseCase', () => {
    it('deletes the slip for that day', async () => {
      const useCase = new RemoveHabitSlipUseCase(
        slipRepository as HabitSlipRepository,
      );

      await useCase.execute('habit-1', '2026-10-12');

      expect(slipRepository.delete).toHaveBeenCalledWith(
        'habit-1',
        new Date('2026-10-12'),
      );
    });
  });
});
