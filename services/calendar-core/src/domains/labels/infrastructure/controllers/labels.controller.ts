import {
  Controller,
  Post,
  Get,
  Put,
  Delete,
  Body,
  Param,
  Query,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { CreateLabelUseCase } from '../../application/use-cases/create-label.use-case';
import { ListLabelsUseCase } from '../../application/use-cases/list-labels.use-case';
import { UpdateLabelUseCase } from '../../application/use-cases/update-label.use-case';
import { DeleteLabelUseCase } from '../../application/use-cases/delete-label.use-case';
import { CreateLabelDto } from '../dtos/create-label.dto';
import { UpdateLabelDto } from '../dtos/update-label.dto';

@Controller('labels')
export class LabelsController {
  constructor(
    private readonly createLabelUseCase: CreateLabelUseCase,
    private readonly listLabelsUseCase: ListLabelsUseCase,
    private readonly updateLabelUseCase: UpdateLabelUseCase,
    private readonly deleteLabelUseCase: DeleteLabelUseCase,
  ) {}

  @Get()
  @HttpCode(HttpStatus.OK)
  async list(@Query('calendarId') calendarId?: string) {
    if (!calendarId) {
      return [];
    }

    const labels = await this.listLabelsUseCase.execute(calendarId);

    return labels.map((label) => ({
      id: label.id,
      calendarId: label.calendarId,
      name: label.name,
      color: label.color,
      isActive: label.isActive,
      createdAt: label.createdAt,
      updatedAt: label.updatedAt,
    }));
  }

  @Post()
  @HttpCode(HttpStatus.CREATED)
  async create(@Body() createLabelDto: CreateLabelDto) {
    const label = await this.createLabelUseCase.execute(createLabelDto);

    return {
      id: label.id,
      calendarId: label.calendarId,
      name: label.name,
      color: label.color,
      isActive: label.isActive,
      createdAt: label.createdAt,
      updatedAt: label.updatedAt,
    };
  }

  @Put(':id')
  @HttpCode(HttpStatus.OK)
  async update(@Param('id') id: string, @Body() updateLabelDto: UpdateLabelDto) {
    const label = await this.updateLabelUseCase.execute(id, updateLabelDto);

    return {
      id: label.id,
      calendarId: label.calendarId,
      name: label.name,
      color: label.color,
      isActive: label.isActive,
      createdAt: label.createdAt,
      updatedAt: label.updatedAt,
    };
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async delete(@Param('id') id: string) {
    await this.deleteLabelUseCase.execute(id);
  }
}
