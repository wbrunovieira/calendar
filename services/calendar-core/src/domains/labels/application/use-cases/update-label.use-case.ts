import { Injectable, NotFoundException } from '@nestjs/common';
import { Label } from '../../domain/entities/label.entity';
import { PrismaLabelRepository } from '../../infrastructure/repositories/label.repository';

export interface UpdateLabelInput {
  name?: string;
  color?: string;
}

@Injectable()
export class UpdateLabelUseCase {
  constructor(private readonly labelRepository: PrismaLabelRepository) {}

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
