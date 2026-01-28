package workout

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// WorkoutStatus represents the status of a workout
type WorkoutStatus string

const (
	WorkoutStatusPlanned    WorkoutStatus = "PLANNED"
	WorkoutStatusInProgress WorkoutStatus = "IN_PROGRESS"
	WorkoutStatusCompleted  WorkoutStatus = "COMPLETED"
	WorkoutStatusSkipped    WorkoutStatus = "SKIPPED"
)

// SetType represents the type of a set
type SetType string

const (
	SetTypeWarmup    SetType = "WARMUP"
	SetTypeNormal    SetType = "NORMAL"
	SetTypeDrop      SetType = "DROP"
	SetTypeFailure   SetType = "FAILURE"
	SetTypeRestPause SetType = "REST_PAUSE"
)

// Workout represents a workout session
type Workout struct {
	ID              string        `json:"id"`
	ProfileID       string        `json:"profileId"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	WorkoutDate     time.Time     `json:"workoutDate"`
	StartTime       string        `json:"startTime,omitempty"`
	EndTime         string        `json:"endTime,omitempty"`
	DurationMinutes int           `json:"durationMinutes,omitempty"`
	Notes           string        `json:"notes,omitempty"`
	Rating          int           `json:"rating,omitempty"`
	Status          WorkoutStatus `json:"status"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`

	// Related data
	Exercises []WorkoutExercise `json:"exercises,omitempty"`
}

// WorkoutExercise represents an exercise performed in a workout
type WorkoutExercise struct {
	ID         string        `json:"id"`
	WorkoutID  string        `json:"workoutId"`
	ExerciseID string        `json:"exerciseId"`
	OrderIndex int           `json:"orderIndex"`
	Notes      string        `json:"notes,omitempty"`
	CreatedAt  time.Time     `json:"createdAt"`
	Sets       []ExerciseSet `json:"sets,omitempty"`
}

// ExerciseSet represents a single set within a workout exercise
type ExerciseSet struct {
	ID                string   `json:"id"`
	WorkoutExerciseID string   `json:"workoutExerciseId"`
	SetNumber         int      `json:"setNumber"`
	SetType           SetType  `json:"setType"`
	Weight            *float64 `json:"weight,omitempty"`
	WeightUnit        string   `json:"weightUnit,omitempty"`
	Reps              *int     `json:"reps,omitempty"`
	DurationSeconds   *int     `json:"durationSeconds,omitempty"`
	DistanceMeters    *float64 `json:"distanceMeters,omitempty"`
	RPE               *int     `json:"rpe,omitempty"`
	Completed         bool     `json:"completed"`
	Notes             string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
}

// NewWorkout creates a new Workout with validation
func NewWorkout(profileID, name string, workoutDate time.Time) (*Workout, error) {
	if profileID == "" {
		return nil, errors.New("profileID is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}

	now := time.Now()
	return &Workout{
		ID:          uuid.New().String(),
		ProfileID:   profileID,
		Name:        name,
		WorkoutDate: workoutDate,
		Status:      WorkoutStatusCompleted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// NewWorkoutExercise creates a new WorkoutExercise
func NewWorkoutExercise(workoutID, exerciseID string, orderIndex int) (*WorkoutExercise, error) {
	if workoutID == "" {
		return nil, errors.New("workoutID is required")
	}
	if exerciseID == "" {
		return nil, errors.New("exerciseID is required")
	}

	return &WorkoutExercise{
		ID:         uuid.New().String(),
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
		OrderIndex: orderIndex,
		CreatedAt:  time.Now(),
	}, nil
}

// NewExerciseSet creates a new ExerciseSet
func NewExerciseSet(workoutExerciseID string, setNumber int, setType SetType) (*ExerciseSet, error) {
	if workoutExerciseID == "" {
		return nil, errors.New("workoutExerciseID is required")
	}

	return &ExerciseSet{
		ID:                uuid.New().String(),
		WorkoutExerciseID: workoutExerciseID,
		SetNumber:         setNumber,
		SetType:           setType,
		WeightUnit:        "kg",
		Completed:         true,
		CreatedAt:         time.Now(),
	}, nil
}
