package model

import "time"

// GitHubProfile stores the public account details used by the works section.
type GitHubProfile struct {
	Username           string
	DisplayName        *string
	Bio                *string
	ProfileURL         string
	RepositoriesURL    string
	AvatarURL          *string
	LastSyncedAt       *time.Time
	LastError          *string
	RateLimitRemaining *int
	RateLimitResetAt   *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// GitHubRepository stores a selected GitHub repository that is safe to show in
// the public works section and compact enough to keep synchronized over time.
type GitHubRepository struct {
	ID              string
	GitHubRepoID    int64
	Username        string
	OwnerLogin      string
	Name            string
	FullName        string
	Description     string
	ReadmeSummary   *string
	TechStack       *string
	GitHubURL       string
	HomepageURL     *string
	PrimaryLanguage *string
	Stars           int
	Forks           int
	Watchers        int
	IsPinned        bool
	IsArchived      bool
	GitHubUpdatedAt time.Time
	PushedAt        *time.Time
	ReadmeSHA       *string
	SyncedAt        time.Time
	CreatedAt       time.Time
}
