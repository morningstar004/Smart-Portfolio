package worker

import (
	"context"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/modules/content/service"
	"github.com/rs/zerolog/log"
)

type GitHubSyncWorker struct {
	service  service.GitHubSyncService
	interval time.Duration
}

func NewGitHubSyncWorker(service service.GitHubSyncService, interval time.Duration) *GitHubSyncWorker {
	return &GitHubSyncWorker{
		service:  service,
		interval: interval,
	}
}

func (w *GitHubSyncWorker) Start(ctx context.Context) {
	if w == nil || w.service == nil || !w.service.Enabled() {
		return
	}

	go func() {
		w.syncOnce(ctx, false)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.syncOnce(ctx, false)
			}
		}
	}()
}

func (w *GitHubSyncWorker) syncOnce(ctx context.Context, force bool) {
	if err := w.service.Sync(ctx, force); err != nil {
		log.Error().Err(err).Msg("github_sync_worker: sync failed")
	}
}
