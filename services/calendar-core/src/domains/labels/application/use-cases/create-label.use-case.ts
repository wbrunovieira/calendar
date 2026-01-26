import { Injectable, Inject } from '@nestjs/common';
import { Label } from '../../domain/entities/label.entity';
import { LabelRepository } from '../../infrastructure/repositories/label.repository.interface';
import { randomUUID } from 'crypto';

export interface CreateLabelInput {
  calendarId: string;
  name: string;
  color: string;
}

@Injectable()
export class CreateLabelUseCase {
  constructor(
    @Inject('LabelRepository')
    private readonly labelRepository: LabelRepository,
  ) {}

  async execute(input: CreateLabelInput): Promise<Label> {
    const now = new Date();

    const label = new Label({
      id: randomUUID(),
      calendarId: input.calendarId,
      name: input.name.trim(),
      color: input.color,
      isActive: true,
      createdAt: now,
      updatedAt: now,
    });

    return this.labelRepository.create(label);
  }
}
