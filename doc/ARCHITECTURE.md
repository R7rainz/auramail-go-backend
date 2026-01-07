# System Architecture

## 🏗️ Overall Architecture

The AuraMail backend follows a **layered architecture pattern** with clear separation of concerns:

```
┌─────────────────────────────────────────────┐
│         HTTP Layer (Handlers)               │
│    - GoogleHandler                          │
│    - AuthHandler                            │
│    - GmailHandler                           │
│    - SSE Stream (emails/stream)             │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│      Business Logic Layer (Services)        │
│    - Auth.Service.Refresh()                 │
│    - Auth.Service.Logout()                  │
│    - Gmail.FetchAndSummarize()              │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│    Data Access Layer (Repository)           │
│    - Repository Interface                   │
│    - PostgresRepository Implementation      │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│      Database Layer (PostgreSQL)            │
│    - users table                            │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│           External Services                 │
│    - Google OAuth 2.0                       │
│    - Gmail API (readonly)                   │
│    - OpenAI API (summaries)                 │
└─────────────────────────────────────────────┘
```

## 📦 Component Details

### 1. User Layer (`internal/user/`)

#### Model

```go
type User struct {
    ID            int       // Unique identifier
    Email         string    // User's email
    Name          string    // User's full name
   Provider      string    // OAuth provider (e.g., "google") (may be empty)
    ProviderID    string    // Provider's unique ID for user
    RefreshToken  string    // JWT refresh token
}
```

#### Repository Interface

```go
type Repository interface {
    FindOrCreateGoogleUser(ctx, email, name, googleSub) (*User, error)
    UpdateRefreshToken(ctx, userID, token) error
    FindByRefreshToken(ctx, token) (*User, error)
    ClearRefreshToken(ctx, userID) error
   FindByID(ctx, id string) (*User, error)
   Save(ctx context.Context, user *User) error
}
```

**Why an interface?**

- Allows multiple implementations (PostgreSQL, MongoDB, etc.)
- Easy to mock for testing
- Decouples business logic from database choice

#### PostgresRepository Implementation

Implements the `Repository` interface with actual SQL queries:

- `FindOrCreateGoogleUser` - Insert-once by email (stores `provider_id`)
- `UpdateRefreshToken` - Save refresh token to database
- `FindByRefreshToken` - Retrieve user by their refresh token
- `ClearRefreshToken` - Delete refresh token (logout)
- `FindByID` - Fetch user by numeric ID (with cast)
- `Save` - Update basic user fields

---

### 2. Authentication Layer (`internal/auth/`)

#### JWT Functions (`jwt.go`)

**Token Claims Structures:**

```go
type AccessTokenClaims struct {
    UserID int                  // User's database ID
    Email  string               // User's email
    Name   string               // User's name
    jwt.RegisteredClaims        // Standard JWT claims (exp, iat, iss)
}

type RefreshTokenClaims struct {
    UserID int                  // User's database ID
    Email  string               // User's email
    jwt.RegisteredClaims        // Standard JWT claims
}
```

**Key Functions:**

- `GenerateAccessToken()` - Creates 15-minute access token
- `GenerateRefreshToken()` - Creates long-lived refresh token
- `ValidateAccessToken()` - Verifies and extracts access token data
- `ValidateRefreshToken()` - Verifies and extracts refresh token data

**Token Flow:**

```
┌─────────────────────────────────────┐
│  User provides credentials          │
└────────────────┬────────────────────┘
                 │
                 ▼
        ┌─────────────────┐
        │ Validate token  │
        │ signature & exp │
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │   Extract data  │
        │   from claims   │
        └────────┬────────┘
                 │
                 ▼
        ┌─────────────────┐
        │  Return claims  │
        │   or error      │
        └─────────────────┘
```

#### Service Layer (`service.go`)

Contains business logic:

```go
type Service struct {
    users user.Repository  // Data access
}
```

**Methods:**

1. **Refresh(ctx, refreshToken)**

   - Validates refresh token
   - Fetches user from database
   - Generates new access token
   - Returns new access token

2. **Logout(ctx, userID)**
   - Clears refresh token from database
   - User must login again to get new tokens

#### Handler Layer (`handler.go`)

Converts HTTP requests to function calls:

```go
type Handler struct {
    oauthConfig *oauth2.Config      // Google OAuth config
    userRepo    user.Repository      // User data access
    service     *Service             // Business logic
}
```

**Endpoints:**

1. **Refresh(w, r)** - `POST /auth/refresh`

   - Accepts: `{"refreshToken": "..."}`
   - Validates refresh token via service
   - Returns: `{"access_token": "..."}`

2. **Logout(w, r)** - `POST /auth/logout`
   - Extracts userID from context
   - Calls service to invalidate token
   - Returns: 200 OK

---

### 3. Gmail Layer (`internal/gmail/`)

- `handler.go` exposes:

  - `GET /emails/sync` (protected): returns recent placement-related emails parsed into a compact structure
  - `GET /emails/stream` (protected, SSE): streams AI summaries with heartbeat support

- `service.go` provides `FetchAndSummarize(ctx, srv, query, userID)`

  - Fetches message metadata from Gmail
  - Extracts subject/body (via `internal/utils/gmail.go`)
  - Calls `ai.AnalyzeEmail()` concurrently with a worker pool
  - Emits validated summaries on a channel

- `internal/utils/gmail.go` includes:
  - `ListPlacementEmails()` — concurrent fetch of messages
  - `ParseBody()` — retrieves and cleans message body
  - `FormatForAI()` and helpers

### 4. AI Layer (`internal/ai/`)

- `summarizer.go`:
  - Uses `go-openai` (`OPENAI_API_KEY` required) with `GPT4oMini`
  - Caches results (in-memory TTL)
  - Returns a structured `AIResult` JSON (fields are nullable where appropriate)

---

### 3. Google OAuth Layer (`internal/auth/google/`)

#### OAuth Configuration (`oauth.go`)

```go
func NewOAuthConfig() *oauth2.Config {
    return &oauth2.Config{
        ClientID:     // From environment
        ClientSecret: // From environment
        RedirectURL:  // Where Google redirects after auth
        Scopes: []string{
            "email",      // Access email
            "profile",    // Access profile info
            "gmail.readonly" // Read Gmail
        }
    }
}
```

#### Google Handler (`handler.go`)

1. **GoogleAuth(w, r)** - `GET /auth/google`

   - Generates OAuth state
   - Redirects user to Google login page
   - Google handles authentication

2. **GoogleCallback(w, r)** - `GET /auth/google/callback`
   - Receives authorization code from Google
   - Exchanges code for Google access token
   - Fetches user info from Google API
   - Creates/updates user in database
   - Generates JWT tokens
   - Returns both tokens to client

**OAuth Flow Diagram:**

```
User                    Backend              Google
  │                        │                   │
  ├─── GET /auth/google ──>│                   │
  │                        │                   │
  │                        ├─ Redirect URL ───>│
  │<─────────────────────────────────────────────┤ (Google login page)
  │                    (User logs in)           │
  │                                             │
  │<─────────────────────────────────────────────┤ (Redirect with code)
  │                        │<──────────────────┤
  │                        │  Callback?code=X  │
  │                        │                   │
  │                        ├─ Exchange code ──>│
  │                        │                   │
  │                        │<─ Access Token ───┤
  │                        │                   │
  │                        ├─ Fetch userinfo ─>│
  │                        │<─ User data ──────┤
  │                        │                   │
  │                    (Save to DB)            │
  │                    (Generate JWTs)         │
  │<────── Tokens ────────┤
```

---

## 🔄 Request Flow Examples

### Login Flow

```
1. GET /auth/google
   → Redirect to Google

2. [User logs in with Google]

3. GET /auth/google/callback?code=XXX
   → Exchange code for tokens
   → Fetch user info from Google
   → FindOrCreateGoogleUser() in DB
   → Generate JWT access + refresh tokens
   → Return tokens to client (camelCase keys)

4. Client stores tokens (access in memory, refresh securely)
```

### Token Refresh Flow

```
1. POST /auth/refresh
   Body: { "refreshToken": "eyJhbGc..." }

2. Handler.Refresh()
   → Service.Refresh()
   → ValidateRefreshToken()
   → FindByRefreshToken() in DB
   → GenerateAccessToken()
   → Return new access token

3. Client uses new access token for next 15 minutes
```

### Logout Flow

```
1. POST /auth/logout
   Context: userID extracted from middleware

2. Handler.Logout()
   → Service.Logout()
   → ClearRefreshToken() in DB

3. User's refresh token is deleted
   → User must login again
```

---

## 🔐 Security Principles

| Principle        | Implementation                                 |
| ---------------- | ---------------------------------------------- |
| Token Separation | Access token (short) + Refresh token (long)    |
| Secure Storage   | Refresh token stored in database (server-side) |
| Token Validation | Signature verification + expiration check      |
| Context Binding  | User info extracted from JWT claims            |
| Stateless Auth   | No session storage, only validate JWTs         |

---

## 🎯 Design Patterns Used

| Pattern                  | Location             | Purpose                      |
| ------------------------ | -------------------- | ---------------------------- |
| **Repository**           | `user/repository.go` | Abstract data access         |
| **Service**              | `auth/service.go`    | Encapsulate business logic   |
| **Dependency Injection** | `handlers`           | Pass dependencies explicitly |
| **Interface-based**      | All repositories     | Enable testing & flexibility |

---

_Last Updated: January 7, 2026_
