import { Injectable, NotFoundException } from '@nestjs/common';
import { PrismaLabelRepository } from '../../infrastructure/repositories/label.repository';

@Injectable()
export class DeleteLabelUseCase {
  constructor(private readonly labelRepository: PrismaLabelRepository) {}

  async execute(id: string): Promise<void> {
    const label = await this.labelRepository.findById(id);

    if (!label) {
      throw new NotFoundException(`Label with id ${id} not found`);
    }

    await this.labelRepository.delete(id);
  }
}
