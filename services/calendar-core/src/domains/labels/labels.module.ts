import { Module } from '@nestjs/common';
import { LabelsController } from './infrastructure/controllers/labels.controller';
import { PrismaLabelRepository } from './infrastructure/repositories/label.repository';
import { CreateLabelUseCase } from './application/use-cases/create-label.use-case';
import { ListLabelsUseCase } from './application/use-cases/list-labels.use-case';
import { UpdateLabelUseCase } from './application/use-cases/update-label.use-case';
import { DeleteLabelUseCase } from './application/use-cases/delete-label.use-case';

@Module({
  controllers: [LabelsController],
  providers: [
    PrismaLabelRepository,
    CreateLabelUseCase,
    ListLabelsUseCase,
    UpdateLabelUseCase,
    DeleteLabelUseCase,
  ],
  exports: [PrismaLabelRepository],
})
export class LabelsModule {}
