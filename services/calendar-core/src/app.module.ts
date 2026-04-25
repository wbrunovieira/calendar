import { Module, NestModule, MiddlewareConsumer } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { CalendarsModule } from './domains/calendars/calendars.module';
import { CategoriesModule } from './domains/categories/categories.module';
import { CategoryTypesModule } from './domains/category-types/category-types.module';
import { EventsModule } from './domains/events/events.module';
import { LabelsModule } from './domains/labels/labels.module';
import { GoogleCalendarModule } from './domains/google-calendar/google-calendar.module';
import { LoggerMiddleware } from './common/middleware/logger.middleware';

@Module({
  imports: [
    CalendarsModule,
    CategoriesModule,
    CategoryTypesModule,
    EventsModule,
    LabelsModule,
    GoogleCalendarModule,
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule implements NestModule {
  configure(consumer: MiddlewareConsumer) {
    consumer.apply(LoggerMiddleware).forRoutes('*');
  }
}
