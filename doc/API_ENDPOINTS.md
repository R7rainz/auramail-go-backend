# API Endpoints Documentation

## 📋 Overview

All endpoints are prefixed with the base URL: `http://localhost:8080`

## 🔍 Endpoint Summary

| HTTP Method | Endpoint                | Purpose               | Auth Required              |
| ----------- | ----------------------- | --------------------- | -------------------------- |
| `GET`       | `/health`               | Health check          | ❌ No                      |
| `GET`       | `/auth/google`          | Initiate Google login | ❌ No                      |
| `GET`       | `/auth/google/callback` | Google OAuth callback | ❌ No                      |
| `POST`      | `/auth/refresh`         | Refresh access token  | ❌ No (uses refresh token) |
| `POST`      | `/auth/logout`          | Logout user           | ✅ Yes (Bearer)            |
| `GET`       | `/emails/sync`          | Fetch recent emails   | ✅ Yes (Bearer)            |
| `GET`       | `/emails/stream`        | Stream AI summaries   | ✅ Yes (Bearer)            |

---

## 🩺 Health Check

### `GET /health`

Simple health check to verify the server is running.

**Request:**

```bash
curl -X GET http://localhost:8080/health
```

**Response:**

```
Status: 200 OK
Body: ok
```

---

## 🔐 Authentication Endpoints

### 1. Google Login Initiation

#### `GET /auth/google`

Initiates the Google OAuth 2.0 login flow. Redirects the user to Google's login page.

**Request:**

```bash
curl -X GET http://localhost:8080/auth/google
```

**Response:**

```
Status: 307 Temporary Redirect
Location: https://accounts.google.com/o/oauth2/v2/auth?client_id=...&redirect_uri=...
```

**Flow:**

1. User accesses this endpoint
2. Redirected to Google login page
3. User logs in with Google account
4. Google redirects back to `/auth/google/callback`

**Query Parameters:**

- None (state is generated server-side)

---

### 2. Google OAuth Callback

#### `GET /auth/google/callback`

Receives the authorization code from Google after successful authentication.

**Request:**

```bash
# User is redirected here automatically by Google
GET http://localhost:8080/auth/google/callback?code=4/0A...&state=random-state-for-now
```

**Response (Success):**

```
Status: 200 OK
Content-Type: application/json

{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (Error - Missing Code):**

```
Status: 400 Bad Request
Body: missing code
```

**Response (Error - OAuth Exchange Failed):**

```
Status: 500 Internal Server Error
Body: oauth exchange failed
```

**Query Parameters:**

- `code` (required) - Authorization code from Google
- `state` (optional) - State parameter for CSRF protection

---

### 3. Refresh Access Token

#### `POST /auth/refresh`

Refreshes an expired access token using a valid refresh token.

**Request:**

```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

**Request Body:**

```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (Success):**

```
Status: 200 OK
Content-Type: application/json

{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (Error - Invalid Token):**

```
Status: 401 Unauthorized
Body: invalid refresh token
```

**Response (Error - Invalid JSON):**

```
Status: 400 Bad Request
Body: invalid request
```

**Notes:**

- The refresh token must be stored from the initial login response
- Access token is valid for 15 minutes
- Refresh token has a longer expiration (depends on configuration)
- Refresh token is also stored in database for server-side validation

---

### 4. Logout

#### `POST /auth/logout`

Logs out the current user by invalidating their refresh token.

**Request:**

```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Request Headers:**

- `Authorization: Bearer {access_token}`

**Response (Success):**

```
Status: 200 OK
```

**Response (Error - Not Authenticated):**

```
Status: 401 Unauthorized
Body: unauthorized
```

**Response (Error - DB Error):**

```
Status: 500 Internal Server Error
Body: logout failed
```

**Notes:**

- Requires user to be authenticated (userID from context)
- Clears the refresh token from database
- User must login again to get new tokens
- Access token remains valid until expiration (15 min)

---

## 📊 Token Structure

### Access Token JWT Claims

```json
{
  "sub": 1,
  "email": "user@gmail.com",
  "name": "John Doe",
  "exp": 1735203900,
  "iat": 1735203000,
  "iss": "AuraMail"
}
```

**Claims:**

- `sub` (subject) - User ID
- `email` - User's email
- `name` - User's name
- `exp` - Expiration timestamp (15 minutes from generation)
- `iat` - Issued at timestamp
- `iss` - Issuer (always "AuraMail")

### Refresh Token JWT Claims

```json
{
  "sub": 1,
  "email": "user@gmail.com",
  "exp": 1740387900,
  "iat": 1735203000,
  "iss": "AuraMail"
}
```

**Claims:**

- `sub` (subject) - User ID
- `email` - User's email
- `exp` - Expiration timestamp (longer expiry)
- `iat` - Issued at timestamp
- `iss` - Issuer (always "AuraMail")

---

## 🔄 Complete Authentication Flow Example

### Step 1: Initiate Login

```bash
# User clicks "Login with Google"
GET http://localhost:8080/auth/google

# Response: Redirect to Google
# Location: https://accounts.google.com/o/oauth2/v2/auth?...
```

### Step 2: User Authenticates with Google

User logs in with their Google account on Google's website.

### Step 3: Google Redirects Back

```bash
# Google redirects to callback
GET http://localhost:8080/auth/google/callback?code=4/0A...&state=...

# Response: 200 OK
# {
#   "accessToken": "eyJhbGc...",
#   "refreshToken": "eyJhbGc..."
# }
```

### Step 4: Store Tokens

Client stores:

- Access token in memory (short-lived)
- Refresh token securely (e.g., httpOnly cookie)

### Step 5: Use Access Token for API Calls

```bash
# Make API requests with access token
GET http://localhost:8080/emails/sync \
  -H "Authorization: Bearer {access_token}"
```

### Step 6: Refresh When Expired

When access token expires:

```bash
POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken": "{refresh_token}"}'

# Response: 200 OK
# {
#   "access_token": "eyJhbGc..."  (new token)
# }
```

### Step 7: Logout

```bash
POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer {access_token}"

# Response: 200 OK
# Refresh token is now invalid in database
```

---

## ⚠️ Error Handling

### Common HTTP Status Codes

| Status | Meaning               | Common Cause           |
| ------ | --------------------- | ---------------------- |
| `200`  | OK                    | Request successful     |
| `307`  | Temporary Redirect    | OAuth redirect         |
| `400`  | Bad Request           | Invalid request format |
| `401`  | Unauthorized          | Missing/invalid token  |
| `500`  | Internal Server Error | Database/server error  |

### Error Response Format

When errors occur, the response includes an error message:

```
Status: {status_code}
Body: {error_message}
```

Example:

```
Status: 401 Unauthorized
Body: invalid refresh token
```

---

## 🔑 Authentication Methods

### Token-Based (Bearer Token)

Used for logout endpoint via middleware:

```bash
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

The middleware extracts the token and validates it, placing the userID in the request context.

---

## ✉️ Email Endpoints

### 1) `GET /emails/sync`

Fetches recent placement-related emails (read-only) for the authenticated user and returns a parsed list.

Request:

```bash
curl -X GET http://localhost:8080/emails/sync \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Response (200):

```json
[
  {
    "id": "187ab...",
    "subject": "Placement Drive: ACME",
    "from": "placementoffice@vitbhopal.ac.in",
    "date": "Mon, 06 Jan 2026 10:15:00 +0530",
    "body": "...cleaned plain text...",
    "snippet": "Short Gmail snippet..."
  }
]
```

Errors:

- 401 Unauthorized: missing/invalid token
- 404 User not found
- 500 Failed to connect to Gmail / Extraction Failed

Notes:

- Uses Gmail scope `gmail.readonly`
- Query currently filters placement emails

### 2) `GET /emails/stream` (SSE)

Streams AI summaries for recent placement emails using Server-Sent Events. Keeps the connection open and periodically sends data or heartbeats.

Request:

```bash
curl -N -H "Accept: text/event-stream" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8080/emails/stream
```

Response events:

```
data: {"summary":"...","category":"...","company":null,...}
:
data: {"summary":"..."}
```

Error event when no emails found:

```
data: {"error": "no_emails_found"}
```

Headers:

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`
- CORS is enabled for development

Notes:

- Heartbeat comment `: heartbeat` is sent every 15s to keep the connection alive
- Requires environment variable `OPENAI_API_KEY` for AI summaries; if not set, summaries are minimal

---

## 📝 Example Implementations

### JavaScript/Fetch

```javascript
// Step 1: Login
function startGoogleLogin() {
  window.location.href = "http://localhost:8080/auth/google";
}

// Step 2: After redirect, you'll have tokens

// Step 3: Refresh token
async function refreshAccessToken(refreshToken) {
  const response = await fetch("http://localhost:8080/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
  });

  if (response.ok) {
    const data = await response.json();
    return data.access_token;
  }
}

// Step 4: Logout
async function logout(accessToken) {
  await fetch("http://localhost:8080/auth/logout", {
    method: "POST",
    headers: { Authorization: `Bearer ${accessToken}` },
  });
}
```

### cURL

```bash
# Health check
curl -X GET http://localhost:8080/health

# Refresh token
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken":"eyJhbGc..."}'

# Logout
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer eyJhbGc..."
```

---

_Last Updated: December 26, 2025_
