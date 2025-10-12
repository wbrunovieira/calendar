/*
  Warnings:

  - A unique constraint covering the columns `[calendar_id,value]` on the table `category_types` will be added. If there are existing duplicate values, this will fail.
  - Added the required column `calendar_id` to the `category_types` table without a default value. This is not possible if the table is not empty.

*/
-- DropIndex
DROP INDEX "category_types_value_key";

-- AlterTable
ALTER TABLE "category_types" ADD COLUMN     "calendar_id" TEXT NOT NULL;

-- CreateIndex
CREATE INDEX "category_types_calendar_id_idx" ON "category_types"("calendar_id");

-- CreateIndex
CREATE UNIQUE INDEX "category_types_calendar_id_value_key" ON "category_types"("calendar_id", "value");

-- AddForeignKey
ALTER TABLE "category_types" ADD CONSTRAINT "category_types_calendar_id_fkey" FOREIGN KEY ("calendar_id") REFERENCES "calendars"("id") ON DELETE CASCADE ON UPDATE CASCADE;
