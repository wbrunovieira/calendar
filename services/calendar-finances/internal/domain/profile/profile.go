package profile

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ProfileType represents the type of financial profile
type ProfileType string

const (
	ProfileTypePersonal ProfileType = "PERSONAL"
	ProfileTypeBusiness ProfileType = "BUSINESS"
)

// Profile represents a financial profile entity
type Profile struct {
	ID         string      `json:"id"`
	CalendarID string      `json:"calendarId"`
	Name       string      `json:"name"`
	Type       ProfileType `json:"type"`
	IsActive   bool        `json:"isActive"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// NewProfile creates a new Profile with validation
func NewProfile(calendarID, name string, profileType ProfileType) (*Profile, error) {
	if calendarID == "" {
		return nil, errors.New("calendarID is required")
	}

	if name == "" {
		return nil, errors.New("name is required")
	}

	if profileType != ProfileTypePersonal && profileType != ProfileTypeBusiness {
		return nil, errors.New("invalid profile type")
	}

	now := time.Now()
	return &Profile{
		ID:         uuid.New().String(),
		CalendarID: calendarID,
		Name:       name,
		Type:       profileType,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Update updates the profile fields
func (p *Profile) Update(name string, profileType ProfileType) error {
	if name == "" {
		return errors.New("name is required")
	}

	if profileType != ProfileTypePersonal && profileType != ProfileTypeBusiness {
		return errors.New("invalid profile type")
	}

	p.Name = name
	p.Type = profileType
	p.UpdatedAt = time.Now()
	return nil
}

// Deactivate marks the profile as inactive
func (p *Profile) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now()
}

// Activate marks the profile as active
func (p *Profile) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now()
}
