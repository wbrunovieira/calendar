package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Connect establishes a connection to the PostgreSQL database
func Connect(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return db, nil
}

// RunMigrations creates the health schema and initial tables
func RunMigrations(db *sql.DB) error {
	log.Println("Running health database migrations...")

	migrations := []string{
		// Ensure required extension
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,

		// Create health schema
		`CREATE SCHEMA IF NOT EXISTS health`,

		// Create profiles table (linked to calendar profiles)
		`CREATE TABLE IF NOT EXISTS health.profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			calendar_id VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create exercise_types table (categories like: Strength, Cardio, Flexibility, Breathing)
		`CREATE TABLE IF NOT EXISTS health.exercise_types (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
			name VARCHAR(120) NOT NULL,
			color VARCHAR(7),
			icon VARCHAR(50),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_exercise_types_profile_name UNIQUE (profile_id, name)
		)`,

		// Create exercises table (specific exercises like: Bench Press, Squat, Wim Hof, HIT)
		`CREATE TABLE IF NOT EXISTS health.exercises (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
			exercise_type_id UUID REFERENCES health.exercise_types(id) ON DELETE SET NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			muscle_group VARCHAR(100),
			equipment VARCHAR(100),
			instructions TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_exercises_profile_name UNIQUE (profile_id, name)
		)`,

		// Create workouts table (a workout session)
		`CREATE TABLE IF NOT EXISTS health.workouts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			workout_date DATE NOT NULL,
			start_time TIME,
			end_time TIME,
			duration_minutes INTEGER,
			notes TEXT,
			rating INTEGER CHECK (rating >= 1 AND rating <= 5),
			status VARCHAR(20) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PLANNED', 'IN_PROGRESS', 'COMPLETED', 'SKIPPED')),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create workout_exercises table (exercises performed in a workout with sets)
		`CREATE TABLE IF NOT EXISTS health.workout_exercises (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			workout_id UUID NOT NULL REFERENCES health.workouts(id) ON DELETE CASCADE,
			exercise_id UUID NOT NULL REFERENCES health.exercises(id) ON DELETE CASCADE,
			order_index INTEGER NOT NULL DEFAULT 0,
			notes TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create exercise_sets table (individual sets within a workout exercise)
		`CREATE TABLE IF NOT EXISTS health.exercise_sets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			workout_exercise_id UUID NOT NULL REFERENCES health.workout_exercises(id) ON DELETE CASCADE,
			set_number INTEGER NOT NULL,
			set_type VARCHAR(20) NOT NULL DEFAULT 'NORMAL' CHECK (set_type IN ('WARMUP', 'NORMAL', 'DROP', 'FAILURE', 'REST_PAUSE')),
			weight NUMERIC(8, 2),
			weight_unit VARCHAR(5) DEFAULT 'kg',
			reps INTEGER,
			duration_seconds INTEGER,
			distance_meters NUMERIC(10, 2),
			rpe INTEGER CHECK (rpe >= 1 AND rpe <= 10),
			completed BOOLEAN NOT NULL DEFAULT true,
			notes TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create activities table (for non-gym activities like Wim Hof, meditation, HIT)
		`CREATE TABLE IF NOT EXISTS health.activities (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
			exercise_id UUID REFERENCES health.exercises(id) ON DELETE SET NULL,
			name VARCHAR(255) NOT NULL,
			activity_type VARCHAR(50) NOT NULL CHECK (activity_type IN ('BREATHING', 'COLD_EXPOSURE', 'MEDITATION', 'HIIT', 'CARDIO', 'STRETCHING', 'OTHER')),
			activity_date DATE NOT NULL,
			start_time TIME,
			duration_minutes INTEGER,
			rounds INTEGER,
			notes TEXT,
			metrics JSONB,
			rating INTEGER CHECK (rating >= 1 AND rating <= 5),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create personal_records table (PRs for exercises)
		`CREATE TABLE IF NOT EXISTS health.personal_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES health.profiles(id) ON DELETE CASCADE,
			exercise_id UUID NOT NULL REFERENCES health.exercises(id) ON DELETE CASCADE,
			record_type VARCHAR(20) NOT NULL CHECK (record_type IN ('MAX_WEIGHT', 'MAX_REPS', 'MAX_VOLUME', 'MAX_DURATION', 'MAX_DISTANCE')),
			value NUMERIC(10, 2) NOT NULL,
			unit VARCHAR(20),
			achieved_at DATE NOT NULL,
			workout_id UUID REFERENCES health.workouts(id) ON DELETE SET NULL,
			notes TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_health_profiles_calendar_id ON health.profiles(calendar_id)`,
		`CREATE INDEX IF NOT EXISTS idx_health_exercise_types_profile ON health.exercise_types(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_health_exercises_profile ON health.exercises(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_health_exercises_type ON health.exercises(exercise_type_id)`,
		`CREATE INDEX IF NOT EXISTS idx_health_workouts_profile_date ON health.workouts(profile_id, workout_date)`,
		`CREATE INDEX IF NOT EXISTS idx_health_workout_exercises_workout ON health.workout_exercises(workout_id)`,
		`CREATE INDEX IF NOT EXISTS idx_health_exercise_sets_workout_exercise ON health.exercise_sets(workout_exercise_id)`,
		`CREATE INDEX IF NOT EXISTS idx_health_activities_profile_date ON health.activities(profile_id, activity_date)`,
		`CREATE INDEX IF NOT EXISTS idx_health_personal_records_profile_exercise ON health.personal_records(profile_id, exercise_id)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("✓ Health migrations completed successfully")
	return nil
}
