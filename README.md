# Deadline-Aggregator 📚⏰

Deadline-Aggregator is a small Go service that watches your **Google Classroom** courses and sends 📢 **Discord** reminders whenever an assignment is due soon.  
The goal is to keep students (and teachers!) on top of deadlines without living inside Classroom all day.

---

## ✨ Features

1. **Google OAuth2 login** – sign in with your Google account; no passwords stored.
2. **Classroom read-only access** – queries active courses & coursework.
3. **Automated reminders** – every day at **18:00 (6 p.m.)** the scheduler checks for assignments due within the next **6 hours** and posts one embed per user to the configured Discord channel.
4. **PostgreSQL persistence** – keeps OAuth tokens, assignments & notification history.
5. **Discord embeds** – colourful, mobile-friendly notifications with title, course and countdown.
6. **Docker-friendly** – run Postgres via Docker, the app itself is just `go run ./cmd/main.go`.
7. **Zero secrets in repo** – configuration lives in a local `.env` file that never gets committed.

---

## 🗺️ Repository layout

```
cmd/                 – main.go (HTTP server + scheduler)
internal/
  handlers/          – Gin HTTP routes (OAuth flow, API placeholders)
  google/            – Classroom helper (fetch & filter assignments)
  discord/           – Discord webhook client
  scheduler/         – daily job that ties everything together
  store/             – Postgres connection & queries (migrations included)
migrations/          – (not used – inline SQL instead)
README.md            – you're here 🙂
.gitignore           – bans .env, binaries, IDE files
```

---

## ⚙️ Requirements

- Go 1.22+ (modules)
- A Discord server where you can create a **webhook**
- A Google Cloud project with the **Google Classroom API** enabled
- PostgreSQL 14+ – local install **or** Docker container (see below)

---

## 🚀 Quick-start (local machine)

### 1. Clone & enter the repo

```bash
git clone https://github.com/your-username/deadline-aggregator.git
cd deadline-aggregator
```

### 2. Create configuration file

Copy the example and edit it:

```bash
cp env.example .env   # if you committed env.example
# OR simply create .env with the following keys 👇
```

```.env
# ── Database ───────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=deadline_aggregator
DB_SSL=disable          # use "require" only if your DB enforces SSL

# ── Google OAuth ───────────────────────────
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# ── Discord ────────────────────────────────
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...

# ── Server ─────────────────────────────────
PORT=8080               # optional, default 8080
```

> **Never commit `.env** – it's in `.gitignore` already.

### 3. Start PostgreSQL (Docker)

```bash
docker run --name deadline-pg \
           -e POSTGRES_PASSWORD=password \
           -e POSTGRES_DB=deadline_aggregator \
           -p 5432:5432 -d postgres:16
```

Feel free to use a local Postgres instead; just match the credentials.

### 4. Install Go deps & run

```bash
go mod tidy           # pulls modules

go run ./cmd/main.go  # or: go build -o deadline && ./deadline
```

Console output:

```
Starting reminder scheduler...
Scheduler sleeping until Sat, 05 Jul 2025 18:00:00 ...
Server running on http://localhost:8080
```

### 5. Authenticate

Open a browser at:

```
http://localhost:8080/auth/google/login
```

Log in with the Google account that has Classroom courses – you'll be redirected back and see a JSON payload with your basic profile and current coursework.

From now on your OAuth tokens are stored in Postgres and the scheduler will process your account once a day.

---

## 🔧 How it works under the hood

1. **Handlers** complete the Google OAuth flow, persist tokens, expose tiny JSON endpoints.
2. **Scheduler** wakes at 18:00 local time:
   - loads every user's token
   - queries Classroom coursework
   - filters items due _> now && ≤ now+6h_
   - logs the list and posts a Discord embed via REST webhook
3. **Store** contains plain SQL migrations that run automatically on startup.

> Tokens are saved unencrypted for simplicity – if you deploy publicly, add encryption (e.g. Cloud KMS).

---

## 🛠️ Customisation

| Change                        | Location                                                         |
| ----------------------------- | ---------------------------------------------------------------- |
| Reminder time                 | `internal/scheduler/scheduler.go` (`18,0,0`)                     |
| Lead-time window (hours)      | same file – `classroomapi.GetUpcomingAssignments(token, cfg, 6)` |
| Discord embed colours / style | `internal/discord/discord.go`                                    |
| Database engine               | replace `store.NewPostgres()` (SQLite variant is easy)           |

---

## 🐳 Docker-compose (optional)

A single-container Postgres is used above; you can also run the whole stack with `docker-compose`/`compose.yaml` so the Go app runs in a container too. PRs welcome!

---

## 📜 License

MIT (see `LICENSE` file).

---

## 🙋‍♀️ FAQ

**Q: Will this spam my Discord channel?**  
A: Scheduler runs once a day and deduplicates per assignment. Adjust as needed.

**Q: Can I host for multiple users?**  
A: Yes, the DB already stores many users, but you'll need a public HTTPS server and to add that domain to Google OAuth redirect URIs.

**Q: Why 6-hour window / 18:00 trigger?**  
A: Fits a common "evening heads-up" use-case. Change constants to suit your workflow.

Enjoy and pull-request away! 🎉
