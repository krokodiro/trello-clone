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
3. API docs: `https://YOUR-trello-api.onrender.com/api/docs`
4. Register or sign in on the web URL.

## Email verification

Registration sends a verification email. Without email configured, the API shows a direct verification link on the verify-email page.

**Render blocks outbound SMTP (ports 25/587/465).** Gmail and other SMTP providers will fail with `connection timed out` on Render. Use the **Resend HTTP API** instead (works over HTTPS).

### Resend on Render (recommended)

On **trello-api** → **Environment**:

| Variable | Value |
|----------|--------|
| `RESEND_API_KEY` | your Resend API key (`re_…`) |
| `EMAIL_FROM` | `Trello Clone <onboarding@resend.dev>` (testing only — see below) |

Remove `SMTP_HOST` / Gmail vars if set — they are not needed and may confuse troubleshooting. Redeploy **trello-api**.

Logs should show `[email] using Resend HTTP API` and `[email] sent (resend) to …`.

**Resend `@resend.dev` testing limit:** with the default sender, Resend only delivers to the email on your Resend account (e.g. `you@gmail.com`). Other recipients get `403 validation_error`. Either:

- **Quick test:** register with your Resend account email, or use the verification link shown in the app when send fails.
- **Real prod:** verify your domain at [resend.com/domains](https://resend.com/domains), then set `EMAIL_FROM` to e.g. `Trello Clone <noreply@yourdomain.com>`.

### SMTP (local dev only)

Works on localhost / Docker where outbound SMTP is not blocked:

| Variable | Example (Gmail) |
|----------|-----------------|
| `SMTP_HOST` | `smtp.gmail.com` |
| `SMTP_PORT` | `587` |
| `SMTP_USER` | `you@gmail.com` |
| `SMTP_PASSWORD` | Google app password |
| `EMAIL_FROM` | `Trello Clone <you@gmail.com>` |

Do **not** set `RESEND_API_KEY` locally if you want to test SMTP.

## Admin login

**trello-api** → **Environment** → `ADMIN_EMAIL` / `ADMIN_PASSWORD` (auto-generated).

## Troubleshooting

- **Register/login fails / CORS error** → `WEB_URL` on API must exactly match your web URL (https, no trailing `/`).
- **`/api/config` shows `"mode":"proxy"`** → set `API_PUBLIC_URL` on trello-web and redeploy.
- **WebSockets / live updates broken** → set `WS_PUBLIC_URL` to `wss://` + same host as API.
- **Free tier cold start** → first request after idle can take ~30s.
- **Can't verify email after register** → configure email above, or use the verification link shown on the login/verify-email page when email is off.
- **Can't reset password without email** → use **Forgot password**; when email is off, the reset link is shown on that page (also logged in API as `password reset link for ...`).
- **Resend `403 validation_error`** → `@resend.dev` only sends to your Resend account email; verify a domain and update `EMAIL_FROM`, or use the in-app verification link.
- **SMTP configured but no email** → on Render, SMTP is blocked; switch to `RESEND_API_KEY`. Check logs for `[email] send failed`.
- **Gmail `connection timed out`** → Render blocks port 587; use Resend HTTP API instead of Gmail SMTP.
- **502 on resend/register** → often cold start; wait and retry, or check API logs after redeploy.

## Local production

```powershell
docker compose -f docker-compose.prod.yml up --build -d
```

App: http://localhost:3000 (uses internal proxy; no extra env needed)
