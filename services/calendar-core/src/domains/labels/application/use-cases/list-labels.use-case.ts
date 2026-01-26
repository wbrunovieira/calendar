import { Injectable, Inject } from '@nestjs/common';
import { Label } from '../../domain/entities/label.entity';
import { LabelRepository } from '../../infrastructure/repositories/label.repository.interface';

@Injectable()
export class ListLabelsUseCase {
  constructor(
    @Inject('LabelRepository')
    private readonly labelRepository: LabelRepository,
  ) {}

  async execute(calendarId: string): Promise<Label[]> {
    return this.labelRepository.findByCalendarId(calendarId);
  }
}
