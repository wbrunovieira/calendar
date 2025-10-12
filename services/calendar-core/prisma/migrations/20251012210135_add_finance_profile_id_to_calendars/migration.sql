-- AlterTable
ALTER TABLE "calendars" ADD COLUMN     "finance_profile_id" TEXT;

-- CreateIndex
CREATE INDEX "calendars_finance_profile_id_idx" ON "calendars"("finance_profile_id");
