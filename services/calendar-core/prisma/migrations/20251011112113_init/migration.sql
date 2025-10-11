-- CreateTable
CREATE TABLE "users" (
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "password_hash" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "avatar_url" TEXT,
    "timezone" TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "calendars" (
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "email" TEXT,
    "color" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "google_calendar_id" TEXT,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "calendars_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "categories" (
    "id" TEXT NOT NULL,
    "calendar_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "icon" TEXT,
    "color" TEXT NOT NULL,
    "type" TEXT,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "categories_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "events" (
    "id" TEXT NOT NULL,
    "calendar_id" TEXT NOT NULL,
    "category_id" TEXT,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "start_time" TEXT NOT NULL,
    "end_time" TEXT,
    "start_date" DATE NOT NULL,
    "end_date" DATE,
    "recurrence_rule" TEXT,
    "recurrence_master_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'CONFIRMED',
    "google_event_id" TEXT,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "events_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "recurrence_exceptions" (
    "id" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "exception_date" DATE NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "recurrence_exceptions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "recurrence_overrides" (
    "id" TEXT NOT NULL,
    "master_event_id" TEXT NOT NULL,
    "occurrence_date" DATE NOT NULL,
    "override_event_id" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "recurrence_overrides_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_completions" (
    "id" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "occurrence_date" DATE NOT NULL,
    "completed" BOOLEAN NOT NULL DEFAULT false,
    "completed_at" TIMESTAMP(3),
    "notes" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "event_completions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "event_reminders" (
    "id" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "minutes_before" INTEGER NOT NULL,
    "method" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "event_reminders_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "monthly_goals" (
    "id" TEXT NOT NULL,
    "category_id" TEXT NOT NULL,
    "month" DATE NOT NULL,
    "target_executions" INTEGER NOT NULL,
    "description" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "monthly_goals_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reports" (
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "month" DATE NOT NULL,
    "data" JSONB NOT NULL,
    "ai_insights" TEXT[],
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "reports_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");

-- CreateIndex
CREATE INDEX "users_email_idx" ON "users"("email");

-- CreateIndex
CREATE INDEX "calendars_user_id_idx" ON "calendars"("user_id");

-- CreateIndex
CREATE INDEX "calendars_google_calendar_id_idx" ON "calendars"("google_calendar_id");

-- CreateIndex
CREATE INDEX "categories_calendar_id_idx" ON "categories"("calendar_id");

-- CreateIndex
CREATE INDEX "events_calendar_id_idx" ON "events"("calendar_id");

-- CreateIndex
CREATE INDEX "events_category_id_idx" ON "events"("category_id");

-- CreateIndex
CREATE INDEX "events_start_date_idx" ON "events"("start_date");

-- CreateIndex
CREATE INDEX "events_recurrence_master_id_idx" ON "events"("recurrence_master_id");

-- CreateIndex
CREATE INDEX "events_google_event_id_idx" ON "events"("google_event_id");

-- CreateIndex
CREATE INDEX "events_status_idx" ON "events"("status");

-- CreateIndex
CREATE INDEX "recurrence_exceptions_event_id_idx" ON "recurrence_exceptions"("event_id");

-- CreateIndex
CREATE INDEX "recurrence_exceptions_exception_date_idx" ON "recurrence_exceptions"("exception_date");

-- CreateIndex
CREATE UNIQUE INDEX "recurrence_exceptions_event_id_exception_date_key" ON "recurrence_exceptions"("event_id", "exception_date");

-- CreateIndex
CREATE INDEX "recurrence_overrides_master_event_id_idx" ON "recurrence_overrides"("master_event_id");

-- CreateIndex
CREATE INDEX "recurrence_overrides_override_event_id_idx" ON "recurrence_overrides"("override_event_id");

-- CreateIndex
CREATE INDEX "recurrence_overrides_occurrence_date_idx" ON "recurrence_overrides"("occurrence_date");

-- CreateIndex
CREATE UNIQUE INDEX "recurrence_overrides_master_event_id_occurrence_date_key" ON "recurrence_overrides"("master_event_id", "occurrence_date");

-- CreateIndex
CREATE INDEX "event_completions_event_id_idx" ON "event_completions"("event_id");

-- CreateIndex
CREATE INDEX "event_completions_occurrence_date_idx" ON "event_completions"("occurrence_date");

-- CreateIndex
CREATE INDEX "event_completions_completed_idx" ON "event_completions"("completed");

-- CreateIndex
CREATE UNIQUE INDEX "event_completions_event_id_occurrence_date_key" ON "event_completions"("event_id", "occurrence_date");

-- CreateIndex
CREATE INDEX "event_reminders_event_id_idx" ON "event_reminders"("event_id");

-- CreateIndex
CREATE INDEX "monthly_goals_category_id_idx" ON "monthly_goals"("category_id");

-- CreateIndex
CREATE INDEX "monthly_goals_month_idx" ON "monthly_goals"("month");

-- CreateIndex
CREATE UNIQUE INDEX "monthly_goals_category_id_month_key" ON "monthly_goals"("category_id", "month");

-- CreateIndex
CREATE INDEX "reports_user_id_idx" ON "reports"("user_id");

-- CreateIndex
CREATE INDEX "reports_month_idx" ON "reports"("month");

-- CreateIndex
CREATE UNIQUE INDEX "reports_user_id_month_key" ON "reports"("user_id", "month");

-- AddForeignKey
ALTER TABLE "calendars" ADD CONSTRAINT "calendars_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "categories" ADD CONSTRAINT "categories_calendar_id_fkey" FOREIGN KEY ("calendar_id") REFERENCES "calendars"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "events" ADD CONSTRAINT "events_calendar_id_fkey" FOREIGN KEY ("calendar_id") REFERENCES "calendars"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "events" ADD CONSTRAINT "events_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "events" ADD CONSTRAINT "events_recurrence_master_id_fkey" FOREIGN KEY ("recurrence_master_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "recurrence_exceptions" ADD CONSTRAINT "recurrence_exceptions_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "recurrence_overrides" ADD CONSTRAINT "recurrence_overrides_master_event_id_fkey" FOREIGN KEY ("master_event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "recurrence_overrides" ADD CONSTRAINT "recurrence_overrides_override_event_id_fkey" FOREIGN KEY ("override_event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "event_completions" ADD CONSTRAINT "event_completions_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "event_reminders" ADD CONSTRAINT "event_reminders_event_id_fkey" FOREIGN KEY ("event_id") REFERENCES "events"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "monthly_goals" ADD CONSTRAINT "monthly_goals_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reports" ADD CONSTRAINT "reports_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;
