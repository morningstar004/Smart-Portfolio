# Deploying the Application

## What You Need in Production

| Component | What | Options |
|-----------|------|---------|
| **Go backend** | The Docker image or binary | Railway, Render, Fly.io, VPS |
| **PostgreSQL + pgvector** | Database with vector extension | Neon, Supabase, Railway Postgres, any PostgreSQL 15+ with pgvector |
| **Environment variables** | API keys and config | Platform's secret/env management |

**Architecture:**

```
Frontend (Vercel/Netlify)
         │
         │  HTTPS
         ▼
Go Backend (Railway/Render/Fly.io/VPS)
         │
         │  PostgreSQL protocol
         ▼
PostgreSQL + pgvector (Neon/Supabase/Railway)
```

---

## Option A: Railway

[Railway](https://railway.app) is the simplest — it detects the Dockerfile automatically.

**Step 1: Create the project**

1. Go to [railway.app](https://railway.app) and sign in with GitHub
2. Click **"New Project"** → **"Deploy from GitHub Repo"**
3. Select your `smart-portfolio` repository
4. Railway detects the Dockerfile in `backend/`. Set the **root directory** to `backend`

**Step 2: Add PostgreSQL**

1. In your Railway project, click **"+ New"** → **"Database"** → **"Add PostgreSQL"**
2. Railway gives you a `DATABASE_URL` automatically
3. **Enable pgvector**: Go to the PostgreSQL service → **Variables** tab → find the connection URL. Then connect via `psql` or the Railway CLI and run:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   ```
   Alternatively, the app's migration will try to create it on startup.

**Step 3: Set environment variables**

In the Go backend service, go to **Variables** and add:

```
DATABASE_URL=${{Postgres.DATABASE_URL}}    ← Railway auto-references the DB
GROQ_API_KEY=your-groq-key
JINA_API_KEY=your-jina-key
ADMIN_API_KEY=a-strong-random-secret
ENV=production
SERVER_PORT=8080
FRONTEND_URL=https://your-frontend-domain.vercel.app
```

**Step 4: Deploy**

Railway auto-deploys on every push to your default branch. The first deploy takes ~2 minutes (building the Docker image). Subsequent deploys use cached layers and take ~30 seconds.

**Step 5: Get your URL**

Railway assigns a URL like `smart-portfolio-production.up.railway.app`. Use this as your backend URL in the frontend's environment.

---

## Option B: Render

[Render](https://render.com) has a generous free tier.

**Step 1: Create a Web Service**

1. Go to [render.com](https://render.com) → **"New"** → **"Web Service"**
2. Connect your GitHub repository
3. Settings:
   - **Root Directory**: `backend`
   - **Runtime**: Docker
   - **Instance Type**: Free or Starter
   - **Health Check Path**: `/healthz`

**Step 2: Add PostgreSQL**

1. **"New"** → **"PostgreSQL"**
2. Note the **Internal Database URL** (use this for `DATABASE_URL` since both services are on Render's network)
3. Connect to the DB and enable pgvector:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   ```

**Step 3: Set environment variables**

In the Web Service settings → **Environment**:

```
DATABASE_URL=<internal-postgres-url>
GROQ_API_KEY=your-groq-key
JINA_API_KEY=your-jina-key
ADMIN_API_KEY=a-strong-random-secret
ENV=production
FRONTEND_URL=https://your-frontend.vercel.app
```

**Step 4: Deploy**

Render auto-deploys on push. Your URL will be `smart-portfolio.onrender.com`.

---

## Option C: Fly.io

[Fly.io](https://fly.io) runs containers close to your users.

```bash
# 1. Install flyctl
curl -L https://fly.io/install.sh | sh

# 2. Sign in
fly auth login

# 3. Navigate to backend
cd backend

# 4. Launch (creates fly.toml)
fly launch --no-deploy
#   - Choose a name: smart-portfolio
#   - Choose a region close to you
#   - Say NO to creating a database here (we'll do it separately)

# 5. Create a PostgreSQL cluster
fly postgres create --name smart-portfolio-db
fly postgres attach smart-portfolio-db --app smart-portfolio
# This sets DATABASE_URL automatically

# 6. Enable pgvector
fly postgres connect --app smart-portfolio-db
# Then in the psql prompt:
CREATE EXTENSION IF NOT EXISTS vector;
\q

# 7. Set secrets (environment variables)
fly secrets set \
  GROQ_API_KEY="your-groq-key" \
  JINA_API_KEY="your-jina-key" \
  ADMIN_API_KEY="a-strong-random-secret" \
  ENV="production" \
  FRONTEND_URL="https://your-frontend.vercel.app" \
  --app smart-portfolio

# 8. Deploy
fly deploy

# 9. Check it's running
fly status
fly logs
curl https://smart-portfolio.fly.dev/healthz
```

---

## Option D: VPS (DigitalOcean, Hetzner, AWS EC2)

For maximum control, run the binary directly on a VPS.

**Step 1: Provision a server**

- Ubuntu 22.04+ or Debian 12+
- At least 512 MB RAM, 1 vCPU
- Open ports: 22 (SSH), 80 (HTTP), 443 (HTTPS)

**Step 2: Install PostgreSQL with pgvector**

```bash
# On the VPS:
sudo apt update && sudo apt install -y postgresql postgresql-contrib

# Install pgvector
sudo apt install -y postgresql-16-pgvector

# Create the database and user
sudo -u postgres psql <<SQL
CREATE USER portfolio WITH PASSWORD 'your-strong-password';
CREATE DATABASE smart_portfolio OWNER portfolio;
\c smart_portfolio
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
SQL
```

**Step 3: Upload and run the binary**

```bash
# On your local machine: build the binary
cd backend
make build-linux

# Upload it to the server
scp bin/smart-portfolio-linux-amd64 user@your-server:/opt/smart-portfolio/server
scp -r migrations/ user@your-server:/opt/smart-portfolio/migrations/
```

**Step 4: Create a systemd service**

On the VPS, create `/etc/systemd/system/smart-portfolio.service`:

```ini
[Unit]
Description=Smart Portfolio Backend
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/smart-portfolio
ExecStart=/opt/smart-portfolio/server
Restart=always
RestartSec=5

# Environment variables
Environment=ENV=production
Environment=SERVER_PORT=8080
Environment=DATABASE_URL=postgres://portfolio:your-strong-password@localhost:5432/smart_portfolio?sslmode=disable
Environment=GROQ_API_KEY=your-groq-key
Environment=JINA_API_KEY=your-jina-key
Environment=ADMIN_API_KEY=your-admin-secret
Environment=FRONTEND_URL=https://your-frontend-domain.com

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadOnlyPaths=/opt/smart-portfolio

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable smart-portfolio
sudo systemctl start smart-portfolio

# Check status
sudo systemctl status smart-portfolio
sudo journalctl -u smart-portfolio -f
```

**Step 5: Set up a reverse proxy (Caddy or Nginx)**

Using **Caddy** (auto-HTTPS with Let's Encrypt):

```bash
sudo apt install -y caddy
```

Edit `/etc/caddy/Caddyfile`:

```
api.yourdomain.com {
    reverse_proxy localhost:8080
}
```

```bash
sudo systemctl restart caddy
```

Caddy automatically obtains and renews TLS certificates from Let's Encrypt. Your API is now live at `https://api.yourdomain.com`.

---

## Database Hosting

If you don't want to manage PostgreSQL yourself, use a managed service with pgvector support:

| Provider | pgvector Support | Free Tier | Notes |
|----------|-----------------|-----------|-------|
| **[Neon](https://neon.tech)** | Built-in | 0.5 GB storage | Serverless, scales to zero, branching |
| **[Supabase](https://supabase.com)** | Built-in | 500 MB storage | Also gives you PostgREST, Auth, Storage |
| **[Railway](https://railway.app)** | Need to enable | $5/month credit | One-click add to Railway projects |
| **[Tembo](https://tembo.io)** | Built-in | 1 GB storage | PostgreSQL-native platform |

After creating the database, enable pgvector if not already enabled:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

Then set the `DATABASE_URL` in your app's environment to the connection string the provider gives you. Make sure to use `sslmode=require` for remote databases.
