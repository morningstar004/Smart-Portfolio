package model

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID             uuid.UUID `json:"id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	PrimaryRole    string    `json:"primary_role"`
	Specialization string    `json:"specialization"`
	Location       string    `json:"location"`
	Summary        string    `json:"summary"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Education struct {
	ID          uuid.UUID `json:"id"`
	Institution string    `json:"institution"`
	Degree      string    `json:"degree"`
	Location    string    `json:"location"`
	StartDate   string    `json:"start_date"`
	EndDate     string    `json:"end_date"`
	GPA         string    `json:"gpa"`
	Coursework  string    `json:"coursework"`
	CreatedAt   time.Time `json:"created_at"`
}

type Experience struct {
	ID        uuid.UUID `json:"id"`
	Company   string    `json:"company"`
	Role      string    `json:"role"`
	Location  string    `json:"location"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Summary   string    `json:"summary"`
	TechStack string    `json:"tech_stack"`
	CreatedAt time.Time `json:"created_at"`
}

type Certification struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Issuer    string    `json:"issuer"`
	IssueDate string    `json:"issue_date"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type Achievement struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Metric      string    `json:"metric"`
	Description string    `json:"description"`
	Date        string    `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

type Skill struct {
	ID        uuid.UUID `json:"id"`
	Category  string    `json:"category"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
