import { beforeAll, afterAll, afterEach, vi } from 'vitest';

// Mock environment variables for tests
beforeAll(() => {
  process.env.DATABASE_URL = 'postgresql://calendar:calendar123@localhost:5433/calendar_test_db';
  process.env.JWT_SECRET = 'test-secret-key-for-testing';
  process.env.NODE_ENV = 'test';
  process.env.PORT = '3334';
  process.env.TZ = 'America/Sao_Paulo';
});

afterEach(() => {
  // Clear all mocks after each test
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

afterAll(() => {
  // Global cleanup
});
