-- Meeting details synced from Google Calendar: Meet link, guests, location, organizer.
-- All nullable; existing rows keep NULL until the next sync repopulates them.
ALTER TABLE "events" ADD COLUMN "location" TEXT;
ALTER TABLE "events" ADD COLUMN "meeting_url" TEXT;
ALTER TABLE "events" ADD COLUMN "attendees" JSONB;
ALTER TABLE "events" ADD COLUMN "organizer" JSONB;
