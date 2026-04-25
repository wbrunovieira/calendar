-- AlterTable
ALTER TABLE "calendars" ADD COLUMN     "google_sync_token" TEXT,
ADD COLUMN     "last_sync_at" TIMESTAMP(3);

-- CreateTable
CREATE TABLE "google_oauth_tokens" (
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "access_token" TEXT NOT NULL,
    "refresh_token" TEXT NOT NULL,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "google_oauth_tokens_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "google_oauth_tokens_email_key" ON "google_oauth_tokens"("email");
