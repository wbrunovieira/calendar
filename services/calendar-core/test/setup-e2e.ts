import { beforeAll, afterAll } from 'vitest';
import { PrismaClient } from '@prisma/client';

let prisma: PrismaClient;

// Setup before all E2E tests
beforeAll(async () => {
  // TEST_DATABASE_URL lets the suite run from inside Docker (host `postgres`) as well as from the
  // host (localhost:5433). It must always point at calendar_test_db — this suite wipes the schema.
  process.env.DATABASE_URL =
    process.env.TEST_DATABASE_URL ??
    'postgresql://calendar:calendar123@localhost:5433/calendar_test_db';
  process.env.NODE_ENV = 'test';
  process.env.TZ = 'America/Sao_Paulo';

  // Initialize Prisma client for test database
  prisma = new PrismaClient({
    datasources: {
      db: {
        url: process.env.DATABASE_URL,
      },
    },
  });

  // Connect to database
  await prisma.$connect();

  // Clean database before tests
  await cleanDatabase();
});

// Cleanup after all E2E tests
afterAll(async () => {
  if (prisma) {
    await prisma.$disconnect();
  }
});

// Helper function to clean database
async function cleanDatabase() {
  // Delete in correct order to respect foreign keys
  await prisma.eventCompletion.deleteMany();
  await prisma.recurrenceException.deleteMany();
  await prisma.recurrenceOverride.deleteMany();
  await prisma.eventReminder.deleteMany();
  await prisma.event.deleteMany();
  await prisma.monthlyGoal.deleteMany();
  await prisma.categoryToType.deleteMany();
  await prisma.category.deleteMany();
  await prisma.categoryType.deleteMany();
  await prisma.calendar.deleteMany();
  await prisma.user.deleteMany();
}

// Export for use in tests
export { prisma, cleanDatabase };
