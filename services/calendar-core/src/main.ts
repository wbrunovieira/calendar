import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  // Habilitar CORS
  app.enableCors({
    origin: [
      'http://localhost:3000',
      'http://localhost:3001',
      'http://localhost:3002',
      'http://localhost:3003',
      'http://localhost:3004',
      'http://localhost:3005',
      'http://localhost:3006',
      'http://localhost:3007',
      'http://localhost:3008',
      'http://localhost:3009',
      'http://localhost:3010',
      'http://192.168.0.17:3000',
      'http://192.168.0.17:3003',
      'https://calendar.wbdigitalsolutions.com',
      'https://finances.wbdigitalsolutions.com',
      'https://health.wbdigitalsolutions.com',
      'https://calendar.app.localhost',
      'https://finances.app.localhost',
      'https://health.app.localhost',
    ],
    credentials: true,
  });

  await app.listen(process.env.PORT ?? 3334);
}
bootstrap();
