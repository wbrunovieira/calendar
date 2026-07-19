import { Controller, Get } from '@nestjs/common';
import { AppService } from './app.service';
import { Public } from './common/auth/public.decorator';

@Controller()
export class AppController {
  constructor(private readonly appService: AppService) {}

  // Health/liveness — must stay reachable without a token (used by the CD health check).
  @Public()
  @Get()
  getHello(): string {
    return this.appService.getHello();
  }
}
