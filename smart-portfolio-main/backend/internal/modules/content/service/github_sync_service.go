package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/morningstar004/smart-portfolio/internal/config"
	airepository "github.com/morningstar004/smart-portfolio/internal/modules/ai/repository"
	aiservice "github.com/morningstar004/smart-portfolio/internal/modules/ai/service"
	"github.com/morningstar004/smart-portfolio/internal/modules/content/model"
	contentrepo "github.com/morningstar004/smart-portfolio/internal/modules/content/repository"
	"github.com/rs/zerolog/log"
)

const (
	githubGraphQLEndpoint = "https://api.github.com/graphql"
	githubRESTEndpoint    = "https://api.github.com"
	maxPassiveSyncAge     = 10 * time.Minute
)

type GitHubSyncService interface {
	Enabled() bool
	Username() string
	Sync(ctx context.Context, force bool) error
}

type githubSyncService struct {
	cfg             config.GitHubConfig
	profileRepo     *contentrepo.GitHubProfileRepository
	repoRepo        *contentrepo.GitHubRepositoryRepository
	embeddingRepo   *airepository.GitHubEmbeddingRepository
	embeddingSvc    aiservice.EmbeddingService
	httpClient      *http.Client
	invalidateCache func()
}

func NewGitHubSyncService(
	cfg config.GitHubConfig,
	profileRepo *contentrepo.GitHubProfileRepository,
	repoRepo *contentrepo.GitHubRepositoryRepository,
	embeddingRepo *airepository.GitHubEmbeddingRepository,
	embeddingSvc aiservice.EmbeddingService,
	invalidateCache func(),
) GitHubSyncService {
	if cfg.ProjectsLimit <= 0 {
		cfg.ProjectsLimit = 6
	}
	if cfg.CandidateLimit <= 0 {
		cfg.CandidateLimit = 40
	}
	if cfg.CandidateLimit > 100 {
		cfg.CandidateLimit = 100
	}
	if cfg.SyncInterval <= 0 {
		cfg.SyncInterval = 24 * time.Hour
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}

	return &githubSyncService{
		cfg:             cfg,
		profileRepo:     profileRepo,
		repoRepo:        repoRepo,
		embeddingRepo:   embeddingRepo,
		embeddingSvc:    embeddingSvc,
		httpClient:      &http.Client{Timeout: cfg.Timeout},
		invalidateCache: invalidateCache,
	}
}

func (s *githubSyncService) Enabled() bool {
	return strings.TrimSpace(s.cfg.Username) != "" && strings.TrimSpace(s.cfg.Token) != ""
}

func (s *githubSyncService) Username() string {
	return strings.ToLower(strings.TrimSpace(s.cfg.Username))
}

func (s *githubSyncService) Sync(ctx context.Context, force bool) error {
	if !s.Enabled() {
		return nil
	}

	username := s.Username()
	if !force {
		profile, err := s.profileRepo.GetByUsername(ctx, username)
		if err != nil {
			return fmt.Errorf("github_sync_service.Sync: load profile: %w", err)
		}
		effectiveMaxAge := s.cfg.SyncInterval
		if effectiveMaxAge <= 0 || effectiveMaxAge > maxPassiveSyncAge {
			effectiveMaxAge = maxPassiveSyncAge
		}
		if profile != nil && profile.LastSyncedAt != nil && time.Since(*profile.LastSyncedAt) < effectiveMaxAge {
			return nil
		}
	}

	profileData, rate, err := s.fetchGitHubData(ctx, username)
	if err != nil {
		return fmt.Errorf("github_sync_service.Sync: fetch github data: %w", err)
	}

	selected := selectTopRepositories(profileData.PinnedRepos, profileData.Repositories, s.cfg.ProjectsLimit)
	selectedIDs := make([]int64, 0, len(selected))
	for _, repo := range selected {
		selectedIDs = append(selectedIDs, repo.DatabaseID)
	}

	existingRepos, err := s.repoRepo.FindByGitHubIDs(ctx, selectedIDs)
	if err != nil {
		return fmt.Errorf("github_sync_service.Sync: load existing repos: %w", err)
	}

	now := time.Now().UTC()
	storedRepos := make([]model.GitHubRepository, 0, len(selected))
	embeddingInputs := make([]string, 0, len(selected)+1)
	embeddingDocsMeta := make([]airepository.GitHubEmbeddingDocument, 0, len(selected)+1)
	keepEmbeddingKeys := make([]string, 0, len(selected)+1)

	for _, repo := range selected {
		existing, found := existingRepos[repo.DatabaseID]
		readmeSummary, readmeSHA := existing.ReadmeSummary, existing.ReadmeSHA

		if !found || existing.GitHubUpdatedAt.Before(repo.UpdatedAt) || existing.ReadmeSummary == nil {
			summary, sha, readmeErr := s.fetchRepositoryReadmeSummary(ctx, repo.OwnerLogin, repo.Name)
			if readmeErr != nil {
				log.Warn().Err(readmeErr).Str("repo", repo.NameWithOwner).Msg("github_sync_service: falling back to repository description for README summary")
			} else {
				readmeSummary = summary
				readmeSHA = sha
			}
		}

		description := chooseRepositoryDescription(repo.Description, readmeSummary)
		techStack := buildTechStack(repo.PrimaryLanguage, repo.Topics)

		storedRepo := model.GitHubRepository{
			GitHubRepoID:    repo.DatabaseID,
			Username:        username,
			OwnerLogin:      repo.OwnerLogin,
			Name:            repo.Name,
			FullName:        repo.NameWithOwner,
			Description:     description,
			ReadmeSummary:   readmeSummary,
			TechStack:       techStack,
			GitHubURL:       repo.URL,
			HomepageURL:     repo.HomepageURL,
			PrimaryLanguage: repo.PrimaryLanguage,
			Stars:           repo.Stars,
			Forks:           repo.Forks,
			Watchers:        repo.Watchers,
			IsPinned:        repo.IsPinned,
			IsArchived:      repo.IsArchived,
			GitHubUpdatedAt: repo.UpdatedAt,
			PushedAt:        contentrepo.PtrTime(repo.PushedAt),
			ReadmeSHA:       readmeSHA,
		}
		storedRepos = append(storedRepos, storedRepo)

		entityKey := fmt.Sprintf("repo:%d", repo.DatabaseID)
		keepEmbeddingKeys = append(keepEmbeddingKeys, entityKey)
		embeddingInputs = append(embeddingInputs, buildRepositoryEmbeddingContent(profileData.Profile, repo, description, techStack))
		repoID := repo.DatabaseID
		embeddingDocsMeta = append(embeddingDocsMeta, airepository.GitHubEmbeddingDocument{
			EntityKey:    entityKey,
			Username:     username,
			EntityType:   "repository",
			GitHubRepoID: &repoID,
			Metadata: map[string]string{
				"repo_id":   fmt.Sprintf("%d", repo.DatabaseID),
				"name":      repo.Name,
				"full_name": repo.NameWithOwner,
				"url":       repo.URL,
			},
		})
	}

	profileModel := &model.GitHubProfile{
		Username:           username,
		DisplayName:        emptyToNil(profileData.Profile.Name),
		Bio:                emptyToNil(profileData.Profile.Bio),
		ProfileURL:         profileData.Profile.URL,
		RepositoriesURL:    profileData.Profile.RepositoriesURL,
		AvatarURL:          emptyToNil(profileData.Profile.AvatarURL),
		LastSyncedAt:       &now,
		LastError:          nil,
		RateLimitRemaining: &rate.Remaining,
		RateLimitResetAt:   contentrepo.PtrTime(rate.ResetAt),
	}

	profileEmbeddingKey := "profile:" + username
	keepEmbeddingKeys = append(keepEmbeddingKeys, profileEmbeddingKey)
	embeddingInputs = append(embeddingInputs, buildProfileEmbeddingContent(profileData.Profile))
	embeddingDocsMeta = append(embeddingDocsMeta, airepository.GitHubEmbeddingDocument{
		EntityKey:    profileEmbeddingKey,
		Username:     username,
		EntityType:   "profile",
		GitHubRepoID: nil,
		Metadata: map[string]string{
			"username":         username,
			"profile_url":      profileData.Profile.URL,
			"repositories_url": profileData.Profile.RepositoriesURL,
		},
	})

	embeddings, err := s.embeddingSvc.EmbedBatch(ctx, embeddingInputs)
	if err != nil {
		return fmt.Errorf("github_sync_service.Sync: embed github content: %w", err)
	}
	for i := range embeddingDocsMeta {
		embeddingDocsMeta[i].Content = embeddingInputs[i]
		embeddingDocsMeta[i].Embedding = embeddings[i]
	}

	if err := s.profileRepo.Upsert(ctx, profileModel); err != nil {
		return fmt.Errorf("github_sync_service.Sync: save profile: %w", err)
	}
	if err := s.repoRepo.UpsertMany(ctx, storedRepos); err != nil {
		return fmt.Errorf("github_sync_service.Sync: save repositories: %w", err)
	}
	if err := s.repoRepo.DeleteMissingByUsername(ctx, username, selectedIDs); err != nil {
		return fmt.Errorf("github_sync_service.Sync: prune repositories: %w", err)
	}
	if err := s.embeddingRepo.UpsertMany(ctx, embeddingDocsMeta); err != nil {
		return fmt.Errorf("github_sync_service.Sync: save embeddings: %w", err)
	}
	if err := s.embeddingRepo.DeleteMissingByUsername(ctx, username, keepEmbeddingKeys); err != nil {
		return fmt.Errorf("github_sync_service.Sync: prune embeddings: %w", err)
	}

	if s.invalidateCache != nil {
		s.invalidateCache()
	}

	log.Info().
		Str("username", username).
		Int("repositories", len(storedRepos)).
		Int("rate_limit_remaining", rate.Remaining).
		Msg("github_sync_service: GitHub works synchronized")

	return nil
}

type githubRateLimit struct {
	Remaining int
	ResetAt   time.Time
}

type githubProfileData struct {
	Profile      githubProfile
	PinnedRepos  []githubRepositoryNode
	Repositories []githubRepositoryNode
}

type githubProfile struct {
	Login           string
	Name            string
	Bio             string
	URL             string
	AvatarURL       string
	RepositoriesURL string
}

type githubRepositoryNode struct {
	DatabaseID      int64
	Name            string
	NameWithOwner   string
	OwnerLogin      string
	URL             string
	HomepageURL     *string
	Description     string
	Stars           int
	Forks           int
	Watchers        int
	IsArchived      bool
	IsPinned        bool
	PrimaryLanguage *string
	Topics          []string
	UpdatedAt       time.Time
	PushedAt        time.Time
}

func (s *githubSyncService) fetchGitHubData(ctx context.Context, username string) (*githubProfileData, githubRateLimit, error) {
	query := `
	query($login: String!, $candidateLimit: Int!, $pinnedLimit: Int!) {
	  user(login: $login) {
	    login
	    name
	    bio
	    url
	    avatarUrl
	    pinnedItems(first: $pinnedLimit, types: REPOSITORY) {
	      nodes {
	        ... on Repository {
	          databaseId
	          name
	          nameWithOwner
	          url
	          homepageUrl
	          description
	          stargazerCount
	          forkCount
	          watchers { totalCount }
	          isArchived
	          primaryLanguage { name }
	          repositoryTopics(first: 8) { nodes { topic { name } } }
	          updatedAt
	          pushedAt
	          owner { login }
	        }
	      }
	    }
	    repositories(first: $candidateLimit, privacy: PUBLIC, ownerAffiliations: OWNER, isFork: false, orderBy: {field: PUSHED_AT, direction: DESC}) {
	      nodes {
	        databaseId
	        name
	        nameWithOwner
	        url
	        homepageUrl
	        description
	        stargazerCount
	        forkCount
	        watchers { totalCount }
	        isArchived
	        primaryLanguage { name }
	        repositoryTopics(first: 8) { nodes { topic { name } } }
	        updatedAt
	        pushedAt
	        owner { login }
	      }
	    }
	  }
	  rateLimit {
	    cost
	    remaining
	    resetAt
	  }
	}`

	reqBody := map[string]any{
		"query": query,
		"variables": map[string]any{
			"login":          username,
			"candidateLimit": s.cfg.CandidateLimit,
			"pinnedLimit":    minInt(s.cfg.ProjectsLimit, 6),
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, githubRateLimit{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubGraphQLEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, githubRateLimit{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, githubRateLimit{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, githubRateLimit{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, githubRateLimit{}, fmt.Errorf("github GraphQL status %d: %s", resp.StatusCode, truncateString(string(respBody), 512))
	}

	var graphResp struct {
		Data struct {
			User struct {
				Login       string `json:"login"`
				Name        string `json:"name"`
				Bio         string `json:"bio"`
				URL         string `json:"url"`
				AvatarURL   string `json:"avatarUrl"`
				PinnedItems struct {
					Nodes []githubRepoNodeGraph `json:"nodes"`
				} `json:"pinnedItems"`
				Repositories struct {
					Nodes []githubRepoNodeGraph `json:"nodes"`
				} `json:"repositories"`
			} `json:"user"`
			RateLimit struct {
				Remaining int       `json:"remaining"`
				ResetAt   time.Time `json:"resetAt"`
			} `json:"rateLimit"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &graphResp); err != nil {
		return nil, githubRateLimit{}, err
	}
	if len(graphResp.Errors) > 0 {
		return nil, githubRateLimit{}, fmt.Errorf("github GraphQL error: %s", graphResp.Errors[0].Message)
	}

	data := &githubProfileData{
		Profile: githubProfile{
			Login:           graphResp.Data.User.Login,
			Name:            graphResp.Data.User.Name,
			Bio:             graphResp.Data.User.Bio,
			URL:             graphResp.Data.User.URL,
			AvatarURL:       graphResp.Data.User.AvatarURL,
			RepositoriesURL: buildRepositoriesURL(graphResp.Data.User.URL),
		},
		PinnedRepos:  make([]githubRepositoryNode, 0, len(graphResp.Data.User.PinnedItems.Nodes)),
		Repositories: make([]githubRepositoryNode, 0, len(graphResp.Data.User.Repositories.Nodes)),
	}
	for _, node := range graphResp.Data.User.PinnedItems.Nodes {
		data.PinnedRepos = append(data.PinnedRepos, node.toRepository(true))
	}
	for _, node := range graphResp.Data.User.Repositories.Nodes {
		data.Repositories = append(data.Repositories, node.toRepository(false))
	}

	rate := githubRateLimit{
		Remaining: graphResp.Data.RateLimit.Remaining,
		ResetAt:   graphResp.Data.RateLimit.ResetAt,
	}
	return data, rate, nil
}

type githubRepoNodeGraph struct {
	DatabaseID     int64   `json:"databaseId"`
	Name           string  `json:"name"`
	NameWithOwner  string  `json:"nameWithOwner"`
	URL            string  `json:"url"`
	HomepageURL    *string `json:"homepageUrl"`
	Description    string  `json:"description"`
	StargazerCount int     `json:"stargazerCount"`
	ForkCount      int     `json:"forkCount"`
	Watchers       struct {
		TotalCount int `json:"totalCount"`
	} `json:"watchers"`
	IsArchived      bool `json:"isArchived"`
	PrimaryLanguage *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	RepositoryTopics struct {
		Nodes []struct {
			Topic struct {
				Name string `json:"name"`
			} `json:"topic"`
		} `json:"nodes"`
	} `json:"repositoryTopics"`
	UpdatedAt time.Time `json:"updatedAt"`
	PushedAt  time.Time `json:"pushedAt"`
	Owner     struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (n githubRepoNodeGraph) toRepository(isPinned bool) githubRepositoryNode {
	topics := make([]string, 0, len(n.RepositoryTopics.Nodes))
	for _, topicNode := range n.RepositoryTopics.Nodes {
		if topicNode.Topic.Name != "" {
			topics = append(topics, topicNode.Topic.Name)
		}
	}

	var primaryLanguage *string
	if n.PrimaryLanguage != nil && n.PrimaryLanguage.Name != "" {
		primaryLanguage = &n.PrimaryLanguage.Name
	}

	return githubRepositoryNode{
		DatabaseID:      n.DatabaseID,
		Name:            n.Name,
		NameWithOwner:   n.NameWithOwner,
		OwnerLogin:      n.Owner.Login,
		URL:             n.URL,
		HomepageURL:     n.HomepageURL,
		Description:     strings.TrimSpace(n.Description),
		Stars:           n.StargazerCount,
		Forks:           n.ForkCount,
		Watchers:        n.Watchers.TotalCount,
		IsArchived:      n.IsArchived,
		IsPinned:        isPinned,
		PrimaryLanguage: primaryLanguage,
		Topics:          topics,
		UpdatedAt:       n.UpdatedAt,
		PushedAt:        n.PushedAt,
	}
}

func (s *githubSyncService) fetchRepositoryReadmeSummary(ctx context.Context, owner, repo string) (*string, *string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/readme", githubRESTEndpoint, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("github README status %d: %s", resp.StatusCode, truncateString(string(body), 256))
	}

	var readme struct {
		SHA      string `json:"sha"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(body, &readme); err != nil {
		return nil, nil, err
	}
	if strings.ToLower(readme.Encoding) != "base64" || strings.TrimSpace(readme.Content) == "" {
		return nil, emptyToNil(readme.SHA), nil
	}

	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(readme.Content, "\n", ""))
	if err != nil {
		return nil, nil, err
	}
	summary := summariseREADME(string(raw))
	return emptyToNil(summary), emptyToNil(readme.SHA), nil
}

func selectTopRepositories(pinned, repos []githubRepositoryNode, limit int) []githubRepositoryNode {
	if limit <= 0 {
		return nil
	}

	byID := make(map[int64]githubRepositoryNode, len(pinned)+len(repos))
	for _, repo := range repos {
		if repo.IsArchived {
			continue
		}
		byID[repo.DatabaseID] = repo
	}
	for _, repo := range pinned {
		if repo.IsArchived {
			continue
		}
		repo.IsPinned = true
		byID[repo.DatabaseID] = repo
	}

	all := make([]githubRepositoryNode, 0, len(byID))
	for _, repo := range byID {
		all = append(all, repo)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].IsPinned != all[j].IsPinned {
			return all[i].IsPinned
		}
		if all[i].Stars != all[j].Stars {
			return all[i].Stars > all[j].Stars
		}
		if all[i].Forks != all[j].Forks {
			return all[i].Forks > all[j].Forks
		}
		return all[i].PushedAt.After(all[j].PushedAt)
	})

	if len(all) > limit {
		all = all[:limit]
	}
	return all
}

func buildTechStack(primaryLanguage *string, topics []string) *string {
	seen := map[string]struct{}{}
	parts := make([]string, 0, 1+len(topics))
	appendPart := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		parts = append(parts, value)
	}

	if primaryLanguage != nil {
		appendPart(*primaryLanguage)
	}
	for _, topic := range topics {
		appendPart(topic)
		if len(parts) >= 8 {
			break
		}
	}

	if len(parts) == 0 {
		return nil
	}
	value := strings.Join(parts, ", ")
	return &value
}

func buildProfileEmbeddingContent(profile githubProfile) string {
	return strings.TrimSpace(strings.Join([]string{
		"GitHub profile",
		"profile: " + profile.Login,
		"name: " + strings.TrimSpace(profile.Name),
		"bio: " + strings.TrimSpace(profile.Bio),
		"url: " + profile.URL,
		"repositories: " + profile.RepositoriesURL,
	}, "\n"))
}

func buildRepositoriesURL(profileURL string) string {
	profileURL = strings.TrimRight(strings.TrimSpace(profileURL), "/")
	if profileURL == "" {
		return ""
	}
	return profileURL + "?tab=repositories"
}

func buildRepositoryEmbeddingContent(profile githubProfile, repo githubRepositoryNode, description string, techStack *string) string {
	var parts []string
	parts = append(parts, "GitHub repository")
	parts = append(parts, "owner: "+profile.Login)
	parts = append(parts, "repository: "+repo.NameWithOwner)
	if description != "" {
		parts = append(parts, "summary: "+description)
	}
	if techStack != nil {
		parts = append(parts, "tech_stack: "+*techStack)
	}
	if repo.URL != "" {
		parts = append(parts, "url: "+repo.URL)
	}
	return strings.Join(parts, "\n")
}

func chooseRepositoryDescription(repoDescription string, readmeSummary *string) string {
	if readmeSummary != nil && strings.TrimSpace(*readmeSummary) != "" {
		return *readmeSummary
	}
	if strings.TrimSpace(repoDescription) != "" {
		return strings.TrimSpace(repoDescription)
	}
	return "Public repository synced from GitHub."
}

var markdownLinkPattern = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
var markdownFormattingPattern = regexp.MustCompile("[*_`>#-]")
var multiWhitespacePattern = regexp.MustCompile(`\s+`)

func summariseREADME(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	var blocks []string
	var current []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, " "))
				current = nil
			}
			continue
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "![") ||
			strings.HasPrefix(trimmed, "<img") ||
			strings.HasPrefix(trimmed, "[![") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, "---") ||
			strings.HasPrefix(trimmed, "##") ||
			strings.Contains(lower, "license") && len(trimmed) < 40 {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, " "))
				current = nil
			}
			continue
		}

		current = append(current, trimmed)
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, " "))
	}

	for _, block := range blocks {
		block = markdownLinkPattern.ReplaceAllString(block, "$1")
		block = markdownFormattingPattern.ReplaceAllString(block, " ")
		block = multiWhitespacePattern.ReplaceAllString(strings.TrimSpace(block), " ")
		if len(block) < 30 {
			continue
		}
		if len(block) > 220 {
			block = strings.TrimSpace(block[:220]) + "..."
		}
		return block
	}

	return ""
}

func truncateString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func emptyToNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
