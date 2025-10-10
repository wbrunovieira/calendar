import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { CategoriesModule } from './domains/categories/categories.module';
import { EventsModule } from './domains/events/events.module';

@Module({
  imports: [CategoriesModule, EventsModule],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
