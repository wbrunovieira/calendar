package exercise

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Exercise represents an exercise entity
type Exercise struct {
	ID             string    `json:"id"`
	ProfileID      string    `json:"profileId"`
	ExerciseTypeID string    `json:"exerciseTypeId,omitempty"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	MuscleGroup    string    `json:"muscleGroup,omitempty"`
	Equipment      string    `json:"equipment,omitempty"`
	Instructions   string    `json:"instructions,omitempty"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ExerciseType represents a category of exercises
type ExerciseType struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profileId"`
	Name      string    `json:"name"`
	Color     string    `json:"color,omitempty"`
	Icon      string    `json:"icon,omitempty"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewExercise creates a new Exercise with validation
func NewExercise(profileID, name string) (*Exercise, error) {
	if profileID == "" {
		return nil, errors.New("profileID is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}

	now := time.Now()
	return &Exercise{
		ID:        uuid.New().String(),
		ProfileID: profileID,
		Name:      name,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// NewExerciseType creates a new ExerciseType with validation
func NewExerciseType(profileID, name string) (*ExerciseType, error) {
	if profileID == "" {
		return nil, errors.New("profileID is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}

	now := time.Now()
	return &ExerciseType{
		ID:        uuid.New().String(),
		ProfileID: profileID,
		Name:      name,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update updates the exercise fields
func (e *Exercise) Update(name, description, muscleGroup, equipment, instructions string) error {
	if name == "" {
		return errors.New("name is required")
	}

	e.Name = name
	e.Description = description
	e.MuscleGroup = muscleGroup
	e.Equipment = equipment
	e.Instructions = instructions
	e.UpdatedAt = time.Now()
	return nil
}
