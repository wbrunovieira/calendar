import { Injectable } from '@nestjs/common';
import { Label } from '../../domain/entities/label.entity';
import { PrismaLabelRepository } from '../../infrastructure/repositories/label.repository';

@Injectable()
export class ListLabelsUseCase {
  constructor(private readonly labelRepository: PrismaLabelRepository) {}

  async execute(calendarId: string): Promise<Label[]> {
    return this.labelRepository.findByCalendarId(calendarId);
  }
}
