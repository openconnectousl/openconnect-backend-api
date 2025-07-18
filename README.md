# OpenConnect Backend API

A backend service written in Go, providing authentication, user management, and email support for the OpenConnect platform. This backend is containerized, uses PostgreSQL, and supports Google OAuth and SMTP email functionality.

---

##  Features

-  User authentication via Google OAuth 2.0
-  Email support using SMTP (e.g., Gmail)
-  PostgreSQL database with migration support via [golang-migrate](https://github.com/golang-migrate/migrate)
-  Dockerized environment for easy setup and deployment
-  Modular Go architecture using `cmd/`, `internal/`, and `scripts/`
-  File upload support

---

##  Getting Started

### Prerequisites

- [Docker](https://www.docker.com/)
- [Go 1.24+](https://golang.org/doc/install) (for local development)

---

##  Running with Docker

```bash
docker-compose up --build
```
This will spin up:
* `openconnect-db`: A PostgreSQL 15 database
* `openconnect-api`: The Go backend service

## Environment Variables
Create a .env file based on the following template:

```env
DB_DSN=postgres://openconnect:1234@db:5432/openconnect?sslmode=disable

SMTPPORT=587
SMTPSENDER=example@gmail.com
SMTPHOST=smtp.example.com
SMTPUSERNAME=example@gmail.com
SMTPPASS=your_password_here

GOOGLE_CLIENT_ID=your_client_id_here
GOOGLE_CLIENT_SECRET=your_client_secret_here
GOOGLE_REDIRECT_URL=http://localhost:4000/auth/google/callback

FRONTEND_URL=http://localhost:5173
```

## Project Structure
```bash
.
├── cmd/api/                  # App entrypoint (main.go)
│   ├── app/                  # Application logic
│   ├── config/               # Configuration loaders
│   └── server/               # HTTP server logic
├── internal/                 # Internal Go modules (auth, mailer, data, etc.)
│   ├── data/                 # DB models
│   ├── jsonlog/              # Custom logger
│   └── mailer/               # Email templates and mail service
├── migrations/               # SQL migrations (up/down)
├── scripts/entrypoint.sh     # Docker init script (runs migrations)
├── uploads/                  # Upload directory
├── Dockerfile                # Backend image build
├── docker-compose.yml        # Multi-container setup
├── go.mod / go.sum           # Go module definitions
└── .env.example              # Env template
```
## Migrations
Migrations are automatically run during container startup via:

```bash
migrate -path /migrations -database ${DB_DSN} up
```
You can also run them manually inside the API container:

```bash
docker-compose exec api migrate -path /migrations -database ${DB_DSN} up
```
## Google OAuth Setup
1. Create OAuth credentials at Google Cloud Console
2. Use:
   * Authorized redirect URI: `http://localhost:4000/auth/google/callback`
3. Add credentials to `.env`

## SMTP Setup (Gmail Example)
1. Enable [App Passwords](https://support.google.com/accounts/answer/185833)  for Gmail.
2. Set the following in `.env`:

```env
SMTPHOST=smtp.gmail.com
SMTPPORT=587
SMTPUSERNAME=your@gmail.com
SMTPPASS=your_app_password
SMTPSENDER=Your Name <your@gmail.com>
```

## Development
Run Locally Without Docker
```bash
go run ./cmd/api
```
Make sure to run migrations manually using:

```bash
migrate -path ./migrations -database ${DB_DSN} up
```

## Dependencies
[github.com/julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) – HTTP routing

[github.com/lib/pq](https://github.com/lib/pq) – PostgreSQL driver

[golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) – Google OAuth 2.0

[github.com/go-mail/mail/v2](https://github.com/go-mail/mail) – SMTP mailing

## Scripts
entrypoint.sh

This script:
1. Waits for PostgreSQL to be ready
2. Runs migrations
3. Starts the API server

## Sample API Flow (coming soon)
* `POST /auth/google` → OAuth login
* `GET /user/profile` → Protected route
* `POST /mail/send` → Trigger SMTP email

##  Contributing
Pull requests are welcome! Please ensure:
* Code is formatted (`gofmt`)
* You add tests for new features
* You document new env vars or routes

## License
MIT — [OpenConnectOUSL](https://github.com/OpenConnectOUSL)