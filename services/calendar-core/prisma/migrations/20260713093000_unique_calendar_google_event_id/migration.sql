-- Guard-rail against duplicates from the Google Calendar sync.
--
-- Events pulled from Google were persisted with google_event_id = NULL, so the idempotency
-- lookup in PullGoogleChangesUseCase never matched and every re-delivery of the same Google
-- event inserted a new row. The repository now persists google_event_id; this constraint makes
-- the database reject a second row for the same (calendar, Google event) pair.
--
-- Postgres treats NULLs as distinct in unique indexes, so locally-created events
-- (google_event_id IS NULL) never collide with each other.

-- CreateIndex
CREATE UNIQUE INDEX "events_calendar_id_google_event_id_key" ON "events"("calendar_id", "google_event_id");
