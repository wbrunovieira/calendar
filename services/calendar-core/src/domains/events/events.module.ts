import { Module } from '@nestjs/common';
import { EventsController } from './infrastructure/controllers/events.controller';
import { CreateEventUseCase } from './application/use-cases/create-event.use-case';
import { DeleteEventUseCase } from './application/use-cases/delete-event.use-case';
import { ListEventsUseCase } from './application/use-cases/list-events.use-case';
import { UpdateEventUseCase } from './application/use-cases/update-event.use-case';
import { EventRepository } from './infrastructure/repositories/event.repository';

@Module({
  controllers: [EventsController],
  providers: [
    CreateEventUseCase,
    DeleteEventUseCase,
    ListEventsUseCase,
    UpdateEventUseCase,
    EventRepository,
  ],
  exports: [EventRepository],
})
export class EventsModule {}
