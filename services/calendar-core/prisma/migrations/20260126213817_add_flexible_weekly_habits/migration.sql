-- AlterTable
ALTER TABLE "events" ADD COLUMN     "recurrence_type" TEXT,
ADD COLUMN     "weekly_preferred_days" TEXT[],
ADD COLUMN     "weekly_target_count" INTEGER;

-- CreateTable
CREATE TABLE "weekly_goal_completions" (
    "id" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "week_start_date" DATE NOT NULL,
    "target_count" INTEGER NOT NULL,
    "completed_count" INTEGER NOT NULL DEFAULT 0,
    "completed_dates" TEXT[],
    "is_goal_met" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "weekly_goal_completions_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "weekly_goal_completions_event_id_idx" ON "weekly_goal_completions"("event_id");

-- CreateIndex
CREATE INDEX "weekly_goal_completions_week_start_date_idx" ON "weekly_goal_completions"("week_start_date");

-- CreateIndex
CREATE INDEX "weekly_goal_completions_is_goal_met_idx" ON "weekly_goal_completions"("is_goal_met");

-- CreateIndex
CREATE UNIQUE INDEX "weekly_goal_completions_event_id_week_start_date_key" ON "weekly_goal_completions"("event_id", "week_start_date");

-- AddForeignKey
ALTER TABLE "weekly_goal_completions" ADD CONSTRAINT "weekly_goal_completions_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;
