# Local Development Setup

## 🛠️ Prerequisites

Before starting, ensure you have the following installed:

- **Go 1.25+** - [Download](https://golang.org/dl/)
- **PostgreSQL 12+** - [Download](https://www.postgresql.org/download/)
- **Git** - [Download](https://git-scm.com/)
- **Docker & Docker Compose** (optional, for database) - [Download](https://www.docker.com/)

---

## 📦 Installation Steps

### Step 1: Clone the Repository

```bash
git clone https://github.com/r7rainz/auramail.git
cd go-backendfor-auramail
```

### Step 2: Install Go Dependencies

```bash
go mod download
go mod tidy
```

This downloads all required packages from `go.mod`.

### Step 3: Set Up PostgreSQL Database

#### Option A: Using Docker Compose (Recommended)

```bash
# Start PostgreSQL in Docker
docker-compose up -d

# Verify database is running
docker-compose ps
```

The database will be available at:

- **Host**: `localhost`
- **Port**: `5432`
- **Database**: Set in `docker-compose.yml`

#### Option B: Local PostgreSQL Installation

```bash
# Create database
createdb auramail

# Verify connection
psql -U postgres -d auramail -c "SELECT 1;"
```

### Step 4: Run Database Migrations

```bash
# Install goose if not already installed
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations from internal/db/migrations
goose -dir internal/db/migrations postgres "postgresql://user:password@localhost/auramail" up

# Verify migrations
goose -dir internal/db/migrations postgres "postgresql://user:password@localhost/auramail" status
```

### Step 5: Configure Environment Variables

Create a `.env` file in the project root:

```bash
# Database
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/auramail
GOOSE_DBSTRING=postgresql://postgres:postgres@localhost:5432/auramail

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# Google OAuth Configuration
GOOGLE_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
GOOGLE_OAUTH_REDIRECT_URI=http://localhost:8080/auth/google/callback
OPENAI_API_KEY=your-openai-key   # optional but required for AI summaries/SSE

# Server
SERVER_PORT=8080
```

**⚠️ Important:**

- Never commit `.env` to version control
- Change `JWT_SECRET` to a strong random string in production
- Replace Google OAuth credentials with your own (see [GOOGLE_OAUTH.md](./GOOGLE_OAUTH.md))

### Step 6: Verify Configuration

Test the database connection:

```bash
# Set environment
export $(cat .env | xargs)

# Test connection
psql $DATABASE_URL -c "SELECT 1;"
```

---

## 🚀 Running the Application

### Build the Application

```bash
go build -o backend ./cmd/backend
```

This creates an executable named `backend`.

### Run the Application

```bash
./backend
```

Expected output:

```
2025/12/26 10:00:00 Google OAuth RedirectURL: http://localhost:8080/auth/google/callback
2025/12/26 10:00:00 Server starting on :8080
```

### Verify Server is Running

````bash
curl http://localhost:8080/health
# Response: ok

### Test Email Endpoints (after logging in & obtaining tokens)

```bash
# Sync recent placement emails
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/emails/sync | jq .

# Stream AI summaries over SSE
curl -N -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/emails/stream
````

````

---

## 🔄 Development Workflow

### Watch Mode (Auto-Restart on Changes)

Install `air` for hot-reload:

```bash
go install github.com/cosmtrek/air@latest
````

Create `.air.toml` in project root:

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/backend"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_patterns = ["_test.go"]
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_patterns = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"
```

Run with:

```bash
air
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/auth
```

### Code Formatting

```bash
# Format all Go files
go fmt ./...

# Run linter (if installed)
golangci-lint run
```

---

## 🗄️ Database Management

### Viewing Database Schema

```bash
# Connect to database
psql $DATABASE_URL

# List tables
\dt

# Describe users table
\d users

# View all data
SELECT * FROM users;

# Exit
\q
```

### Common Database Operations

```bash
# Check migrations applied
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING status

# Rollback last migration
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING down

# Create new migration
goose -dir internal/db/migrations create create_users sql
```

### Database Reset (Development Only)

```bash
# ⚠️ WARNING: This will delete all data!
dropdb auramail
createdb auramail
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING up
```

---

## 🧪 Testing the API

### Using cURL

```bash
# Health check
curl http://localhost:8080/health

# Initiate Google login
curl http://localhost:8080/auth/google

# Refresh token (requires valid refresh token)
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken":"YOUR_REFRESH_TOKEN"}'

# Logout (requires valid access token)
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Emails (requires valid access token)
curl http://localhost:8080/emails/sync \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

curl -N http://localhost:8080/emails/stream \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Using Postman

1. Open Postman
2. Create a new collection "AuraMail API"
3. Add requests:
   - **GET** `/health`
   - **GET** `/auth/google`
   - **POST** `/auth/refresh` with body: `{"refreshToken":"..."}`
   - **POST** `/auth/logout` with header: `Authorization: Bearer ...`
4. Set base URL to `http://localhost:8080`
5. Set environment variables for tokens

### Using REST Client (VS Code)

Create `requests.rest` in project root:

```rest
### Health Check
GET http://localhost:8080/health

### Get Refresh Token (manual from Google callback)
@refreshToken = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

### Refresh Access Token
POST http://localhost:8080/auth/refresh
Content-Type: application/json

{
  "refreshToken": "@refreshToken"
}

### Logout
@accessToken = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

POST http://localhost:8080/auth/logout
Authorization: Bearer @accessToken
```

---

## 🐛 Troubleshooting

### Issue: `DATABASE_URL not set`

**Solution:**

```bash
# Check .env file exists
ls -la .env

# Load environment variables
export $(cat .env | xargs)

# Verify
echo $DATABASE_URL
```

### Issue: `connection refused` (database)

**Solution:**

```bash
# Check if PostgreSQL is running
ps aux | grep postgres

# If using Docker:
docker-compose up -d

# Verify connection
psql $DATABASE_URL -c "SELECT 1;"
```

### Issue: `JWT_SECRET not set`

**Solution:**

```bash
# Verify .env has JWT_SECRET
grep JWT_SECRET .env

# If missing, add it:
echo "JWT_SECRET=your-secret-key" >> .env
```

### Issue: `Google OAuth credentials not working`

**Solution:**
See [GOOGLE_OAUTH.md](./GOOGLE_OAUTH.md) for:

- Creating Google OAuth credentials
- Setting up redirect URI
- Testing OAuth flow

### Issue: `Migration failed`

**Solution:**

```bash
# Check migration status
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING status

# Rollback and retry
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING down
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING up

# Check logs
tail -f build-errors.log
```

---

## 📋 Development Checklist

Before committing code:

- [ ] Tests pass: `go test ./...`
- [ ] Code formatted: `go fmt ./...`
- [ ] No unused imports: `go mod tidy`
- [ ] Linter passes: `golangci-lint run`
- [ ] Environment variables set
- [ ] Database migrations applied
- [ ] Server starts without errors
- [ ] Health check works: `curl http://localhost:8080/health`

---

## 🔗 Useful Resources

- [Go Documentation](https://golang.org/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [JWT Introduction](https://jwt.io/)
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)
- [HTTP Status Codes](https://httpwg.org/specs/rfc7231.html#status.codes)

---

_Last Updated: December 26, 2025_
