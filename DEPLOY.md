# Deploy to Render (easiest cloud option)

## Prerequisites

- Free [GitHub](https://github.com) + [Render](https://render.com) accounts
- Repo: https://github.com/krokodiro/trello-clone

## 1. Create services (one time)

Open **[Create Blueprint](https://dashboard.render.com/blueprint/new?repo=https://github.com/krokodiro/trello-clone)** → connect GitHub → **Apply**.

Wait until **trello-api** and **trello-web** both show **Live**.

## 2. Set three environment variables (required)

Copy your public URLs from the Render dashboard (no trailing slash).

| Service | Variable | Example value |
|---------|----------|---------------|
| **trello-api** | `WEB_URL` | `https://trello-web-xxxx.onrender.com` |
| **trello-web** | `API_PUBLIC_URL` | `https://trello-api-xxxx.onrender.com` |
| **trello-web** | `WS_PUBLIC_URL` | `wss://trello-api-xxxx.onrender.com` |

Save each service — Render redeploys automatically.

## 3. Verify

1. Open `https://YOUR-trello-web.onrender.com/api/config`  
   You should see `"mode":"direct"` and your API URL.
2. Open `https://YOUR-trello-api.onrender.com/health`  
   Should return `{"status":"ok"}`.
3. Register or sign in on the web URL.

## Email verification (SMTP)

Registration sends a verification email. Without SMTP configured, the API logs `[email] SMTP not configured` and the **verify-email** page shows a direct verification link instead.

To send real emails on Render, add these on **trello-api** → **Environment**:

| Variable | Example (Resend) |
|----------|------------------|
| `SMTP_HOST` | `smtp.resend.com` |
| `SMTP_PORT` | `587` |
| `SMTP_USER` | `resend` |
| `SMTP_PASSWORD` | your Resend API key |
| `SMTP_FROM` | `Your App <onboarding@yourdomain.com>` |

Use a verified sender domain in Resend (or another SMTP provider). Redeploy **trello-api** after saving.

## Admin login

**trello-api** → **Environment** → `ADMIN_EMAIL` / `ADMIN_PASSWORD` (auto-generated).

## Troubleshooting

- **Register/login fails / CORS error** → `WEB_URL` on API must exactly match your web URL (https, no trailing `/`).
- **`/api/config` shows `"mode":"proxy"`** → set `API_PUBLIC_URL` on trello-web and redeploy.
- **WebSockets / live updates broken** → set `WS_PUBLIC_URL` to `wss://` + same host as API.
- **Free tier cold start** → first request after idle can take ~30s.
- **Can't verify email after register** → configure SMTP above, or use the link shown on the verify-email page when SMTP is off.

## Local production

```powershell
docker compose -f docker-compose.prod.yml up --build -d
```

App: http://localhost:3000 (uses internal proxy; no extra env needed)
