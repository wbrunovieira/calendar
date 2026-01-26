import { Injectable, Inject, NotFoundException } from '@nestjs/common';
import { Label } from '../../domain/entities/label.entity';
import { LabelRepository } from '../../infrastructure/repositories/label.repository.interface';

export interface UpdateLabelInput {
  name?: string;
  color?: string;
}

@Injectable()
export class UpdateLabelUseCase {
  constructor(
    @Inject('LabelRepository')
    private readonly labelRepository: LabelRepository,
  ) {}

  async execute(id: string, input: UpdateLabelInput): Promise<Label> {
    const label = await this.labelRepository.findById(id);

    if (!label) {
      throw new NotFoundException(`Label with id ${id} not found`);
    }

    label.update({
      name: input.name?.trim(),
      color: input.color,
    });

    return this.labelRepository.update(label);
  }
}
