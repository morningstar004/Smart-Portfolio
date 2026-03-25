# Guide Overview

This guide covers everything you need to run, test, build, deploy, and automate Smart Portfolio.

---

## What's in this Guide

| Section | What You'll Learn |
|---------|------------------|
| [Prerequisites](prerequisites.md) | Tools, API keys, and accounts you need before starting |
| [Running](running.md) | Three ways to run the stack: Docker Compose, local Go, and live reload |
| [Testing](testing.md) | Unit tests, coverage reports, benchmarks, race detector, and manual curl testing |
| [Building](building.md) | Compiling binaries for all platforms and building the Docker image |
| [Deploying](deploying.md) | Deploying to Railway, Render, Fly.io, or a VPS with systemd |
| [CI/CD](cicd.md) | GitHub Actions pipelines for automated testing, releases, and Docker pushes |

---

## Quick Reference

| Task | Command |
|------|---------|
| Start backend (Docker) | `cd backend && docker compose up -d --build` |
| Run locally | `cd backend && make run` |
| Live reload dev server | `cd backend && make dev` |
| Run tests | `cd backend && make test` |
| Run tests with coverage | `cd backend && make cover` |
| Run benchmarks | `cd backend && make bench` |
| Lint code | `cd backend && make lint` |
| Format code | `cd backend && make fmt` |
| Full pre-commit check | `cd backend && make check` |
| Build binary | `cd backend && make build` |
| Build Linux binary | `cd backend && make build-linux` |
| Build Docker image | `cd backend && make docker-build` |
| View logs | `cd backend && docker compose logs -f app` |
| Stop everything | `cd backend && docker compose down` |
| Clean build artifacts | `cd backend && make clean` |
| Tag a release | `git tag v1.0.0 && git push origin v1.0.0` |
| Open Swagger UI | `http://localhost:8080/docs` |
| Health check | `curl http://localhost:8080/healthz` |

---

## Server Endpoints at a Glance

Once running, the server exposes:

| URL | What |
|-----|------|
| `http://localhost:8080/healthz` | Liveness probe |
| `http://localhost:8080/docs` | Interactive Swagger UI |
| `http://localhost:8080/api/projects` | Projects CRUD |
| `http://localhost:8080/api/contact` | Contact form |
| `http://localhost:8080/api/chat` | AI chat (JSON) |
| `http://localhost:8080/api/chat/stream` | AI chat (SSE streaming) |
| `http://localhost:8080/api/ingest` | Resume PDF ingestion (admin) |
| `http://localhost:8080/api/admin/stats` | Dashboard statistics (admin) |
| `http://localhost:8080/api/webhooks/razorpay` | Razorpay payment webhook |
