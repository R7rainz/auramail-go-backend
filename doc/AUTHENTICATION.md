# Authentication Flow & Token Management

## 🔐 Authentication Overview

AuraMail uses a two-tier token system for secure, stateless authentication:

1. **Access Token** - Short-lived JWT for API requests
2. **Refresh Token** - Long-lived JWT for obtaining new access tokens

---

## 📊 Token Comparison

| Aspect         | Access Token         | Refresh Token                    |
| -------------- | -------------------- | -------------------------------- |
| **Duration**   | 15 minutes           | Long (days/months)               |
| **Usage**      | API authorization    | Get new access token             |
| **Storage**    | Memory/localStorage  | Secure storage (httpOnly cookie) |
| **Validation** | Fast signature check | DB lookup + signature check      |
| **Risk**       | Lower (short expiry) | Higher (stored longer)           |
| **Scope**      | Limited claims       | User identity only               |

---

## 🔄 Complete Authentication Flows

### Flow 1: Initial Login with Google

```
┌──────────────┐
│   User       │
└──────┬───────┘
       │
       │ 1. Click "Login with Google"
       ▼
┌──────────────────────────┐
│  GET /auth/google        │
│  (Backend generates state)
└──────────┬───────────────┘
           │
           │ 2. Redirect to Google
           ▼
    ┌─────────────────────┐
    │  Google OAuth Page  │
    │  (User logs in)     │
    └────────┬────────────┘
             │
             │ 3. User authenticates
             ▼
┌────────────────────────────────────┐
│ GET /auth/google/callback          │
│ (Google redirects with code)       │
└────────┬─────────────────────────┤
         │
         │ 4. Exchange code for token
         ▼
    ┌──────────────┐
    │  Google API  │
    └────────┬─────┘
             │
             │ 5. Get user info
             ▼
  ┌─────────────────────┐
  │  Database           │
  │  FindOrCreateUser   │
  └────────┬────────────┘
           │
           │ 6. Generate JWTs
           ▼
┌──────────────────────────────────┐
│ Return to Client:                │
│ {                                │
│   "accessToken": "...",          │
│   "refreshToken": "..."          │
│ }                                │
└──────────────────────────────────┘
           │
           │ 7. Store tokens
           ▼
      ┌──────────┐
      │  Client  │
      └──────────┘
```

**Step-by-Step Explanation:**

1. **User Initiates Login**

   - User clicks "Login with Google" button
   - Frontend redirects to `/auth/google`

2. **Backend Generates State**

   - Backend creates random state string (CSRF protection)
   - Redirects user to Google OAuth consent page

3. **User Authenticates**

   - User logs in with Google credentials
   - User grants permission for email & profile access

4. **Google Redirect with Code**

   - Google redirects back to callback URL with authorization code
   - Authorization code is temporary (expires in minutes)

5. **Backend Exchanges Code**

   - Backend uses the code to request access token from Google
   - Access token allows reading user info

6. **Fetch User Info from Google**

   - Backend retrieves user email, name, and unique ID from Google

7. **Database Operation**

   - Backend checks if user exists
   - If exists: update user data
   - If new: create user record
   - Store Google refresh token if provided (for Gmail API)

8. **Generate JWTs**

   - `GenerateAccessToken()` - creates 15-minute token
   - `GenerateRefreshToken()` - creates long-lived token

9. **Return to Client**
   - Send both tokens (camelCase keys) to frontend
   - Frontend stores them securely

---

### Flow 2: Token Refresh

When access token expires (after 15 minutes):

```
┌────────────────┐
│   Client       │
│ (Access token  │
│  expired)      │
└────────┬───────┘
         │
         │ 1. POST /auth/refresh
         │    { "refreshToken": "..." }
         ▼
┌─────────────────────────┐
│  Handler.Refresh()      │
│  Decode request         │
└────────┬────────────────┘
         │
         │ 2. Validate refresh token
         ▼
┌──────────────────────┐
│  JWT Validation      │
│  - Check signature   │
│  - Check expiry      │
└────────┬─────────────┘
         │
         │ 3. If invalid
         ▼
    ┌──────────────┐
    │ 401 Error    │
    │ "invalid     │
    │  token"      │
    └──────────────┘
         │
         │ 4. If valid, fetch user
         ▼
┌──────────────────────┐
│  Database            │
│  FindByRefreshToken()│
└────────┬─────────────┘
         │
         │ 5. Generate new access token
         ▼
┌───────────────────────────┐
│  GenerateAccessToken()    │
└────────┬──────────────────┘
         │
         │ 6. Return new token
         ▼
┌────────────────────────┐
│  {                     │
│    "access_token": ... │
│  }                     │
└────────────────────────┘
         │
         │ 7. Client stores new token
         ▼
      ┌────────┐
      │ Client │
      └────────┘
```

**Key Points:**

- Refresh token doesn't change (still valid for future use)
- New access token is issued
- User continues using API with new access token

---

### Flow 3: Logout

```
┌─────────────┐
│   User      │
│ (clicks     │
│  Logout)    │
└─────┬───────┘
      │
      │ 1. POST /auth/logout
      │    Header: Authorization: Bearer {access_token}
      ▼
┌──────────────────────┐
│ Middleware           │
│ Validate JWT (Bearer │
│ Authorization header)│
└──────┬───────────────┘
       │
       │ 2. Extract userID from token
       ▼
┌──────────────────────┐
│ Handler.Logout()     │
│ userID from context  │
└──────┬───────────────┘
       │
       │ 3. Clear refresh token from DB
       ▼
┌─────────────────────────┐
│ Database                │
│ ClearRefreshToken(id)   │
└──────┬──────────────────┘
       │
       │ 4. Return success
       ▼
  ┌──────────┐
  │ 200 OK   │
  └────┬─────┘
       │
       │ 5. Client clears tokens
       ▼
┌──────────────────┐
│ Client           │
│ - Clear localStorage
│ - Clear memory
│ - Delete cookie
└──────────────────┘
```

**Key Points:**

- Refresh token is deleted from database
- Access token still valid until expiration (can't be revoked in stateless JWT)
- User must login again to get new tokens
- Old refresh token cannot be reused

---

## 🛡️ Security Mechanisms

### 1. JWT Signature Verification

Every token is signed with a secret key:

```go
key, _ := jwtSecret()  // Get JWT_SECRET from environment
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenString, _ := token.SignedString(key)  // Sign with secret
```

**Verification:**

- Parse token and verify signature
- If tampering detected: reject token
- If signature valid: trust claims inside token

### 2. Token Expiration (exp claim)

```go
expirationTime := time.Now().Add(15 * time.Minute)
claims := &AccessTokenClaims{
    UserID: userID,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(expirationTime),
    },
}
```

**Validation:**

- Check `exp` claim against current time
- If expired: reject token
- Forces periodic token refresh

### 3. Refresh Token Database Storage

```
┌─────────────────┐
│  Database       │
├─────────────────┤
│ id | email | ...|
│ 1  | user@ |...|
│    |refresh |...|
│    | token |...|
└─────────────────┘
```

- App refresh token stored in database (in addition to client)
- During refresh, validate token exists in database
- Logout clears token from database
- Prevents use of old/stolen tokens (server-side revocation)

### 4. HTTPS-Only Recommendation

For production:

- Set refresh token as httpOnly cookie (can't be accessed via JavaScript)
- Use HTTPS only (token encrypted in transit)
- Set Secure flag on cookie (sent only over HTTPS)
- Set SameSite=Strict (CSRF protection)

### 5. CSRF Protection

State parameter in OAuth flow:

```go
state := "random-state-for-now"  // TODO: generate cryptographically random value
authURL := h.oauthConfig.AuthCodeURL(state)
```

- User's state parameter must match returned state
- Prevents cross-site request forgery

---

## 🔑 Token Claims Details

### Access Token Claims

```go
type AccessTokenClaims struct {
    UserID int    `json:"sub"`         // Subject (user ID)
    Email  string `json:"email"`       // User email
    Name   string `json:"name,omitempty"` // User name
    jwt.RegisteredClaims               // Standard JWT claims
}
```

**RegisteredClaims includes:**

- `exp` - Expiration time (15 minutes from issue)
- `iat` - Issued at time
- `iss` - Issuer (always "AuraMail")

**Example Payload (decoded):**

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

### Refresh Token Claims

```go
type RefreshTokenClaims struct {
    UserID int    `json:"sub"`   // Subject (user ID)
    Email  string `json:"email"` // User email
    jwt.RegisteredClaims
}
```

**Example Payload (decoded):**

```json
{
  "sub": 1,
  "email": "user@gmail.com",
  "exp": 1740387900,
  "iat": 1735203000,
  "iss": "AuraMail"
}
```

---

## 🔍 Token Validation Process

### Validating Access Token

```
1. Extract token from Authorization header
   ↓
2. Parse JWT (check format)
   ↓
3. Verify signature with secret key
   ↓
4. Check expiration time (exp claim)
   ↓
5. Extract and use claims (userID, email, name)
   ↓
6. Proceed with request
```

### Validating Refresh Token

```
1. Parse JWT from request body
   ↓
2. Verify signature with secret key
   ↓
3. Check expiration time (exp claim)
   ↓
4. Look up token in database (optional but recommended)
   ↓
5. If found in DB: token is valid
   ↓
6. Generate new access token
```

---

## 🚀 Best Practices for Clients

### Token Storage

```javascript
// ❌ DON'T - Vulnerable to XSS
localStorage.setItem("token", accessToken);

// ✅ DO - Secure httpOnly cookie (set by backend)
// Cookie is automatically included in requests
// Cannot be accessed by JavaScript

// ✅ DO - Short-term memory for access token
let accessToken = response.data.accessToken;
```

### Using Access Token

```javascript
// With fetch
fetch("/api/emails", {
  headers: {
    Authorization: `Bearer ${accessToken}`,
  },
});

// With axios
axios.get("/api/emails", {
  headers: {
    Authorization: `Bearer ${accessToken}`,
  },
});
```

### Refresh Logic

```javascript
async function makeAuthenticatedRequest(url) {
  let token = getAccessToken();

  try {
    return await fetch(url, {
      headers: { Authorization: `Bearer ${token}` },
    });
  } catch (error) {
    if (error.status === 401) {
      // Access token expired
      token = await refreshAccessToken();
      // Retry with new token
      return await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });
    }
  }
}
```

---

_Last Updated: January 7, 2026_
