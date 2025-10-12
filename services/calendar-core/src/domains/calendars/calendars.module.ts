import { Module } from '@nestjs/common';
import { CalendarsController } from './infrastructure/http/calendars.controller';
import { CalendarRepository } from './infrastructure/persistence/calendar.repository';
import { ListCalendarsUseCase } from './application/use-cases/list-calendars.use-case';
import { CreateCalendarUseCase } from './application/use-cases/create-calendar.use-case';
import { UpdateCalendarUseCase } from './application/use-cases/update-calendar.use-case';
import { DeleteCalendarUseCase } from './application/use-cases/delete-calendar.use-case';

@Module({
  controllers: [CalendarsController],
  providers: [
    CalendarRepository,
    ListCalendarsUseCase,
    CreateCalendarUseCase,
    UpdateCalendarUseCase,
    DeleteCalendarUseCase,
  ],
  exports: [CalendarRepository],
})
export class CalendarsModule {}
