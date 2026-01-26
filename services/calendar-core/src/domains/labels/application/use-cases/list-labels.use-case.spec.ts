import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ListLabelsUseCase } from './list-labels.use-case';
import { Label } from '../../domain/entities/label.entity';

describe('ListLabelsUseCase', () => {
  let useCase: ListLabelsUseCase;
  let mockLabelRepository: {
    findByCalendarId: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    mockLabelRepository = {
      findByCalendarId: vi.fn(),
    };
    useCase = new ListLabelsUseCase(mockLabelRepository as any);
  });

  it('should return labels for a calendar', async () => {
    const labels = [
      new Label({
        id: 'label-1',
        calendarId: 'calendar-123',
        name: 'Urgente',
        color: '#FF0000',
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      }),
      new Label({
        id: 'label-2',
        calendarId: 'calendar-123',
        name: 'Importante',
        color: '#00FF00',
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      }),
    ];

    mockLabelRepository.findByCalendarId.mockResolvedValue(labels);

    const result = await useCase.execute('calendar-123');

    expect(result).toHaveLength(2);
    expect(result[0].name).toBe('Urgente');
    expect(result[1].name).toBe('Importante');
    expect(mockLabelRepository.findByCalendarId).toHaveBeenCalledWith('calendar-123');
  });

  it('should return empty array when no labels exist', async () => {
    mockLabelRepository.findByCalendarId.mockResolvedValue([]);

    const result = await useCase.execute('calendar-123');

    expect(result).toHaveLength(0);
  });

  it('should only return active labels', async () => {
    const labels = [
      new Label({
        id: 'label-1',
        calendarId: 'calendar-123',
        name: 'Urgente',
        color: '#FF0000',
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      }),
    ];

    mockLabelRepository.findByCalendarId.mockResolvedValue(labels);

    const result = await useCase.execute('calendar-123');

    expect(result.every((label) => label.isActive)).toBe(true);
  });
});
