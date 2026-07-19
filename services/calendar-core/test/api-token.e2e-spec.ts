import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from 'vitest';
import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { AppModule } from './../src/app.module';

/**
 * Proves the token guard is actually wired as a global APP_GUARD and that @Public survives the
 * real HTTP pipeline — a unit test of the guard can't catch a missing registration.
 */
describe('API token auth (e2e)', () => {
  let app: INestApplication;
  const TOKEN = 'e2e-api-token';

  beforeAll(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule],
    }).compile();
    app = moduleFixture.createNestApplication();
    await app.init();
  });

  afterAll(async () => {
    await app.close();
  });

  afterEach(() => {
    delete process.env.API_TOKEN;
  });

  describe('when API_TOKEN is set', () => {
    beforeEach(() => {
      process.env.API_TOKEN = TOKEN;
    });

    it('blocks a protected route with no token (401)', () =>
      request(app.getHttpServer()).get('/events').expect(401));

    it('allows a protected route with the correct Bearer token', () =>
      request(app.getHttpServer())
        .get('/events')
        .set('Authorization', `Bearer ${TOKEN}`)
        .expect(200));

    it('allows a protected route with the correct X-API-Key', () =>
      request(app.getHttpServer()).get('/events').set('X-API-Key', TOKEN).expect(200));

    it('keeps the @Public health route open without a token', () =>
      request(app.getHttpServer()).get('/').expect(200).expect('Hello World!'));
  });

  describe('when API_TOKEN is not set', () => {
    it('leaves protected routes open (backward-compatible rollout)', () =>
      request(app.getHttpServer()).get('/events').expect(200));
  });
});
