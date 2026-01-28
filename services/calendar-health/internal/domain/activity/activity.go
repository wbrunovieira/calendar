package activity

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ActivityType represents the type of activity
type ActivityType string

const (
	ActivityTypeBreathing    ActivityType = "BREATHING"
	ActivityTypeColdExposure ActivityType = "COLD_EXPOSURE"
	ActivityTypeMeditation   ActivityType = "MEDITATION"
	ActivityTypeHIIT         ActivityType = "HIIT"
	ActivityTypeCardio       ActivityType = "CARDIO"
	ActivityTypeStretching   ActivityType = "STRETCHING"
	ActivityTypeOther        ActivityType = "OTHER"
)

// Activity represents a non-gym health activity (Wim Hof, meditation, HIT, etc.)
type Activity struct {
	ID              string          `json:"id"`
	ProfileID       string          `json:"profileId"`
	ExerciseID      string          `json:"exerciseId,omitempty"`
	Name            string          `json:"name"`
	ActivityType    ActivityType    `json:"activityType"`
	ActivityDate    time.Time       `json:"activityDate"`
	StartTime       string          `json:"startTime,omitempty"`
	DurationMinutes int             `json:"durationMinutes,omitempty"`
	Rounds          int             `json:"rounds,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	Metrics         json.RawMessage `json:"metrics,omitempty"`
	Rating          int             `json:"rating,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// WimHofMetrics represents metrics specific to Wim Hof breathing
type WimHofMetrics struct {
	Rounds          int   `json:"rounds"`
	BreathsPerRound []int `json:"breathsPerRound,omitempty"`
	RetentionTimes  []int `json:"retentionTimes,omitempty"` // in seconds
	ColdExposure    int   `json:"coldExposure,omitempty"`   // in seconds
}

// HIITMetrics represents metrics specific to HIIT workouts
type HIITMetrics struct {
	Intervals     int `json:"intervals"`
	WorkSeconds   int `json:"workSeconds"`
	RestSeconds   int `json:"restSeconds"`
	TotalCalories int `json:"totalCalories,omitempty"`
	AvgHeartRate  int `json:"avgHeartRate,omitempty"`
	MaxHeartRate  int `json:"maxHeartRate,omitempty"`
}

// NewActivity creates a new Activity with validation
func NewActivity(profileID, name string, activityType ActivityType, activityDate time.Time) (*Activity, error) {
	if profileID == "" {
		return nil, errors.New("profileID is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}

	now := time.Now()
	return &Activity{
		ID:           uuid.New().String(),
		ProfileID:    profileID,
		Name:         name,
		ActivityType: activityType,
		ActivityDate: activityDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Update updates the activity fields
func (a *Activity) Update(name string, durationMinutes, rounds, rating int, notes string) error {
	if name == "" {
		return errors.New("name is required")
	}

	a.Name = name
	a.DurationMinutes = durationMinutes
	a.Rounds = rounds
	a.Rating = rating
	a.Notes = notes
	a.UpdatedAt = time.Now()
	return nil
}

// SetWimHofMetrics sets Wim Hof specific metrics
func (a *Activity) SetWimHofMetrics(metrics WimHofMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	a.Metrics = data
	a.UpdatedAt = time.Now()
	return nil
}

// SetHIITMetrics sets HIIT specific metrics
func (a *Activity) SetHIITMetrics(metrics HIITMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	a.Metrics = data
	a.UpdatedAt = time.Now()
	return nil
}
