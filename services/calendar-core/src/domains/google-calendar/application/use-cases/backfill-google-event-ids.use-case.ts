import { Injectable, Logger } from '@nestjs/common';
import { calendar_v3 } from 'googleapis';
import { CalendarRepository } from '../../../calendars/infrastructure/persistence/calendar.repository';
import { EventRepository } from '../../../events/infrastructure/repositories/event.repository';
import { GoogleCalendarApiClient } from '../../infrastructure/services/google-calendar-api.client';
import { GoogleEventMapper } from '../../domain/services/google-event-mapper';
import { DuplicateResolutionPolicy } from '../../domain/services/duplicate-resolution-policy';

interface PlanEntry {
  eventId: string;
  googleEventId: string;
  title: string;
  calendarId: string;
}

interface Conflict {
  googleEventId: string;
  title: string;
  calendarId: string;
  reason: string;
}

export interface BackfillPlan {
  linked: PlanEntry[];
  deleted: PlanEntry[];
  conflicts: Conflict[];
  unmatchedGoogleEvents: number;
  /**
   * Active top-level EVENTs still carrying no googleEventId once this plan is applied. A raw
   * denominator, not a defect count: most of them are events the user created by hand that Google
   * never knew about. It also covers the orphans this run cannot repair — those whose title or
   * time changed in Google since the import, and those outside the API's lookback window.
   */
  unlinkedEventsRemaining: number;
}

/**
 * One-shot repair for events the broken sync inserted with google_event_id = NULL.
 *
 * Google is the source of truth: a local row is only ever touched if it matches an event that
 * actually exists in the connected account. Rows matching nothing — including the pre-existing
 * duplicate groups that came from the bulk import — are never even looked at.
 *
 * The governing rule for deletion: the sync writes only the dozen columns Google owns. Category,
 * label, reminders, completions and recurrence structure are user data it cannot reconstruct, so a
 * row carrying any of them is never deleted — it becomes a conflict for a human to merge.
 */
@Injectable()
export class BackfillGoogleEventIdsUseCase {
  private readonly logger = new Logger(BackfillGoogleEventIdsUseCase.name);

  constructor(
    private readonly calendarRepository: CalendarRepository,
    private readonly eventRepository: EventRepository,
    private readonly apiClient: GoogleCalendarApiClient,
  ) {}

  async execute({ dryRun }: { dryRun: boolean }): Promise<BackfillPlan> {
    const plan: BackfillPlan = {
      linked: [],
      deleted: [],
      conflicts: [],
      unmatchedGoogleEvents: 0,
      unlinkedEventsRemaining: 0,
    };

    // Rows already spoken for in this run. Without this, dry-run and apply walk different paths:
    // apply mutates the rows a later iteration would query, dry-run does not, so the plan a human
    // reviews would not be the plan that executes.
    const consumed = new Set<string>();

    const calendars = await this.calendarRepository.findAllGoogleConnected();

    for (const calendar of calendars) {
      if (!calendar.googleCalendarId || !calendar.email) continue;

      // Counted up front, then adjusted by what this run plans to do, so dry-run and apply report
      // the same projected residue rather than a pre- and a post-state.
      const unlinkedBefore = await this.eventRepository.countUnlinkedEvents(calendar.id);
      const before = { linked: plan.linked.length, deleted: plan.deleted.length };

      try {
        // syncToken = null forces a full listing rather than a delta.
        const { events } = await this.apiClient.fetchChanges(
          calendar.googleCalendarId,
          calendar.email,
          null,
        );

        for (const googleEvent of this.usableEvents(events, calendar.id, plan)) {
          try {
            await this.processGoogleEvent(googleEvent, calendar.id, plan, consumed, dryRun);
          } catch (err) {
            plan.conflicts.push({
              googleEventId: googleEvent.id!,
              title: googleEvent.summary ?? '(no title)',
              calendarId: calendar.id,
              reason: `unexpected failure, left untouched: ${(err as Error).message}`,
            });
          }
        }
      } catch (err) {
        // A whole calendar failing (expired refresh token, Google down) must not discard the plan
        // already applied for the calendars before it.
        plan.conflicts.push({
          googleEventId: '-',
          title: '-',
          calendarId: calendar.id,
          reason: `calendar skipped, could not be read from Google: ${(err as Error).message}`,
        });
      }

      const planned =
        plan.linked.length - before.linked + (plan.deleted.length - before.deleted);
      plan.unlinkedEventsRemaining += Math.max(0, unlinkedBefore - planned);
    }

    this.logger.log(
      `Backfill ${dryRun ? '(dry run) ' : ''}— linked: ${plan.linked.length}, deleted: ${plan.deleted.length}, conflicts: ${plan.conflicts.length}, Google events with no local copy: ${plan.unmatchedGoogleEvents}, rows left unlinked: ${plan.unlinkedEventsRemaining}`,
    );

    return plan;
  }

  /**
   * Drops cancelled events, and refuses to touch any group of Google events that share one natural
   * key. Matching is by (title, date, time); when two Google events collide on it there is no way
   * to tell which local row belongs to which, and guessing would delete the wrong one.
   */
  private usableEvents(
    events: calendar_v3.Schema$Event[],
    calendarId: string,
    plan: BackfillPlan,
  ): calendar_v3.Schema$Event[] {
    const live = events.filter((e) => e.id && e.status !== 'cancelled');
    const byKey = new Map<string, calendar_v3.Schema$Event[]>();

    for (const event of live) {
      let key: string;
      try {
        const mapped = GoogleEventMapper.fromGoogleEvent(event, calendarId);
        key = `${mapped.title}|${mapped.startDate.toISOString()}|${mapped.startTime}`;
      } catch {
        // Unmappable payload — processGoogleEvent will report it as a conflict.
        key = `unmappable|${event.id}`;
      }
      byKey.set(key, [...(byKey.get(key) ?? []), event]);
    }

    const usable: calendar_v3.Schema$Event[] = [];

    for (const group of byKey.values()) {
      if (group.length === 1) {
        usable.push(group[0]);
        continue;
      }

      for (const event of group) {
        plan.conflicts.push({
          googleEventId: event.id!,
          title: event.summary ?? '(no title)',
          calendarId,
          reason: `${group.length} Google events share the same title and start time; cannot tell which local row belongs to which`,
        });
      }
    }

    return usable;
  }

  private async processGoogleEvent(
    googleEvent: calendar_v3.Schema$Event,
    calendarId: string,
    plan: BackfillPlan,
    consumed: Set<string>,
    dryRun: boolean,
  ): Promise<void> {
    const googleEventId = googleEvent.id!;
    const mapped = GoogleEventMapper.fromGoogleEvent(googleEvent, calendarId);

    const candidates = (
      await this.eventRepository.findUnlinkedMatches(
        calendarId,
        mapped.title,
        mapped.startDate,
        mapped.startTime,
      )
    ).filter((c) => !consumed.has(c.id));

    const entry = { googleEventId, title: mapped.title, calendarId };

    // The fixed sync may already have re-imported this event as a correctly linked row. The UNIQUE
    // index means that row owns the identity — we cannot link a second one. But it is a *bare* row:
    // the sync writes no category, label or reminders. So a stale copy is only dropped when it
    // carries nothing of the user's; otherwise a human has to merge the two.
    const alreadyLinked = await this.eventRepository.findByGoogleEventId(googleEventId, calendarId);

    if (alreadyLinked) {
      for (const stale of candidates) {
        await this.discard(stale, entry, plan, consumed, dryRun);
      }
      return;
    }

    if (candidates.length === 0) {
      plan.unmatchedGoogleEvents++;
      return;
    }

    const resolution = DuplicateResolutionPolicy.resolve(candidates);

    if (resolution.kind === 'conflict') {
      plan.conflicts.push({ ...entry, reason: resolution.reason });
      return;
    }

    const { survivor, losers } = resolution;

    consumed.add(survivor.id);
    plan.linked.push({ eventId: survivor.id, ...entry });
    if (!dryRun) {
      await this.eventRepository.updateGoogleEventId(survivor.id, googleEventId);
    }

    for (const loser of losers) {
      await this.discard(loser, entry, plan, consumed, dryRun);
    }
  }

  private async discard(
    candidate: { id: string; hasUserData: boolean },
    entry: { googleEventId: string; title: string; calendarId: string },
    plan: BackfillPlan,
    consumed: Set<string>,
    dryRun: boolean,
  ): Promise<void> {
    consumed.add(candidate.id);

    if (candidate.hasUserData) {
      plan.conflicts.push({
        ...entry,
        reason: `copy ${candidate.id} carries user data (category/label/reminders/history) and was left in place`,
      });
      return;
    }

    plan.deleted.push({ eventId: candidate.id, ...entry });
    if (!dryRun) {
      await this.eventRepository.delete(candidate.id);
    }
  }
}
