-- AlterTable
ALTER TABLE "events" ADD COLUMN     "due_date" DATE,
ADD COLUMN     "event_type" TEXT NOT NULL DEFAULT 'EVENT',
ADD COLUMN     "priority" INTEGER;

-- CreateIndex
CREATE INDEX "events_event_type_idx" ON "events"("event_type");
