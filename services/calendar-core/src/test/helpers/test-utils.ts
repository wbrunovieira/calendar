import { vi, expect } from 'vitest';

/**
 * Test Utilities
 *
 * Common utilities for testing
 */

/**
 * Sets up fake timers with a fixed date
 */
export function useFakeTimers(date: Date = new Date('2024-11-16T10:00:00-03:00')) {
  vi.useFakeTimers();
  vi.setSystemTime(date);
}

/**
 * Restores real timers
 */
export function useRealTimers() {
  vi.useRealTimers();
}

/**
 * Waits for a promise to resolve
 */
export async function waitFor(ms: number = 0): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Creates a spy on console methods
 */
export function spyOnConsole() {
  const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
  const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

  return {
    log: logSpy,
    error: errorSpy,
    warn: warnSpy,
    restore: () => {
      logSpy.mockRestore();
      errorSpy.mockRestore();
      warnSpy.mockRestore();
    },
  };
}

/**
 * Helper to test error throwing
 */
export async function expectToThrow(fn: () => any, errorMessage?: string) {
  try {
    await fn();
    throw new Error('Expected function to throw an error');
  } catch (error: any) {
    if (errorMessage) {
      expect(error.message).toContain(errorMessage);
    }
  }
}

/**
 * Helper to create a delay
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Helper to test async execution order
 */
export class ExecutionOrder {
  private order: string[] = [];

  add(name: string) {
    this.order.push(name);
  }

  expect(expected: string[]) {
    expect(this.order).toEqual(expected);
  }

  reset() {
    this.order = [];
  }
}
