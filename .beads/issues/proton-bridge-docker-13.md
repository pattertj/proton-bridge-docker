---
type: task
title: "Frontend SPA (setup wizard + dashboard)"
status: done
priority: high
epic: proton-bridge-docker-8
blocked_by: [proton-bridge-docker-11, proton-bridge-docker-12]
created: 2026-02-27
---

# Frontend SPA (setup wizard + dashboard)

Create `webui/frontend/` — React SPA loaded via CDN (no build tooling):

## Files
- `index.html` — SPA shell, loads React/ReactDOM from CDN
- `app.js` — React app with Babel standalone for JSX
- `styles.css` — minimal styling, Proton purple (#6D4AFF) accent

## Views/States
1. **Setup Wizard** — email + password form → 2FA input if needed → sync progress
2. **Syncing** — progress bar with percentage, account name
3. **Dashboard** — green status, per-account IMAP/SMTP settings with copy buttons, add/remove account buttons
4. **Error** — bridge crashed, login failed, sync failed

## Key behaviors
- Poll `/api/status` every 5s on dashboard
- Bridge-generated password clearly distinguished from Proton password (callout)
- "Copy All Settings" button for formatted connection block
- Host note: "Use this container's IP or hostname on your Docker network"
