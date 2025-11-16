import { mock, mockDeep, MockProxy, DeepMockProxy } from 'vitest-mock-extended';
import { PrismaClient } from '@prisma/client';

/**
 * Mock Builders for Testing
 *
 * These builders provide convenient ways to create mocks for testing.
 */

/**
 * Creates a mock repository
 */
export function createMockRepository<T>(): MockProxy<T> {
  return mock<T>();
}

/**
 * Creates a deep mock of PrismaClient
 */
export function createMockPrisma(): DeepMockProxy<PrismaClient> {
  return mockDeep<PrismaClient>();
}

/**
 * Creates a mock use case
 */
export function createMockUseCase<T>(): MockProxy<T> {
  return mock<T>();
}

/**
 * Type helper for mocking functions
 */
export type MockedFunction<T extends (...args: any[]) => any> = MockProxy<T>;

/**
 * Helper to create a resolved promise mock
 */
export function mockResolved<T>(value: T) {
  return vi.fn().mockResolvedValue(value);
}

/**
 * Helper to create a rejected promise mock
 */
export function mockRejected(error: Error) {
  return vi.fn().mockRejectedValue(error);
}

/**
 * Helper to reset all mocks
 */
export function resetAllMocks() {
  vi.clearAllMocks();
  vi.restoreAllMocks();
}
