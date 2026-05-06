package service

import (
	"testing"
	"time"
)

func TestSummariseREADME(t *testing.T) {
	readme := `
# smart-portfolio

[![build](https://example.com/badge.svg)](https://example.com)

Smart Portfolio is a full-stack portfolio application with GitHub-synced works, AI chat, and payments.

## Features

- Thing one
`

	got := summariseREADME(readme)
	if got == "" {
		t.Fatal("expected non-empty summary")
	}
	if got[:15] != "Smart Portfolio" {
		t.Fatalf("expected README summary to start with content paragraph, got %q", got)
	}
}

func TestSelectTopRepositories(t *testing.T) {
	now := time.Now()
	repos := []githubRepositoryNode{
		{DatabaseID: 1, Name: "older-starred", Stars: 20, Forks: 1, PushedAt: now.Add(-48 * time.Hour)},
		{DatabaseID: 2, Name: "recent-mid", Stars: 10, Forks: 5, PushedAt: now.Add(-2 * time.Hour)},
	}
	pinned := []githubRepositoryNode{
		{DatabaseID: 3, Name: "pinned-low", Stars: 1, Forks: 0, PushedAt: now.Add(-72 * time.Hour), IsPinned: true},
	}

	selected := selectTopRepositories(pinned, repos, 2)
	if len(selected) != 2 {
		t.Fatalf("expected 2 selected repositories, got %d", len(selected))
	}
	if selected[0].DatabaseID != 3 {
		t.Fatalf("expected pinned repository first, got id %d", selected[0].DatabaseID)
	}
	if selected[1].DatabaseID != 1 {
		t.Fatalf("expected higher-star repository second, got id %d", selected[1].DatabaseID)
	}
}
