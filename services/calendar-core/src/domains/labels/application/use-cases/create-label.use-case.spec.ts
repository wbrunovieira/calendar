import { describe, it, expect, beforeEach, vi } from 'vitest';
import { CreateLabelUseCase } from './create-label.use-case';
import { Label } from '../../domain/entities/label.entity';

describe('CreateLabelUseCase', () => {
  let useCase: CreateLabelUseCase;
  let mockLabelRepository: {
    create: ReturnType<typeof vi.fn>;
    findByCalendarId: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    mockLabelRepository = {
      create: vi.fn(),
      findByCalendarId: vi.fn(),
    };
    useCase = new CreateLabelUseCase(mockLabelRepository as any);
  });

  it('should create a label successfully', async () => {
    const input = {
      calendarId: 'calendar-123',
      name: 'Urgente',
      color: '#FF0000',
    };

    mockLabelRepository.create.mockResolvedValue(
      new Label({
        id: 'label-123',
        calendarId: input.calendarId,
        name: input.name,
        color: input.color,
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      }),
    );

    const result = await useCase.execute(input);

    expect(result.name).toBe('Urgente');
    expect(result.color).toBe('#FF0000');
    expect(result.calendarId).toBe('calendar-123');
    expect(mockLabelRepository.create).toHaveBeenCalledOnce();
  });

  it('should throw error when name is empty', async () => {
    const input = {
      calendarId: 'calendar-123',
      name: '',
      color: '#FF0000',
    };

    await expect(useCase.execute(input)).rejects.toThrow('Label name is required');
  });

  it('should throw error when color is invalid', async () => {
    const input = {
      calendarId: 'calendar-123',
      name: 'Urgente',
      color: 'invalid',
    };

    await expect(useCase.execute(input)).rejects.toThrow('Label color must be a valid hex color');
  });

  it('should throw error when calendarId is empty', async () => {
    const input = {
      calendarId: '',
      name: 'Urgente',
      color: '#FF0000',
    };

    await expect(useCase.execute(input)).rejects.toThrow('Calendar ID is required');
  });

  it('should trim the label name', async () => {
    const input = {
      calendarId: 'calendar-123',
      name: '  Urgente  ',
      color: '#FF0000',
    };

    mockLabelRepository.create.mockImplementation((label: Label) => {
      return Promise.resolve(label);
    });

    const result = await useCase.execute(input);

    expect(result.name).toBe('Urgente');
  });
});
