import { describe, it, expect, beforeEach, vi } from 'vitest';
import { UpdateLabelUseCase } from './update-label.use-case';
import { Label } from '../../domain/entities/label.entity';
import { NotFoundException } from '@nestjs/common';

describe('UpdateLabelUseCase', () => {
  let useCase: UpdateLabelUseCase;
  let mockLabelRepository: {
    findById: ReturnType<typeof vi.fn>;
    update: ReturnType<typeof vi.fn>;
  };

  const existingLabel = new Label({
    id: 'label-123',
    calendarId: 'calendar-456',
    name: 'Urgente',
    color: '#FF0000',
    isActive: true,
    createdAt: new Date('2024-01-15'),
    updatedAt: new Date('2024-01-15'),
  });

  beforeEach(() => {
    mockLabelRepository = {
      findById: vi.fn(),
      update: vi.fn(),
    };
    useCase = new UpdateLabelUseCase(mockLabelRepository as any);
  });

  it('should update label name', async () => {
    mockLabelRepository.findById.mockResolvedValue(existingLabel);
    mockLabelRepository.update.mockImplementation((label: Label) => Promise.resolve(label));

    const result = await useCase.execute('label-123', { name: 'Importante' });

    expect(result.name).toBe('Importante');
    expect(mockLabelRepository.update).toHaveBeenCalledOnce();
  });

  it('should update label color', async () => {
    mockLabelRepository.findById.mockResolvedValue(existingLabel);
    mockLabelRepository.update.mockImplementation((label: Label) => Promise.resolve(label));

    const result = await useCase.execute('label-123', { color: '#00FF00' });

    expect(result.color).toBe('#00FF00');
  });

  it('should update both name and color', async () => {
    mockLabelRepository.findById.mockResolvedValue(existingLabel);
    mockLabelRepository.update.mockImplementation((label: Label) => Promise.resolve(label));

    const result = await useCase.execute('label-123', { name: 'Importante', color: '#00FF00' });

    expect(result.name).toBe('Importante');
    expect(result.color).toBe('#00FF00');
  });

  it('should throw NotFoundException when label does not exist', async () => {
    mockLabelRepository.findById.mockResolvedValue(null);

    await expect(useCase.execute('non-existent', { name: 'Test' })).rejects.toThrow(
      NotFoundException,
    );
  });

  it('should throw error when updating with invalid color', async () => {
    mockLabelRepository.findById.mockResolvedValue(existingLabel);

    await expect(useCase.execute('label-123', { color: 'invalid' })).rejects.toThrow(
      'Label color must be a valid hex color',
    );
  });

  it('should throw error when updating with empty name', async () => {
    mockLabelRepository.findById.mockResolvedValue(existingLabel);

    await expect(useCase.execute('label-123', { name: '' })).rejects.toThrow(
      'Label name is required',
    );
  });
});
