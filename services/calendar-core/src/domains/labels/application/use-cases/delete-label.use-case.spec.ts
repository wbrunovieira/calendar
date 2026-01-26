import { describe, it, expect, beforeEach, vi } from 'vitest';
import { DeleteLabelUseCase } from './delete-label.use-case';
import { Label } from '../../domain/entities/label.entity';
import { NotFoundException } from '@nestjs/common';

describe('DeleteLabelUseCase', () => {
  let useCase: DeleteLabelUseCase;
  let mockLabelRepository: {
    findById: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
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
      delete: vi.fn(),
    };
    useCase = new DeleteLabelUseCase(mockLabelRepository as any);
  });

  it('should delete an existing label', async () => {
    mockLabelRepository.findById.mockResolvedValue(existingLabel);
    mockLabelRepository.delete.mockResolvedValue(undefined);

    await useCase.execute('label-123');

    expect(mockLabelRepository.delete).toHaveBeenCalledWith('label-123');
  });

  it('should throw NotFoundException when label does not exist', async () => {
    mockLabelRepository.findById.mockResolvedValue(null);

    await expect(useCase.execute('non-existent')).rejects.toThrow(NotFoundException);
  });
});
