/**
 * One-shot repair for the Google Calendar duplication bug.
 *
 * The sync used to persist imported events with google_event_id = NULL, so it could never
 * recognise them again and kept inserting copies. This links each surviving row back to its Google
 * event and removes the duplicates.
 *
 * The production image ships only `dist`, so run the compiled file:
 *
 *   Dry run (prints the plan, writes nothing):
 *     docker compose exec calendar-core node dist/scripts/backfill-google-event-ids.js
 *
 *   Apply:
 *     docker compose exec calendar-core node dist/scripts/backfill-google-event-ids.js --apply
 *
 * Stop the polling cron before applying — set GOOGLE_SYNC_ENABLED=false and recreate the service.
 * While the cron runs it keeps importing events into the very rows this script is linking, and any
 * orphan it re-imports meanwhile becomes a duplicate this script then has to reconcile.
 */
import { NestFactory } from '@nestjs/core';
import { Logger } from '@nestjs/common';
import { BackfillModule } from '../domains/google-calendar/backfill.module';
import { BackfillGoogleEventIdsUseCase } from '../domains/google-calendar/application/use-cases/backfill-google-event-ids.use-case';

async function main() {
  const apply = process.argv.includes('--apply');
  const logger = new Logger('BackfillScript');

  const app = await NestFactory.createApplicationContext(BackfillModule, {
    logger: ['error', 'warn', 'log'],
  });

  try {
    const plan = await app.get(BackfillGoogleEventIdsUseCase).execute({ dryRun: !apply });

    logger.log(apply ? '=== APPLIED ===' : '=== DRY RUN (nothing was written) ===');

    logger.log(`Linked to their Google event: ${plan.linked.length}`);
    for (const item of plan.linked) {
      logger.log(`  link  ${item.eventId}  ->  ${item.googleEventId}  "${item.title}"`);
    }

    logger.log(`Duplicates removed: ${plan.deleted.length}`);
    for (const item of plan.deleted) {
      logger.log(`  drop  ${item.eventId}  "${item.title}"  [${item.calendarId}]`);
    }

    logger.log(`Left for a human to decide: ${plan.conflicts.length}`);
    for (const item of plan.conflicts) {
      logger.warn(`  skip  "${item.title}"  [${item.calendarId}]  ${item.reason}`);
    }

    logger.log(`Google events with no local copy: ${plan.unmatchedGoogleEvents}`);
    logger.log(
      `Rows left with no googleEventId: ${plan.unlinkedEventsRemaining} ` +
        '(mostly events the user created locally; also any orphan whose title or time changed in ' +
        'Google, or that falls outside the API lookback window)',
    );

    if (!apply && (plan.linked.length > 0 || plan.deleted.length > 0)) {
      logger.log('Re-run with --apply to execute this plan.');
    }
  } finally {
    await app.close();
  }
}

main().catch((err) => {
  new Logger('BackfillScript').error(err);
  process.exit(1);
});
