# Google OAuth 2.0 Setup Guide

## 🎯 Overview

This guide walks you through setting up Google OAuth 2.0 credentials for the AuraMail backend.

---

## 📋 Prerequisites

- Google Account
- Access to [Google Cloud Console](https://console.cloud.google.com)
- Project created in Google Cloud Platform

---

## 🔑 Step-by-Step Credential Setup

### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Click **Select a Project** → **New Project**
3. Enter project name: `AuraMail`
4. Click **Create**
5. Wait for project creation (may take a minute)

### Step 2: Enable Required APIs

1. In the Cloud Console, go to **APIs & Services** → **Library**
2. Enable the following:

- Gmail API
- Google OAuth 2.0 endpoints (Identity Services)

**Tip:** Search for "Gmail" and "Identity".

### Step 3: Create OAuth 2.0 Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth client ID**
3. You may be prompted to create a consent screen first:

   - Click **Create Consent Screen**
   - Select **External** as user type
   - Fill in required fields:
     - **App name**: `AuraMail`
     - **User support email**: Your email
     - **Developer contact**: Your email
   - Click **Save and Continue**
   - Skip scopes (we'll configure in the app)
   - Click **Save and Continue** again

4. Back to credentials, click **Create Credentials** → **OAuth client ID** again
5. Choose **Web Application**
6. Configure:

   - **Name**: `AuraMail Backend`
   - **Authorized JavaScript origins**:
     ```
     http://localhost:8080
     http://localhost:3000
     https://yourdomain.com (production)
     ```
   - **Authorized redirect URIs**:
     ```
     http://localhost:8080/auth/google/callback
     http://localhost:3000/callback (if frontend is separate)
     https://yourdomain.com/auth/google/callback (production)
     ```

7. Click **Create**

### Step 4: Copy Your Credentials

After creation, a dialog shows:

```
Client ID: YOUR_CLIENT_ID.apps.googleusercontent.com
Client Secret: YOUR_CLIENT_SECRET
```

Save these securely.

---

## 🔐 Configure Environment Variables

Update your `.env` file:

```bash
GOOGLE_OAUTH_CLIENT_ID=YOUR_CLIENT_ID.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=YOUR_CLIENT_SECRET
GOOGLE_OAUTH_REDIRECT_URI=http://localhost:8080/auth/google/callback
OPENAI_API_KEY=your-openai-key   # optional but required for AI summaries
```

**⚠️ Never commit credentials to version control!**

---

## 🧪 Testing OAuth Flow

### Step 1: Start the Backend

```bash
./backend
```

Expected output:

```
2025/12/26 10:00:00 Google OAuth RedirectURL: http://localhost:8080/auth/google/callback
2025/12/26 10:00:00 Server starting on :8080
```

### Step 2: Test in Browser

1. Open: `http://localhost:8080/auth/google`
2. You should be redirected to Google login
3. Sign in with a Google account
4. Grant permission to access email and profile
5. You'll be redirected to callback with tokens

### Step 3: Verify Response

Check that you receive:

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

---

## 🚀 Scopes Configuration

### Current Scopes

The backend requests these scopes from Google:

```go
Scopes: []string{
    "https://www.googleapis.com/auth/userinfo.email",
    "https://www.googleapis.com/auth/userinfo.profile",
    "https://www.googleapis.com/auth/gmail.readonly",
}
```

**What they do:**

- **email**: Access user's email address
- **profile**: Access user's name and basic profile info
- **gmail.readonly**: Read-only access to Gmail

### Adding More Scopes

Edit [internal/auth/google/oauth.go](../internal/auth/google/oauth.go):

```go
Scopes: []string{
    "https://www.googleapis.com/auth/userinfo.email",
    "https://www.googleapis.com/auth/userinfo.profile",
    "https://www.googleapis.com/auth/gmail.readonly",
    // Add more:
    "https://www.googleapis.com/auth/calendar.readonly",
    "https://www.googleapis.com/auth/contacts.readonly",
}
```

---

## 🌐 Production Deployment

### Update Redirect URI

1. Go to [Google Cloud Console](https://console.cloud.google.com) → **Credentials**
2. Click on your OAuth client
3. Add production redirect URI:
   ```
   https://yourdomain.com/auth/google/callback
   ```
4. Save changes

### Update Environment Variables

```bash
GOOGLE_OAUTH_REDIRECT_URI=https://yourdomain.com/auth/google/callback
```

### Enable HTTPS

Google OAuth requires HTTPS for production:

```bash
# Example with Let's Encrypt
certbot certonly -d yourdomain.com
```

### Update Backend Configuration

Ensure your backend is accessible at the registered domain:

```bash
# .env production
GOOGLE_OAUTH_REDIRECT_URI=https://yourdomain.com/auth/google/callback
SERVER_PORT=443  # HTTPS port
```

---

## 🐛 Troubleshooting

### Issue: `redirect_uri_mismatch`

**Cause:** Redirect URI in request doesn't match registered URI

**Solution:**

1. Check `.env` file has correct `GOOGLE_OAUTH_REDIRECT_URI`
2. Verify it matches exactly in Google Cloud Console
3. No trailing slashes or extra parameters

Example:

```bash
# ✅ Correct
http://localhost:8080/auth/google/callback

# ❌ Wrong
http://localhost:8080/auth/google/callback/
http://localhost:8080/auth/google/callback?state=x
```

### Issue: `invalid_client`

**Cause:** Client ID or secret is incorrect

**Solution:**

1. Go to [Google Cloud Console](https://console.cloud.google.com) → **Credentials**
2. Delete old credential
3. Create new OAuth 2.0 credential
4. Copy exact ID and secret again
5. Update `.env`

### Issue: `access_blocked`

**Cause:** OAuth consent screen not properly configured

**Solution:**

1. Go to **APIs & Services** → **OAuth consent screen**
2. Ensure it's configured as "External"
3. Fill all required fields
4. Add test users if still in development
5. Try again

### Issue: `scope_mismatch`

**Cause:** Requesting scopes not configured in consent screen

**Solution:**

1. Go to **APIs & Services** → **OAuth consent screen**
2. Add scopes under "Scopes"
3. Include all scopes from [internal/auth/google/oauth.go](../internal/auth/google/oauth.go)

### Issue: API not enabled

**Cause:** Google+ API not enabled in project

**Solution:**

1. Go to **APIs & Services** → **Library**
2. Search `Google+ API` or `OAuth`
3. Click result and press **Enable**

---

## 🔍 Debugging Tips

### Enable Debug Logging

Update [internal/auth/google/handler.go](../internal/auth/google/handler.go):

```go
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
    codeStr := r.URL.Query().Get("code")
    log.Printf("DEBUG: code=%s", codeStr)  // Add this

    // ... rest of code

    log.Printf("DEBUG: token=%v", token)  // Add this
}
```

### Check Token Validity

```bash
# Decode JWT at jwt.io to inspect claims
# https://jwt.io/

# Copy your token and paste in "Encoded" section
# View payload to verify claims
```

### Test Manually

```bash
# Step 1: Get authorization code manually
# Open in browser and allow access
curl "https://accounts.google.com/o/oauth2/v2/auth?client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:8080/auth/google/callback&response_type=code&scope=email%20profile"

# Step 2: Exchange code for token
curl -X POST https://oauth2.googleapis.com/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "code=YOUR_CODE&client_id=YOUR_CLIENT_ID&client_secret=YOUR_SECRET&redirect_uri=http://localhost:8080/auth/google/callback&grant_type=authorization_code"
```

---

## 📚 Useful Resources

- [Google OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [Google Cloud Console](https://console.cloud.google.com)
- [JWT.io - Inspect tokens](https://jwt.io/)
- [OAuth 2.0 Flow Diagram](https://tools.ietf.org/html/rfc6749#section-1.3.1)

---

## ✅ Checklist

- [ ] Google Cloud Project created
- [ ] Google+ API enabled
- [ ] OAuth consent screen configured
- [ ] OAuth 2.0 credentials created
- [ ] Client ID copied to `.env`
- [ ] Client Secret copied to `.env`
- [ ] Redirect URI matches in both places
- [ ] Backend started successfully
- [ ] Can access `/auth/google` without errors
- [ ] Google login redirects properly
- [ ] Callback receives tokens

---

_Last Updated: January 7, 2026_
