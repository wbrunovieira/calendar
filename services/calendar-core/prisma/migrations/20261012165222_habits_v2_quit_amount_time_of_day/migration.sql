-- AlterTable
ALTER TABLE "event_completions" ADD COLUMN     "value" DOUBLE PRECISION;

-- AlterTable
ALTER TABLE "events" ADD COLUMN     "daily_target" DOUBLE PRECISION,
ADD COLUMN     "habit_kind" TEXT,
ADD COLUMN     "is_paused" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "target_unit" TEXT,
ADD COLUMN     "time_of_day" TEXT;

-- CreateTable
CREATE TABLE "habit_slips" (
    "id" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "slip_date" DATE NOT NULL,
    "trigger" TEXT,
    "note" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "habit_slips_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "habit_slips_event_id_idx" ON "habit_slips"("event_id");

-- CreateIndex
CREATE UNIQUE INDEX "habit_slips_event_id_slip_date_key" ON "habit_slips"("event_id", "slip_date");

-- AddForeignKey
ALTER TABLE "habit_slips" ADD CONSTRAINT "habit_slips_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;
