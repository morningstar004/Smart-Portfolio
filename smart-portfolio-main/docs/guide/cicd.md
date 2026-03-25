# CI/CD Pipelines

## Repository Structure

The workflows are already created in your project:

```
smart-portfolio/
├── .github/
│   └── workflows/
│       ├── ci.yml      ← Runs on every push and PR (lint, test, build)
│       └── cd.yml      ← Runs on version tags (build + push Docker image + GitHub Release)
├── backend/
├── frontend/
└── README.md
```

---

## CI Workflow (ci.yml)

**Triggers:** Every push to any branch, every pull request.

**Jobs:**

| Job | What It Does | Depends On |
|-----|-------------|------------|
| **lint** | `go mod tidy` check, `go vet`, `staticcheck`, `gofmt` check | — |
| **test** | `go test ./... -race -coverprofile=coverage.out` | lint |
| **build** | Compile binary, build Docker image, upload binary as artifact | test |

**What happens on a PR:**

1. Developer pushes a branch or opens a PR
2. GitHub Actions runs all three jobs sequentially (lint → test → build)
3. If any job fails, the PR gets a red ❌
4. If all pass, the PR gets a green ✅
5. Coverage report is uploaded as an artifact (downloadable for 14 days)
6. The compiled binary is uploaded as an artifact (downloadable for 7 days)

---

## CD Workflow (cd.yml)

**Triggers:** Pushing a tag that matches `v*` (e.g., `v1.0.0`, `v1.2.3-beta`).

**Jobs:**

| Job | What It Does | Depends On |
|-----|-------------|------------|
| **test** | Full test suite with race detector (safety gate) | — |
| **release** | Build 5 platform binaries, build + push multi-arch Docker image to GHCR, create GitHub Release | test |

**What it produces:**

| Output | Location |
|--------|----------|
| Docker image (amd64 + arm64) | `ghcr.io/<your-username>/smart-portfolio:1.0.0` |
| Docker image (latest) | `ghcr.io/<your-username>/smart-portfolio:latest` |
| Linux amd64 binary | GitHub Release assets |
| Linux arm64 binary | GitHub Release assets |
| macOS Apple Silicon binary | GitHub Release assets |
| macOS Intel binary | GitHub Release assets |
| Windows amd64 binary | GitHub Release assets |
| Auto-generated release notes | GitHub Release body |

---

## Required GitHub Setup

**Step 1: Push the workflows to GitHub**

```bash
# Make sure you're in the project root
cd smart-portfolio

# Initialize git if not already done
git init
git add .
git commit -m "initial commit"

# Add your GitHub remote
git remote add origin https://github.com/YOUR_USERNAME/smart-portfolio.git

# Push
git push -u origin main
```

**Step 2: Verify GitHub Actions is enabled**

1. Go to your repository on GitHub
2. Click **Settings** → **Actions** → **General**
3. Make sure **"Allow all actions and reusable workflows"** is selected
4. Under **"Workflow permissions"**, select **"Read and write permissions"**
5. Click **Save**

The "Read and write permissions" setting is needed because:
- The CD workflow pushes Docker images to GitHub Container Registry (GHCR) using `GITHUB_TOKEN`
- The CD workflow creates GitHub Releases using `GITHUB_TOKEN`

**Step 3: No additional secrets needed!**

Both workflows use only `GITHUB_TOKEN`, which GitHub provides automatically. No need to create PATs or add Docker Hub credentials.

If you want to add deployment secrets for auto-deploy (see below), you would add those under **Settings** → **Secrets and variables** → **Actions**.

---

## Creating a Release

The CD pipeline triggers when you push a version tag. Here's the full flow:

```bash
# 1. Make sure all your changes are committed and pushed
git add .
git commit -m "feat: add new endpoint"
git push origin main

# 2. Create a version tag
git tag v1.0.0

# 3. Push the tag — this triggers the CD workflow
git push origin v1.0.0
```

**Tag naming conventions:**

| Tag | Type | Docker Tags Generated |
|-----|------|----------------------|
| `v1.0.0` | Stable release | `1.0.0`, `1.0`, `1`, `latest` |
| `v1.0.1` | Patch release | `1.0.1`, `1.0`, `1`, `latest` |
| `v2.0.0` | Major release | `2.0.0`, `2.0`, `2`, `latest` |
| `v1.1.0-beta` | Pre-release | `1.1.0-beta` (marked as pre-release on GitHub) |
| `v1.1.0-rc.1` | Release candidate | `1.1.0-rc.1` (marked as pre-release) |

**After the CD workflow completes (~3-5 minutes):**

1. A **GitHub Release** appears at `github.com/YOUR_USERNAME/smart-portfolio/releases/tag/v1.0.0`
2. The release has 5 downloadable binaries attached
3. Auto-generated release notes show all commits since the last tag
4. The **Docker image** is available at `ghcr.io/YOUR_USERNAME/smart-portfolio:1.0.0`

**Pull the Docker image:**

```bash
docker pull ghcr.io/YOUR_USERNAME/smart-portfolio:1.0.0
docker pull ghcr.io/YOUR_USERNAME/smart-portfolio:latest
```

---

## Adding Deploy-on-Push to a Platform

To auto-deploy to Railway, Render, or Fly.io after the Docker image is pushed, add a deploy job to `cd.yml`.

**Example: Railway deploy webhook**

1. In Railway, go to your service → **Settings** → **Deploy** → copy the **Deploy Webhook URL**
2. Add it as a GitHub secret: **Settings** → **Secrets** → `RAILWAY_WEBHOOK_URL`
3. Add this job to `.github/workflows/cd.yml` after the `release` job:

```yaml
  deploy:
    name: Deploy to Railway
    runs-on: ubuntu-latest
    needs: release
    steps:
      - name: Trigger Railway deploy
        run: |
          curl -X POST "${{ secrets.RAILWAY_WEBHOOK_URL }}" \
            --fail --silent --show-error
```

**Example: Render deploy hook**

Same concept — Render gives you a deploy hook URL under **Settings** → **Deploy Hook**.

```yaml
  deploy:
    name: Deploy to Render
    runs-on: ubuntu-latest
    needs: release
    steps:
      - name: Trigger Render deploy
        run: |
          curl "${{ secrets.RENDER_DEPLOY_HOOK_URL }}" \
            --fail --silent --show-error
```

**Example: Fly.io deploy**

```yaml
  deploy:
    name: Deploy to Fly.io
    runs-on: ubuntu-latest
    needs: release
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Fly CLI
        uses: superfly/flyctl-actions/setup-flyctl@master

      - name: Deploy to Fly.io
        working-directory: backend
        run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

For Fly.io, create a deploy token: `fly tokens create deploy --app smart-portfolio`, then add it as the `FLY_API_TOKEN` secret in GitHub.

---

## Branch Protection Rules

To enforce that CI passes before merging PRs:

1. Go to **Settings** → **Branches** → **Add branch protection rule**
2. **Branch name pattern**: `main`
3. Check these boxes:
   - ✅ **Require a pull request before merging**
   - ✅ **Require status checks to pass before merging**
     - Search and add: `Lint`, `Test`, `Build`
   - ✅ **Require branches to be up to date before merging**
   - ✅ **Do not allow bypassing the above settings** (optional but recommended)
4. Click **Create**

Now, any PR targeting `main` must pass all three CI jobs before it can be merged. No one can push directly to `main` or merge with failing tests.
