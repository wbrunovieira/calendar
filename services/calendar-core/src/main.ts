import { NestFactory } from '@nestjs/core';
import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  // OpenAPI docs — UI at /docs, spec JSON at /docs-json. Kept public (the ApiTokenGuard exempts
  // the /docs paths) so agents can read the spec by link; calling the endpoints still needs the token.
  const swaggerConfig = new DocumentBuilder()
    .setTitle('WB Calendar API')
    .setDescription(
      'Events, habits, todos, reminders and calendars. Events/habits/todos are the same `/events` ' +
        'resource, distinguished by `eventType`. Most routes require an API token — click Authorize ' +
        'and paste it as a Bearer token (same value as the backend API_TOKEN env var).',
    )
    .setVersion('1.0')
    .addBearerAuth({
      type: 'http',
      scheme: 'bearer',
      description: 'API token — same value as the calendar-core API_TOKEN env var',
    })
    .build();
  SwaggerModule.setup('docs', app, SwaggerModule.createDocument(app, swaggerConfig));

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
