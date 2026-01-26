/*
  Warnings:

  - Made the column `display_order` on table `events` required. This step will fail if there are existing NULL values in that column.

*/
-- DropIndex
DROP INDEX "public"."idx_events_display_order";

-- AlterTable
ALTER TABLE "events" ADD COLUMN     "label_id" TEXT,
ALTER COLUMN "display_order" SET NOT NULL;

-- CreateTable
CREATE TABLE "labels" (
    "id" TEXT NOT NULL,
    "calendar_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "color" TEXT NOT NULL,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "labels_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "labels_calendar_id_idx" ON "labels"("calendar_id");

-- CreateIndex
CREATE INDEX "events_label_id_idx" ON "events"("label_id");

-- AddForeignKey
ALTER TABLE "labels" ADD CONSTRAINT "labels_calendar_id_fkey" FOREIGN KEY ("calendar_id") REFERENCES "calendars"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "events" ADD CONSTRAINT "events_label_id_fkey" FOREIGN KEY ("label_id") REFERENCES "labels"("id") ON DELETE SET NULL ON UPDATE CASCADE;
