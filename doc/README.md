# AuraMail Backend API Documentation

## 📧 Project Overview

AuraMail is an email management backend service that integrates with Google OAuth 2.0 for user authentication. It provides JWT-based token management with secure refresh token handling for stateless API authentication.

## 🎯 Key Features

- **Google OAuth 2.0 Integration** - Seamless login with Google accounts
- **JWT Token Management** - Secure access and refresh tokens
- **Gmail Integration** - Read-only access to Gmail placement emails
- **User Management** - Automatic user creation and management
- **Token Refresh** - Ability to refresh expired access tokens
- **Logout** - Secure token invalidation
- **AI Summaries (Optional)** - Summarize emails via OpenAI if `OPENAI_API_KEY` is set

## 🏗️ Project Structure

```
go-backendfor-auramail/
├── cmd/
│   └── backend/
│       └── main.go                 # Application entry point
├── internal/
│   ├── auth/
│   │   ├── handler.go              # HTTP handlers for auth endpoints
│   │   ├── service.go              # Business logic for auth
│   │   ├── jwt.go                  # JWT token generation & validation
│   │   └── google/
│   │       ├── handler.go          # Google OAuth handlers
│   │       └── oauth.go            # OAuth configuration
│   ├── user/
│   │   ├── model.go                # User data model
│   │   ├── repository.go           # User data access interface
│   │   └── postgres.go             # PostgreSQL implementation
│   ├── server/
│   ├── middleware/
│   └── ...other modules
├── docker-compose.yml              # Database setup
├── go.mod                          # Go module definition
└── doc/                            # Documentation (this folder)
```

## 🔐 Technology Stack

- **Language**: Go 1.25+
- **Database**: PostgreSQL
- **Authentication**: Google OAuth 2.0, JWT
- **HTTP Framework**: Standard Go `net/http`
- **Token Library**: `github.com/golang-jwt/jwt/v5`
- **Database Driver**: `github.com/jackc/pgx/v5`
- **Gmail API**: `google.golang.org/api/gmail/v1`
- **OpenAI**: `github.com/sashabaranov/go-openai`

## 📚 Documentation Structure

This documentation is organized into the following sections:

1. **README.md** (this file) - Project overview
2. **ARCHITECTURE.md** - System design and component explanation
3. **API_ENDPOINTS.md** - Complete API reference with examples
4. **AUTHENTICATION.md** - Authentication flows and token management
5. **SETUP.md** - Local development setup instructions
6. **GOOGLE_OAUTH.md** - Google OAuth configuration guide
7. **DATABASE.md** - Database schema and structure

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 12+
- Docker & Docker Compose (optional)

### Environment Setup

Create a `.env` file in the project root:

```bash
DATABASE_URL=postgresql://user:password@localhost:5432/auramail
GOOSE_DBSTRING=postgresql://user:password@localhost:5432/auramail
JWT_SECRET=your-very-secret-jwt-key
GOOGLE_OAUTH_CLIENT_ID=your-google-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-google-client-secret
GOOGLE_OAUTH_REDIRECT_URI=http://localhost:8080/auth/google/callback
OPENAI_API_KEY=your-openai-key   # optional but enables AI summaries & streaming
```

### Running the Server

```bash
# Build the application
go build ./cmd/backend

# Run the application
./backend
```

The server will start on `http://localhost:8080`

## 🔄 API Endpoints Summary

| Method | Endpoint                | Purpose                             |
| ------ | ----------------------- | ----------------------------------- |
| `GET`  | `/health`               | Health check                        |
| `GET`  | `/auth/google`          | Initiate Google login               |
| `GET`  | `/auth/google/callback` | Google OAuth callback               |
| `POST` | `/auth/refresh`         | Refresh access token                |
| `POST` | `/auth/logout`          | Logout and invalidate refresh token |
| `GET`  | `/emails/sync`          | Fetch recent placement emails       |
| `GET`  | `/emails/stream`        | Stream AI summaries (SSE)           |

## 💡 Core Concepts

### JWT Tokens

- **Access Token**: Short-lived (15 minutes), used for API requests
- **Refresh Token**: Long-lived, stored in database, used to obtain new access tokens

### User Flow

1. User initiates login → redirected to Google
2. User authenticates with Google
3. Google redirects back with authorization code
4. Backend exchanges code for tokens
5. User receives access and refresh tokens
6. User can refresh tokens or logout

## 📖 Next Steps

- Read [ARCHITECTURE.md](./ARCHITECTURE.md) to understand system design
- Check [SETUP.md](./SETUP.md) for development setup
- Review [API_ENDPOINTS.md](./API_ENDPOINTS.md) for endpoint details
- Configure Google OAuth following [GOOGLE_OAUTH.md](./GOOGLE_OAUTH.md)

---

_Last Updated: January 7, 2026_
