# Deploy to Render (easiest cloud option)

## Prerequisites

- A free [GitHub](https://github.com) account
- A free [Render](https://render.com) account

## Steps (about 5 minutes)

### 1. Push code to GitHub

Create a new repository on GitHub, then from this folder:

```powershell
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/YOUR_USERNAME/trello-clone.git
git push -u origin main
```

### 2. Deploy on Render

1. Open **[Create Blueprint](https://dashboard.render.com/blueprint/new?repo=https://github.com/krokodiro/trello-clone)**
2. Connect your GitHub account and select the `trello-clone` repository
3. Click **Apply** — Render creates Postgres, Redis, API, and Web services
4. Wait for all services to finish deploying (first build takes ~5–10 min)

If services already exist with wrong paths, open each web service → **Settings** → set **Root Directory** to `backend` or `frontend`, then **Manual Deploy**.

### 3. One-time config after first deploy

1. Open the **trello-web** service and copy its URL (e.g. `https://trello-web-xxxx.onrender.com`)
2. Open **trello-api** → **Environment** → set `WEB_URL` to that URL
3. Save — Render redeploys the API automatically

### 4. Sign in

Open your **trello-web** URL. Admin credentials are in the Render dashboard under **trello-api** → **Environment** (`ADMIN_EMAIL` / generated `ADMIN_PASSWORD`).

## Notes

- Free services spin down after inactivity; first load may take ~30 seconds
- Free Postgres expires after 90 days on Render — upgrade or export data for long-term use
- To redeploy after code changes: push to GitHub; Render auto-deploys

## Local production (alternative)

```powershell
docker compose -f docker-compose.prod.yml up --build -d
```

App: http://localhost:3000 · API: http://localhost:8080
