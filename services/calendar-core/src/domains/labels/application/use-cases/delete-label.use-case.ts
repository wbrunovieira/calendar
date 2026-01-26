import { Injectable, Inject, NotFoundException } from '@nestjs/common';
import { LabelRepository } from '../../infrastructure/repositories/label.repository.interface';

@Injectable()
export class DeleteLabelUseCase {
  constructor(
    @Inject('LabelRepository')
    private readonly labelRepository: LabelRepository,
  ) {}

  async execute(id: string): Promise<void> {
    const label = await this.labelRepository.findById(id);

    if (!label) {
      throw new NotFoundException(`Label with id ${id} not found`);
    }

    await this.labelRepository.delete(id);
  }
}
