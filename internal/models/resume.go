package models

import (
	"time"

	"github.com/google/uuid"
)

// Resume represents a curriculum vitae owned by a user.
type Resume struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Title         string
	TemplateName  string
	PersonalName  string
	PersonalTitle string
	Email         string
	Phone         string
	Location      string
	Summary       string
	PhotoURL      string
	ShareToken    *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ResumeDetail is a Resume with all associated child entities loaded.
type ResumeDetail struct {
	Resume
	Experience []Experience
	Education  []Education
	Skills     []Skill
}

// Experience represents a professional experience entry within a resume.
type Experience struct {
	ID           uuid.UUID
	ResumeID     uuid.UUID
	Company      string
	Role         string
	Period       string
	Description  string
	DisplayOrder int
}

// Education represents an education entry within a resume.
type Education struct {
	ID           uuid.UUID
	ResumeID     uuid.UUID
	Institution  string
	Degree       string
	Period       string
	DisplayOrder int
}

// Skill represents a skill entry within a resume.
type Skill struct {
	ID           uuid.UUID
	ResumeID     uuid.UUID
	Name         string
	DisplayOrder int
}
