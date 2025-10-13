-- AlterTable
ALTER TABLE "events" ADD COLUMN     "category_type_id" TEXT;

-- CreateIndex
CREATE INDEX "events_category_type_id_idx" ON "events"("category_type_id");

-- AddForeignKey
ALTER TABLE "events" ADD CONSTRAINT "events_category_type_id_fkey" FOREIGN KEY ("category_type_id") REFERENCES "category_types"("id") ON DELETE SET NULL ON UPDATE CASCADE;
