import { Module, NestModule, MiddlewareConsumer } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { CategoriesModule } from './domains/categories/categories.module';
import { EventsModule } from './domains/events/events.module';
import { LoggerMiddleware } from './common/middleware/logger.middleware';

@Module({
  imports: [CategoriesModule, EventsModule],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule implements NestModule {
  configure(consumer: MiddlewareConsumer) {
    consumer.apply(LoggerMiddleware).forRoutes('*');
  }
}
