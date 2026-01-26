import { Module } from '@nestjs/common';
import { EventsController } from './infrastructure/controllers/events.controller';
import { CreateEventUseCase } from './application/use-cases/create-event.use-case';
import { DeleteEventUseCase } from './application/use-cases/delete-event.use-case';
import { ListEventsUseCase } from './application/use-cases/list-events.use-case';
import { UpdateEventUseCase } from './application/use-cases/update-event.use-case';
import { ToggleEventExecutionUseCase } from './application/use-cases/toggle-event-execution.use-case';
import { GetEventExecutionsUseCase } from './application/use-cases/get-event-executions.use-case';
import { GetEventsStatsUseCase } from './application/use-cases/get-events-stats.use-case';
import { GetHabitsStatsUseCase } from './application/use-cases/get-habits-stats.use-case';
import { ReorderEventsUseCase } from './application/use-cases/reorder-events.use-case';
import { EventRepository } from './infrastructure/repositories/event.repository';
import { EventExecutionRepository } from './infrastructure/repositories/event-execution.repository';

@Module({
  controllers: [EventsController],
  providers: [
    CreateEventUseCase,
    DeleteEventUseCase,
    ListEventsUseCase,
    UpdateEventUseCase,
    ToggleEventExecutionUseCase,
    GetEventExecutionsUseCase,
    GetEventsStatsUseCase,
    GetHabitsStatsUseCase,
    ReorderEventsUseCase,
    EventRepository,
    EventExecutionRepository,
  ],
  exports: [EventRepository, EventExecutionRepository],
})
export class EventsModule {}
